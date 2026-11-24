package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestD2SherpaRuntimeRootUsesRunbookDefaultAndExplicitOverride(t *testing.T) {
	for _, tt := range []struct {
		name string
		env  string
		want string
	}{
		{name: "documented local attach path", want: "/tmp/sherpa-root"},
		{name: "explicit runtime root", env: "/opt/sherpa/v1.13.2", want: "/opt/sherpa/v1.13.2"},
		{name: "blank override uses runbook path", env: "  ", want: "/tmp/sherpa-root"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := d2SherpaRuntimeRoot(func(key string) string {
				require.Equal(t, "SHERPA_ONNX_ROOT", key)
				return tt.env
			})
			require.Equal(t, tt.want, got)
		})
	}
}
