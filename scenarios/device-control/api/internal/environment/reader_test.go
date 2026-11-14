package environment

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapReaderProvidesDeterministicOptionalValues(t *testing.T) {
	reader := MapReader{"configured": " value "}

	value, ok := reader.LookupEnv("configured")
	require.True(t, ok)
	require.Equal(t, " value ", value)
	require.Empty(t, Value(reader, "missing"))
	require.Empty(t, Value(nil, "configured"))
}
