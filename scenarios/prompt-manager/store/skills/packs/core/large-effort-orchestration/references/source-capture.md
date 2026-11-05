# Preserve the assigned source before editing

Read once before the first source write, or when resolving attribution after source drift.

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

