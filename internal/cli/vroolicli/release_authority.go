package vroolicli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/credentialauthority"
	"github.com/vrooli/vrooli/internal/releaseauthority"
)

// runReleaseAuthorityCommand is the project-owned release trust control plane.
// Private key material never crosses this command boundary: it is generated
// inside Go and retained by the existing native credential authority.
func (app *App) runReleaseAuthorityCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		fmt.Fprintln(ctx.Stdout, "Usage:\n  vrooli release-authority init [--replace-trust-anchor]\n  vrooli release-authority status [--format text|json]\n  vrooli release-authority add-evidence --stage <release-directory> --source <file> --name <safe-file-name> --role <role> --provenance <source> [--os <os>] [--arch <arch>]\n  vrooli release-authority sign --stage <release-directory> [--overwrite]\n  vrooli release-authority regenerate --replace-trust-anchor\n\nThe private key is generated and retained in the native secure store. It is never written to a project file or printed. Adding evidence invalidates any prior signature; sign the completed stage explicitly. Regenerating replaces the release trust root and makes artifacts signed by the prior key unverifiable unless that prior public key remains trusted.")
		return nil
	}
	authority, err := app.releaseAuthority()
	if err != nil {
		return err
	}
	switch args[0] {
	case "init":
		return releaseAuthorityInit(ctx, authority, args[1:])
	case "status":
		return releaseAuthorityStatus(ctx, authority, args[1:])
	case "sign":
		return releaseAuthoritySign(ctx, authority, args[1:])
	case "add-evidence":
		return releaseAuthorityAddEvidence(ctx, authority, args[1:])
	case "regenerate":
		return releaseAuthorityRegenerate(ctx, authority, args[1:])
	default:
		return fmt.Errorf("unknown release-authority command %q", args[0])
	}
}

func releaseAuthorityAddEvidence(ctx *CommandContext, authority *releaseauthority.Authority, args []string) error {
	fs := flag.NewFlagSet("release-authority add-evidence", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stage, source, name, role, provenance, osName, arch := "", "", "", "", "", "", ""
	fs.StringVar(&stage, "stage", "", "staged release directory")
	fs.StringVar(&source, "source", "", "durable evidence file to stage")
	fs.StringVar(&name, "name", "", "safe file name to use in the stage")
	fs.StringVar(&role, "role", "", "evidence role, such as bas-run")
	fs.StringVar(&provenance, "provenance", "", "human-readable source provenance")
	fs.StringVar(&osName, "os", "", "target operating system")
	fs.StringVar(&arch, "arch", "", "target architecture")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || strings.TrimSpace(stage) == "" || strings.TrimSpace(source) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(role) == "" || strings.TrimSpace(provenance) == "" {
		return fmt.Errorf("release-authority add-evidence requires --stage, --source, --name, --role, and --provenance")
	}
	artifact, err := authority.AddEvidence(stage, source, name, role, osName, arch, provenance)
	if err != nil {
		return err
	}
	return cliout.WriteJSONValue(ctx.Stdout, artifact)
}

func (app *App) releaseAuthority() (*releaseauthority.Authority, error) {
	credentials, err := credentialauthority.DefaultAuthority()
	if err != nil {
		return nil, err
	}
	return releaseauthority.New(credentials)
}

func releaseAuthorityInit(ctx *CommandContext, authority *releaseauthority.Authority, args []string) error {
	fs := flag.NewFlagSet("release-authority init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	replace := false
	fs.BoolVar(&replace, "replace-trust-anchor", false, "explicitly replace a mismatched existing public trust anchor")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 {
		return fmt.Errorf("release-authority init accepts no positional arguments")
	}
	status, err := authority.Initialize(ctx.Root, replace)
	if err != nil {
		return err
	}
	return renderReleaseAuthorityStatus(ctx, status)
}

func releaseAuthorityStatus(ctx *CommandContext, authority *releaseauthority.Authority, args []string) error {
	fs := flag.NewFlagSet("release-authority status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	format := "text"
	fs.StringVar(&format, "format", "text", "output format: text or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || (format != "text" && format != "json") {
		return fmt.Errorf("release-authority status accepts only --format text or json")
	}
	status, err := authority.Status(ctx.Root)
	if err != nil {
		return err
	}
	if format == "json" || ctx.Globals.JSON {
		return cliout.WriteJSONValue(ctx.Stdout, status)
	}
	return renderReleaseAuthorityStatus(ctx, status)
}

func releaseAuthoritySign(ctx *CommandContext, authority *releaseauthority.Authority, args []string) error {
	fs := flag.NewFlagSet("release-authority sign", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	stage := ""
	overwrite := false
	fs.StringVar(&stage, "stage", "", "staged release directory containing release-manifest.json")
	fs.BoolVar(&overwrite, "overwrite", false, "replace an existing detached signature")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || strings.TrimSpace(stage) == "" {
		return fmt.Errorf("release-authority sign requires --stage")
	}
	envelope, err := authority.SignStage(ctx.Root, stage, overwrite)
	if err != nil {
		return err
	}
	if ctx.Globals.JSON {
		return cliout.WriteJSONValue(ctx.Stdout, envelope)
	}
	_, err = fmt.Fprintf(ctx.Stdout, "Signed release manifest with managed authority key %s.\n", envelope.KeyID)
	return err
}

func releaseAuthorityRegenerate(ctx *CommandContext, authority *releaseauthority.Authority, args []string) error {
	fs := flag.NewFlagSet("release-authority regenerate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	replace := false
	fs.BoolVar(&replace, "replace-trust-anchor", false, "acknowledge destructive release trust-root replacement")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || !replace {
		return fmt.Errorf("release-authority regenerate requires --replace-trust-anchor")
	}
	status, err := authority.Regenerate(ctx.Root)
	if err != nil {
		return err
	}
	return renderReleaseAuthorityStatus(ctx, status)
}

func renderReleaseAuthorityStatus(ctx *CommandContext, status releaseauthority.Status) error {
	if ctx.Globals.JSON {
		return cliout.WriteJSONValue(ctx.Stdout, status)
	}
	state := "uninitialized"
	if status.Configured {
		state = "configured"
	}
	_, err := fmt.Fprintf(ctx.Stdout, "Release authority: %s (provider: %s, key: %s, trust anchor match: %t)\n", state, status.Provider, status.KeyID, status.TrustAnchorMatch)
	return err
}
