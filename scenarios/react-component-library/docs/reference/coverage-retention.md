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
