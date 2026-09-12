package channelmanager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// Register exposes the permanent manual executor floor. It uses the scenario
// API exclusively; secrets remain credential-authority references and are never CLI flags.
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
		{Name: "create", Description: "Create an identity with non-secret metadata and an optional credential-authority reference", NeedsAPI: true, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "id", Required: true, Description: "Stable identity id"}, {Name: "platform", Required: true, Description: "Platform descriptor id"}, {Name: "purpose", Required: true, Description: "brand or persona-actor"}, {Name: "environment", Required: true, Description: "Attested environment reference"}, {Name: "credential-ref", Description: "Optional credential-authority identity/field reference; required before browser automation, never a secret"}, {Name: "handle", Description: "Public account handle"}, {Name: "label", Description: "Operator display label"}, {Name: "persona-ref", Description: "Persona metadata reference"}, {Name: "goals", Description: "Comma-separated account goals"}, {Name: "notes", Description: "Private operator notes"}, {Name: "owner-ref", Description: "Operator ownership reference"}, {Name: "lifecycle", Description: "Lifecycle state; defaults to draft"}, {Name: "d009-ref", Description: "Operator D-009 acceptance reference"}, {Name: "automation-mode", Description: "manual or operator-gated; defaults to manual"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			return request(ctx, http.MethodPost, "/channel-manager/identities", map[string]any{"id": ctx.Flag("id"), "platform_id": ctx.Flag("platform"), "handle": ctx.Flag("handle"), "display_label": ctx.Flag("label"), "persona_ref": ctx.Flag("persona-ref"), "goals": splitGoals(ctx.Flag("goals")), "notes": ctx.Flag("notes"), "owner_ref": ctx.Flag("owner-ref"), "purpose": ctx.Flag("purpose"), "environment_ref": ctx.Flag("environment"), "credential_ref": ctx.Flag("credential-ref"), "lifecycle": ctx.Flag("lifecycle"), "d009_acceptance_ref": ctx.Flag("d009-ref"), "automation_mode": ctx.Flag("automation-mode"), "attestations": map[string]bool{"region_locked": true, "unique_fingerprint": true}}, "Identity created.")
		}},
		{Name: "edit", Description: "Update non-secret identity metadata", NeedsAPI: true, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "id", Required: true, Description: "Identity id"}, {Name: "handle", Description: "Public account handle"}, {Name: "label", Description: "Operator display label"}, {Name: "goals", Description: "Comma-separated goals"}, {Name: "notes", Description: "Private operator notes"}, {Name: "owner-ref", Description: "Operator ownership reference"}, {Name: "purpose", Required: true, Description: "Identity purpose"}, {Name: "environment", Required: true, Description: "Attested environment reference"}, {Name: "lifecycle", Required: true, Description: "Lifecycle state"}, {Name: "d009-ref", Description: "Operator D-009 acceptance reference"}, {Name: "automation-mode", Required: true, Description: "manual or operator-gated"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			id := ctx.Flag("id")
			return request(ctx, http.MethodPut, "/channel-manager/identities/"+id, map[string]any{"id": id, "handle": ctx.Flag("handle"), "display_label": ctx.Flag("label"), "goals": splitGoals(ctx.Flag("goals")), "notes": ctx.Flag("notes"), "owner_ref": ctx.Flag("owner-ref"), "purpose": ctx.Flag("purpose"), "environment_ref": ctx.Flag("environment"), "lifecycle": ctx.Flag("lifecycle"), "d009_acceptance_ref": ctx.Flag("d009-ref"), "automation_mode": ctx.Flag("automation-mode")}, "Identity updated.")
		}},
		{Name: "retire", Description: "Retire an identity without deleting its audit history", NeedsAPI: true, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "identity", Required: true, Description: "Identity id"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			return request(ctx, http.MethodPost, "/channel-manager/identities/"+ctx.Positional("identity")+"/retire", map[string]string{}, "Identity retired.")
		}},
		{Name: "timeline", Description: "Show the immutable, redacted activity timeline for an identity", NeedsAPI: true, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "identity", Required: true, Description: "Identity id"}}, Flags: []cliapp.Flag{{Name: "action-id", Description: "Optional action id filter"}, {Name: "event-type", Description: "Optional event type filter"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			query := url.Values{}
			if actionID := ctx.Flag("action-id"); actionID != "" {
				query.Set("action_id", actionID)
			}
			if eventType := ctx.Flag("event-type"); eventType != "" {
				query.Set("event_type", eventType)
			}
			bytes, err := ctx.Core().Get("/channel-manager/identities/"+ctx.Positional("identity")+"/timeline", query)
			if err != nil {
				return err
			}
			return ctx.RenderList(cliapp.ListReport{Summary: []string{"Channel Manager activity timeline"}, ResultsHeading: "Events", Results: []string{string(bytes)}})
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
		{Name: "assign-automation", Description: "Assign an operator-approved BAS profile and workflow reference", NeedsAPI: true, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "identity-id", Required: true, Description: "Identity id"}, {Name: "consumer-profile-key", Required: true, Description: "Profile key declared in this scenario's BAS consumer declaration"}, {Name: "session-profile-ref", Required: true, Description: "Opaque BAS profile reference"}, {Name: "workflow-ref", Required: true, Description: "Operator-approved BAS workflow UUID"}, {Name: "enabled-action-kind", Required: true, Description: "Permitted action kind"}, {Name: "operator-note", Required: true, Description: "Operator acceptance decision"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			return request(ctx, http.MethodPost, "/channel-manager/identities/"+ctx.Flag("identity-id")+"/automation", map[string]any{"consumer_profile_key": ctx.Flag("consumer-profile-key"), "session_profile_ref": ctx.Flag("session-profile-ref"), "workflow_ref": ctx.Flag("workflow-ref"), "enabled_action_kinds": []string{ctx.Flag("enabled-action-kind")}, "operator_note": ctx.Flag("operator-note")}, "Browser automation gate saved.")
		}},
		{Name: "dispatch-browser", Description: "Dispatch an approved durable action to BAS", NeedsAPI: true, Args: cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "action-id", Required: true, Description: "Scheduled action id"}}}, RunCtx: func(ctx cliapp.RunContext) error {
			return request(ctx, http.MethodPost, "/channel-manager/actions/"+ctx.Positional("action-id")+"/dispatch-browser", map[string]string{}, "Browser action dispatched.")
		}},
	}}
}

func splitGoals(raw string) []string {
	goals := make([]string, 0)
	for _, goal := range strings.Split(raw, ",") {
		if goal = strings.TrimSpace(goal); goal != "" {
			goals = append(goals, goal)
		}
	}
	return goals
}
