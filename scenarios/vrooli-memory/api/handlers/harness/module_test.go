package harness

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtoFileDeclaresHarnessService(t *testing.T) {
	services := ProtoFile.Services()
	require.Equal(t, 1, services.Len())
	require.Equal(t, "HarnessService", string(services.Get(0).Name()))
}
