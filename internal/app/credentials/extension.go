package credentials

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/credentialextension"
)

// Extensions installs, removes, or inspects the public browser registration
// for the Secrets Manager native host.
func (app *Service) Extensions(ctx context.Context, out io.Writer, opts ExtensionOptions) error {
	_ = ctx
	format := strings.TrimSpace(opts.Format)
	if format == "" {
		format = string(cliout.FormatHuman)
	}
	if format != string(cliout.FormatHuman) && format != string(cliout.FormatJSON) {
		return fmt.Errorf("credentials extensions accepts only --format text|json")
	}
	registration := credentialextension.Options{
		HostPath: opts.HostPath, ExtensionID: opts.ExtensionID, Browser: opts.Browser,
	}
	var (
		result credentialextension.Result
		err    error
	)
	switch strings.TrimSpace(opts.Operation) {
	case "install":
		result, err = credentialextension.Install(registration)
	case "uninstall":
		if !opts.Yes {
			return fmt.Errorf("credentials extensions uninstall requires explicit --yes confirmation")
		}
		result, err = credentialextension.Uninstall(registration)
	case "status":
		result, err = credentialextension.Status(registration)
	default:
		return fmt.Errorf("credentials extensions: unknown operation %q", opts.Operation)
	}
	if err != nil {
		return err
	}
	if format == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, result)
	}
	fmt.Fprintf(out, "Native host registration: %s\nHost: %s\nExtension: %s\n", result.Operation, result.HostPath, result.ExtensionID)
	for _, target := range result.Targets {
		state := "not installed"
		if target.ManifestInstalled && (target.RegistryKey == "" || target.RegistryConfigured) {
			state = "installed"
		}
		fmt.Fprintf(out, "  %s: %s (%s)\n", target.Browser, state, target.ManifestPath)
	}
	return nil
}
