# Testing

## What proves what

| Layer | Proves |
|---|---|
| Go unit (`api/`, `cli/`) | Read-model derivation, closure resolution, merge-patch semantics, apply ordering, readiness composition, exit codes |
| Control-plane unit (`internal/operatorstate/`) | Field preservation, schema rejection, single-writer enforcement, concurrency |
| UI unit (`ui/src/**`) | Step rendering, derived consequences, credential entry with no disclosure, accessibility |
| Integration | Surface parity — the same selection through three surfaces yields byte-identical state |
| Journey (browser-automation-studio) | The declared journeys in `experience/`, recorded as evidence |
| Bundle mode | Every V2 endpoint with only `BUNDLE_ROOT` set |

## Non-negotiable tests

These pin defect classes rather than instances. Do not weaken them to make a
change land.

- **Field preservation** — a document carrying every schema field plus one the
  binary has never seen survives a write from every writer, byte for byte.
- **Single writer** — exactly one write site for the operator-state file.
- **Bundle mode** — all four V2 endpoints return 200 with only `BUNDLE_ROOT` set.
- **Credential non-disclosure** — no response, log, request line, or written
  document contains a submitted value.
- **Endpoint contract** — the declared endpoint set and the router's registered
  set match in both directions.
- **No dead commands** — every route a CLI command targets is registered.
- **Surface parity** — three surfaces, one selection, identical state.

## Running

```bash
vrooli scenario test vrooli-onboarding
```

The run is server-owned and survives your cancel. To wait, block **once**:
`test-genie runs wait --json vrooli-onboarding <run-id>`. Never poll. Cancelling
is not aborting; use `vrooli scenario test abort` for that. The full protocol is
in [`/docs/TESTING.md`](../../../../docs/TESTING.md).

For a fast inner loop:

```bash
cd api && GOWORK=off go test ./...
cd cli && GOWORK=off go test ./...
cd ui  && pnpm exec vitest run
```

## Standards

- Test the **desired** behaviour, not the current implementation. Several tests
  here are expected to fail until the target lands; that is the point of writing
  them first.
- Tag every test with its `[REQ:ID]`. An untagged test is invisible to
  requirement sync, so it earns no status.
- Fixtures carry no real credential values, on any tier.
- Coverage floors come from `.vrooli/testing.json`: 75% for both Go roles, 85%
  for the UI. The policy's required test-utility projections must exist.

## Known environmental limits

Recorded in [Problems](PROBLEMS.md): the storage phase's routed-isolation
detector expects a SQL handle a file-only scenario does not have, the
accessibility metric is capped by a missing host tool, and the Lighthouse and
playbook phases are intermittently infrastructure-bound.
