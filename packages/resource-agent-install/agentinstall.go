// Package agentinstall is the root-module bridge for nested resource CLI
// modules. The implementation lives in packages/harness-lifecycle;
// standalone resource modules cannot import Go internal packages directly.
package agentinstall

import (
	"context"

	"github.com/vrooli/cli-core/cliapp"
	internal "github.com/vrooli/vrooli/packages/harness-lifecycle"
)

// AntigravityManifestBase is the auto-updater service Antigravity releases through.
const AntigravityManifestBase = internal.AntigravityManifestBase

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

func Update(ctx context.Context, spec Spec) error { return internal.Update(ctx, spec) }

// Uninstall removes only receipt-owned lifecycle copies. It leaves upstream
// copies discovered on PATH untouched.
func Uninstall(ctx context.Context, root, resource string) error {
	return internal.Uninstall(ctx, root, resource)
}

func DirectInstallCommand(spec Spec) cliapp.Command {
	return internal.DirectInstallCommand(spec)
}

func UpdateCommand(spec Spec) cliapp.Command {
	return internal.UpdateCommand(spec)
}
