---
name: "safe-operator-script"
description: "Author preview-first, resumable, self-testing operator scripts for destructive host maintenance, and promote recurring work to a CLI command or governed program when it earns that rung."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["scripts", "maintenance", "safety", "dry-run", "operator", "repair", "resumable"]
  icon: "terminal"
  status: "active"
  revision: 6
  createdAt: "2026-11-19T00:00:00Z"
  updatedAt: "2026-11-19T12:00:00Z"
  requires:
    scenarios: ["prompt-manager"]
    commands: ["prompt-manager skill read", "bash -n", "shellcheck"]
  origin:
    kind: "authored"
---

## Practice focus: Safe Operator Scripts

Prioritize a script that proves it will not destroy the wrong thing, reports every refusal before it acts, and can be re-run after any interruption. A safe operator script is a reviewed runbook with an executable preflight, not a shell wrapper around a destructive command.

Required reading:
- `path:docs/agent-system/SKILL_AUTHORING.md` — universal quality bars and the promotion ladder citation rule.
- `path:docs/agent-system/SKILL_AUTHORING.md#promotion-ladder` — when a script should become a CLI command, Action, or governed program.
- `prompt-manager skill read writing-standards` — STE-100 applies to the script's help text and this skill's procedural prose.

### 1. Scope

In scope: one-off or rare destructive host maintenance that needs operator judgment, a verified safety copy, and a human confirmation. Examples: convert a filesystem, migrate a repository between disks, reclaim a wedged mount, rebuild a corrupted service volume.

Out of scope: deterministic operations that already have a CLI command; recurring multi-step workflows with stable inputs and joins; product behavior that belongs in a scenario API; policy decisions about whether maintenance is authorized.

### 1.1 Where the script lives

Scenario-private operator scripts live at `scenarios/<scenario>/.private/scripts/<name>.sh`. They are host-specific and are never repository source.

| Rule | Check |
|---|---|
| The scenario's `.gitignore` contains `.private/` | `git check-ignore -v scenarios/<scenario>/.private/scripts/<name>.sh` returns the rule. |
| The script is executable | `test -x scenarios/<scenario>/.private/scripts/<name>.sh` passes. |
| The script is not tracked | `git status --porcelain` does not list it. |
| The name states the operator task | `repair-drive.sh`, `reclaim-mount.sh`, not `script1.sh`. |

Create the directory and the ignore rule when they are absent. Do not commit a script that embeds host paths, device serials, or operator notes. If the script becomes reusable across hosts, promote it to a scenario CLI command or a governed program; do not move it into tracked `scripts/`.

Worked example: `scenarios/data-backup-manager/.private/scripts/repair-drive.sh`.

### 2. Choose the artifact first

| Situation | Artifact | Reason |
|---|---|---|
| One-off destructive host maintenance, operator present, judgment required | operator script | The work is rare, the risk is high, and the confirmation belongs to a person. |
| One deterministic operation many agents call | scenario CLI command | The operation has a stable contract, typed output, and tests. |
| Recurring multi-step workflow with stable joins and effects | governed program | The composition repeats; budgets, envelopes, and failure evidence belong to Program Runtime. |
| Judgment about when to act, what to refuse, and how to read evidence | skill | The decision is not a command and must stay transferable. |

A script that starts to encode product behavior is a promotion candidate. Record the candidate in the owning scenario's `docs/internal/PROBLEMS.md`; do not let the script become a second implementation.

Host remediation ownership is a repo rule: detection and remediation of host state belong in the control plane (`internal/`). A scenario script is a transitional operator tool. State that status in the script header and name the control-plane owner that should eventually absorb it.

### 3. The safe-script contract

Every row is a verifiable property. A script that misses a row is not ready for an operator.

| # | Property | Verification |
|---|---|---|
| 1 | Zero-argument entry is useful | Running with no arguments prints help and the available targets, then exits 0. It never prints a bare required-argument error. |
| 2 | Dry run is first-class | `--dry-run` runs every preflight check, prints a pass/fail table, changes nothing, and exits 2 when any check fails. It does not stop at the first failure. |
| 3 | Destructive default is preview | The script prints the plan and exits unless `--apply` is present. |
| 4 | One typed confirmation | The operator types a phrase that names the destructive target and the safety copy, for example `REPAIR /dev/sda1 USING /dev/sdb`. `--yes` is documented as unattended use, not as the default. |
| 5 | Identity re-read before mutation | The script re-reads the target's stable identity (serial, UUID, size, parent disk) immediately before the destructive command. A changed identity aborts. |
| 6 | Safety copy verified before mutation | The script checks structure, path and size, and a checksum pass (or documents why the checksum pass is optional) before it formats or deletes anything. |
| 7 | Refusals run before effects | The script refuses the system disk, a staging target on the same disk, insufficient space, an unsupported filesystem, and a target that already holds unexpected data. |
| 8 | Phases are resumable | Each phase checks its postcondition; re-running skips completed work. The safety copy is never deleted automatically. |
| 9 | Failure is diagnosable | A failure prints the failed phase, the log path, the journal path, the source state, and the re-run instruction. |
| 10 | Self-test exists | A synthetic fixture exercises the copy, verification, restore, corruption detection, log parsing, and plan output without touching a real device. |
| 11 | Static gates pass | `bash -n` and `shellcheck -S warning` pass. |
| 12 | Output is human-first | Phase headers, a findings table, a final result, and the next command. JSON is an option, not the only output. |
| 13 | Selection is human-first | `--destination` accepts an id, a name, or an index; a single candidate is selected without a prompt; a selector that identifies a unit (a drive) lists every subject on that unit and handles all of them. |

### 3.1 Fallback ladders

A cheaper path may be attempted only when a read-only probe proves it possible. A destructive attempt is never automatically followed by the fallback: a failed attempt can invalidate the fallback's precondition.

| Step | Form | Rule |
|---|---|---|
| Probe | read-only command or inspection | Change nothing. Report whether the cheaper path is possible and why not. |
| Choose | decision table over observed facts | Prefer the path with a verified external copy when both paths are possible. |
| Destructive attempt | explicit, confirmed, not automatic | Do not auto-fallback after it. Stop, report the state, and let the operator decide. |
| Fallback | the verified-copy path | Run only when its precondition (free space, healthy target) still holds. |

A probe that unmounts a live service is still read-only for data, but it is a service interruption; say so and restore the mount and service before returning. Run the probe automatically when it is read-only and the cheaper path is relevant; do not require a separate flag for a check that changes nothing. If the probe needs elevation, ask for it in place (re-exec through `sudo`) instead of failing and making the operator retype the command.

### 4. Dry-run design

A dry run is a complete read-only rehearsal, not a help alias.

| Check | Pass condition | Failure output |
|---|---|---|
| Required commands | Every command the apply path calls resolves on `PATH`. | Name the missing commands. |
| Target identity | The observed serial, UUID, filesystem, and mountpoint match the recorded identity. | Name both values and refuse. |
| System-disk guard | The target's parent disk differs from the root disk. | Name the parent disks. |
| Staging | The staging target is a different disk, a repairable filesystem, and large enough. | Name the required bytes and the available bytes. |
| Confirmation reachability | A TTY exists, or `--yes` was supplied. | Refuse before any effect. |
| Plan | The copy, mutation, restore, and verification commands render with real paths. | Print the plan with placeholders only when a value is genuinely unknown. |

Exit codes: `0` means every check passed and nothing was changed; `2` means the dry run found issues and nothing was changed; `1` means an unexpected failure in the script itself.

### 5. Anti-patterns

| Anti-pattern | Why it fails | Fix |
|---|---|---|
| Bare `--destination is required` as the only entry | The operator cannot discover the valid targets or the next command. | Print help and list the targets; exit 0. |
| `--dry-run` mutates or stops at the first failure | The operator learns one problem per run and cannot judge readiness. | Collect all findings, print the table, change nothing. |
| Confirmation defaults to yes | One mistyped command destroys a disk. | Require the exact phrase; make `--yes` explicit. |
| Format or delete before the copy verifies | The safety copy becomes the only copy without proof. | Verify structure, manifest, and checksums first. |
| Reuse a device path without re-reading identity | A replug or renumber targets another disk. | Re-read serial, UUID, size, and parent disk immediately before the effect. |
| `rm -rf` on a computed path | A variable bug deletes the wrong tree. | Contain the path under a verified root; refuse `..` and root. |
| Clean up the safety copy automatically | The second copy disappears before the result is verified. | Keep it and print its path. |
| Hidden `sudo` during a claimed dry run | The dry run is not read-only. | Do not elevate on the dry-run path. |
| JSON-only output | An operator cannot read the result at a glance. | Human-first table and summary; JSON optional. |

### 6. Output expectations

You may create or update:
- The operator script under the owning scenario's `scripts/` directory.
- A short header comment that states the purpose, the destructive step, the safety copy, and the control-plane owner.
- The scenario's `docs/internal/PROBLEMS.md` or `PROGRESS.md` entry that records why the script exists and when to retire it.

You must:
- Include every row of §3 or state a dated reason for the exception.
- Keep the dry-run path free of side effects.
- Use STE-100 for help text and phase output.
- Name the exact commands and paths the operator will see.
- Test the non-destructive path with a self-test and a live dry run.

You must not:
- Add a second copy of product logic to the script.
- Require the operator to read the source to use the script.
- Claim the script fixes a condition the underlying tool cannot fix.

### 7. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Dry run reports the target is unreachable | The owning service is stopped | Start it through its lifecycle command, then re-run the dry run. | Do not let the dry run start services; that is a side effect. |
| Staging is rejected for space | The copy needs headroom for the filesystem and the restore | Compare `used × 1.17 + 5 GiB` with the available bytes. | Use a larger staging target; do not compress the requirement away. |
| Copy verification fails on a committed file | The source is damaged beyond the known skips | Read the verification log and the skipped-path list. | Abort before mutation; the source is unchanged. |
| Failure after the destructive step | The safety copy is now the primary copy | Check the staging manifest and the journal phase. | Re-run to resume the restore; do not delete the staging copy. |
| Unmount fails with "target is busy" | A process or shell holds the mount | `findmnt` and `fuser -m` on the mountpoint | Stop the holder, or move the operator's shell out of the mount. |
| Confirmation prompt is unavailable | No TTY (cron, CI) | Inspect stdin | Refuse unless `--yes` was supplied deliberately. |
| System-disk detection misses a mapper or LVM volume | The device is not a plain partition | Compare the parent disk and the root mount's source | Add the platform's mapper resolution before trusting the guard. |
| A probe run as root remounts the volume under root's session | A session-aware mounter (udisks) chooses `/media/root/<label>` | Compare the observed mount target with the recorded mountpoint | Unmount the wrong path and mount explicitly at the recorded path with the original options; offer a `--remount` recovery command. |
| Several subjects share one destructive unit | The script selects one subject but formats a disk that holds several | List the subjects by their common device | Group by the destructive unit, copy the whole unit, and handle every subject in the restore and re-bind phase. |

Promotion note: when the same script runs unchanged more than twice with the same inputs and joins, propose a governed program under the owning scenario. Keep the confirmation and the dry run in the program's contract, not only in prose.
