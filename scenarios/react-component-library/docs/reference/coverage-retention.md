# Coverage cache retention

The `coverage/` directory contains generated browser traces, screenshots, and
derived validation artifacts. It is a local cache and is ignored by Git.

`react-component-library coverage prune` enforces these limits:

- Maximum age: 14 days.
- Maximum total size: 2 GiB.
- Default mode: dry run. The command lists every selected file before it can
  delete anything. Pass `--apply` only after reviewing that list.

The command never removes durable evidence: verdict ledgers, evaluator output,
calibration verdicts, content hashes, and timing records. A cache can be
recreated from the asset version, viewport, theme, kit, seeded fixture data,
font set, and determinism controls recorded by the capture instrument.

## Run snapshots

Run snapshots are disposable evidence produced while a validation command is
in flight. Capture manifests, Vite result caches, and dated suite exports do
not belong in the scenario source tree. They are written to the scenario's
ignored runtime/cache locations and are subject to the same fourteen-day,
two-gigabyte retention policy when they are materialized under `coverage/`.

The checked-in `docs/reports/` directory is different: it contains durable
design records that humans read, such as the setup architecture review. A
report is retained there only when it is a design record, not because a test
run happened to emit HTML. Durable gate verdicts belong in the evidence store,
where the coverage join can read them without depending on a snapshot file.
