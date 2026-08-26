package runtimecli

import (
	"io"

	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
)

// WriteSupervisorStatusJSON keeps the typed runtime output available at the
// CLI boundary without duplicating protobuf mapping logic.
func WriteSupervisorStatusJSON(w io.Writer, report runtimesupervisor.StatusReport) error {
	return runtimeapp.WriteSupervisorStatusJSON(w, report)
}

// WriteSupervisorServiceResultJSON keeps install/uninstall output on the same
// typed contract as the application layer.
func WriteSupervisorServiceResultJSON(w io.Writer, result runtimesupervisor.ServiceInstallResult) error {
	return runtimeapp.WriteSupervisorServiceResultJSON(w, result)
}
