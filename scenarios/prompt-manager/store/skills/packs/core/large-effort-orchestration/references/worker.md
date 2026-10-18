# Epoch worker card

You deliver one epoch of an orchestrated goal. Read this card and your epoch file
(`<goal home>/epochs/E<n>.md`); read other protocol docs once at epoch start, never
on resume. Your prompt names the epoch file, the orchestrator run ID and your
resume set.

**Resume set (about 20 KB).** The brief (header fields, modules and budgets,
deletion list, exit gate, non-goals), the last ~20 slice-log lines, open
directives, and the scenario docs the brief names.

**Before the first write.** Retain a bounded byte snapshot and hash manifest of
the exact admitted source paths through the owner's existing artifact route,
including untracked files and explicit absence for files you will create. Record
the artifact identity and capture time in the slice log. Capture generated
owner outputs before a permitted generator/materializer can replace them. If
this attempt resumes, preserve its original baseline and add a dated checkpoint;
never relabel later bytes as the original pre-run state. At handoff compare the
allowed write set with that baseline, naming concurrent/unknown attribution and
validation side effects. Do not restore over another writer or claim a whole
dirty checkout diff is your contribution. This is evidence inside the existing
admission, not a new product or parent-release gate.

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

**Loop.**
1. Make one change and run its focused checks. A change whose checks pass is a
   work unit. Size units as meaningful steps (a module, an owner or a whole
   pattern across a package), not single edits; the overrun trigger counts them.
2. Append one line to `## Slice log`:
   `<ISO time> | <what changed> | exit metric=<value> | net runtime lines=<±n> | net test lines=<±n> | ack=<directive IDs or ->`
   Net lines are the unit's delta from before/after inventory snapshots of the
   epoch scope. Never use git. Moved or extracted code is not removal.
3. Read `## Directives`. Acknowledge each new ID in your next line and act on it.
4. Run `agent-manager effort epoch-check <epoch file> --runs <your run ID> --wake-key <orchestrator run ID>`.

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
inside a shrunk scope; the orchestrator records the choice in `QUEUE.md`.

**Broken infrastructure** (sandbox, shadow, Test Genie, Program Runtime, bridge)
never blocks you. Append a `WORKAROUNDS.md` entry (what failed, the fallback, when
to retry), continue unsandboxed in the shared tree or restart in a quiet window,
and stop retrying that tool.

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
