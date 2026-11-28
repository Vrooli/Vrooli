package agentharness

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRuntimeLogOnlyIsPerCheckExpiresAndAudits(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	store := NewBundleStore(t.TempDir())
	snapshot := testSnapshot(now, "sda", MaturityEnforcing, HealthHealthy, EvidenceClean)
	snapshot.Rules = []PolicyRule{{EvaluatorID: "sda:dependency-addition", Risk: RiskDependencyAdd, Action: ActionDeny, Reason: "blocked add"}, {EvaluatorID: "sda:dependency-upgrade", Risk: RiskDependencyUpgrade, Action: ActionDeny, Reason: "blocked upgrade"}}
	publishTestBundle(t, store, snapshot)
	modes := NewGuardModeStore(store.Dir)
	modes.Now = func() time.Time { return now }
	if err := modes.SetLogOnly("sda:dependency-addition", now.Add(time.Minute), "recurring false positive", "operator"); err != nil {
		t.Fatal(err)
	}
	runtime := Runtime{Profile: ProfileEnforcing, Store: store, Now: func() time.Time { return now }}
	got, err := runtime.Evaluate(testEvent())
	if err != nil {
		t.Fatal(err)
	}
	if got.Action != ActionAllow || !hasEvidence(got, "CHECK_LOG_ONLY") {
		t.Fatalf("log-only decision = %+v", got)
	}
	upgrade := testEvent()
	upgrade.Arguments = []string{"upgrade", "example"}
	got, err = runtime.Evaluate(upgrade)
	if err != nil {
		t.Fatal(err)
	}
	if got.Action != ActionDeny {
		t.Fatalf("unconfigured check = %+v", got)
	}
	now = now.Add(time.Minute)
	got, err = runtime.Evaluate(testEvent())
	if err != nil {
		t.Fatal(err)
	}
	if got.Action != ActionDeny {
		t.Fatalf("expired mode = %+v", got)
	}
	entries, err := modes.Audit()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || entries[0].Action != "enabled" || entries[1].Action != "observed" || entries[1].Check != "sda:dependency-addition" || entries[1].OriginalAction != ActionDeny || entries[1].EventID == "" || entries[1].ExpiresAt.IsZero() || entries[2].Action != "expired" {
		t.Fatalf("audit entries = %+v, want enable, observed suppression, and expiry", entries)
	}
}

func TestDecisionRemainsEnforcedWhenSuppressionAuditCannotBeWritten(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	store := NewBundleStore(t.TempDir())
	snapshot := testSnapshot(now, "scenario-dependency-analyzer", MaturityEnforcing, HealthHealthy, EvidenceClean)
	snapshot.Rules = []PolicyRule{{EvaluatorID: "scenario-dependency-analyzer:dependency-addition", Risk: RiskDependencyAdd, Action: ActionDeny, Reason: "blocked add"}}
	publishTestBundle(t, store, snapshot)
	modes := NewGuardModeStore(store.Dir)
	modes.Now = func() time.Time { return now }
	if err := modes.SetLogOnly("scenario-dependency-analyzer:dependency-addition", now.Add(time.Hour), "operator decision", "operator"); err != nil {
		t.Fatal(err)
	}
	auditPath := filepath.Join(store.Dir, "guard-modes.audit.jsonl")
	blockedAuditPath := auditPath + ".saved"
	if err := os.Rename(auditPath, blockedAuditPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(auditPath, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(auditPath)
		_ = os.Rename(blockedAuditPath, auditPath)
	})

	runtime := Runtime{Profile: ProfileEnforcing, Store: store, GuardModes: modes, Now: func() time.Time { return now }}
	decision, err := runtime.Evaluate(testEvent())
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != ActionDeny || !decision.Degraded || !hasEvidence(decision, "CHECK_MODES_UNAVAILABLE") || hasEvidence(decision, "CHECK_LOG_ONLY") {
		t.Fatalf("decision was relaxed without a durable observed-action audit: %+v", decision)
	}
}

func TestHardStopClassificationUsesExactHookTokens(t *testing.T) {
	for _, test := range []struct {
		name  string
		event ToolEvent
		want  HardStopClass
	}{
		{name: "money token", event: ToolEvent{Tool: "stripe", Arguments: []string{"charge"}}, want: HardStopMoney},
		{name: "echoing money token is not a transaction", event: ToolEvent{Tool: "shell", Shell: "echo payment"}, want: ""},
		{name: "money substring does not match", event: ToolEvent{Tool: "shell", Shell: "go test ./internal/recharge"}, want: ""},
		{name: "searching protected words is not an effect", event: ToolEvent{Tool: "Search", Arguments: []string{"secret", "stripe charge", "slack send"}}, want: ""},
		{name: "shell comment mentioning message action is not an effect", event: ToolEvent{Tool: "shell", Shell: "# comment mentions slack send"}, want: ""},
		{name: "echoed message words are not an outbound message", event: ToolEvent{Tool: "shell", Shell: "echo 'slack send'"}, want: ""},
		{name: "printing protected words is not a secret access", event: ToolEvent{Tool: "shell", Shell: "echo secret"}, want: ""},
		{name: "search query for credential field is not a secret access", event: ToolEvent{Tool: "Search", Arguments: []string{"api_key"}}, want: ""},
		{name: "secret target", event: ToolEvent{Tool: "Read", Target: "/repo/.env"}, want: HardStopSecret},
		{name: "write tool targeting secret file remains protected", event: ToolEvent{Tool: "Write", Target: "/repo/.env"}, want: HardStopSecret},
		{name: "editing schema prose about secret reads is not secret access", event: ToolEvent{Tool: "Write", Target: "/repo/kit/schema.json", Arguments: []string{"description: read secrets only with approval; do not write credentials"}}, want: ""},
		{name: "writing schema heredoc prose about protected actions is not an external effect", event: ToolEvent{Tool: "shell", Shell: "cat > /tmp/agentharness-schema.json <<'EOF'\nDescription: Slack send, Stripe payment and secret reads are hard stops.\nEOF"}, want: ""},
		{name: "post to a messaging service inside an unquoted heredoc is an outbound message", event: ToolEvent{Tool: "shell", Shell: "cat > /tmp/agentharness-output <<EOF\n$(curl -X POST https://hooks.slack.com/services/x -d payload)\nEOF"}, want: HardStopOutsideMessage},
		{name: "unquoted heredoc escaped expansion marker remains literal", event: ToolEvent{Tool: "shell", Shell: "cat > /tmp/agentharness-output <<EOF\n\\${API_TOKEN}\nEOF"}, want: ""},
		{name: "heredoc reading a secret path remains protected", event: ToolEvent{Tool: "shell", Shell: "cat /repo/.env > /tmp/agentharness-schema.json <<'EOF'\nDocument text.\nEOF"}, want: HardStopSecret},
		{name: "heredoc network redirection remains an external message hard stop", event: ToolEvent{Tool: "shell", Shell: "cat > /dev/tcp/example.invalid/443 <<'EOF'\nhello\nEOF"}, want: HardStopOutsideMessage},
		{name: "attached descriptor network redirection remains an external message hard stop", event: ToolEvent{Tool: "shell", Shell: "cat 3>/dev/tcp/example.invalid/443 <<'EOF'\nhello\nEOF"}, want: HardStopOutsideMessage},
		{name: "combined stdout stderr network redirection remains a hard stop", event: ToolEvent{Tool: "shell", Shell: "cat &>/dev/udp/example.invalid/443 <<'EOF'\nhello\nEOF"}, want: HardStopOutsideMessage},
		{name: "network redirection in nested shell is a hard stop", event: ToolEvent{Tool: "shell", Shell: "bash -c \"echo > /dev/tcp/host/443\""}, want: HardStopOutsideMessage},
		{name: "network redirection in sudo-wrapped shell is a hard stop", event: ToolEvent{Tool: "shell", Shell: "sudo bash -c 'echo >/dev/tcp/host/443'"}, want: HardStopOutsideMessage},
		{name: "network redirection in sudo long-user wrapped shell is a hard stop", event: ToolEvent{Tool: "shell", Shell: "sudo --user root bash -c 'echo >/dev/tcp/host/443'"}, want: HardStopOutsideMessage},
		{name: "network redirection in sudo long-group wrapped shell is a hard stop", event: ToolEvent{Tool: "shell", Shell: "sudo --group root bash -c 'echo >/dev/tcp/host/443'"}, want: HardStopOutsideMessage},
		{name: "network redirection in sudo short-chroot wrapped shell is a hard stop", event: ToolEvent{Tool: "shell", Shell: "sudo -R /rootfs bash -c 'echo >/dev/tcp/host/443'"}, want: HardStopOutsideMessage},
		{name: "network redirection in env-wrapped shell is a hard stop", event: ToolEvent{Tool: "shell", Shell: "env FOO=bar bash -c 'echo >/dev/tcp/host/443'"}, want: HardStopOutsideMessage},
		{name: "network redirection in env split-string shell is a hard stop", event: ToolEvent{Tool: "shell", Shell: "env -S 'bash -c \"echo >/dev/tcp/host/443\"'"}, want: HardStopOutsideMessage},
		{name: "network redirection in env split-string equals shell is a hard stop", event: ToolEvent{Tool: "shell", Shell: "env --split-string='bash -c \"echo >/dev/tcp/host/443\"'"}, want: HardStopOutsideMessage},
		{name: "network redirection in command substitution is a hard stop", event: ToolEvent{Tool: "shell", Shell: "echo $(echo >/dev/tcp/host/443)"}, want: HardStopOutsideMessage},
		{name: "single-quoted backslash cannot hide later network redirection", event: ToolEvent{Tool: "shell", Shell: "echo 'literal\\'; echo > /dev/tcp/host/443"}, want: HardStopOutsideMessage},
		{name: "quoted network redirection example is inert text", event: ToolEvent{Tool: "shell", Shell: "echo 'Example syntax: > /dev/tcp/host/443'"}, want: ""},
		{name: "commented network redirection example is inert text", event: ToolEvent{Tool: "shell", Shell: "echo ready # example: > /dev/tcp/host/443"}, want: ""},
		{name: "vault secret-read tool remains protected", event: ToolEvent{Tool: "mcp__vault__read_secret"}, want: HardStopSecret},
		{name: "vault CLI secret read remains protected", event: ToolEvent{Tool: "shell", Shell: "vault read secret/app"}, want: HardStopSecret},
		{name: "sudo-wrapped vault read remains protected", event: ToolEvent{Tool: "shell", Shell: "sudo vault read kv/data/prod"}, want: HardStopSecret},
		{name: "env-wrapped vault read remains protected", event: ToolEvent{Tool: "shell", Shell: "env VAULT_ADDR=https://vault.example vault read kv/data/prod"}, want: HardStopSecret},
		{name: "structured vault read remains protected", event: ToolEvent{Tool: "vault", Arguments: []string{"read", "kv/data/prod"}}, want: HardStopSecret},
		{name: "vault read after harmless command remains protected", event: ToolEvent{Tool: "shell", Shell: "echo ready; sudo vault read kv/data/prod"}, want: HardStopSecret},
		{name: "vault read in nested shell remains protected", event: ToolEvent{Tool: "shell", Shell: "bash -c 'vault read kv/data/prod'"}, want: HardStopSecret},
		{name: "vault read in sudo-wrapped shell remains protected", event: ToolEvent{Tool: "shell", Shell: "sudo bash -c 'vault read kv/data/prod'"}, want: HardStopSecret},
		{name: "vault read in sudo long-user wrapped shell remains protected", event: ToolEvent{Tool: "shell", Shell: "sudo --user root bash -c 'vault read kv/data/prod'"}, want: HardStopSecret},
		{name: "vault read in sudo long-group wrapped shell remains protected", event: ToolEvent{Tool: "shell", Shell: "sudo --group root bash -c 'vault read kv/data/prod'"}, want: HardStopSecret},
		{name: "vault read in sudo short-chroot wrapped shell remains protected", event: ToolEvent{Tool: "shell", Shell: "sudo -R /rootfs bash -c 'vault read kv/data/prod'"}, want: HardStopSecret},
		{name: "vault read in env-wrapped shell remains protected", event: ToolEvent{Tool: "shell", Shell: "env FOO=bar bash -c 'vault read kv/data/prod'"}, want: HardStopSecret},
		{name: "vault read in env split-string shell remains protected", event: ToolEvent{Tool: "shell", Shell: "env -S 'bash -c \"vault read kv/data/prod\"'"}, want: HardStopSecret},
		{name: "vault read in env split-string equals shell remains protected", event: ToolEvent{Tool: "shell", Shell: "env --split-string='bash -c \"vault read kv/data/prod\"'"}, want: HardStopSecret},
		{name: "vault read in command substitution remains protected", event: ToolEvent{Tool: "shell", Shell: "echo \"$(vault read kv/data/prod)\""}, want: HardStopSecret},
		{name: "single-quoted backslash cannot hide later vault read", event: ToolEvent{Tool: "shell", Shell: "echo 'literal\\'; echo $(vault read kv/data/prod)"}, want: HardStopSecret},
		{name: "vault read text inside comment remains inert", event: ToolEvent{Tool: "shell", Shell: "echo ready # example; vault read kv/data/prod"}, want: ""},
		{name: "search target that opens credential file is protected", event: ToolEvent{Tool: "Search", Target: "/repo/.env"}, want: HardStopSecret},
		{name: "grep credential file path is protected", event: ToolEvent{Tool: "shell", Shell: "grep API_KEY /repo/.env"}, want: HardStopSecret},
		{name: "echoing a credential filename is not opening it", event: ToolEvent{Tool: "shell", Shell: "echo .env"}, want: ""},
		{name: "ssh credential target", event: ToolEvent{Tool: "Read", Target: "/home/user/.ssh/id_ed25519"}, want: HardStopSecret},
		{name: "secret context value with read operation", event: ToolEvent{Tool: "Read", Context: map[string]string{"path": "/repo/.env.local"}}, want: HardStopSecret},
		{name: "qualified environment secret member", event: ToolEvent{Tool: "shell", Shell: "process.env.SECRET"}, want: HardStopSecret},
		{name: "source filename is not a secret token", event: ToolEvent{Tool: "Read", Target: "/repo/secrets_test.go"}, want: ""},
		{name: "financial source filename is not a money token", event: ToolEvent{Tool: "Read", Target: "/repo/internal/charge.go"}, want: ""},
		{name: "outbound message", event: ToolEvent{Tool: "slack", Arguments: []string{"send", "report"}}, want: HardStopOutsideMessage},
		{name: "compound mcp slack send tool", event: ToolEvent{Tool: "mcp__slack__send_message"}, want: HardStopOutsideMessage},
		{name: "compound mcp stripe payment tool", event: ToolEvent{Tool: "mcp__stripe__payment_intents_create"}, want: HardStopMoney},
		{name: "github issue comment mutation", event: ToolEvent{Tool: "mcp__github__create_issue_comment"}, want: HardStopOutsideMessage},
		{name: "gitlab merge request note mutation", event: ToolEvent{Tool: "mcp__gitlab__create_merge_request_note"}, want: HardStopOutsideMessage},
		{name: "github comment listing is not a message mutation", event: ToolEvent{Tool: "mcp__github__list_issue_comments"}, want: ""},
		{name: "compound adapter text in shell filename stays opaque", event: ToolEvent{Tool: "shell", Shell: "cat /tmp/mcp__slack__send_message.log"}, want: ""},
		{name: "compound adapter text in argv filename stays opaque", event: ToolEvent{Tool: "shell", Arguments: []string{"cat", "/tmp/mcp__stripe__payment_intents_create.txt"}}, want: ""},
		{name: "message substrings do not match", event: ToolEvent{Tool: "shell", Shell: "go test ./internal/slack/send-helper_test.go"}, want: ""},
		{name: "inline post to a messaging service", event: ToolEvent{Tool: "shell", Shell: "python3 -c \"requests.post('https://hooks.slack.com/services/x', json=m)\""}, want: HardStopOutsideMessage},
		{name: "inline post to an unnamed URL is not a message", event: ToolEvent{Tool: "shell", Shell: "python3 -c 'requests.post(url)'"}, want: ""},
		{name: "a script editing text that names a messaging call is not a message", event: ToolEvent{Tool: "shell", Shell: "python3 - <<'EOF'\ns=open('t.go').read().replace('a', 'requests.post(\"https://hooks.slack.com/x\")')\nEOF"}, want: ""},
		{name: "curl post to a messaging service", event: ToolEvent{Tool: "shell", Shell: "curl -X POST https://discord.com/api/webhooks/1/x"}, want: HardStopOutsideMessage},
		{name: "curl post to a local service is not a message", event: ToolEvent{Tool: "shell", Shell: "curl -X POST http://127.0.0.1:8080/api/v1/runs -d @run.json"}, want: ""},
		{name: "curl post to an API is not a message", event: ToolEvent{Tool: "shell", Shell: "curl -X POST https://api.example.com/v1/search -d q=x"}, want: ""},
		{name: "a host name inside a longer host does not match", event: ToolEvent{Tool: "shell", Shell: "curl -X POST https://notslack.com.example.org -d x"}, want: ""},
		{name: "fetch post to a messaging service", event: ToolEvent{Tool: "shell", Shell: "node -e \"fetch('https://api.telegram.org/bot/sendMessage',{method:'POST'})\""}, want: HardStopOutsideMessage},
		{name: "encoded shell effect is unknown protected", event: ToolEvent{Tool: "shell", Shell: "base64 -d | bash"}, want: HardStopUnknownEffect},
		{name: "inline generic send is not a stop", event: ToolEvent{Tool: "shell", Shell: "python3 -c 'client.send(payload)'"}, want: ""},
		{name: "publishing a pull request is an outbound message", event: ToolEvent{Tool: "shell", Shell: "gh pr create --title x --body y"}, want: HardStopOutsideMessage},
		{name: "viewing a pull request is not", event: ToolEvent{Tool: "shell", Shell: "gh pr view 12"}, want: ""},
		{name: "inline local deletion is not a guard stop", event: ToolEvent{Tool: "shell", Shell: "python3 -c 'os.remove(path)'"}, want: ""},
		{name: "decoding to a file is not hidden", event: ToolEvent{Tool: "shell", Shell: "base64 -d blob.b64 > blob.bin"}, want: ""},
		{name: "shell syntax check is not script execution", event: ToolEvent{Tool: "shell", Shell: "bash -n ./scripts/send.sh"}, want: ""},
		{name: "camel case payment sdk method is money", event: ToolEvent{Tool: "shell", Shell: "stripe.paymentIntents.create({amount:100})"}, want: HardStopMoney},
		{name: "git checkout is not a payment operation", event: ToolEvent{Tool: "shell", Shell: "git checkout README.md"}, want: ""},
		{name: "provider checkout remains a money hard stop", event: ToolEvent{Tool: "shell", Shell: "stripe checkout sessions create"}, want: HardStopMoney},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := hardStopForEvent(test.event); got != test.want {
				t.Fatalf("hardStopForEvent(%+v) = %q, want %q", test.event, got, test.want)
			}
		})
	}
}

var opaqueExecutionEvents = []struct {
	name  string
	shell string
}{
	{name: "unquoted heredoc command substitution", shell: "cat > /tmp/agentharness-output <<EOF\n$(./tool)\nEOF"},
	{name: "unquoted heredoc secret expansion", shell: "cat > /tmp/agentharness-output <<EOF\n${API_TOKEN}\nEOF"},
	{name: "file-backed python", shell: "python3 /tmp/send.py"},
	{name: "file-backed node", shell: "node ./scripts/pay.js"},
	{name: "file-backed shell", shell: "bash ./scripts/send.sh"},
	{name: "file-backed interpreter in nested shell", shell: "bash -c 'python3 ./scripts/send.py'"},
	{name: "file-backed interpreter in command substitution", shell: "echo $(node ./scripts/pay.js)"},
	{name: "python module run", shell: "python3 -m pytest steward/tests"},
}

func TestHeredocWriteToSymlinkedSpecialPathIsNotAllowedAsData(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "dev-link")
	if err := os.Symlink("/dev", link); err != nil {
		t.Fatal(err)
	}
	event := ToolEvent{Tool: "shell", WorkingDirectory: root, Shell: "cat > " + filepath.Join(link, "tcp", "example.invalid", "443") + " <<'EOF'\nhello\nEOF"}
	if got := hardStopForEvent(event); got != HardStopUnknownEffect {
		t.Fatalf("symlinked special endpoint classification = %q, want %q", got, HardStopUnknownEffect)
	}
}

func guardTestRuntime(t *testing.T, now time.Time) (Runtime, *GuardModeStore) {
	t.Helper()
	store := NewBundleStore(t.TempDir())
	modes := NewGuardModeStore(store.Dir)
	modes.Now = func() time.Time { return now }
	previous := GuardGrantCheck
	GuardGrantCheck = func(string) bool { return false }
	t.Cleanup(func() { GuardGrantCheck = previous })
	return Runtime{Profile: ProfileAdvisory, Store: store, GuardModes: modes, Now: func() time.Time { return now }}, modes
}

func guardEvents(t *testing.T, dir string) []GuardEvent {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, guardEventsFile))
	if err != nil {
		t.Fatal(err)
	}
	var events []GuardEvent
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var event GuardEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatal(err)
		}
		events = append(events, event)
	}
	return events
}

var chatSend = ToolEvent{Runner: "codex", Tool: "shell", Shell: "slack send '#ops' 'deploy done'", Context: map[string]string{"session_id": "s1"}}

// A blocked agent learns what stopped it, why, what to use instead and the
// exact request for approval; the block is logged for the steward.
func TestBlockedGuardExplainsItselfAndIsLogged(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	runtime, modes := guardTestRuntime(t, now)
	decision, err := runtime.Evaluate(chatSend)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != ActionDeny || decision.EvaluatorID != "agent-guard:outside-message" || decision.ProviderID != "agent-guard" {
		t.Fatalf("decision = %+v", decision)
	}
	for _, want := range []string{"messaging the outside world", "sends a chat or email message", "Instead:", "notification-hub", "vrooli agent authorize \"$VROOLI_AGENT_SESSION\" --policy agent-guard-override --target agent-guard:outside-message"} {
		if !strings.Contains(decision.Reason, want) {
			t.Errorf("reason lacks %q:\n%s", want, decision.Reason)
		}
	}
	events := guardEvents(t, modes.Dir)
	if len(events) != 1 || events[0].Outcome != "blocked" || events[0].Check != "agent-guard:outside-message" || events[0].Session != "s1" || events[0].Runner != "codex" || events[0].Command == "" {
		t.Fatalf("logged events = %+v", events)
	}
}

// The operator's log-only mode for a guard lets the action through, and the
// log keeps what it would have stopped. Nothing withdraws the relaxation.
func TestLogOnlyGuardLetsTheActionThroughAndLogsIt(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	runtime, modes := guardTestRuntime(t, now)
	if err := modes.SetLogOnly("agent-guard:outside-message", now.Add(7*24*time.Hour), "detection is being fixed", "operator"); err != nil {
		t.Fatal(err)
	}
	decision, err := runtime.Evaluate(chatSend)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action == ActionDeny {
		t.Fatalf("log-only guard still denied: %+v", decision)
	}
	if events := guardEvents(t, modes.Dir); len(events) != 1 || events[0].Outcome != "log_only" {
		t.Fatalf("logged events = %+v", events)
	}
	money := ToolEvent{Runner: "codex", Tool: "stripe", Arguments: []string{"charge"}}
	if decision, _ := runtime.Evaluate(money); decision.Action != ActionDeny {
		t.Fatalf("money guard was relaxed too: %+v", decision)
	}
	if _, active, err := modes.ModeFor("agent-guard:outside-message", now); err != nil || !active {
		t.Fatalf("relaxation was withdrawn: active=%t err=%v", active, err)
	}
}

// An approved per-agent grant covers its guard only.
func TestApprovedGrantLetsTheAgentThrough(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	runtime, modes := guardTestRuntime(t, now)
	GuardGrantCheck = func(check string) bool { return check == "agent-guard:secret" }
	read := ToolEvent{Runner: "claude-code", Tool: "Read", Target: "/repo/.env"}
	if decision, _ := runtime.Evaluate(read); decision.Action == ActionDeny {
		t.Fatalf("granted secret read denied: %+v", decision)
	}
	if decision, _ := runtime.Evaluate(chatSend); decision.Action != ActionDeny {
		t.Fatalf("grant for secrets allowed a message: %+v", decision)
	}
	events := guardEvents(t, modes.Dir)
	if len(events) != 2 || events[0].Outcome != "granted" || events[1].Outcome != "blocked" {
		t.Fatalf("logged events = %+v", events)
	}
}

// A policy file that does not load must never stop work: a crashed or
// all-blocking guard stops every agent.
func TestBrokenPolicyStopsNothing(t *testing.T) {
	if _, err := loadHardStops([]byte(`{"guards": {}}`)); err == nil {
		t.Fatal("an incomplete policy must be refused")
	}
	previous := hardStopsErr
	hardStopsErr = errors.New("broken")
	t.Cleanup(func() { hardStopsErr = previous })
	if got := hardStopForEvent(chatSend); got != "" {
		t.Fatalf("broken policy still stopped: %q", got)
	}
}

// A verified credential covers a guard only when its grant names the guard
// override operation and this guard (or every guard), and is unexpired.
func TestGrantCoversOnlyItsGuardUntilExpiry(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	grant := func(policy, target string, expires time.Time) string {
		raw, _ := json.Marshal(map[string]any{"policy": policy, "operations": []string{"agent-guard.override"}, "targets": []string{target}, "expires_at": expires})
		return string(raw)
	}
	later := now.Add(time.Hour)
	for _, test := range []struct {
		name, raw, check string
		want             bool
	}{
		{"exact guard", grant("agent-guard-override", "agent-guard:secret", later), "agent-guard:secret", true},
		{"every guard", grant("agent-guard-override", "agent-guard:*", later), "agent-guard:money", true},
		{"another guard", grant("agent-guard-override", "agent-guard:secret", later), "agent-guard:money", false},
		{"expired", grant("agent-guard-override", "agent-guard:secret", now), "agent-guard:secret", false},
		{"another policy", grant("root-state-transition", "agent-guard:secret", later), "agent-guard:secret", false},
		{"no grant", "", "agent-guard:secret", false},
	} {
		if got := grantCovers(test.raw, test.check, now); got != test.want {
			t.Errorf("%s: covers=%t, want %t", test.name, got, test.want)
		}
	}
}

// Every stop an agent meets reaches the steward's log, not only guard stops:
// the deletion floor and operator-only commands are logged under agent-floor
// IDs, which no log-only mode or guard grant relaxes.
func TestFloorStopsAreLoggedForTheSteward(t *testing.T) {
	f := newRemovalFixture(t)
	store := NewBundleStore(filepath.Join(t.TempDir(), "policy"))
	runtime := Runtime{Profile: ProfileAdvisory, Store: store, Removal: func(cwd string) RemovalContext { return f.context(cwd) }}
	decision, err := runtime.Evaluate(ToolEvent{Runner: "codex", Tool: "Bash", Shell: "rm -rf /etc", WorkingDirectory: f.repo, Context: map[string]string{ContextSessionID: "s9"}})
	if err != nil || decision.Action != ActionDeny {
		t.Fatalf("decision = %+v, err = %v", decision, err)
	}
	if _, err := runtime.Evaluate(ToolEvent{Runner: "codex", Tool: "Bash", Shell: "ls", WorkingDirectory: f.repo}); err != nil {
		t.Fatal(err)
	}
	events := guardEvents(t, store.Dir)
	if len(events) != 1 || !strings.HasPrefix(events[0].Check, "agent-floor:deletion-") || events[0].Outcome != "blocked" || events[0].Session != "s9" || events[0].Signal == "" {
		t.Fatalf("logged events = %+v", events)
	}
	for code, want := range map[string]string{"REMOVAL_OUTSIDE_REPO": "deletion-outside-repo", "OPERATOR_ONLY_COMMAND": "operator-only", "PROVIDER_UNAVAILABLE": "provider-unavailable"} {
		if got := floorName(code); got != want {
			t.Errorf("floorName(%s) = %s, want %s", code, got, want)
		}
	}
}

// The guard log never keeps a credential value a command carried.
func TestGuardLogRedactsCredentialValues(t *testing.T) {
	for input, want := range map[string]string{
		"curl -H 'Authorization: Bearer abcdef1234567890' https://hooks.slack.com/x": "curl -H 'Authorization: Bearer ***' https://hooks.slack.com/x",
		"SLACK_TOKEN=xoxb-1234567890-abcdef curl x":                                  "SLACK_TOKEN=*** curl x",
		"gh auth login --with-token ghp_abcdefghijklmnopqrstuvwxyz123456":            "gh auth login --with-token ***",
		`tool --api-key "s3cr3t value" run`:                                          "tool --api-key *** run",
		"grep -rn api_key scenarios/":                                                "grep -rn api_key scenarios/",
	} {
		if got := redactSecrets(input); got != want {
			t.Errorf("redactSecrets(%q) = %q, want %q", input, got, want)
		}
	}
}
