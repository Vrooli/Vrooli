package vroolicli

import (
	"google.golang.org/protobuf/proto"

	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func cliVersionMessage(out versionOutput) *cliv1.CliVersion {
	return &cliv1.CliVersion{CliVersion: out.CLIVersion, PlatformVersion: out.PlatformVersion, Root: out.Root}
}

var _ proto.Message = (*cliv1.CliVersion)(nil)
