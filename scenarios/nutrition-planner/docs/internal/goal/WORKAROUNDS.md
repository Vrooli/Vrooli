# Nooch infrastructure workarounds

## E3 exact-build capture route remains unavailable
The earlier CaptureService action was denied. BAS health/list evidence did not establish a supported Nooch workflow. Keep the visual/connected gates open and preserve that denial. Retry only after the existing BAS/CaptureService owner returns a qualified route, exact build and authorized fixture/state; do not use an alternate wrapper or infer global BAS outage. Detailed source comparisons and receipts are archived.

## Agent Manager worker artifact route may lack worker credentials
E8 and E16 required the documented local exact-path snapshot/hash fallback because the artifact route was unavailable in their worker shells. E17 also used that fallback. Root cause remains unknown; the E16 recurrence was recorded for the supervisor with writer receipt `knw-1791372915606432487`, but read visibility was not verified. Until owner confirms route recovery, do not retry it within an attempt; retain immutable prewrite snapshots and manifests without restoring over shared edits. See each epoch file for identities/hashes.

## Archive
Earlier run-control, capture, validation and fallback details are preserved in `archive/WORKAROUNDS-before-resume-compaction-20261007T1157Z.md`. No history was deleted.
