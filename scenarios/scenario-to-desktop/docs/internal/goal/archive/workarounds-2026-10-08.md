# Archived workarounds — 2026-10-08

### DPF-WA-009 — terminal Test Genie run absent from artifact index

For scenario-to-desktop run `20261008-025632-dc30f188`, awaited terminal JSON
reported `FAIL`, but the supported `test-genie runs artifacts ... --kinds
findings.report --json` lookup returned `not_found: run not found in index`. The
terminal payload remains the evidence; the detailed underlying test output was
unavailable from that artifact route. Fallback: retain the terminal JSON, treat
the unit gate as failed, and do not rerun or retry the artifact lookup.

### DPF-WA-008 — interactive epoch worker launch could not find its Web Console session

Worker `3a13d5cf-c5af-4456-9abd-a1fea0d13361` (task
`98101542-4cbe-4d14-b102-ee13d6e7ac9a`, interactive delivery-epoch-worker)
failed before prompt delivery. Agent Manager recorded `wait for web-console
shell ... not_found: session`; continuation reported that the failed run had no
session. No Test Genie phase was submitted by that worker. The later G7 repair
worker completed successfully and the required replacement unit receipt passed.

### DPF-WA-003 — local resource-artifact staging stopped at Kokoro

The documented `vrooli-dist --resource-artifacts` source-build attempt failed
while staging Kokoro linux-amd64 because `docopt==0.6.2` had no usable wheel for
Python 3.12.11. It left a partial 45-file directory without `SHA256SUMS`; no
manifest was accepted. The attempt was not retried. The Scenario Dependency
Analyzer-governed wheel-only lock repair later built Kokoro linux-amd64 and
verified its four-artifact manifest.
