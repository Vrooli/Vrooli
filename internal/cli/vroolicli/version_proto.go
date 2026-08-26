package vroolicli

import (
	"io"

	"google.golang.org/protobuf/proto"

	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func cliVersionMessage(out versionOutput) *cliv1.CliVersion {
	return &cliv1.CliVersion{CliVersion: out.CLIVersion, PlatformVersion: out.PlatformVersion, Root: out.Root}
}

func writeCliVersionJSON(w io.Writer, out versionOutput) error {
	return cliout.WriteProtoJSON(w, cliVersionMessage(out))
}

func writeCliSupervisorStatusJSON(w io.Writer, report runtimesupervisor.StatusReport) error {
	return runtimeapp.WriteSupervisorStatusJSON(w, report)
}

func writeCliSupervisorServiceResultJSON(w io.Writer, result runtimesupervisor.ServiceInstallResult) error {
	return runtimeapp.WriteSupervisorServiceResultJSON(w, result)
}

var _ proto.Message = (*cliv1.CliVersion)(nil)
