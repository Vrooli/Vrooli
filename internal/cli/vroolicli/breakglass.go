package vroolicli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/trustposture"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

const breakGlassHelpText = `vrooli break-glass - Provision and issue a local, target-bound credential

Usage:
  vrooli break-glass provision --account-id <id> --audience <purpose> --target <hostname> --scopes <scope,...> < passphrase
  vrooli break-glass issue --purpose <purpose> --target <hostname> --scopes <scope,...> [--ttl 15m] < passphrase
  vrooli break-glass rotate < passphrase
  vrooli break-glass reset
  vrooli break-glass status

Passphrases are read only from standard input. Provision and issue refuse a
terminal standard input so a forgotten command cannot silently wait for a
secret. The private key is stored only in an authenticated encrypted envelope.
`

func breakGlassArgSchema() commandtree.ArgSchema {
	return commandtree.ArgSchema{Options: []commandtree.OptionArg{
		{Name: "--account-id", ValueName: "id", Description: "Stable local account/principal identifier (provision only)"},
		{Name: "--audience", ValueName: "purpose", Description: "Credential purpose/audience"},
		{Name: "--purpose", ValueName: "purpose", Description: "Alias for --audience (credential purpose)"},
		{Name: "--target", ValueName: "hostname", Description: "Exact local machine target claim"},
		{Name: "--scopes", ValueName: "scope,...", Description: "Comma-separated scope ceiling or requested scopes"},
		{Name: "--scope", ValueName: "scope", Description: "Cleanup scope binding (apply capability only)"},
		{Name: "--operator-id", ValueName: "id", Description: "Cleanup operator identity binding"},
		{Name: "--machine-id", ValueName: "id", Description: "Cleanup machine identity binding"},
		{Name: "--node-id", ValueName: "id", Description: "Cleanup node identity binding"},
		{Name: "--plan-hash", ValueName: "hash", Description: "Frozen cleanup plan hash binding"},
		{Name: "--operation-id", ValueName: "id", Description: "Cleanup operation identity binding"},
		{Name: "--ttl", ValueName: "duration", Description: "Credential lifetime for issue (default 15m)"},
	}}
}

func (app *App) runBreakGlassCommand(ctx *CommandContext, args []string) error {
	return app.runBreakGlassCommandWithInput(ctx, args, os.Stdin)
}

func (app *App) runBreakGlassCommandWithInput(ctx *CommandContext, args []string, input io.Reader) error {
	if len(args) == 0 || commandtree.WantsHelp(args) {
		fmt.Fprint(ctx.Stdout, breakGlassHelpText)
		return nil
	}
	operation := strings.TrimSpace(args[0])
	parsed, err := commandtree.ParseArgs("break-glass "+operation, breakGlassHelpText, breakGlassArgSchema(), args[1:])
	if err != nil {
		if rootcli.HandleHelp(ctx.Stdout, err) {
			return nil
		}
		return rootcli.UsageErrorf("break-glass", "%s", err.Error())
	}
	paths, err := trustposture.ResolveKeyPaths()
	if err != nil {
		return err
	}
	switch operation {
	case "status":
		if breakGlassHasAnyFlag(parsed, "--account-id", "--audience", "--purpose", "--target", "--scopes", "--scope", "--operator-id", "--machine-id", "--node-id", "--plan-hash", "--operation-id", "--ttl") {
			return rootcli.UsageErrorf("break-glass status", "status accepts no provisioning or issuance flags")
		}
		return renderBreakGlassStatus(ctx, paths)
	case "provision":
		passphrase, err := readBreakGlassPassphrase(input, "provision")
		if err != nil {
			return err
		}
		target, err := requiredBreakGlassTarget(parsed.FlagValue("--target"))
		if err != nil {
			return err
		}
		scopes, err := parseBreakGlassScopes(parsed.FlagValue("--scopes"))
		if err != nil {
			return err
		}
		if err := trustposture.ProvisionWrapped(paths, passphrase, parsed.FlagValue("--account-id"), breakGlassPurpose(parsed), target, scopes, time.Now().UTC()); err != nil {
			return err
		}
		_, err = fmt.Fprintln(ctx.Stdout, "Break-glass material provisioned. The private key is encrypted and was not printed.")
		return err
	case "issue":
		passphrase, err := readBreakGlassPassphrase(input, "issue")
		if err != nil {
			return err
		}
		target, err := requiredBreakGlassTarget(parsed.FlagValue("--target"))
		if err != nil {
			return err
		}
		scopes, err := parseBreakGlassScopes(parsed.FlagValue("--scopes"))
		if err != nil {
			return err
		}
		ttl := 15 * time.Minute
		if raw := strings.TrimSpace(parsed.FlagValue("--ttl")); raw != "" {
			ttl, err = time.ParseDuration(raw)
			if err != nil || ttl <= 0 || ttl > time.Hour {
				return fmt.Errorf("break-glass issue: --ttl must be between 1ns and 1h")
			}
		}
		now := time.Now().UTC()
		binding := trustposture.BreakGlassBinding{
			OperatorID: parsed.FlagValue("--operator-id"), MachineID: parsed.FlagValue("--machine-id"), NodeID: parsed.FlagValue("--node-id"),
			Scope: parsed.FlagValue("--scope"), PlanHash: parsed.FlagValue("--plan-hash"), OperationID: parsed.FlagValue("--operation-id"),
		}
		var token string
		if bindingComplete(binding) {
			token, err = trustposture.IssueFromWrappedProvisionBound(paths, passphrase, breakGlassPurpose(parsed), target, scopes, binding, now, ttl)
		} else if bindingPresent(binding) {
			return fmt.Errorf("break-glass issue: incomplete cleanup binding")
		} else {
			token, err = trustposture.IssueFromWrappedProvision(paths, passphrase, breakGlassPurpose(parsed), target, scopes, now, ttl)
		}
		if err != nil {
			return err
		}
		if err := trustposture.WriteCredential(paths, token); err != nil {
			return err
		}
		return renderBreakGlassCredential(ctx, paths.Credential, now.Add(ttl))
	case "rotate":
		passphrase, err := readBreakGlassPassphrase(input, "rotate")
		if err != nil {
			return err
		}
		if err := trustposture.RotateWrapped(paths, passphrase, time.Now().UTC()); err != nil {
			return err
		}
		_, err = fmt.Fprintln(ctx.Stdout, "Break-glass material rotated. The private key is encrypted and was not printed.")
		return err
	case "reset":
		if breakGlassHasAnyFlag(parsed, "--account-id", "--audience", "--purpose", "--target", "--scopes", "--scope", "--operator-id", "--machine-id", "--node-id", "--plan-hash", "--operation-id", "--ttl") {
			return rootcli.UsageErrorf("break-glass reset", "reset accepts no provisioning or issuance flags")
		}
		if err := trustposture.ResetWrapped(paths); err != nil {
			return err
		}
		_, err = fmt.Fprintln(ctx.Stdout, "Break-glass material retired. No replacement credential was created.")
		return err
	default:
		return rootcli.UsageErrorf("break-glass", "unknown operation %q; choose provision, issue, rotate, or status", operation)
	}
}

func bindingPresent(binding trustposture.BreakGlassBinding) bool {
	return strings.TrimSpace(binding.OperatorID) != "" || strings.TrimSpace(binding.MachineID) != "" || strings.TrimSpace(binding.NodeID) != "" || strings.TrimSpace(binding.Scope) != "" || strings.TrimSpace(binding.PlanHash) != "" || strings.TrimSpace(binding.OperationID) != ""
}

func bindingComplete(binding trustposture.BreakGlassBinding) bool {
	return strings.TrimSpace(binding.OperatorID) != "" && strings.TrimSpace(binding.MachineID) != "" && strings.TrimSpace(binding.NodeID) != "" && strings.TrimSpace(binding.Scope) != "" && strings.TrimSpace(binding.PlanHash) != "" && strings.TrimSpace(binding.OperationID) != ""
}

func breakGlassPurpose(parsed commandtree.ParsedArgs) string {
	if purpose := strings.TrimSpace(parsed.FlagValue("--purpose")); purpose != "" {
		return purpose
	}
	return strings.TrimSpace(parsed.FlagValue("--audience"))
}

func breakGlassHasAnyFlag(parsed commandtree.ParsedArgs, names ...string) bool {
	for _, name := range names {
		if parsed.HasFlag(name) {
			return true
		}
	}
	return false
}

func readBreakGlassPassphrase(input io.Reader, operation string) (string, error) {
	if err := refuseInteractiveStdin(input, fmt.Sprintf("printf '%%s' \"$PASSPHRASE\" | vrooli break-glass %s ...", operation)); err != nil {
		return "", err
	}
	raw, err := io.ReadAll(io.LimitReader(input, 64*1024))
	if err != nil {
		return "", fmt.Errorf("read break-glass passphrase: %w", err)
	}
	passphrase := strings.TrimSpace(string(raw))
	if passphrase == "" {
		return "", fmt.Errorf("break-glass passphrase is required on standard input")
	}
	return passphrase, nil
}

func requiredBreakGlassTarget(raw string) (string, error) {
	target := strings.TrimSpace(raw)
	if target == "" {
		return "", fmt.Errorf("break-glass target is required; use the local hostname")
	}
	return target, nil
}

func parseBreakGlassScopes(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	if len(parts) == 1 && strings.TrimSpace(parts[0]) == "" {
		return nil, fmt.Errorf("break-glass scopes are required")
	}
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		scope := strings.TrimSpace(part)
		if scope == "" {
			return nil, fmt.Errorf("break-glass scopes cannot contain an empty value")
		}
		result = append(result, scope)
	}
	return result, nil
}

func renderBreakGlassStatus(ctx *CommandContext, paths trustposture.KeyPaths) error {
	status, err := trustposture.Status(paths)
	if err != nil {
		return err
	}
	if ctx.Globals.JSON {
		return json.NewEncoder(ctx.Stdout).Encode(status)
	}
	_, err = fmt.Fprintf(ctx.Stdout, "Break-glass material: %s (wrapped private=%t, public=%t, metadata=%t) account=%s audience=%s target=%s scopes=%s provisioned=%d\n", map[bool]string{true: "ready", false: "incomplete"}[status.Complete], status.WrappedPrivate, status.Public, status.Metadata, status.AccountID, status.Audience, status.Target, strings.Join(status.Scopes, ","), status.ProvisionedAt)
	return err
}

type breakGlassCredentialOutput struct {
	Path      string `json:"path"`
	ExpiresAt string `json:"expires_at"`
}

func renderBreakGlassCredential(ctx *CommandContext, path string, expiresAt time.Time) error {
	output := breakGlassCredentialOutput{Path: path, ExpiresAt: expiresAt.UTC().Format(time.RFC3339)}
	if ctx.Globals.JSON {
		return json.NewEncoder(ctx.Stdout).Encode(output)
	}
	_, err := fmt.Fprintf(ctx.Stdout, "Break-glass credential written to %s\nExpires at: %s\n", output.Path, output.ExpiresAt)
	return err
}
