package credentials

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/api-core/trustposture"
	"github.com/vrooli/vrooli/internal/cliout"
)

const (
	breakglassParameterA = 1024
)

const (
	breakglassParameterB = 64
)

//nolint:gocyclo // break-glass execution keeps authorization, audit, input, and cleanup branches visible.
func (app *Service) BreakGlass(ctx context.Context, out io.Writer, opts BreakGlassOptions, input io.Reader) error {
	operation := strings.TrimSpace(opts.Operation)
	paths, err := trustposture.ResolveKeyPaths()
	if err != nil {
		return err
	}
	switch operation {
	case "status":
		return runBreakGlassStatus(out, opts, paths)
	case "provision":
		passphrase, err := readBreakGlassPassphrase(input, "provision")
		if err != nil {
			return err
		}
		target, err := requiredBreakGlassTarget(opts.Target)
		if err != nil {
			return err
		}
		scopes, err := parseBreakGlassScopes(opts.Scopes)
		if err != nil {
			return err
		}
		if err := trustposture.ProvisionWrapped(paths, passphrase, opts.AccountID, breakGlassPurpose(opts), target, scopes, time.Now().UTC()); err != nil {
			return err
		}
		_, err = fmt.Fprintln(out, "Break-glass material provisioned. The private key is encrypted and was not printed.")
		return err
	case "issue":
		passphrase, err := readBreakGlassPassphrase(input, "issue")
		if err != nil {
			return err
		}
		target, err := requiredBreakGlassTarget(opts.Target)
		if err != nil {
			return err
		}
		scopes, err := parseBreakGlassScopes(opts.Scopes)
		if err != nil {
			return err
		}
		ttl := tuning.CopyRetentionWindow()
		if opts.TTL > 0 {
			ttl = opts.TTL
			if ttl > time.Hour {
				return fmt.Errorf("break-glass issue: --ttl must be between 1ns and 1h")
			}
		}
		now := time.Now().UTC()
		binding := trustposture.BreakGlassBinding{
			OperatorID: opts.OperatorID, MachineID: opts.MachineID, NodeID: opts.NodeID,
			Scope: opts.Scope, PlanHash: opts.PlanHash, OperationID: opts.OperationID,
		}
		var token string
		if bindingComplete(binding) {
			token, err = trustposture.IssueFromWrappedProvisionBound(paths, passphrase, breakGlassPurpose(opts), target, scopes, binding, now, ttl)
		} else if bindingPresent(binding) {
			return fmt.Errorf("break-glass issue: incomplete cleanup binding")
		} else {
			token, err = trustposture.IssueFromWrappedProvision(paths, passphrase, breakGlassPurpose(opts), target, scopes, now, ttl)
		}
		if err != nil {
			return err
		}
		if err := trustposture.WriteCredential(paths, token); err != nil {
			return err
		}
		return renderBreakGlassCredential(out, opts.Format, paths.Credential, now.Add(ttl))
	case "rotate":
		return runBreakGlassRotate(out, input, paths)
	case "reset":
		return runBreakGlassReset(out, opts, paths)
	default:
		return fmt.Errorf("break-glass: unknown operation %q; choose provision, issue, rotate, or status", operation)
	}
}

func runBreakGlassStatus(out io.Writer, opts BreakGlassOptions, paths trustposture.KeyPaths) error {
	if breakGlassHasAnyFlag(opts) {
		return fmt.Errorf("break-glass status: status accepts no provisioning or issuance flags")
	}
	return renderBreakGlassStatus(out, opts.Format, paths)
}

func runBreakGlassRotate(out io.Writer, input io.Reader, paths trustposture.KeyPaths) error {
	passphrase, err := readBreakGlassPassphrase(input, "rotate")
	if err != nil {
		return err
	}
	if err := trustposture.RotateWrapped(paths, passphrase, time.Now().UTC()); err != nil {
		return err
	}
	_, err = fmt.Fprintln(out, "Break-glass material rotated. The private key is encrypted and was not printed.")
	return err
}

func runBreakGlassReset(out io.Writer, opts BreakGlassOptions, paths trustposture.KeyPaths) error {
	if breakGlassHasAnyFlag(opts) {
		return fmt.Errorf("break-glass reset: reset accepts no provisioning or issuance flags")
	}
	if err := trustposture.ResetWrapped(paths); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, "Break-glass material retired. No replacement credential was created.")
	return err
}

func bindingPresent(binding trustposture.BreakGlassBinding) bool {
	return strings.TrimSpace(binding.OperatorID) != "" || strings.TrimSpace(binding.MachineID) != "" || strings.TrimSpace(binding.NodeID) != "" || strings.TrimSpace(binding.Scope) != "" || strings.TrimSpace(binding.PlanHash) != "" || strings.TrimSpace(binding.OperationID) != ""
}

func bindingComplete(binding trustposture.BreakGlassBinding) bool {
	return strings.TrimSpace(binding.OperatorID) != "" && strings.TrimSpace(binding.MachineID) != "" && strings.TrimSpace(binding.NodeID) != "" && strings.TrimSpace(binding.Scope) != "" && strings.TrimSpace(binding.PlanHash) != "" && strings.TrimSpace(binding.OperationID) != ""
}

func breakGlassPurpose(opts BreakGlassOptions) string {
	if purpose := strings.TrimSpace(opts.Purpose); purpose != "" {
		return purpose
	}
	return strings.TrimSpace(opts.Audience)
}

func breakGlassHasAnyFlag(opts BreakGlassOptions) bool {
	return strings.TrimSpace(opts.AccountID) != "" || strings.TrimSpace(opts.Audience) != "" || strings.TrimSpace(opts.Purpose) != "" || strings.TrimSpace(opts.Target) != "" || strings.TrimSpace(opts.Scopes) != "" || strings.TrimSpace(opts.Scope) != "" || strings.TrimSpace(opts.OperatorID) != "" || strings.TrimSpace(opts.MachineID) != "" || strings.TrimSpace(opts.NodeID) != "" || strings.TrimSpace(opts.PlanHash) != "" || strings.TrimSpace(opts.OperationID) != "" || opts.TTL > 0
}

func readBreakGlassPassphrase(input io.Reader, operation string) (string, error) {
	if err := refuseInteractiveStdin(input, fmt.Sprintf("printf '%%s' \"$PASSPHRASE\" | vrooli break-glass %s ...", operation)); err != nil {
		return "", err
	}
	raw, err := io.ReadAll(io.LimitReader(input, breakglassParameterB*breakglassParameterA))
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

func renderBreakGlassStatus(out io.Writer, format string, paths trustposture.KeyPaths) error {
	status, err := trustposture.Status(paths)
	if err != nil {
		return err
	}
	if strings.TrimSpace(format) == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, status)
	}
	_, err = fmt.Fprintf(out, "Break-glass material: %s (wrapped private=%t, public=%t, metadata=%t) account=%s audience=%s target=%s scopes=%s provisioned=%d\n", map[bool]string{true: "ready", false: "incomplete"}[status.Complete], status.WrappedPrivate, status.Public, status.Metadata, status.AccountID, status.Audience, status.Target, strings.Join(status.Scopes, ","), status.ProvisionedAt)
	return err
}

type breakGlassCredentialOutput struct {
	Path      string `json:"path"`
	ExpiresAt string `json:"expires_at"`
}

func renderBreakGlassCredential(out io.Writer, format string, path string, expiresAt time.Time) error {
	output := breakGlassCredentialOutput{Path: path, ExpiresAt: expiresAt.UTC().Format(time.RFC3339)}
	if strings.TrimSpace(format) == string(cliout.FormatJSON) {
		return cliout.WriteJSONValue(out, output)
	}
	_, err := fmt.Fprintf(out, "Break-glass credential written to %s\nExpires at: %s\n", output.Path, output.ExpiresAt)
	return err
}
