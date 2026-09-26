---
name: "unit-testing-architecture-steer"
description: "Establishes robust unit test infrastructure including file organization, test isolation, mock organization, testable production code patterns, and systematic edge case coverage structures."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["steer","test"]
  tags: ["testing","unit-tests","architecture","mocks","testcontainers","dependency-injection"]
  status: "active"
  targetDimensions: ["tests"]
  programmaticHome: "unit-health:unit"
  revision: 10
  createdAt: "2026-02-04T12:50:18Z"
  updatedAt: "2026-09-07T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "vrooli"]
    commands: ["prompt-manager skill", "prompt-manager skill read", "vrooli"]
  origin:
    kind: "authored"
---
## Steer focus: Unit Testing Architecture

Make the dependencies needed by a behavioral test controllable in
`scenarios/{{TARGET}}/`. Preserve production behavior while adding the smallest
useful seam, fixture, or harness.

This skill owns test infrastructure. The `test` skill owns behavior assertions.
Use `path:docs/testing/UNIT-TEST-AUTHORING.md` for boundary selection, independent
expectations, fixture design, isolation, and special test kinds. A pure-function
test needs no scenario-wide architecture prerequisite. Introduce infrastructure
when the selected behavior needs it.

## Scope and evidence owner

In scope: co-located tests, reusable fixtures, dependency substitution,
production-import guards, and repository/transport harnesses.

Out of scope: changing product policy, certifying all scenario behavior, or
reimplementing Unit Health checks. Use `seam-discovery-and-enforcement` for a
missing dependency boundary and `requirements-traceability-steer` for evidence
linkage.

Read existing findings in `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` and
the boundary map in `scenarios/{{TARGET}}/docs/internal/SEAMS.md`, when present.
Reuse the active plan's evidence record if it already owns those findings.

Unit Health owns the programmatic maturity report:

```bash
unit-health validate scenario {{TARGET}}
```

This is static assessment unless execution is explicitly requested. Interpret
each finding against its support profile and source evidence. A search hit is a
candidate to inspect; a missing hit does not prove isolation. Unsupported inputs
and unresolved helpers remain unknown. Do not add cosmetic assertions or
unnecessary interfaces to satisfy a heuristic.

Use `path:docs/TESTING.md` to select focused regressions and scoped Test Genie
phases. A green construction report does not prove behavioral adequacy.

## Select the test boundary

| Test subject | Required implementation | Substitute |
|---|---|---|
| Pure policy/value | Real function and policy | None unless it consumes an external input |
| Service orchestration | Real service and relevant domain rules | Repository or external service boundary; an in-memory fake is valid here |
| Repository | Real repository, production schema, matching engine | Unrelated services |
| Handler mapping | Real decoder, mapping, relevant middleware | Service; a recorder is valid when transport is irrelevant |
| Streaming/connection | Real handler and socket transport | Business dependencies |
| UI feature | Real component and relevant providers | API boundary |
| Time-dependent policy | Real policy with a controlled clock | Clock and external services |

A SQLite harness proves SQLite behavior. Use PostgreSQL for PostgreSQL-specific
constraints or transactions. A repository fake cannot prove SQL behavior.
Use a real server for flushing, hijacking, streaming, and connection closure.

The template reference is `path:templates/scenarios/react-vite/api/internal/notes/`:
service tests use repository fakes; `sqlite_test.go` applies production schemas.
The notes domain is removable teaching scaffolding, not required scenario vocabulary.

## Organize for actual reuse

| Artifact | Location |
|---|---|
| Go behavior test | Beside the production file as `*_test.go` |
| UI behavior test | Beside the feature as `*.test.tsx` or `*.test.ts` |
| Python behavior test | Package-local or the established pytest layout |
| One-domain fake or builder | `internal/<domain>/mocks/` or feature-local test helpers |
| Shared seam fake | `internal/testutil/mocks/` |
| Cross-domain fixture | `internal/testutil/fixtures/` or `src/test-utils/` |
| Shared database/HTTP harness | Existing testutil database/transport package |

Keep a small one-use setup local when extraction would hide the claim. Create a
shared helper only for actual reuse or to encapsulate a meaningful harness.
When policy declares a shared helper root, preserve its native projection; an
empty directory does not establish useful test infrastructure.

Factories return fresh deterministic data. Keep case-defining overrides visible.
Use schema constructors for valid wire data and raw bytes for malformed input.
The canonical guide owns literals, counts, and oracle-independence rules.

## Substitute real sources of variation

Inject time, randomness, environment, external HTTP, or repository behavior when
the test needs to control it. Prefer plain constructors and narrow consumer-owned
interfaces or function parameters; do not introduce a DI framework.

Use temporary directories and automatic cleanup for filesystem behavior. Restore
process-global state and avoid concurrent global mutations. Declare live
integration prerequisites separately. Check actual host/runner enforcement before
claiming network denial or filesystem isolation.

Keep domain fakes with their domain and shared fakes with their shared seam.
For reusable typed fakes, use interface-satisfaction checks and focused behavior
tests where the fake could hide errors or shared state. Do not require an assertion
helper package solely to make every test look alike.

## Diagnose and repair

1. Read the selected Unit Health finding and its evidence.
2. Locate the production dependency that affects the case.
3. Choose the boundary from the table above.
4. Add or reuse the narrow seam and wire its production implementation.
5. Exercise the desired behavior through that seam with deterministic inputs.
6. Run focused checks and the relevant owner phases.
7. Record the outcome and remaining limitations.

For import leakage, use resolved imports and the existing native production-import
guard. A filename search cannot resolve aliases or generated bridges.
For policy projection drift, compare `unit.policy_profile` in
`scenarios/{{TARGET}}/.vrooli/testing.json` with the supported native adapter
configuration. Keep policy interpretation in Unit Health and dependency changes
in Scenario Dependency Analyzer.

## Output expectations

Preserve the scenario's observable behavior and existing protective tests.
Record infrastructure decisions in the existing testing section of
`scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md`, seam changes in
`scenarios/{{TARGET}}/docs/internal/SEAMS.md`, and unresolved findings in
`scenarios/{{TARGET}}/docs/internal/PROBLEMS.md`. Reuse a plan-owned record when
the active workflow already captures this evidence. Do not create another
`UNIT_TEST_ARCHITECTURE.md` or a private maturity ladder.

Hand off the controlled boundary, focused validation, and limitations to the
`test` skill. If an actual dependency remains uncontrollable, name that dependency;
do not use an aggregate architecture level as a reason to defer unrelated tests.

## Troubleshooting & Edge Cases

| Observation | Action |
|---|---|
| Valid helper is reported assertion-free | Inspect its implementation and supported resolution; preserve unknown evidence instead of adding a cosmetic assertion |
| Map-backed fake appears in a service test | Check that the service is real; retain the fake when it matches the boundary |
| Repository test substitutes the database | Use production schema and the relevant engine for the repository claim |
| Recorder passes but streaming fails | Exercise the real socket and transport-capability middleware |
| Static search finds an ambient call | Trace whether it is an injected implementation or an uncontrolled domain dependency before changing it |
| Policy requests unsupported isolation | Return the explicit unsupported/refused result; do not silently relax policy |

Universal skill quality bars: `path:docs/agent-system/SKILL_AUTHORING.md`.
