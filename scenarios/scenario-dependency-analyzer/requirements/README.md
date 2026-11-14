# Requirements Registry

Organize requirement modules by PRD operational targets, keeping the filesystem structure aligned with the "what" articulated in the PRD. Create folders such as `01-<target-name>/` as needed (numbers preserve ordering but do **not** imply priority).

## Lifecycle
1. Operational targets in PRD map to folders here.
2. `requirements/index.json` imports each module; tests auto-sync their status when they run.
3. Coverage summaries live in `coverage/phase-results/` after each test phase.

## Contributor Notes
- Add folders/modules that match your scenario’s PRD targets (P0/P1/P2) instead of reusing other scenarios’ names.
- Tag tests with `[REQ:ID]` so auto-sync can update status.
- After running targeted tests, re-run `vrooli scenario requirements sync <name>` so requirement statuses only change via auto-sync—never edit them by hand.
- `PRD.md` checkboxes flip automatically when sync runs; if you see a mismatch, rerun the suite instead of toggling markdown manually.
- Use `vrooli scenario requirements snapshot <name>` to review the last synced commands, operational-target completion counts, and any expiring manual validations before making changes.
- Run `vrooli scenario requirements lint-prd <name>` whenever you add or rename operational targets to ensure every `OT-P*-###` entry has at least one requirement (and that no requirement references a missing target).
- Never add compatibility shims (duplicate folders or alias imports) during migrations—let things fail temporarily instead of adding debt.
- Manual validations are a temporary escape hatch; if you must use one, record it with `vrooli scenario requirements manual-log <scenario> <REQ-ID>` so drift detection knows when it expires, then replace it with Vrooli Ascension workflows (`docs/testing/guides/ui-automation-with-bas.md`) or other automated phases so `scenario status` stays green.
- Keep this README short and link to shared docs (`docs/testing/guides/requirement-tracking-quick-start.md`) for schema details.

## Current Operational Targets

- Actual interface graph: compute scenario edges from upstream proto/import facts, not SDA source scanning.
- Drift accuracy: compare declared `service.json` scenario dependencies with actual evidence and preserve asymmetric severity.
- Programmatic seams: expose `DescribeInterfaceGraph` for planners and `health <scenario> --json` as Test Genie's single dependencies-phase producer.
- Dependency health: validate Code Facts-backed surfaces, runtime dependency status, package readiness, approved governance, release-age policy, Security Health index context, and graph drift without mutating dependency files.
- Governance memory: keep approved dependency records advisory by default, with dry-run mutation commands, review guidance, and Security Health evidence consumption for denied vulnerable ranges.
- Security boundary: Security Health owns scanners and security-phase findings; SDA dependency health reports index availability only.
- Storage cutover: keep derived graph on demand while moving history-bearing records to SQLite domain schemas.
