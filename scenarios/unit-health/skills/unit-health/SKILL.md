---
name: "unit-health"
description: "Operate Unit Health: read a scenario's test maturity, findings, coverage, and advisory test-quality evidence without treating unknown as clean."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["unit-health", "testing", "quality", "evidence", "test-genie"]
  icon: "flask-conical"
  status: "active"
  revision: 3
  createdAt: "2026-09-04T00:00:00Z"
  updatedAt: "2026-09-09T00:00:00Z"
  learning:
    scope: "unit-health-usage"
    capture: "on novel outcome"
  requires:
    scenarios: ["unit-health"]
    commands: ["unit-health validate scenario", "test-genie runs wait", "program-runtime library run"]
    skills: ["test", "unit-testing-architecture-steer", "vrooli-memory"]
  origin:
    kind: "authored"
---
## Tools focus: Unit Health

Unit Health is the owner of a scenario's unit-test surface assessment: discovery,
per-workspace test plan, bounded execution, coverage, test architecture, the
advisory test-quality rule catalog, and the six-capability local maturity ladder.
This skill holds the judgment that `unit-health validate scenario --help` does not
print: which read answers which question, how to read a result without
over-claiming, and where the known limits are recorded. The authoring standard for
the tests themselves is [`/docs/testing/UNIT-TEST-AUTHORING.md`](/docs/testing/UNIT-TEST-AUTHORING.md);
the maturity spec is the `maturity` block of `path:.vrooli/test-genie.json`,
described in `path:docs/reference/maturity.md`.

### Scope

In scope: reading a scenario's assessment, choosing plan-only versus executed
reads, interpreting finding codes and rule statuses, bounded sampled review, and
routing repairs to the owning skill. Out of scope: editing tests or source through
Unit Health, changing the rule catalog or corpus (`unit-health-improve`), and
running Test Genie phases other than through their owner lifecycle.

### Before acting

Run `vrooli-memory recall wake --scope unit-health-usage` and apply any pinned
note before choosing a leaf. Memory mechanics are `prompt-manager skill read
vrooli-memory`'s.

### The tree: I need to know about a scenario's tests

```
What question am I answering?
├─ "What is this scenario's test maturity and what blocks the next rung?"
│   → unit-health validate scenario <name>                                              [S1]
│     Read Summary, then the capability with next_unlock. Plan-only is enough for the
│     level: required findings come from static analysis. Coverage, execution, and
│     reliability findings need the executed leaf below.
├─ "Do the tests actually pass, and what is the coverage?"
│   → unit-health validate scenario <name> --execution                                  [S1]
│     Bounded per workspace (timeout, no-output watchdog, worker caps). A
│     TEST_EXECUTION_FAILURE, TEST_TIMEOUT_HANG, or TEST_DEPENDENCY_MISSING is the
│     classified result of the run; read command output before touching tests.
│     Inside a Test Genie run the unit phase makes this read for you; do not run both.
├─ "Is one workspace the problem?"
│   → unit-health validate scenario <name> --workspace <id> [--execution]               [S1]
├─ "Is this test file well constructed?" (assertion, focus, matcher, async, skip, requirement link)
│   → read test_quality results: Go-profile rules from the plan-only response; the Vitest
│     rules (focused-test, malformed-expectation, async-assertion, skip-declaration) only
│     from a `--execution` response, otherwise their status is unknown/NOT_EXECUTED     [S1]
│     Branch on status:
│       violation      → the named rule fired inside its support profile; repair through `test`
│       checked_clean  → only that check passed; it proves neither execution nor an independent oracle
│       unknown        → unresolved helper, unsupported version, parse limit; never a pass
│       not_applicable → TestMain, build-excluded, registered subprocess helper
│     Every rule is advisory today; a violation does not fail the phase.
├─ "Does this test protect the line it covers?"
│   → unit-health mutation pilot <scenario> --package <p> --seed <seed>             [S1]
│     The owner applies bounded Go AST mutations only in disposable scenario-cache
│     copies and runs the owning tests through Unit Health's executor. Read killed,
│     survived, invalid, equivalent, out_of_contract, infrastructure_failure, and
│     unknown receipts separately; only valid killed/survived mutants contribute to
│     kill_rate. The pilot is a signal, not proof of behavioral adequacy.
├─ "Which tests should a reviewer look at first?"
│   → run unit-health.test-quality-sample                                               [S3]
│     Inputs: scenario, seed, source_identity, sample_size. source_identity is the
│     source identity of the same-day validate read of <name>; seed is <goal>-<YYYY-MM-DD>
│     so a re-read reproduces the cohort. Branch on status:
│       ok/partial → signals.cohort is the reviewed set; labels are advisory
│       unavailable → inventory empty or source drifted: select a new cohort, do not infer clean
│       refused → a selected body was privacy-refused; retain the row as insufficient_context
│     The owner reads one bounded, redacted test body per selected row through
│     `unit-health validate test-body`; the excerpt is capped at 4096 bytes and
│     privacy-pattern files are refused. AI labels judge the supplied excerpt,
│     remain advisory, and become `insufficient_context` when the body read is
│     unavailable or refused. Attach a supplied cohort to a validation receipt
│     with `--reviewed-cohort`, `--reviewed-source-identity`, and
│     `--reviewed-observation-count` before reading `reviewed=supplied`.
├─ "Is a finding about my scenario honest?"
│   → compare the finding against `path:docs/internal/PROBLEMS.md` before repairing      [S0]
│     Known disagreements and dead codes are recorded there; add one entry when you
│     find a new limit, then continue with the repair the finding actually justifies.
└─ "The rule catalog or the corpus is wrong"
    → prompt-manager skill read unit-health-improve                                     [S0]
```

### Reading a result without over-claiming

| Field | Read it as |
|---|---|
| `status` passed / failed / degraded | Error findings fail; degraded means discovery or a dependency was unavailable and the result must not be read as clean |
| `maturity.label` and `blocking_finding_codes` | The lowest capability with a required finding decides the level; advisory codes never block |
| `evidence_stages` | `executed=not_requested` means no test ran; `reviewed=not_supplied` means no sampled review was attached; `reviewed=supplied` means a cohort with an observed label was attached; `analyzed=partial` means at least one adapter returned unknown |
| `coverage` and `LOW_COVERAGE` | Per-file floors from the scenario's policy profile in `.vrooli/testing.json`; advisory, capped at `num[threshold]:25` findings per workspace |
| `traceability.unavailable_reason` | `OWNER_UNAVAILABLE` means the requirements owner did not answer; links are unknown, not missing |

### In-use settings

| Symptom | Move | Journal |
|---|---|---|
| A slow suite makes the executed read exceed the phase budget | `--fast-test-only` for the iteration loop; the full executed read before claiming coverage | the workspace and the fast command's duration |
| Only one workspace changed | `--workspace <id>` | the workspace id |
| Validating a path that is not a registered scenario | `--path <dir>` | the path and target kind reported |

### After acting, always

Capture one line to scope `unit-health-usage` when the read taught something the
skill does not say: a finding that disagreed with the code, a status that needed
an executed read to become true, or a rule status that surprised you. Pin after
the third confirmation; supersede when advice fails.

### Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `NOT_EXECUTED` or `executed=not_requested` | Plan-only read | the flag | Re-read with `--execution`, or request the Test Genie unit phase and wait once with `test-genie runs wait` |
| `degraded` with a Code Facts note | Discovery owner unavailable | `vrooli scenario status code-facts` | Start it through the lifecycle; a degraded plan never passes cleanly |
| The response tells you to rerun with a flag the CLI does not have | Message drift between API field and CLI flag | `unit-health validate scenario --help` | The CLI flag is `--execution`; add a ledger entry if the text regresses |
| `TEST_UTIL_MISSING` on a workspace whose projection check for the testutil root passes | The analyzers disagree about the same directory | the finding's `evidence` and the projection row | Ledger entry 2026-09-09; repair the workspace only when a consumer test needs the shared root |
| A changed source digest makes an old cohort non-comparable | Source drift | `source_identity` | Select a new cohort with a new seed |
| Provider error during sampled review | AI Gateway unavailable | `vrooli scenario status ai-gateway` | Status stays `partial` or `unavailable`; never convert to clean |
| `reviewed=not_supplied` after a cohort run | Cohort was not attached to the validation receipt | `unit-health validate scenario <name> --help` | Re-run with the cohort id, source identity, and observed count; labels remain advisory |
