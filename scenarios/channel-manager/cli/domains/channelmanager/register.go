package channelmanager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/vrooli/cli-core/cliapp"
)

// Register exposes the permanent manual executor floor. It uses the scenario
// API exclusively; secrets remain Vault references and are never CLI flags.
func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	request := func(ctx cliapp.RunContext, method, path string, body any, result string) error {
		bytes, err := ctx.Core().Request(method, path, url.Values{}, body)
		if err != nil {
			return fmt.Errorf("channel-manager API: %w", err)
		}
		var decoded any
		if err := json.Unmarshal(bytes, &decoded); err != nil {
			return err
		}
		return ctx.RenderMutation(cliapp.MutationReport{Result: []string{result}, Changes: []string{string(bytes)}})
	}
	return cliapp.SubcommandGroup{Name: "channel", Description: "Manage platform identities and manual warming actions", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "overview", Description: "Show identities, due actions, and flags", NeedsAPI: true, RunCtx: func(ctx cliapp.RunContext) error {
			bytes, err := ctx.Core().Get("/channel-manager/overview", url.Values{})
			if err != nil {
				return err
			}
			return ctx.RenderList(cliapp.ListReport{Summary: []string{"Channel Manager overview"}, ResultsHeading: "State", Results: []string{string(bytes)}})
		}},
		{Name: "create", Description: "Create an identity with a Vault reference", NeedsAPI: true, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "id", Required: true, Description: "Stable identity id"}, {Name: "platform", Required: true, Description: "Platform descriptor id"}, {Name: "purpose", Required: true, Description: "brand or persona-actor"}, {Name: "environment", Required: true, Description: "Attested environment reference"}, {Name: "vault-ref", Required: true, Description: "Vault path only; never a secret"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			return request(ctx, http.MethodPost, "/channel-manager/identities", map[string]any{"id": ctx.Flag("id"), "platform_id": ctx.Flag("platform"), "purpose": ctx.Flag("purpose"), "environment_ref": ctx.Flag("environment"), "vault_ref": ctx.Flag("vault-ref"), "attestations": map[string]bool{"region_locked": true, "unique_fingerprint": true}}, "Identity created.")
		}},
		{Name: "start", Description: "Start a warming program after recorded attestations", NeedsAPI: true, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "identity", Required: true, Description: "Identity id"}}, Flags: []cliapp.Flag{{Name: "program", Required: true, Description: "Warming program id"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			return request(ctx, http.MethodPost, "/channel-manager/identities/"+ctx.Positional("identity")+"/start", map[string]string{"program_id": ctx.Flag("program")}, "Warming started.")
		}},
		{Name: "queue", Description: "Queue a manual platform action", NeedsAPI: true, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "identity", Required: true, Description: "Identity id"}, {Name: "kind", Required: true, Description: "Declared action kind"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			return request(ctx, http.MethodPost, "/channel-manager/actions", map[string]string{"identity_id": ctx.Flag("identity"), "kind": ctx.Flag("kind")}, "Manual action queued.")
		}},
		{Name: "complete", Description: "Record manual action completion and optional evidence", NeedsAPI: true, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "action", Required: true, Description: "Action id"}}, Flags: []cliapp.Flag{{Name: "evidence", Description: "URL, metric, or screenshot reference"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			return request(ctx, http.MethodPost, "/channel-manager/actions/"+ctx.Positional("action")+"/complete", map[string]string{"evidence": ctx.Flag("evidence")}, "Manual completion recorded.")
		}},
		{Name: "observe", Description: "Record a manually observed reach measurement", NeedsAPI: true, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "identity", Required: true, Description: "Identity id"}}, Flags: []cliapp.Flag{{Name: "value", Required: true, Description: "Observed reach or impressions"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			return request(ctx, http.MethodPost, "/channel-manager/identities/"+ctx.Positional("identity")+"/observations", map[string]string{"metric": "reach", "value": ctx.Flag("value")}, "Observation recorded.")
		}},
		{Name: "assign-automation", Description: "Assign an operator-approved BAS profile and workflow reference", NeedsAPI: true, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "identity-id", Required: true, Description: "Identity id"}, {Name: "session-profile-ref", Required: true, Description: "Opaque BAS profile reference"}, {Name: "workflow-ref", Required: true, Description: "Operator-approved BAS workflow UUID"}, {Name: "enabled-action-kind", Required: true, Description: "Permitted action kind"}, {Name: "operator-note", Required: true, Description: "Operator acceptance decision"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			return request(ctx, http.MethodPost, "/channel-manager/identities/"+ctx.Flag("identity-id")+"/automation", map[string]any{"session_profile_ref": ctx.Flag("session-profile-ref"), "workflow_ref": ctx.Flag("workflow-ref"), "enabled_action_kinds": []string{ctx.Flag("enabled-action-kind")}, "operator_note": ctx.Flag("operator-note")}, "Browser automation gate saved.")
		}},
		{Name: "dispatch-browser", Description: "Dispatch an approved durable action to BAS", NeedsAPI: true, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "action-id", Required: true, Description: "Scheduled action id"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			return request(ctx, http.MethodPost, "/channel-manager/actions/"+ctx.Positional("action-id")+"/dispatch-browser", map[string]string{}, "Browser action dispatched.")
		}},
	}}
}
