# Template Factory Guide

Template Manager owns the operational loop for scenario templates, design kits,
and resource templates. This guide is for agents and engineers maintaining the
factory itself, not for generated-scenario authors.

> Cutover note: command examples use the Template Manager surfaces planned for
> the hard cutover phases. Until those phases land, the equivalent
> `vrooli scenario ...` commands remain the live engine behind the additive
> APIs.

## Validation

Use shallow validation for routine template edits:

```bash
template-manager template validate --template react-vite --mode shallow
```

Shallow validation checks the source template contract without generating a
throwaway scenario. It is the default gate before landing documentation,
manifest, or orientation changes.

Use deep validation for first-run-sensitive changes:

```bash
template-manager template validate --template react-vite --mode deep --test-preset comprehensive
```

Deep validation generates an isolated scenario, runs post hooks, and executes
test-genie against the generated scenario. Treat it as the release-candidate
gate for template changes that affect setup, UI boot, proto relocation, or
test policy.

Persist additive validation evidence through Template Manager:

```bash
template-manager runs run --template react-vite --mode shallow
template-manager runs show RUN_ID
```

## Drift

Fleet drift compares generated scenarios against their recorded template
provenance and current template hashes:

```bash
template-manager runs drift-record
template-manager runs drift --template react-vite
```

Use drift snapshots to decide whether debt is isolated to one scenario or is
inherited from the factory. A repeated drift finding should become a debt entry
with a stable key so reruns update the same row.

## Cleanup

Deep validation can retain temporary workspaces for debugging. Remove retained
runs after inspection:

```bash
template-manager template cleanup --run RUN_ID
```

Cleanup must be marker-backed: delete only workspaces that carry the validation
run marker reported by the engine.

## Changelog And Migration Protocol

`templates/scenarios/react-vite/template.json::version` and
`templates/scenarios/react-vite/CHANGELOG.md` are a lockstep contract.

When a generated scenario records an older
`.vrooli/service.json::generation.template.version`, read every changelog entry
greater than that version in descending file order until reaching the recorded
version. Apply each entry's **Migration** checklist, then update the scenario
provenance to the latest template version after validation passes.

Maintainers follow these rules:

- every template version bump has exactly one matching changelog entry
- entries are ordered newest to oldest
- migration bullets are concrete and verifiable
- changelog holes are template debt, not documentation polish

## Debt Ledger Workflow

Record inherited template defects as Template Manager debt entries instead of
debugging them anew in each generated scenario:

```bash
template-manager debt list --template react-vite
template-manager debt show STABLE_KEY
```

Stable keys should describe the factory defect, not the scenario where it was
first noticed. For example, use `react-vite.router.future-flags-missing`
rather than a generated scenario path. Validation and drift reruns should update
the same debt entry's last-seen timestamp instead of creating duplicates.

Close source debt only after the template source is fixed, shallow validation
passes, and at least one generated-scenario proof shows the defect no longer
appears. Test Genie deep-validation summaries are different: their canonical
failure-class entry represents the latest terminal deep run, so a later
terminal deep run supersedes the older summary while preserving it as resolved
history. This prevents changing provider prose from creating fresh debt rows.

## Orientation Guidance

Use the guidance surface to turn orientation into a small-model work order:

```bash
template-manager guidance next template-manager --json
```

The response includes the next incomplete gate, check status, contract docs, and
remediation pointers. Generated scenarios remain self-contained, but factory
agents should prefer this structured surface over re-reading START-HERE prose
for every step.

## Orientation Gate Checks

Template Manager owns the shared orientation check vocabulary. Individual
templates keep their gate data in `template.json`; do not add per-template Go
branches for one-off gates.

Supported checks:

- `file_exists`, `file_absent`, `directory_exists`: path existence checks.
- `glob_present`, `glob_absent`, `glob_min_count`: path glob checks; count
  checks require `minCount`.
- `json_path_exists`, `json_min_entries`: dotted JSON path checks; entry-count
  checks require `minCount` and the selected value must be an array or object.
- `text_contains`, `text_absent`, `text_absent_tree`: exact text checks for
  placeholder removal and residue scans.
- `command`: lifecycle command checks with an explicit timeout.

Use content-quality checks for generated documents that already exist at birth.
A gate should pass because the scenario made a durable decision or shipped real
behavior, not because the template copied a file into place.
