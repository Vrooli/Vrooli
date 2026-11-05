package backup

import (
	"fmt"
	"strings"
)

// RunbookEntry is one situation an operator may face and the owner
// operation that resolves it. The recovery runbook document is rendered from
// this table by a test so the procedure and the verbs cannot drift apart.
type RunbookEntry struct {
	Situation     string
	OwnerVerb     string
	Argv          string
	Preconditions []string
	Refuses       []string
	Evidence      string
}

// Runbook is the ordered owner-verb table.
var Runbook = []RunbookEntry{
	{
		Situation: "Capture a recovery point before a destructive schema change or activation",
		OwnerVerb: "cloud-target:data.backup",
		Argv:      "vrooli cloud-target data backup --deployment <id> --operation <op> --step data.backup --fence <n> --binding '<json>'... --key-ref <logical_id:field> --schema-version <v> --configuration-digest <d> --credential-version-ref <ref>... --provider data-backup-manager --migration-posture <posture>",
		Preconditions: []string{
			"Every binding is declared in the closure persistent_data with an owner and a provider; a database is captured only by its owner's database-native tooling (resources/postgres deployment.backup).",
			"The recovery key reference resolves through the credential authority on the operator side, never on the target being protected.",
			"The application quiesce hook is declared when the scenario owns migrations; otherwise the point records write_quiescence not_declared (snapshot_safe for database-native providers).",
		},
		Refuses:  []string{"fence_stale", "receipt_input_mismatch", "backup_provider_unavailable", "recovery_key_unavailable", "invalid_argument"},
		Evidence: "Receipt operations/<op>/data.backup.json and recovery-points/<id>/recovery-point.json (bindings, refs, checksums, consistency token, digest).",
	},
	{
		Situation: "Verify a recovery point (checksums, key availability, expected inventory) without restoring",
		OwnerVerb: "cloud-target:data.verify",
		Argv:      "vrooli cloud-target data verify --deployment <id> --recovery-point <id> [--open] [--expect '<json>']",
		Preconditions: []string{
			"--open resolves the recovery key and decrypts every artifact: run it from the operator side to prove the key lives outside the target failure domain.",
		},
		Refuses:  []string{"recovery_point_corrupt", "recovery_key_unavailable", "verify_failed"},
		Evidence: "Verify report: artifacts_intact, key_resolved, artifacts_opened, invariants[].",
	},
	{
		Situation: "Restore a recovery point onto a replacement host",
		OwnerVerb: "cloud-target:data.restore",
		Argv:      "vrooli cloud-target data restore --deployment <id> --operation <op> --step data.restore --fence <n> --recovery-point <id> --into <binding>=<locator>...",
		Preconditions: []string{
			"The replacement host holds the release (release verify + stage) and the credentials the workload needs (credentials.provision) before data is restored.",
			"Every target binding is clean: an empty directory or a database without user tables. Restore never overwrites operator-owned data.",
			"The workload is stopped or not yet started; restoring under writes is not a supported path.",
		},
		Refuses:  []string{"recovery_point_corrupt", "recovery_key_unavailable", "restore_target_not_clean", "fence_stale"},
		Evidence: "Receipt operations/<op>/data.restore.json with measured_rto_ms, recovery_point_age_ms and per-binding captured/restored inventories; the cloud restore receipt records the same against certification/budgets.json.",
	},
	{
		Situation: "Roll back code when the predecessor cannot read the current schema",
		OwnerVerb: "scenario-to-cloud:rollback-admission (api/backup.EvaluateRollback)",
		Argv:      "POST /api/v1/deployments/{id}/recovery-points/rollback-admission {target_schema, current_schema, readable_by[]}",
		Preconditions: []string{
			"The scenario declares deployment.recovery.{code_rollback, schema_strategy}; admission fails closed when a schema version is unobserved.",
		},
		Refuses:  []string{"rollback_incompatible (carries a typed forward_repair or restore plan)"},
		Evidence: "RollbackVerdict {compatible, reason_code, plan{kind, recovery_point_id, preconditions[], steps[]}}.",
	},
	{
		Situation: "Prune recovery points under a retention policy",
		OwnerVerb: "scenario-to-cloud:recovery-points.prune (api/backup.Service.Prune)",
		Argv:      "POST /api/v1/deployments/{id}/recovery-points/prune {keep_last, max_age_seconds}",
		Preconditions: []string{
			"Points referenced by the active or retained release, by a non-terminal operation or by a pin are protected and never deleted; the plan names their holders.",
		},
		Refuses:  []string{"recovery_point_protected"},
		Evidence: "PrunePlan {keep[], delete[], protected{id: holders[]}}.",
	},
}

// RenderRunbook renders the recovery runbook markdown from the verb table.
func RenderRunbook() string {
	var b strings.Builder
	b.WriteString("# Recovery runbook\n\n")
	b.WriteString("Generated from `api/backup.Runbook` by `TestRunbookDocumentIsRendered`; edit the table, not this file.\n\n")
	b.WriteString("Code rollback and data restoration are separate operations with different prerequisites. A backup receipt is not proof that the application can restore from it; only a restore receipt with passing invariants is. Every verb below is fenced and receipted per (operation, step) on the target, prints typed JSON, and exits 0 ok / 1 failed / 2 refused.\n\n")
	b.WriteString("| Situation | Owner verb | Refuses |\n|---|---|---|\n")
	for _, e := range Runbook {
		fmt.Fprintf(&b, "| %s | `%s` | %s |\n", e.Situation, e.OwnerVerb, "`"+strings.Join(e.Refuses, "`, `")+"`")
	}
	for i, e := range Runbook {
		fmt.Fprintf(&b, "\n## %d. %s\n\n", i+1, e.Situation)
		fmt.Fprintf(&b, "Owner verb: `%s`\n\n```\n%s\n```\n\n", e.OwnerVerb, e.Argv)
		b.WriteString("Preconditions:\n\n")
		for _, p := range e.Preconditions {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		fmt.Fprintf(&b, "\nRefusals: `%s`\n\nEvidence: %s\n", strings.Join(e.Refuses, "`, `"), e.Evidence)
	}
	b.WriteString("\n## Budgets\n\n")
	b.WriteString("Measured against `certification/budgets.json` qualification: `fresh_host_restore_seconds_max` (RTO) and `backup_recovery_point_seconds_max` (recovery-point age at restore start). A weaker budget may not be introduced after a failed drill.\n")
	b.WriteString("\n## Recovery keys\n\n")
	b.WriteString("Recovery points are sealed with AES-256-GCM (`internal/credentialpolicy.Seal`) under a key derived (PBKDF2-SHA256) from material resolved through the credential authority by reference (`logical_id:field`). The reference is recorded on the manifest and every receipt; the material never is. A restore on a replacement host needs the reference to resolve there, which is why the key lives with the operator's credential store, not on the protected target.\n")
	return b.String()
}
