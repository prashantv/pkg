package jsonobj

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T {
	return &v
}

func mustMarshal(t testing.TB, v any) string {
	marshalled, err := json.Marshal(v)
	require.NoError(t, err)
	return string(marshalled)
}
