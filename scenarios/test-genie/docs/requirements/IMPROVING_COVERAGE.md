# Improving Requirement Coverage

This is the practical companion to [STATUS_MODEL.md](STATUS_MODEL.md). It
answers the question the execute report points you to: *"How do I raise the
number of passing requirements / complete operational targets?"*

## The short version

1. Make sure the requirement exists and links to an operational target.
2. Write a test for it and **tag the test** so Test Genie can attribute the
   result to that requirement.
3. Run the **full suite** so the status is re-derived and the PRD checkbox can
   flip.

## 1. Declare the requirement

Requirements live under `requirements/`:

- `requirements/index.json` — the registry; it `imports` each module file.
- `requirements/NN-domain/module.json` — the requirements for one domain.

A requirement entry looks like:

```json
{
  "id": "REQ-AUTH-002",
  "title": "Session cookie is HttpOnly and Secure",
  "prd_ref": "OT-P0-001",
  "status": "in_progress",
  "validation": [
    { "type": "test", "ref": "api/auth_test.go", "status": "planned" }
  ]
}
```

- **`prd_ref`** is what links the requirement up to a PRD operational target
  (`OT-P0-001`). Without it, the requirement is counted but does **not**
  contribute to any operational-target checkbox.
- **`validation[]`** declares how the requirement is proven. `type: test` is the
  common case; `automation`, `manual`, and `lighthouse` also exist.
- You do **not** hand-edit `status` to `complete` — Test Genie derives it from
  the validations (see STATUS_MODEL.md). Setting it manually is overwritten on
  the next full sync.

## 2. Tag your tests with `[REQ:ID]`

Test Genie attributes a test result to a requirement by scanning test output
for a `[REQ:<id>]` marker. Put the tag in the test name or a log line so it
appears in the phase output:

```go
// Go
func TestSessionCookieIsHttpOnly(t *testing.T) { // [REQ:REQ-AUTH-002]
    // ...
}
```

```ts
// TypeScript / vitest
it("[REQ:REQ-AUTH-002] session cookie is HttpOnly and Secure", () => {
  // ...
});
```

When that test passes, the matching validation becomes `implemented`; when
**all** of a requirement's validations are `implemented`, the requirement
becomes `complete`; when all requirements behind an operational target are
`complete`, the OT checkbox flips to `[x]`.

## 3. Run the full suite to refresh status

Status is only re-derived (and the PRD rewritten) on a **full** run, because a
partial run hasn't seen all the evidence. A targeted run shows the *last*
counts with a `⚠ Not updated this run` notice.

```bash
# Refresh requirement status and PRD checkboxes for a scenario:
test-genie execute <scenario>

# Or, if you only changed one area, you can sync explicitly without a full run
# of every phase (advanced; uses whatever evidence is already on disk):
test-genie requirements sync --scenario <scenario>
```

After a full run, the report's **Changes this run** section tells you exactly
which requirements moved — and flags any that regressed.

## Fixing a regression

A regression (`complete → in_progress`) means a previously-passing test now
fails. The report names the requirement and its `prd_ref`. To resolve it:

1. Find the failing test (the same run's failure digest lists it).
2. Fix the code or the test.
3. Re-run the full suite; the requirement should return to `complete` and the
   OT checkbox re-check.

## Inspecting status without a run

```bash
# Human-readable rollup of requirement/OT status:
test-genie requirements report --scenario <scenario>

# Machine-readable:
test-genie requirements report --scenario <scenario> --format json
```

## See also

- [STATUS_MODEL.md](STATUS_MODEL.md) — how the numbers are derived.
- `PRD.md` in your scenario — the operational targets themselves.
- `concepts/requirement-flow.md` — the end-to-end validation flow.
