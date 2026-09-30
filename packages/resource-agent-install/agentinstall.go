// Package agentinstall is the root-module bridge for nested resource CLI
// modules. The implementation lives in internal/resources/agentinstall;
// standalone resource modules cannot import Go internal packages directly.
package agentinstall

import (
	"context"

	"github.com/vrooli/cli-core/cliapp"
	internal "github.com/vrooli/vrooli/internal/resources/agentinstall"
)

type (
	Spec                     = internal.Spec
	StatusArgs               = internal.StatusArgs
	UnsupportedPlatformError = internal.UnsupportedPlatformError
)

var (
	ParseStatusArgs       = internal.ParseStatusArgs
	WarnIfShadowed        = internal.WarnIfShadowed
	BlockingSystemInstall = internal.BlockingSystemInstall
	InstalledVersion      = internal.InstalledVersion
	ResolveURL            = internal.ResolveURL
)

func Install(ctx context.Context, spec Spec) error { return internal.Install(ctx, spec) }

func DirectInstallCommand(spec Spec) cliapp.Command {
	return internal.DirectInstallCommand(spec)
}
