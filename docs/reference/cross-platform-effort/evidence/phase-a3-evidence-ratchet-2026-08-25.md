# Phase A3 evidence ratchet — 2026-08-25

The honesty ladder now requires a structured evidence object for every manifest
claim above `build-verified`. The object fields are `run_id`, `host`, `os`,
`arch`, `date`, `surface`, and `artifact_uri`; a bare evidence sentence cannot
qualify a cell.

## Changes and measured movement

- Added the evidence type and schema definition, plus loader propagation into
  the portability grid and typed API projection.
- Added negative and positive validation tests. The negative case names the
  manifest path, host OS, and missing structured evidence in its error.
- Converted all 184 occurrences of the boilerplate evidence sentence in tools
  and safeguards to explicit `null` values. The remaining 98 evidence strings
  are mechanism/test descriptions that remain meaningful at lower rungs.
- Downgraded 25 unevidenced macOS/Windows resource entries from `supported` to
  `build-verified`.
- Downgraded 18 unevidenced Linux service capability claims to
  `build-verified`; otherwise the new manifest gate would reject their legacy
  free-form evidence strings. No platform claim remains at `supported` in the
  checked-in manifest set.

## Validation transcript

```text
$ go test ./internal/deployability/...
ok   github.com/vrooli/vrooli/internal/deployability

$ cd scenarios/infrastructure-manager/api && go test ./internal/portability/... ./handlers/portability/...
ok   github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/portability
ok   github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/portability

$ vrooli capability ledger --json
manifestsRead=114 capabilities=45
linux:   qualified=0 build_verified=36 unqualified=52 undeclared=2 ineligible=0
macos:   qualified=0 build_verified=16 unqualified=60 undeclared=10 ineligible=4
windows: qualified=0 build_verified=18 unqualified=52 undeclared=14 ineligible=6

$ docs/reference/generate-platform-support.sh --check
PASS (no diff)
```

The ledger currently reports zero `qualified` cells. This is the intended A3
result: no manifest can reach that rung until a later hardware run attaches a
complete evidence object naming its run. The generated document records the
empty qualified set instead of turning declarations into proof.
