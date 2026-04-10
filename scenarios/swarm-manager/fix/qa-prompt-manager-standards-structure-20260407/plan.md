# Implementation Plan: Resolve prompt-manager standards — required structure

## Purpose

Eliminate the critical "Scenario Required Structure" standards violation for `prompt-manager` by rebuilding the CLI binary with the correct name (`cli/prompt-manager` instead of `cli/pm`). This is a targeted naming fix — the binary source code is unchanged.

## Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

- `scenarios/prompt-manager/cli/main.go` — CLI entry point
- `scenarios/prompt-manager/cli/app.go` — CLI command definitions
- `scenarios/prompt-manager/cli/install.sh` — installer (already uses `--name "prompt-manager"`)
- `scenarios/scenario-auditor/api/rules/structure/required_layout.go:112-116` — the rule that checks for `cli/<scenario-name>`

## Problem Statement

The scenario-auditor's `Scenario Required Structure` rule expects a CLI binary at `cli/<scenario-name>`. For `prompt-manager`, this means `cli/prompt-manager` must exist.

Currently the CLI binary is committed as `cli/pm` — a short alias that does not match the scenario name. This causes a critical standards violation that blocks the standards audit.

## Scope

### In Scope
- Rebuild the CLI binary as `cli/prompt-manager` (same Go source, different output name)
- Remove the old `cli/pm` binary from git tracking
- Verify the standards audit passes for this specific violation

### Out of Scope
- Other standards violations (P0 target requirements, docs, unit tests, etc.)
- Changes to `install.sh` (already correctly uses `--name "prompt-manager"`)
- Changes to `service.json` (no CLI binary path referenced there)
- Refactoring CLI source code

## Current Technical Context

| File / Path | Role |
|---|---|
| `scenarios/prompt-manager/cli/pm` | Current binary (8.3 MB ELF Go executable, tracked in git) |
| `scenarios/prompt-manager/cli/main.go` | Entry point — `package main` |
| `scenarios/prompt-manager/cli/app.go` | CLI command definitions |
| `scenarios/prompt-manager/cli/go.mod` | Module definition |
| `scenarios/prompt-manager/cli/install.sh` | Installer — already passes `--name "prompt-manager"` |

Other scenarios follow the `cli/<scenario-name>` convention correctly:
- `swarm-manager/cli/swarm-manager`
- `scenario-auditor/cli/scenario-auditor`

## Target End State

- `scenarios/prompt-manager/cli/prompt-manager` exists as a working binary
- `scenarios/prompt-manager/cli/pm` no longer exists in the working tree or git index
- `scenario-auditor audit prompt-manager --standards-only` no longer reports the "Scenario Required Structure" critical violation for `cli/prompt-manager`
- All existing CLI functionality is preserved (same Go source, just different output name)

## Greenfield Declaration

This is a rename/rebuild of a single binary. No compatibility shims, aliases, or symlinks for the old `pm` name are needed. The installed binary in `~/.vrooli/bin/` is already named `prompt-manager` via `install.sh`.

## Implementation Strategy

### Phase 1: Rebuild and replace the CLI binary

1. Rebuild with the correct output name:
   ```bash
   cd scenarios/prompt-manager/cli && GOWORK=off go build -o prompt-manager .
   ```
2. Remove the old binary and stage the new one:
   ```bash
   git rm scenarios/prompt-manager/cli/pm
   git add scenarios/prompt-manager/cli/prompt-manager
   ```

This is a single-phase fix. No phased rollout is needed.

## Contract Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Binary naming | `cli/prompt-manager` (match scenario name) | Follows established convention across all scenarios |
| Git tracking | Track the binary in git | Consistent with other scenarios (swarm-manager, scenario-auditor) |
| Old `cli/pm` | Remove completely, no alias/symlink | Greenfield — `install.sh` already installs as `prompt-manager` |

These decisions were confirmed in workshop round-001 (decisions d1=A, d2=A).

## Testing Plan

All tests are automated verification commands:

1. **Binary exists and is executable:**
   ```bash
   test -x scenarios/prompt-manager/cli/prompt-manager && echo PASS || echo FAIL
   ```

2. **Binary runs correctly:**
   ```bash
   scenarios/prompt-manager/cli/prompt-manager --help
   # Expected: exits 0, prints help text
   ```

3. **Old binary removed:**
   ```bash
   test ! -f scenarios/prompt-manager/cli/pm && echo PASS || echo FAIL
   ```

4. **Standards audit passes for this violation:**
   ```bash
   scenario-auditor audit prompt-manager --standards-only --timeout 60
   # Expected: "Scenario Required Structure" violation at cli/prompt-manager no longer present
   ```

## Rollout / Validation Checklist

- [ ] Rebuild binary: `cd scenarios/prompt-manager/cli && GOWORK=off go build -o prompt-manager .`
- [ ] Remove old binary: `git rm scenarios/prompt-manager/cli/pm`
- [ ] Stage new binary: `git add scenarios/prompt-manager/cli/prompt-manager`
- [ ] Verify: `scenarios/prompt-manager/cli/prompt-manager --help` exits 0
- [ ] Verify: `scenario-auditor audit prompt-manager --standards-only --timeout 60` — no "Scenario Required Structure" critical violation
- [ ] Run: `vrooli scenario restart prompt-manager` to confirm scenario health with renamed binary

## Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Other code references `cli/pm` path directly | Low | Low | Grep for `cli/pm` references before committing; `install.sh` already uses `--name prompt-manager` |
| Binary won't build due to dependency issues | Low | Medium | Same Go source that produced `pm`; only the output name changes |
| Stale binary if Go deps have changed | Low | Low | Rebuild ensures binary matches current source |

## Non-goals / Prohibited Patterns

- **No symlinks or aliases** from `pm` → `prompt-manager`. Clean break.
- **No changes to CLI source code** — this is purely a build output rename.
- **No changes to `install.sh`** — it already works correctly.
- **Do not address other standards violations** (P0 targets, docs, unit tests) — those are separate backlog items.

## Definition of Done

1. `scenarios/prompt-manager/cli/prompt-manager` exists and is executable
2. `scenarios/prompt-manager/cli/pm` does not exist
3. `scenarios/prompt-manager/cli/prompt-manager --help` exits 0
4. `scenario-auditor audit prompt-manager --standards-only` does not report "Scenario Required Structure" critical violation for `cli/prompt-manager`
5. `vrooli scenario restart prompt-manager` completes without error
6. Greenfield: no compatibility shims, aliases, or symlinks exist for the old name
