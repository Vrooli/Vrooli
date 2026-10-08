# Source distribution runbook

## Inspect

Use the lifecycle-managed API or CLI to analyze the source root. Preserve the
returned source and closure digests. Do not run the API binary directly and do
not use Git commands to prepare a fixture.

## Assemble and verify

Assembly writes a temporary file beside the requested output and atomically
renames it only after deterministic tar/gzip completion. A failed assembly must
leave no ready artifact. Verification checks external archive bytes, safe and
unique paths, regular-file entries, bounded uncompressed size, and the exact
canonical manifest.

## Restart and recovery

Artifact and distribution identity is stored in the scenario-owned SQLite
tables. After restart, read the distribution by ID and re-run verification if
the archive is unavailable or the standing is not passed. Never reinterpret a
missing record as a successful export.

## Retention and backup

Retain an artifact while a distribution or approval references it. Backup the
SQLite database together with the content-addressed archive directory. Restore
both before declaring a distribution recoverable. Deletion, archival, and
remote publication remain human-authorized operations.

## Limits and unsupported integrations

The initial scanner bounds source file count/bytes and archive uncompressed
size. The supported archive is deterministic `tar_gzip`. Provider connectors,
remote repository writes, and history-preserving mirrors are unavailable.
