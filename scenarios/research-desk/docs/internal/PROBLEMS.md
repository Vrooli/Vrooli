# Problems — Research Desk

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-11-12 — not built or not proven

- **Web Console credential client: not built.** Automatic launch uses Web Console's local
  availability policy; `web_console_access` says so on every launch. Real fix: the desk's
  service principal (`scripts/issue-service-principal.sh`, run by the operator) and a client
  in `webConsoleCredential` (`api/wiring.go`).
- **Remaining-quota lens: not built.** `limited` shows only once Codex has hit a limit (goal
  status `usage_limited`, `budget_limited` or `blocked`), never before a launch. Real fix: a
  quota read from the owner of usage data, shown in preflight. Owner: unassigned.
- **Seen live once, on the operator's debt-crypto-models-20261112 launch (2026-11-12
  06:01Z).** Automatic launch (goal verified in 26 s), delivery of two needs-you items as Hub
  notices (06:05:16Z, read by the supervisor), and a supersede stop: programme-four-20261110
  derives `ended` with no Codex window left for its session. Launches since then record each
  stop's outcome in `LAUNCH.json` `controls` ("window stopped" or "no window found").
- **Not proven live, and staying so until the operator acts.** Proven only against fakes
  (`internal/courier`, `internal/launch` tests): a decision ask, and an answer coming back
  from the Hub into `OPERATOR.md` (no decision event has been published); Say, Answer, Pause,
  Resume and Stop on a real commission; a relaunch; an optional scenario being down.
  `courier-test` is the live decision check and runs only when the operator asks.
- **Hub ask state: not built.** The desk never asks Notification Hub whether an ask was
  answered, expired or withdrawn; an item that says to answer a Hub ask shows that instruction,
  and the answer reaches the desk only when written into `OPERATOR.md`. Real fix: a read of
  ask state from Notification Hub, shown on the item. Owner: unassigned.
- **Stop test: passed (2026-11-12, 07:09:26Z to 07:10:52Z).** Stopping the desk changed
  nothing for the running debt-crypto-models-20261112 commission. While it was stopped, all
  five skills were read through `prompt-manager` and by path, `packet.py preview` and `check`
  and `mission_check.py` ran, the supervisor stayed alive, and an owner wrote its checkpoint
  (07:10:23Z). Log: [`evidence/stop-test.txt`](evidence/stop-test.txt).
- **`--supersede` and `--alongside` have no request field of their own.** On `preflight` and `build` both fill `resolutions` (a repeated message), so the contracts phase reports `binding.arg_unmapped` and a program caller cannot fill them.
  Real fix: requests carry `repeated string supersede` and `repeated string alongside` in place of `resolutions` (a wire change for the API, CLI and UI). Owner: research-desk.
- **Storage check misreads a foreign read-only database.** `api/internal/home/codex.go` opens Codex's own `state_5.sqlite` with `mode=ro`; storage reports `RAW_SQL_OPEN` and `ROUTED_SEAMS_UNWIRED`
  because no declaration marks a foreign read-only read, and the scenario database seam must not carry it. Real fix: such a declaration in storage-manager. Owner: storage-manager.
- **Accepted limit: a caller with no token passes as the operator.** A request from this host
  with no token, an agent's included, is the operator; a failed verification never falls back
  to operator access. Real fix: none planned; any local process can already read and write
  the research home.
- **Old skill paths still linked.** `scenarios/prompt-manager/store/skills/packs/core/` holds
  links for `research-coordination`, `adaptive-research` and `local-experimentation` because
  launched charters name the old paths. Real fix: delete the three links when `research-desk
  commissions` shows no `live`, `paused` or `limited` commission whose charter names an old
  path.

### 2026-11-11 — generated scaffold lacked committed CLI evidence

**Symptom:** the first unit phase failed `TestPrimitiveEvidenceArtifactCurrent` because
`.vrooli/generated/cli-primitive-evidence.json` did not exist.

**Root cause:** `template-manager generate` does not write the CLI primitive evidence file the
CLI tests require (template defect, outside this scenario).

**Workaround:** regenerated with `UPDATE_CLI_EVIDENCE=1 go test -run
TestPrimitiveEvidenceArtifactCurrent .` in `cli/`.

**Real fix:** the react-vite template generates the evidence at scaffold time.

**Owner:** template-manager.

### 2026-11-11 — dependencies phase exceeds its 15-minute deadline

**Symptom:** `vrooli scenario test research-desk --phases dependencies` failed with "phase
dependencies did not complete before its 15m0s deadline" (run 20261111-212302-cd9f5c02).

**Root cause:** unknown; not diagnosed (shared test infrastructure).

**Workaround:** validate with the `structure`, `unit` and `docs` phases.

**Owner:** unassigned.

### 2026-11-12 — four unused UI dependencies stay in the manifest

**Symptom:** `ui/package.json` lists `@radix-ui/react-slot`, `class-variance-authority`,
`clsx` and `tailwind-merge`; nothing in `ui/src` imports them.

**Root cause:** template leftovers; `scenario-dependency-analyzer deps remove` removes Go
requirements only, and raw package managers are not allowed.

**Real fix:** a governed JavaScript remove verb in scenario-dependency-analyzer, then remove
the four. Owner: scenario-dependency-analyzer.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
