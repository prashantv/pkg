package jsonobj

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Retain preserves unknown fields when marshalling JSON.
// It should be added as an unexported field in a struct,
// with UnmarshalJSON / MarshalJSON methods that call
// FromJSON and ToJSON.
type Retain struct {
	raw map[string]json.RawMessage
}

// FromJSON should be called from obj.UnmarshalJSON where obj is the struct for
// which unknown fields should be retained.
func (r *Retain) FromJSON(data []byte, obj any) error {
	rv, ok := ensureStruct(obj, true /* requirePtr */)
	if !ok {
		return fmt.Errorf("FromJSON requires a struct pointer, got %T", obj)
	}

	if err := json.Unmarshal(data, &r.raw); err != nil {
		return err
	}

	if err := forJSONField(rv, func(t jsonTag, v reflect.Value) error {
		fieldJSON, ok := r.raw[t.name()]
		if !ok {
			return nil
		}

		delete(r.raw, t.name())
		return json.Unmarshal(fieldJSON, v.Addr().Interface())
	}); err != nil {
		return err
	}

	if len(r.raw) == 0 {
		r.raw = nil
	}

	return nil
}

// ToJSON should be called from obj.MarshalJSON where obj is the struct being
// marshalled with unknown fields (retained in FromJSON).
func (r *Retain) ToJSON(obj any) ([]byte, error) {
	rv, ok := ensureStruct(obj, false /* requirePtr */)
	if !ok {
		return nil, fmt.Errorf("ToJSON requires a struct, got %T", obj)
	}

	// Create a copy since we mutate the map below
	// and ToJSON should be safe for concurrent-use.
	all := make(map[string]any, len(r.raw))
	for k, v := range r.raw {
		all[k] = v
	}

	forJSONField(rv, func(t jsonTag, v reflect.Value) struct{} {
		if t.omitEmpty() && isZero(v) {
			return struct{}{}
		}

		all[t.name()] = v.Interface()
		return struct{}{}
	})

	return json.Marshal(all)
}

// MustRetainable panics if the passed in object is not Retainable.
//
// The return value is so it can be used in a var declaration such as:
//
//	var _ any = MustRetainable(&ObjType{})
func MustRetainable(obj interface {
	json.Marshaler
	json.Unmarshaler
}) any {
	if err := Retainable(obj); err != nil {
		panic(err)
	}
	return obj
}

// Retainable checks that the provided type is supported for Retain marshalling
// by checking for:
// * The type is a struct pointer (for `UnmarshalJSON` to work correctly).
// * The type has no duplicate JSON field names.
// * The type has no unsupported json tags.
func Retainable(obj interface {
	json.Marshaler
	json.Unmarshaler
}) (retErr error) {
	defer func() {
		if retErr != nil {
			retErr = fmt.Errorf("%T not Retainable: %v", obj, retErr)
		}
	}()

	rv, ok := ensureStruct(obj, true /* requirePtr */)
	if !ok {
		return errors.New("requires struct pointer")
	}

	if err := verifyNoDuplicateFieldNames(rv); err != nil {
		return err
	}

	if err := verifyNoUnsupportedTags(rv); err != nil {
		return err
	}

	return nil
}

func verifyNoDuplicateFieldNames(rv reflect.Value) error {
	exists := make(map[string]struct{})
	return forJSONField(rv, func(t jsonTag, v reflect.Value) error {
		name := t.name()
		if _, ok := exists[name]; ok {
			return fmt.Errorf("duplicate JSON field %q", name)
		}
		exists[name] = struct{}{}
		return nil
	})
}

func verifyNoUnsupportedTags(rv reflect.Value) error {
	return forJSONField(rv, func(jt jsonTag, v reflect.Value) error {
		if len(jt.tag) <= 1 {
			return nil
		}

		for _, t := range jt.tag[1:] {
			if t != "" && t != "omitempty" {
				return fmt.Errorf("field %q has unsupported tag %q", jt.name(), t)
			}
		}

		return nil
	})
}

func ensureStruct(obj any, requirePtr bool) (reflect.Value, bool) {
	rv := reflect.ValueOf(obj)
	if rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	} else if requirePtr {
		return rv, false
	}
	return rv, rv.Kind() == reflect.Struct
}

func forJSONField[R comparable](rv reflect.Value, fn func(t jsonTag, v reflect.Value) R) R {
	var zeroRet R
	rt := rv.Type()

	for f := 0; f < rt.NumField(); f++ {
		ft := rt.Field(f)
		if !ft.IsExported() {
			continue
		}

		tagValue := ft.Tag.Get("json")
		if tagValue == "-" {
			// json package ignores tag with "-"
			continue
		}

		jt := jsonTag{
			tag:   strings.Split(tagValue, ","),
			field: ft,
		}

		if ret := fn(jt, rv.Field(f)); ret != zeroRet {
			return ret
		}
	}

	return zeroRet
}

type jsonTag struct {
	tag   []string
	field reflect.StructField
}

func (t jsonTag) name() string {
	if name := t.tag[0]; name != "" {
		return name
	}
	return t.field.Name
}

func (t jsonTag) omitEmpty() bool {
	if len(t.tag) < 2 {
		return false
	}
	return t.tag[1] == "omitempty"
}

func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}
