# prompt-manager API testing guide

This guide describes the test substrate that exists. Every identifier and path
below resolves under `scenarios/prompt-manager`, and a test enforces that.

The previous version of this file named 13 helper symbols and file names across
45 references, none of which existed anywhere in the repository, and documented
"Campaign CRUD operations" for a scenario that has no campaigns. It was a
template copied from elsewhere. An agent following it wrote code that did not
compile.

Treat that as the failure mode this guide is written against: **do not describe
a helper here unless you can `grep` it.**

## Shared substrate: `internal/testutil`

`api/internal/testutil` is the only shared test-support root. It must never
import a production package — `TestNoProductionImports` in
`internal/testutil/no_prod_import_test.go` parses every file under that root and
fails when one does. The ban exists so a helper cannot quietly re-encode the
behavior it is meant to check.

| Package | Provides |
|---|---|
| `internal/testutil/fixtures` | `WriteTeam`, `WriteJSON`, `RepositoryRoot`, and the `TeamOption` builders (`WithEnabled`, `WithContractAgents`, …) |
| `internal/testutil/httpx` | `Request`, `JSONRequest`, `Recorder`, `DecodeJSON[T]` for handler tests |
| `internal/testutil/assertx` | `Eventually`, `Contains` |
| `internal/testutil/mocks` | Hand-written doubles |

For repository-backed tests, call `fixtures.RepositoryRoot`. A missing checkout
returns `os.ErrNotExist` and is a reason to skip a `liverepo` test; a discovered
but malformed checkout is an error and must fail the test. Default tests should
use `t.TempDir()` trees only.

The `store` package keeps its own local team builder instead of using
`fixtures`. That is deliberate and is not duplication to clean up: all 16
`store` test files are internal `package store`, so importing `fixtures` would
create an import cycle, and a builder returning `*store.Team` cannot live in a
leaf package that must not import `store`.

## The agent-system packages

| Package | What its tests cover |
|---|---|
| `finding` | The one type every validation surface produces |
| `memberflow` | Topic declarations, the operating-graph and operating-model validators, the rule registry and catalog, plan-of-record manifests, objectives |
| `heartbeat` | Prompt assembly, prompt section identity, contract findings, executor and run lifecycle |
| `teamcontract` | `team.json` operating-contract validation and member-policy rendering |

## Three patterns

### Fixture then mutate

Build a valid object, break exactly one thing, assert exactly one finding. The
mutation is the test's subject, so it must be the only difference from a passing
case.

`operatingModelDocumentFixture` in `memberflow/operating_graph_test.go` is the
worked example. The table-driven validator tests take it, apply one `mutate`
closure, and assert the rule id that closure should trigger.

### Golden tree

For a validator that walks a directory, check the tree in and compare against
recorded output. `memberflow/testdata/prose_scan` is the worked example: a small
committed tree whose scan result is asserted whole, so a change in what the
scanner finds appears as a diff rather than as a silently different count.

Golden validation tests assert the complete sorted findings set — rule id,
severity, attribution, advisory status, and detail. Regenerate expected output
only after reviewing every added or removed finding as a validator behavior
change; a count change alone is not evidence of a correct precision change.

### Typed writer

A test that persists a shape must encode it through the production struct, never
as a string literal. A string literal is a second, untyped copy of the schema
that keeps compiling after the real one changes.

`writeMemberTopics` in `memberflow/fixtures_test.go` is the worked example: it
takes a `Topics` value and marshals it, so a field rename breaks the test at
compile time instead of at assertion time.

## Coverage

Coverage thresholds are not configured in `.vrooli/testing.json` — `unit`
carries a `policy_profile` and no numeric gate. Do not quote a coverage number
here; the previous guide did, and there was nothing behind it.

## Running

```bash
cd scenarios/prompt-manager/api && go test ./...
cd scenarios/prompt-manager/cli && go test ./...
```

The full suite runs through test-genie, which owns the run:

```bash
vrooli scenario test prompt-manager
```

Two validation gates run inside the `unit` phase rather than as separate
commands:

- `TestLiveTreeHasNoDeclarationErrors` (`memberflow`) fails when the checked-in
  tree has a declaration error. It deliberately ignores runtime findings, which
  report what an agent did and cannot be cleared by editing the tree.
- `TestGeneratedRuleTablesMatchTheCatalog` (`memberflow`) fails when a generated
  documentation table has drifted from the rule catalog.

## Complete friction capture fixture (I10 A)

`handlers/teams/friction_capture_test.go` builds the actual CLI into a
`t.TempDir`, then invokes `team friction-capture --report-file` and receiver
`team knowledge-list meta-optimization --topic-prefix=friction-inbox/` through
the production Teams Connect mount and heartbeat knowledge handlers.
`paths.RootsForTest` supplies all roots; the standalone `FileTeamStore` has no
Source Ledger attached. Its in-memory corpus belongs only to that test instance.
The server closes and Go removes the temporary build/report/tree at test end.
No fixture is written into live team knowledge; no live deletion is performed.

Run from `scenarios/prompt-manager/api`:

```bash
go test -v ./handlers/teams -run TestFrictionCapture
```

The synthetic report preserves declared identity, six context anchors, exact
multiline/special-character content, date, flags and three explanatory paragraphs.
Prefix intake matches its receipt without a record-ID lookup; repeating identical
input returns that receipt without another case. The transport fixture simulates
denied/unavailable writes and a response lost after the real handler commits.
Read failure and conflicting content prevent another mutation. Legacy typed
operator calls remain narrower and supported. This proves bounded fixture
behavior, not live ledger retention, concurrent exactly-once writes, authenticated
declared identity, curator execution or downstream repair authority.
