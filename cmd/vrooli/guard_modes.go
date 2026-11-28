package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/cli-core/guardmodes"
	"github.com/vrooli/vrooli/internal/agentscope"
)

var (
	guardModeCallerEvidence = agentscope.CallerSessionEvidence
	guardModeCallerActor    = func() string { return cliutil.ResolveCallerActor("") }
	guardModeHubCommand     = runGuardModeHubCommand
	guardModeHubRead        = readGuardModeHubAsk
	guardModeNow            = time.Now
)

const stewardGuardSubjectPrefix = "portfolio-steward:guard-friction:"
const stewardGuardQuestion = "Should I change the system check that repeatedly stopped work?"
const stewardGuardLoosenLabel = "Log this check only for 24 hours"

type guardModeHubPayload struct {
	Ask map[string]any `json:"ask"`
}

func readGuardModeHubAsk(subject string) (map[string]any, error) {
	output, err := guardModeHubCommand(subject)
	if err != nil {
		return nil, fmt.Errorf("read Notification Hub subject: %w", err)
	}
	var payload guardModeHubPayload
	if err := json.Unmarshal(output, &payload); err != nil {
		return nil, fmt.Errorf("decode Notification Hub response: %w", err)
	}
	if payload.Ask == nil {
		return nil, fmt.Errorf("Notification Hub response has no ask")
	}
	return payload.Ask, nil
}

func runGuardModeHubCommand(subject string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, "notification-hub", "conversations", "get", "--subject-key", subject, "--json")
	return command.Output()
}

func guardModeAnswerValue(ask map[string]any) (string, map[string]any) {
	changes, _ := ask["answer_changes"].(map[string]any)
	if changes == nil {
		changes, _ = ask["changes"].(map[string]any)
	}
	answer, _ := ask["answer"].(map[string]any)
	if nested, ok := answer["changes"].(map[string]any); ok {
		changes = nested
	}
	if changes == nil {
		changes = map[string]any{}
	}
	key, _ := changes["answer"].(string)
	plainAnswer, _ := ask["answer"].(string)
	if key != "" && plainAnswer != "" && key != plainAnswer {
		return "", changes
	}
	if key == "" {
		key = strings.TrimSpace(plainAnswer)
	}
	return strings.TrimSpace(key), changes
}

func validateStewardGuardAnswer(ask map[string]any, subject, askID string) (string, error) {
	if subject == "" || !strings.HasPrefix(subject, stewardGuardSubjectPrefix) {
		return "", fmt.Errorf("subject is not a steward structural-friction decision")
	}
	check := strings.TrimPrefix(subject, stewardGuardSubjectPrefix)
	if ask["subject_key"] != subject || ask["id"] != askID || ask["state"] != "answered" || ask["answered_by_identity_kind"] != "verified_human" {
		return "", fmt.Errorf("Hub ask identity, state, or verified-human attribution does not match")
	}
	source, _ := ask["answered_by_identity_source"].(string)
	if source != "cloudflare_access" && source != "personal_local" && source != "scenario_authenticator_step_up" {
		return "", fmt.Errorf("Hub answer lacks a documented verified-human authorization source")
	}
	if ask["template_name"] != "structural-change" {
		return "", fmt.Errorf("Hub answer is not for the structural-change template")
	}
	if ask["question"] != stewardGuardQuestion {
		return "", fmt.Errorf("Hub answer question does not match the steward structural decision")
	}
	data, _ := ask["template_data"].(map[string]any)
	meta, _ := data["meta"].(map[string]any)
	if meta["title"] != stewardGuardQuestion {
		return "", fmt.Errorf("template title does not match the steward structural decision")
	}
	rows, _ := data["technical"].([]any)
	bound := false
	for _, raw := range rows {
		row, _ := raw.(map[string]any)
		if row["label"] == "Evaluator ID" && row["value"] == check {
			bound = true
		}
	}
	if !bound {
		return "", fmt.Errorf("template evidence does not bind the subject to its exact evaluator ID")
	}
	key, _ := guardModeAnswerValue(ask)
	if key != "loosen_24h" {
		return "", fmt.Errorf("Hub answer does not select the 24-hour log-only action")
	}
	opts, _ := ask["options"].([]any)
	optionPresent := false
	for _, raw := range opts {
		option, _ := raw.(map[string]any)
		if option["key"] == "loosen_24h" && option["label"] == stewardGuardLoosenLabel {
			optionPresent = true
		}
	}
	if !optionPresent {
		return "", fmt.Errorf("the answered ask did not offer the 24-hour log-only action")
	}
	return check, nil
}

// runGuardModeCommand owns mutations in the root control plane, where operator
// decisions use the same kernel-backed session check as root-update verbs.
func runGuardModeCommand(args []string, stdout, stderr io.Writer) (bool, int) {
	if len(args) == 0 || args[0] != "guard-mode" {
		return false, 0
	}
	if len(args) < 2 {
		fmt.Fprintln(stderr, "usage: vrooli guard-mode enable|disable|list|audit [options]")
		return true, 2
	}
	command := args[1]
	if command != "enable" && command != "disable" && command != "list" && command != "audit" && command != "apply-answer" {
		fmt.Fprintf(stderr, "unknown guard-mode command %q (want enable, disable, list, audit, or apply-answer)\n", command)
		return true, 2
	}
	fs := flag.NewFlagSet("guard-mode "+command, flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("snapshot-dir", "", "policy snapshot and guard mode directory")
	check := fs.String("check", "", "provider-issued evaluator id (<provider-id>:<rule-name>)")
	expires := fs.String("expires-at", "", "RFC3339 expiry (maximum seven days)")
	reason := fs.String("reason", "", "audited reason for the mode change")
	subject := fs.String("subject-key", "", "exact answered steward structural-decision subject")
	askID := fs.String("ask-id", "", "stable Notification Hub ask ID")
	if err := fs.Parse(args[2:]); err != nil {
		return true, 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected guard-mode arguments: %s\n", strings.Join(fs.Args(), " "))
		return true, 2
	}
	dataDir := strings.TrimSpace(*dir)
	if dataDir == "" {
		var err error
		dataDir, err = guardmodes.DefaultDataDir()
		if err != nil {
			fmt.Fprintf(stderr, "resolve guard mode directory: %v\n", err)
			return true, 1
		}
	}
	store := guardmodes.NewStore(dataDir)
	store.Now = guardModeNow
	switch command {
	case "apply-answer":
		if *subject == "" || *askID == "" {
			fmt.Fprintln(stderr, "guard-mode apply-answer requires --subject-key and --ask-id")
			return true, 2
		}
		ask, err := guardModeHubRead(strings.TrimSpace(*subject))
		if err != nil {
			fmt.Fprintf(stderr, "read answered structural decision: %v\n", err)
			return true, 1
		}
		check, err := validateStewardGuardAnswer(ask, strings.TrimSpace(*subject), strings.TrimSpace(*askID))
		if err != nil {
			fmt.Fprintf(stderr, "validate answered structural decision: %v\n", err)
			return true, 1
		}
		actor := "verified operator via Notification Hub"
		expiresAt := guardModeNow().UTC().Add(24 * time.Hour)
		if err := store.SetLogOnlyFromAnswer(check, expiresAt, "operator selected 24-hour log-only mode in the exact structural-friction decision", actor, strings.TrimSpace(*askID)); err != nil {
			fmt.Fprintf(stderr, "apply answered guard mode: %v\n", err)
			if !strings.Contains(err.Error(), "already consumed") {
				fmt.Fprintln(stderr, "Before retrying, inspect guard-mode list and audit: a durable answer-consumption record may exist even when a later state write failed. Replaying the same answer ID can never extend or reinstate its interval.")
			}
			return true, 1
		}
		_ = json.NewEncoder(stdout).Encode(map[string]any{"applied": true, "check": check,
			"mode": "log_only", "expires_at": expiresAt.Format(time.RFC3339), "answer_id": strings.TrimSpace(*askID)})
		return true, 0
	case "enable":
		if *check == "" || *expires == "" || *reason == "" {
			fmt.Fprintln(stderr, "guard-mode enable requires --check, --expires-at, and --reason")
			return true, 2
		}
		actor, ok := authorizeGuardModeMutation(stderr)
		if !ok {
			return true, 1
		}
		expiresAt, err := time.Parse(time.RFC3339, strings.TrimSpace(*expires))
		if err != nil {
			fmt.Fprintf(stderr, "parse --expires-at: %v\n", err)
			return true, 2
		}
		if err := store.SetLogOnly(strings.TrimSpace(*check), expiresAt, *reason, actor); err != nil {
			fmt.Fprintf(stderr, "enable guard mode: %v\n", err)
			return true, 1
		}
		fmt.Fprintf(stdout, "log-only enabled for %s until %s\n", strings.TrimSpace(*check), expiresAt.UTC().Format(time.RFC3339))
		return true, 0
	case "disable":
		if *check == "" || *reason == "" {
			fmt.Fprintln(stderr, "guard-mode disable requires --check and --reason")
			return true, 2
		}
		actor, ok := authorizeGuardModeMutation(stderr)
		if !ok {
			return true, 1
		}
		if err := store.Disable(strings.TrimSpace(*check), actor, *reason); err != nil {
			fmt.Fprintf(stderr, "disable guard mode: %v\n", err)
			return true, 1
		}
		fmt.Fprintf(stdout, "log-only disabled for %s\n", strings.TrimSpace(*check))
		return true, 0
	case "list":
		active, err := store.Active(time.Now().UTC())
		if err != nil {
			fmt.Fprintf(stderr, "list guard modes: %v\n", err)
			return true, 1
		}
		for _, mode := range active {
			fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", mode.Check, mode.Mode, mode.ExpiresAt.UTC().Format(time.RFC3339), mode.Reason)
		}
		return true, 0
	case "audit":
		entries, err := store.Audit()
		if err != nil {
			fmt.Fprintf(stderr, "read guard mode audit: %v\n", err)
			return true, 1
		}
		for _, entry := range entries {
			fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\t%s\n", entry.At.UTC().Format(time.RFC3339), entry.Action, entry.Check, entry.EventID, entry.OriginalAction)
		}
		return true, 0
	default:
		return true, 1
	}
}

func authorizeGuardModeMutation(stderr io.Writer) (string, bool) {
	if evidence := guardModeCallerEvidence(); evidence != "" {
		fmt.Fprintf(stderr, "guard mode changes are operator decisions and cannot run from a coding-agent session (%s); use the approved Notification Hub answer flow or run this command from the operator's terminal\n", evidence)
		return "", false
	}
	return guardModeCallerActor(), true
}
