package exp

// func ToJSONWithUnknown(obj any, r *Retain) ([]byte, error) {
// 	var fields []reflect.StructField
// 	rt := reflect.TypeOf(obj)
// 	for i := 0; i < rt.NumField(); i++ {
// 		fields = append(fields, rt.Field(i))
// 	}
// 	newTyp := reflect.StructOf(fields)

// 	noMarshal := reflect.ValueOf(obj).Convert(newTyp).Interface()

// 	bytes, err := json.Marshal(noMarshal)
// 	if err != nil {
// 		return nil, err
// 	}

// 	l := len(bytes)
// 	bytes[l-1] =

// 		reflect.New(newTyp)

// 	type copyT T
// 	copyT()

// }
