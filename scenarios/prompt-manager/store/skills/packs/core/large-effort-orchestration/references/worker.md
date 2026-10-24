# Epoch worker card

You deliver one epoch of an orchestrated goal. Your assignment prompt carries
the applicable directive IDs, authority, gates, source references, owner/run
identity and resume set. Read this card and that assignment first. Read the
epoch sections and evidence needed to understand or verify those requirements;
read older history or the full epoch when a requirement depends on it. The epoch
file remains the complete worker execution record and `epoch-check` always checks
that full file. Your prompt names the epoch file, the orchestrator run ID and the
applicable resume set.

**Resume set.** The admitted brief (header fields, modules and
budgets, deletion list, exit gate, non-goals), applicable directives named in
the assignment, the last ~20 slice-log lines, and the scenario docs the brief
names. Read earlier slice-log entries or other epoch sections when they contain
evidence needed for the assigned work or a gate; do not reload historical
sections by default.

**Before the first write.** Retain a bounded byte snapshot and hash manifest of
the exact admitted source paths through the owner's existing artifact route,
including untracked files and explicit absence for files you will create. Record
the artifact identity and capture time in the slice log. Capture generated
owner outputs before a permitted generator/materializer can replace them. If
this attempt resumes, preserve its original baseline and add a dated checkpoint;
never relabel later bytes as the original pre-run state. At handoff, lead with
this run ID, its final result, and its final source/evidence references. Compare
this attempt's allowed write set with its preserved baseline; name concurrent or
unknown attribution and validation side effects. Label older attempt tables as
historical. A hash of a handoff Markdown file identifies that document, not the
product files it describes. If final source identities are unavailable, say so
and preserve unknown attribution; do not create extra capture requirements.
Do not restore over another writer or claim a whole dirty checkout diff is your
contribution. This is evidence inside the existing admission, not a new product
or parent-release gate.
Keep the surviving baseline in durable goal/owner artifact storage through acceptance and handoff; use `/tmp` only for staging. Record the compared source revision and date. After source inputs change, label the earlier measurement historical until it can be recomputed from the preserved baseline.

When the admitted directive permits a same-task local fallback, use that route
without retrying an opaque managed-artifact request. Put one exact repository-
relative file path per line in a task-local `prewrite.paths`; include untracked
files, intended new files, and the generated owner outputs that may change.
Expand admitted directories to files explicitly. Do not snapshot the whole tree.
Run this recipe with a new task-local baseline directory:

```bash
python3 - /absolute/task/prewrite.paths /absolute/task/baseline-original <<'PY'
import datetime, hashlib, json, pathlib, sys
root = pathlib.Path.cwd().resolve()
paths = pathlib.Path(sys.argv[1]).read_text().splitlines()
out = pathlib.Path(sys.argv[2])
out.mkdir(parents=True, exist_ok=False)
rows = []
for name in paths:
    rel = pathlib.Path(name)
    if not name or rel.is_absolute() or '..' in rel.parts:
        raise SystemExit(f'Expected exact relative file path: {name!r}')
    source = root / rel
    if source.resolve().is_relative_to(root) is False or source.is_symlink():
        raise SystemExit(f'Unsafe snapshot path: {name}')
    row = {'path': name, 'observed_at': datetime.datetime.now(datetime.timezone.utc).isoformat()}
    if not source.exists():
        row['absent'] = True
    else:
        before = source.stat()
        data = source.read_bytes()
        after = source.stat()
        if (before.st_ino, before.st_size, before.st_mtime_ns) != (after.st_ino, after.st_size, after.st_mtime_ns):
            raise SystemExit(f'Concurrent change; retain partial snapshot: {name}')
        target = out / 'bytes' / rel
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_bytes(data)
        row.update(bytes=len(data), sha256=hashlib.sha256(data).hexdigest())
    rows.append(row)
    (out / 'manifest.json').write_text(json.dumps(rows, indent=2) + '\n')
print(out / 'manifest.json')
PY
```

Run from the admitted checkout root. Record the manifest path/hash and capture
time in the slice log. A capture error leaves partial evidence, not a baseline;
reconcile it before a source write. Keep the original directory on continuation.
This recipe records bytes and absences; it neither proves exclusive ownership
nor removes a separate managed-tool recovery hold.

**Coherent worker scope.** For an implementation assignment, define the goal
around the smallest meaningful behavior and include focused self-validation and
handoff when those checks are authorized and available. Keep independent review
separate. Do not split one change into source-only and test-only worker goals
solely to create another dispatch.

A parent-owned review or acceptance is not a child prerequisite once the
assigned work and worker-owned checks are complete. Return the bounded result,
evidence, and pending parent gate through the normal terminal handoff; the
completion watcher delivers it, and the parent retains epoch acceptance. Do not keep a completed task open or wake/continue the parent solely
to request review. This does not waive required worker self-validation: a failed
check remains failed and an unavailable check remains unverified. Repair it
within scope, or hand off the exact finding, remaining check, and needed owner
dependency or scope amendment; never mark it passed. If a retained summary
conflicts with current authoritative run or await fields, current owner fields
win. Report a lost wait to the orchestrator or supervisor; do not wake the
parent based on a stale summary.

Do not stretch work to meet a runtime or token target.

**Owner-issued IDs.** Use returned IDs, not predicted suffixes. Preserve a verified
handle for the admitted owner and asset; ask the orchestrator to reconcile
derived references in this goal instead of creating another. Stop before writes
only on owner, asset or scope mismatch, or an unverified identity.

**Loop.**
1. Make one change and run its focused checks. A change whose checks pass is a
   work unit. Size units as meaningful steps (a module, an owner or a whole
   pattern across a package), not single edits; the overrun trigger counts them.
2. Append one line to `## Slice log`:
   `<RFC3339 timestamp with seconds and timezone, e.g. 2026-10-20T11:30:00Z> | <what changed> | exit metric=<value> | net runtime lines=<±n> | net test lines=<±n> | ack=<directive IDs or ->`
   Net lines are the unit's delta from before/after inventory snapshots of the
   epoch scope. Never use git. Moved or extracted code is not removal.
3. Reconcile the current IDs in `## Directives` with the assignment and prior
   acknowledgements in `## Slice log`. Read and act on each new applicable
   directive, and acknowledge its ID in your next slice-log line. If a new
   applicable directive is not synchronized with the assignment, or its scope
   or authority is unclear, use the existing amendment route with the
   orchestrator before affected work. Read other directive text when an
   assigned requirement depends on it.
4. Run `agent-manager effort epoch-check <repository-relative epoch file path> --runs <your run ID> --wake-key <orchestrator run ID>`.
   Use the full repository-relative epoch path and orchestrator run ID from your
   assignment. If the admitted brief names an active effort reference, pass its
   exact value with `--effort <ref>`; omit `--effort` otherwise. The completion
   watcher delivers your terminal result to the parent.
   `--wake-key` signals only a fired step-back. Do not call `run continue` on the
   parent or request global-management authority to deliver your result.

**Growth budget.** `refactor` epochs close at cumulative net runtime lines ≤ 0.
`feature` epochs stay within 1.5× `Expected size`. Tests follow
`path:docs/testing/UNIT-TEST-AUTHORING.md#mature-a-suite-instead-of-growing-it`.

**Step-back.** `epoch-check` exits 3 with `STEP_BACK <reason>` when a trigger fires:

| Trigger | Signal | Default |
|---|---|---|
| Flatline | exit metric unchanged across the last K lines | K = 10 |
| Overrun | log lines above the estimate | 2× |
| Growth | refactor: cumulative net runtime lines; feature: above expected size | > +300; > 1.5× |
| Spend | weighted non-cache tokens of the epoch's workers | set after 3 epochs |
| Wall clock | now minus `Started` | 24 h; prints `REVIEW`, never forces a stop |

Then stop adding more of the same. Write about 10 lines under the slice log: why
the epoch is stuck, what change elsewhere would make it tractable, and whether to
**park**, **switch** to an enabling slice, or **shrink** the scope. Continue only
inside a shrunk scope; the orchestrator records the choice in `STATUS.md` `## Next`.

**Broken infrastructure** (sandbox, shadow, Test Genie, Program Runtime, bridge)
does not block other implementation work. Append a `STATUS.md` `## Workarounds` entry (what
failed, the fallback, when to retry), continue unsandboxed in the shared tree or
restart in a quiet window, and stop retrying that tool. A fallback does not waive
required proof: use equivalent authorized evidence for the required behavior or
report the unmet gate and return condition. Do not mark the epoch complete while
a required gate remains unmet; the orchestrator keeps the affected slice open
while other admissible work continues.

**Blocked** means a decision, credential or approval you lack. Name it in a slice-log
line. Friction you can diagnose is not blocked.

**Never:** write `ACCEPTED` or directives, edit the brief's gates, read
`FEEDBACK.md`, install a tool outside the project (request it in a slice-log line),
or stop at a compaction, a green test or a status request.

**Exit.** Follow the scope named in your admitted assignment.

- **Full epoch:** Run every exit-gate command in the epoch brief. When all pass,
  log a final line with their results, mark your native goal complete, and end
  with changed, verified, remaining, and unverified. If any gate fails, report
  the failing command and remaining holds; do not mark the epoch complete.
- **Bounded task or slice:** If the orchestrator explicitly assigns a bounded
  unit and excludes epoch gates or acceptance, do not run the full epoch exit
  gates. When the assigned work and required handoff evidence are complete, mark
  only the task's native goal complete. End with changed, verified, remaining,
  and unverified, including unrun epoch gates and other unresolved holds. This
  closes only the assigned task; it never records or implies epoch acceptance.
  The orchestrator retains responsibility for epoch acceptance.
