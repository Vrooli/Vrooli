# Manual Scenario Recovery — The Trusted-Base Fallback

**Audience:** an operator (or agent) facing a scenario that will not start, stop, or
return to a working state — *especially when the automated recovery tooling itself is
broken*.

This page is the single hand-operated escape hatch beneath every automated safety
mechanism. It is a **prerequisite of the Baseline Modes work** (`baseline start/check/
promote/abandon`, the shadow lifecycle, the restore point): those mechanisms refactor
the unshadowable recovery floor — the lifecycle, ports, and registry layers — and they
are built *with no net*. If a refactor of the floor leaves a scenario wedged and
`vrooli scenario …` / `baseline …` cannot fix it, the steps here restore it **by hand,
using only primitives that do not depend on the floor being healthy.**

> **Hard rule — no git as an undo.** Do **not** use `git stash`, `git reset`,
> `git checkout --`, or `git revert` to snapshot or roll back a scenario. Uncommitted
> work on the `agi` branch gets clobbered. The sanctioned code-level undo is a
> filesystem copy (see [Restore code from a copy](#4-restore-code-from-a-copy)), and —
> once Baseline Modes lands — `baseline abandon`. To preserve a known-good tree, copy it
> aside with `cp -a` to `/tmp`, never a git operation.

---

## 0. The recovery ladder (try in order)

Most problems are resolved by the first two rungs. Drop to a lower rung only when the
one above it fails or is itself broken.

| Rung | Tool | Use when |
|------|------|----------|
| 1 | `vrooli scenario restart <name>` | The scenario is just unhealthy/stuck. |
| 2 | `vrooli cleanup locks` then restart | A stale lock or registry claim blocks start/stop. |
| 3 | Hand-kill the process (§2) + clear records (§3) | The lifecycle cannot stop it. |
| 4 | Restore code from a copy (§4) | A bad edit broke the scenario and you need the prior tree. |
| 5 | Restore data from a backup (§5) | The store (DB/Redis/Qdrant/files) is corrupt. |
| 6 | Last-resort full reset (§6) | Everything above failed. |

Where every scenario's runtime state lives (all under `~/.vrooli/`):

| What | Path |
|------|------|
| Process records (PID/PGID/log per step) | `~/.vrooli/processes/scenarios/<name>/` |
| Lifecycle log | `~/.vrooli/logs/<name>.log` |
| Per-step logs | `~/.vrooli/logs/scenarios/<name>/` |
| Advisory locks | `~/.vrooli/state/locks/scenario-<name>.lock` |
| Runtime registry (SQLite) | `~/.vrooli/state/runtime.db` |
| Start-operation records (progress-only; inside the registry) | `runtime_start_operations` / `runtime_phase_durations` tables in `runtime.db` |
| Scenario state | `~/.vrooli/state/scenarios/` |

> **Start-operation records are safe to ignore during recovery.** They are derived
> progress state for `vrooli scenario wait`/`status`, never authority: a record whose
> initiator pid is dead reads as `abandoned` and the next `vrooli scenario start`
> supersedes it. No manual-recovery step needs to clear them; deleting `runtime.db`
> (last resort) also removes them harmlessly.

> **Variant note (Baseline Modes).** Once instance variants ship, a non-live instance is
> namespaced with `@<variant>`: its records live at
> `~/.vrooli/processes/scenarios/<name>@<variant>/`, its lock is
> `scenario-<name>@<variant>.lock`, and its Postgres DB is `vrooli_<name>_<variant>`. The
> **live** instance keeps the bare forms shown above, unchanged. Everything in this guide
> applies to a variant by substituting `<name>@<variant>` for `<name>` (and
> `vrooli_<name>_<variant>` for the DB). When in doubt, recover **live** first — it is the
> one that must always work.

---

## 1. Diagnose first

```bash
vrooli status                 # overall health
vrooli scenario status <name> # this scenario's registry/health view
vrooli orphans                # stray Vrooli processes the registry lost track of
tail -n 100 ~/.vrooli/logs/<name>.log
```

Identify which failure you have:

- **Won't start, "already running" / busy** → stale lock or registry claim → §2–§3.
- **Won't stop** → process escaped the lifecycle → §2.
- **Starts but unhealthy after a code change** → §4 (restore code).
- **Starts but data is wrong/corrupt** → §5 (restore data).

---

## 2. Stop a scenario the lifecycle can't

Prefer the sanctioned path; only hand-kill if it fails.

```bash
vrooli scenario stop <name>          # try this first
vrooli cleanup orphans               # SIGTERMs registry-confirmed orphans (inspect with `vrooli orphans` first)
```

If a process is still wedged, find and kill it by hand. The process records hold the
PID and process-group ID:

```bash
ls ~/.vrooli/processes/scenarios/<name>/        # one <step>.json per running step (e.g. develop.json)
cat ~/.vrooli/processes/scenarios/<name>/develop.json   # contains "pid" and "pgid"

# Kill the whole process group (negative PGID) — graceful first, then hard:
kill -TERM -<pgid> ; sleep 3 ; kill -KILL -<pgid> 2>/dev/null

# If you only have a PID and no clean PGID, fall back to:
pkill -TERM -f 'VROOLI_SCENARIO=<name>' ; sleep 3 ; pkill -KILL -f 'VROOLI_SCENARIO=<name>'
```

Managed Vrooli processes carry `VROOLI_SCENARIO=<name>` in their environment, which makes
them identifiable with `ps eww` / `pgrep -f`.

---

## 3. Clear stale locks and registry records

After the process is dead, clear the bookkeeping that still claims it is alive.

```bash
vrooli cleanup locks            # expires stale advisory locks + non-authoritative registry claims (preferred)
```

If `cleanup locks` is itself broken, remove the artifacts by hand (only after confirming
no live process owns them — §2):

```bash
rm -f ~/.vrooli/state/locks/scenario-<name>.lock
rm -f ~/.vrooli/processes/scenarios/<name>/*.json
```

The runtime registry (`~/.vrooli/state/runtime.db`) self-heals: on the next
`vrooli scenario start`, the reaper finalizes instances whose owner PID is dead. You do
**not** normally edit `runtime.db` by hand. If it is corrupt and blocking everything, see
§6 — deleting it discards *all* scenarios' runtime bookkeeping, so it is a last resort.

Then restart cleanly:

```bash
vrooli scenario start <name>
vrooli scenario status <name>
```

---

## 4. Restore code from a copy

When a bad edit broke a scenario and you need the previous tree back — **without git**.

The scenario's source is `scenarios/<name>/`. Build artifacts (`dist/`, `generated/`,
`node_modules/`) regenerate on restart and should be excluded from any copy.

```bash
# BEFORE risky edits — snapshot the good tree aside (NOT git):
cp -a scenarios/<name> /tmp/<name>.good

# To roll back — restore source, leave build artifacts to regenerate:
rsync -a --delete \
  --exclude node_modules --exclude dist --exclude generated --exclude .git \
  /tmp/<name>.good/ scenarios/<name>/

vrooli scenario restart <name>
```

> The Baseline Modes **restore point** is the automated form of the copy above: every
> `baseline start` copies the good tree to
> `~/.cache/vrooli/<name>/baseline-<slug>/restore-point/`, and the engagement verbs use it
> as the code-level undo. The `vrooli recovery` surface drives it directly:
>
> ```bash
> vrooli recovery capture  --scenario <name> --slug <slug>   # snapshot the working tree
> vrooli recovery restore  --scenario <name> --slug <slug>   # overlay the snapshot back
> vrooli recovery clean    --scenario <name> --slug <slug>   # drop the restore point + manifest
> ```
>
> The `cp -a` / `rsync` pair above remains the engagement-free manual fallback.

### The live-from-copy isolation model

While a **shadow** engagement is open, the two code locations hold different things and
the two instances run from different places:

| Role | Location | Served by |
|---|---|---|
| **Candidate** (in-progress edits) | the working tree `scenarios/<name>/` | the `@shadow` instance |
| **Baseline** (known-good) | the restore-point copy | the **live** instance |

The lifecycle resolves a live (re)start's build/run directory through the engagement
resolver: **while a shadow engagement is open, a live `start`/`restart` runs from the
frozen copy, not the working tree the agent is editing.** Opening a shadow does *not*
restart live — live keeps serving on its existing process; the copy is the source it
re-resolves to only on a later restart. This is safe for any scenario at storage-steer
maturity (all persistent state resolved through the shared variant-aware packages), since
the running process never reads repo-relative state at runtime.

**Invariant.** While an engagement is open for scenario `S`, the instance serving the
**Baseline** role MUST NOT run from the location receiving the agent's merge (the working
tree under the standard layout). The directionality is declared once in
`internal/engagementlayout` and enforced by a property-based test
(`TestServingInstanceIsolated`), so flipping the layout cannot silently expose live.

**Engagement timeline.**

```
baseline start    → capture Baseline into the restore-point copy
   (shadow run)   → [agent edits the working tree = the Candidate; live unaffected]
baseline check    → validate the Candidate via the @shadow instance
baseline promote  → re-point live off the baseline copy onto the blessed working tree:
                      `recovery set-mode --mode live` collapses the split so a live
                      restart resolves to the working tree (the copy is kept as the
                      rollback source until the health probe passes), then restart +
                      probe; on failure, flip back to shadow + restart = live re-resolves
                      to the baseline copy (working tree keeps the Candidate, retryable)
baseline abandon  → restore the Baseline over the working tree (discard the Candidate),
                      tear down the shadow; live (still on the copy) is untouched
```

> `recovery set-mode` is the non-lossy re-point lever: it flips only the engagement's
> recorded mode (preserving the restore point and all manifest metadata), which is what
> makes the resolver stop/start redirecting live without tearing the engagement down.

---

## 5. Restore data from a backup

Code recovery (§4) never touches stateful stores. If the failure is corrupt data, restore
the store from a `data-backup-manager` snapshot.

```bash
# List what is captured for this scenario's targets:
vrooli scenario start data-backup-manager
# (via data-backup-manager CLI)
restores restore --target <scenario-target> --to <destination>
restores verify  --target <scenario-target>     # checksum the restored data
```

`data-backup-manager` is itself a **trusted-base member**: it backs every storage engine
(filesystem, SQLite, Postgres, Redis, Qdrant, object storage) through one type-agnostic
path, and a restore can target an arbitrary destination (a different DB name, Qdrant
collection, Redis prefix, or path). If `data-backup-manager` itself is the broken
scenario, recover *it* via §2–§4 first, since the rest of the data-recovery floor depends
on it.

---

## 6. Last-resort full reset

Only when every rung above has failed. This is destructive to runtime bookkeeping.

```bash
# 1. Stop everything you can:
vrooli stop
vrooli cleanup orphans

# 2. Hand-kill any survivors (§2).

# 3. If the registry DB itself is corrupt and blocking ALL scenarios, move it aside
#    (this discards runtime tracking for every scenario; they re-register on next start):
mv ~/.vrooli/state/runtime.db ~/.vrooli/state/runtime.db.broken.$(date +%s)

# 4. Clear stale locks/records (§3), then bring the system back up:
vrooli develop
vrooli status
```

If even this fails, the host or `~/.vrooli` state may be corrupt at a level beyond
scenario recovery — capture `~/.vrooli/logs/` and escalate.

---

## Why this doc exists (and is a hard gate)

The Baseline Modes plan refactors the lifecycle / ports / registry layers (P1) so a
scenario can run a second isolated **shadow** instance and be safely promoted. That floor
is what *all* automated recovery sits on. Refactoring it without a written manual fallback
means a single bug in the refactor could leave a core scenario unrecoverable with no
sanctioned path back. This page is that path. It must exist **before** the floor is
touched, and it stays the documented trusted-base escape hatch afterward.

See also: [troubleshooting.md](troubleshooting.md) (first-line checks),
[logging.md](logging.md) (diagnostic surfaces).
