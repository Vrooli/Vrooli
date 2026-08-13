package control

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/vrooli/cli-core/cliapp"
)

func AuthGroup(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	return cliapp.SubcommandGroup{Name: "auth", Description: "Manage reference-only device authentication profiles", NeedsAPI: true, Subcommands: []cliapp.Command{
		command("list", "List authentication profiles without secret values", cliapp.ArgSchema{}, func(ctx cliapp.RunContext) error {
			b, err := core.Request(http.MethodGet, "/auth/profiles", nil, nil)
			if err != nil {
				return err
			}
			return emit(ctx, b, "Authentication profiles")
		}),
		command("create", "Create a reference-only authentication profile", cliapp.ArgSchema{Flags: []cliapp.Flag{
			{Name: "device", Required: true}, {Name: "method", Required: true}, {Name: "credential-identity", Required: true}, {Name: "credential-field", Required: true},
			{Name: "verification", Default: "fresh_lock_state_unlocked"}, {Name: "max-attempts", Default: "1"}, {Name: "attempt-limit-ms", Default: "15000"}, {Name: "settle-ms", Default: "750"}, {Name: "actor", Default: "cli"},
		}}, func(ctx cliapp.RunContext) error {
			maxAttempts, err := strconv.Atoi(ctx.Flag("max-attempts"))
			if err != nil {
				return fmt.Errorf("max-attempts must be an integer")
			}
			attemptLimit, err := strconv.Atoi(ctx.Flag("attempt-limit-ms"))
			if err != nil {
				return fmt.Errorf("attempt-limit-ms must be an integer")
			}
			settle, err := strconv.Atoi(ctx.Flag("settle-ms"))
			if err != nil {
				return fmt.Errorf("settle-ms must be an integer")
			}
			body := map[string]any{"actor": ctx.Flag("actor"), "profile": map[string]any{
				"device_id": ctx.Flag("device"), "method": ctx.Flag("method"), "credential_identity": ctx.Flag("credential-identity"), "credential_field": ctx.Flag("credential-field"), "verification": ctx.Flag("verification"),
				"policy": map[string]any{"max_attempts": maxAttempts, "attempt_limit": attemptLimit * 1000000, "settle": settle * 1000000},
			}}
			return post(ctx, core, "/auth/profiles", body, "Authentication profile created")
		}),
		command("get", "Inspect a profile and provider status", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}}, func(ctx cliapp.RunContext) error {
			b, err := core.Request(http.MethodGet, "/auth/profiles/"+ctx.Positional("id"), nil, nil)
			if err != nil {
				return err
			}
			return emit(ctx, b, "Authentication profile")
		}),
		command("update", "Update reference-only authentication profile metadata", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}, Flags: []cliapp.Flag{
			{Name: "device"}, {Name: "method"}, {Name: "credential-identity"}, {Name: "credential-field"}, {Name: "verification"}, {Name: "max-attempts"}, {Name: "attempt-limit-ms"}, {Name: "settle-ms"}, {Name: "actor", Default: "cli"},
		}}, func(ctx cliapp.RunContext) error {
			profile := map[string]any{}
			for _, item := range []struct{ flag, key string }{{"device", "device_id"}, {"method", "method"}, {"credential-identity", "credential_identity"}, {"credential-field", "credential_field"}, {"verification", "verification"}} {
				if value := ctx.Flag(item.flag); value != "" {
					profile[item.key] = value
				}
			}
			policy := map[string]any{}
			for _, item := range []struct{ flag, key string }{{"max-attempts", "max_attempts"}, {"attempt-limit-ms", "attempt_limit"}, {"settle-ms", "settle"}} {
				if value := ctx.Flag(item.flag); value != "" {
					parsed, err := strconv.Atoi(value)
					if err != nil {
						return fmt.Errorf("%s must be an integer", item.flag)
					}
					if item.key == "attempt_limit" || item.key == "settle" {
						policy[item.key] = parsed * 1000000
					} else {
						policy[item.key] = parsed
					}
				}
			}
			if len(policy) > 0 {
				profile["policy"] = policy
			}
			return put(ctx, core, "/auth/profiles/"+ctx.Positional("id"), map[string]any{"actor": ctx.Flag("actor"), "profile": profile}, "Authentication profile updated")
		}),
		command("test", "Check provider/profile readiness without attempting unlock", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}}, func(ctx cliapp.RunContext) error {
			b, err := core.Request(http.MethodPost, "/auth/profiles/"+ctx.Positional("id")+"/test", nil, nil)
			if err != nil {
				return err
			}
			return emit(ctx, b, "Authentication profile test")
		}),
		command("revoke", "Revoke an authentication profile", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}, Flags: []cliapp.Flag{{Name: "actor", Default: "cli"}}}, func(ctx cliapp.RunContext) error {
			b, err := core.Request(http.MethodDelete, "/auth/profiles/"+ctx.Positional("id"), nil, nil)
			if err != nil {
				return err
			}
			return emit(ctx, b, "Authentication profile revoked")
		}),
		command("provision", "Provision the credential from stdin into the credential authority", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}}, func(ctx cliapp.RunContext) error {
			value, err := io.ReadAll(io.LimitReader(os.Stdin, 4097))
			if err != nil {
				return fmt.Errorf("read credential from stdin: %w", err)
			}
			defer func() {
				for i := range value {
					value[i] = 0
				}
			}()
			b, err := core.Request(http.MethodPost, "/auth/profiles/"+ctx.Positional("id")+"/provision", nil, value)
			if err != nil {
				return err
			}
			return emit(ctx, b, "Credential provisioned")
		}),
		command("delete-credential", "Delete the authority-held credential for a profile", cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id", Required: true}}, Flags: []cliapp.Flag{{Name: "actor", Default: "cli"}}}, func(ctx cliapp.RunContext) error {
			b, err := core.Request(http.MethodDelete, "/auth/profiles/"+ctx.Positional("id")+"/credential", nil, nil)
			if err != nil {
				return err
			}
			return emit(ctx, b, "Credential deleted")
		}),
		command("unlock", "Unlock a locked device through a held lease", cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "profile", Required: true}, {Name: "device", Required: true}, {Name: "lease", Required: true}, {Name: "actor", Default: "cli"}}}, func(ctx cliapp.RunContext) error {
			return post(ctx, core, "/auth/unlock", map[string]string{"profile_id": ctx.Flag("profile"), "device_id": ctx.Flag("device"), "lease_token": ctx.Flag("lease"), "actor": ctx.Flag("actor")}, "Device unlock")
		}),
	}}
}
