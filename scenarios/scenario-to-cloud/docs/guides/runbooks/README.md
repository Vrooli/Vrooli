# Operator runbooks

Six procedures, one per lifecycle task, each using only commands declared in
`cli/manifest.json` (rendered in `docs/reference/cli-commands.md`) or the
REST routes classified in `docs/reference/authorization-matrix.md`. The
conformance test `api/archtest/docs_test.go` fails the build when a fenced
`scenario-to-cloud …` or `vrooli cloud-target …` line here names a command
the manifest does not declare.

| Task | Runbook | Owner operation |
|---|---|---|
| First deployment onto a prepared target | [deploy.md](deploy.md) | `deployment plan` → `deployment apply` (one reviewed plan digest) |
| New release onto an existing deployment | [update.md](update.md) | recovery point → plan → apply → verify → rollback if needed |
| Credential rotation, revocation, store recovery | [rotate.md](rotate.md) | `credential rotate\|revoke\|recover` (values on stdin) |
| Data restore onto the same or a replacement host | [restore.md](restore.md) | `deployment recovery-points verify\|restore` |
| Alert or outage | [incident.md](incident.md) | diagnose → contain → recover → verify → record |
| Retire a deployment | [retire.md](retire.md) | `POST …/retire/plan` → `POST …/retire/apply` |

Conventions shared by every runbook:

- **One identity.** Resolve the deployment once and reuse the id. Every
  selector form is accepted (`--deployment <id>`, positional id,
  `--scenario <id> --environment <env>`, `--scenario <id> --domain <domain>`,
  `--scenario <id> --host <host>`); an ambiguous selector is refused with the
  candidates (exit 2) and nothing runs.
- **Preview is apply.** Every effect is a plan with a digest; the digest you
  reviewed is the digest you apply. A changed precondition between the two is
  `plan_stale` (exit 2), never a silent re-plan.
- **Wait once.** Operations are durable and server-owned. `operation wait`
  blocks once; exit 124 means the bound elapsed and the operation is
  unchanged: reattach with `operation resume <operation-id>`, do not re-issue.
- **Exit codes.** 0 done or `no_op`, 1 failed, 2 refused before any effect,
  3 pending or input required (the durable handoff is printed), 124 observer
  timeout.
- **No shell on the target.** Target effects are `vrooli cloud-target` verbs
  invoked by the operation owner with `--operation --step --fence`; nothing in
  these runbooks asks you to SSH in and type commands.
- **Values never on argv.** Credential values travel on standard input
  (`--value-stdin`, `--passphrase-stdin`) and are never printed, logged or
  stored in a plan.
- **REST when no verb exists.** Two procedures (retire, owner-wide
  reconciliation) have no CLI verb yet; they call the API with the same
  bearer the CLI uses:

  ```bash
  export STC_API=http://localhost:15672      # the scenario-to-cloud API
  export STC_TOKEN=<bearer with scenario-to-cloud:destructive>
  ```
