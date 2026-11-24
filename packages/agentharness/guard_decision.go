package agentharness

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// GuardOverridePolicy is the reviewed grant policy an agent asks for when a
// guard stops a legitimate action; its target is the guard's check ID.
const (
	GuardOverridePolicy    = "agent-guard-override"
	GuardOverrideOperation = "agent-guard.override"
	guardProviderID        = "agent-guard"
	guardEventsFile        = "guard-events.jsonl"
	guardEventsMaxBytes    = 4 << 20
	guardEventCommandChars = 240
)

// GuardGrantCheck reports whether an operator-approved grant covers a guard
// check for the calling agent session. It is a variable so tests and other
// hosts can replace the broker lookup.
var GuardGrantCheck = brokerGuardGrant

// hardStopDecision decides an event a guard recognized. It returns stopped
// false when the operator has put the guard in log-only mode or an approved
// grant covers it; the caller then evaluates the event as usual. Every
// outcome is appended to the guard event log the steward reads.
func (r Runtime) hardStopDecision(event ToolEvent, risk RiskClass, match hardStopMatch, now time.Time) (Decision, bool) {
	guard := guardFor(match.Class)
	if store := r.guardModeStore(); store != nil {
		if _, active, err := store.ModeFor(guard.ID, now); err == nil && active {
			r.recordGuardEvent(event, guard, match, "log_only", now)
			return Decision{}, false
		}
	}
	if GuardGrantCheck != nil && GuardGrantCheck(guard.ID) {
		r.recordGuardEvent(event, guard, match, "granted", now)
		return Decision{}, false
	}
	r.recordGuardEvent(event, guard, match, "blocked", now)
	return Decision{
		ContractVersion: ContractVersion, EventID: event.EventID, Action: ActionDeny, Risk: risk,
		ProviderID: guardProviderID, EvaluatorID: guard.ID,
		Reason: guardMessage(guard, match),
		Evidence: []Evidence{{Code: "AGENT_GUARD", Message: "an agent guard stopped this action", Source: "policy-runtime",
			Facts: map[string]string{"check": guard.ID, "signal": match.Signal}, Severity: "warning"}},
	}, true
}

// guardMessage tells the agent what stopped, why, what to use instead, and
// how to get the action approved when it is legitimate.
func guardMessage(guard guardInfo, match hardStopMatch) string {
	var b strings.Builder
	b.WriteString("Stopped by the agent guard for " + guard.Name + " (" + guard.ID + "): this command " + match.Signal + ".\n")
	b.WriteString("Why: " + guard.Why + "\n")
	if len(guard.Instead) > 0 {
		b.WriteString("Instead: " + strings.Join(guard.Instead, " ") + "\n")
	}
	b.WriteString("If the action is legitimate, ask the operator to approve it, then retry once approved:\n")
	b.WriteString("  vrooli agent authorize \"$VROOLI_AGENT_SESSION\" --policy " + GuardOverridePolicy + " --target " + guard.ID + " --reason \"<what you need to do and why>\"\n")
	b.WriteString("If the guard misread the command, say so in your report. Repeated stops go to the operator as a decision to relax or fix the guard.")
	return b.String()
}

// brokerGuardGrant asks the local credential broker for an approved grant
// covering this guard. It answers false quickly outside an Agent Manager
// session, when the broker is absent, and on any refusal.
func brokerGuardGrant(checkID string) bool {
	if _, err := os.Stat(cliutil.AgentCredentialSocket()); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	credentials, err := cliutil.ResolveAgentCredentialsForOperation(ctx, GuardOverrideOperation, checkID)
	if err != nil {
		return false
	}
	// An Agent Manager run holds a baseline plus an operation credential; a
	// session the launcher attached holds one grant-bearing credential. Either
	// way the verified claims must carry a grant that covers this guard.
	var claims *cliutil.VerifiedClaims
	switch {
	case credentials.BaselineToken != "" && credentials.OperationToken != "":
		verified, err := (cliutil.IdentityEnv{Token: credentials.BaselineToken}).VerifyIdentityPair(ctx, credentials.OperationToken)
		if err == nil && verified != nil && verified.Valid && verified.Operation != nil && verified.Operation.Valid {
			claims = verified.Operation.Claims
		}
	case credentials.LegacyToken != "":
		verified, err := (cliutil.IdentityEnv{Token: credentials.LegacyToken}).VerifyIdentityWithSessionBinding(ctx)
		if err == nil && verified != nil && verified.Valid {
			claims = verified.Claims
		}
	}
	return claims != nil && grantCovers(claims.Meta["operation_grant"], checkID, time.Now())
}

// grantCovers reads the operation grant a verified credential carries and
// reports whether it names the guard override for this check and is current.
func grantCovers(raw, checkID string, now time.Time) bool {
	var grant struct {
		Policy     string    `json:"policy"`
		Operations []string  `json:"operations"`
		Targets    []string  `json:"targets"`
		ExpiresAt  time.Time `json:"expires_at"`
	}
	if raw == "" || json.Unmarshal([]byte(raw), &grant) != nil || grant.Policy != GuardOverridePolicy || !now.Before(grant.ExpiresAt) {
		return false
	}
	operation, target := false, false
	for _, value := range grant.Operations {
		operation = operation || value == GuardOverrideOperation
	}
	for _, value := range grant.Targets {
		target = target || value == checkID || value == guardProviderID+":*"
	}
	return operation && target
}

// GuardEvent is one line of the guard event log. The steward reads the log to
// find guards that keep stopping work and asks the operator about them.
type GuardEvent struct {
	At      time.Time `json:"at"`
	Check   string    `json:"check"`
	Outcome string    `json:"outcome"`
	Signal  string    `json:"signal"`
	Runner  string    `json:"runner,omitempty"`
	Session string    `json:"session,omitempty"`
	Run     string    `json:"run,omitempty"`
	Cwd     string    `json:"cwd,omitempty"`
	Command string    `json:"command,omitempty"`
}

// recordGuardEvent appends to the guard event log. It is best effort: the log
// informs the steward and never decides an event, so a write failure is
// ignored rather than turned into a block.
func (r Runtime) recordGuardEvent(event ToolEvent, guard guardInfo, match hardStopMatch, outcome string, now time.Time) {
	r.recordEvent(event, guard.ID, match.Signal, outcome, now)
}

// floorCheckPrefix names the checks no log-only mode or guard grant relaxes:
// the deletion floor, operator-only commands and provider-state refusals.
// They are logged so the steward sees them too; a fix is their only route.
const floorCheckPrefix = "agent-floor:"

// recordStop logs a refusal or confirmation request that did not come from a
// guard, so every stop an agent meets reaches the steward. A provider's own
// evaluator ID is kept; any other stop is named from its evidence code.
func (r Runtime) recordStop(event ToolEvent, decision Decision, now time.Time) {
	if decision.ProviderID == guardProviderID {
		return
	}
	if decision.Action == ActionAllow {
		// A provider check in log-only mode let this through; log what it
		// would have stopped so a relaxed check is visible, not silent.
		for _, evidence := range decision.Evidence {
			if evidence.Code == "CHECK_LOG_ONLY" && evidence.Facts["check"] != "" {
				r.recordEvent(event, evidence.Facts["check"], "would have been "+evidence.Facts["original_action"]+"ed; the check is log-only", "log_only", now)
			}
		}
		return
	}
	if decision.Action != ActionDeny && decision.Action != ActionAsk {
		return
	}
	check := strings.TrimSpace(decision.EvaluatorID)
	if !strings.Contains(check, ":") {
		code := "unexplained"
		if len(decision.Evidence) > 0 && decision.Evidence[0].Code != "" {
			code = decision.Evidence[0].Code
		}
		check = floorCheckPrefix + floorName(code)
	}
	outcome := "blocked"
	if decision.Action == ActionAsk {
		outcome = "asked"
	}
	signal := strings.TrimSpace(strings.SplitN(decision.Reason, "\n", 2)[0])
	if runes := []rune(signal); len(runes) > 160 {
		signal = string(runes[:160]) + "…"
	}
	r.recordEvent(event, check, signal, outcome, now)
}

// floorName turns an evidence code into a check name: REMOVAL_REPO_TREE
// becomes deletion-repo-tree and OPERATOR_ONLY_COMMAND operator-only.
func floorName(code string) string {
	name := strings.ToLower(strings.ReplaceAll(code, "_", "-"))
	switch {
	case strings.HasPrefix(name, "removal-"):
		return "deletion-" + strings.TrimPrefix(name, "removal-")
	case name == "operator-only-command":
		return "operator-only"
	}
	return name
}

func (r Runtime) recordEvent(event ToolEvent, check, signal, outcome string, now time.Time) {
	// The log lives beside the runtime's own guard-mode store. A runtime with
	// no store (a test, a library caller) logs nothing, so it can never put
	// synthetic blocks in front of the steward.
	store := r.guardModeStore()
	if store == nil || store.Dir == "" {
		return
	}
	dir := store.Dir
	command := event.Shell
	if command == "" {
		command = strings.TrimSpace(event.Tool + " " + strings.Join(event.Arguments, " "))
	}
	command = redactSecrets(command)
	if runes := []rune(command); len(runes) > guardEventCommandChars {
		command = string(runes[:guardEventCommandChars]) + "…"
	}
	session := event.Context["session_id"]
	if session == "" {
		session = os.Getenv("VROOLI_AGENT_SESSION")
	}
	line, err := json.Marshal(GuardEvent{At: now.UTC(), Check: check, Outcome: outcome, Signal: signal,
		Runner: event.Runner, Session: session, Run: os.Getenv("AGENT_MANAGER_RUN_ID"), Cwd: event.WorkingDirectory, Command: command})
	if err != nil {
		return
	}
	_ = os.MkdirAll(dir, 0o700)
	path := filepath.Join(dir, guardEventsFile)
	if info, err := os.Stat(path); err == nil && info.Size() > guardEventsMaxBytes {
		_ = os.Rename(path, path+".1")
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.Write(append(line, '\n'))
}

// secretShapes match credential values a command line can carry: a
// credential-named assignment or flag, an authorization header, and the
// well-known token prefixes. The guard log is kept and read back, so values
// are replaced before it is written.
var secretShapes = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b([a-z0-9_-]*(?:token|secret|password|passwd|api[_-]?key|credential)[a-z0-9_-]*)([=:]\s*)("[^"]*"|'[^']*'|[^\s'"]+)`),
	regexp.MustCompile(`(?i)(\s--?[a-z0-9-]*(?:token|secret|password|passwd|api-?key|credential)[a-z0-9-]*)(\s+)("[^"]*"|'[^']*'|[^\s'"-][^\s'"]*)`),
	regexp.MustCompile(`(?i)\b(bearer|basic)\s+[a-z0-9._~+/=-]{8,}`),
	regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|xox[abprs]-[A-Za-z0-9-]{10,}|sk-[A-Za-z0-9_-]{16,}|AKIA[0-9A-Z]{16}|AIza[0-9A-Za-z_-]{30,})`),
}

func redactSecrets(text string) string {
	text = secretShapes[0].ReplaceAllString(text, "${1}${2}***")
	text = secretShapes[1].ReplaceAllString(text, "${1}${2}***")
	text = secretShapes[2].ReplaceAllString(text, "${1} ***")
	return secretShapes[3].ReplaceAllString(text, "***")
}
