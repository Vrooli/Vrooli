# Testing — Scenario to Plugin

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

Existing local entry points (choose the domain test for the behavior you change):

- [api/handlers/health/handler_test.go](../../api/handlers/health/handler_test.go)
- [ui/src/App.test.tsx](../../ui/src/App.test.tsx)
- [ui/src/features/health/HealthCard.test.tsx](../../ui/src/features/health/HealthCard.test.tsx)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## Validation strategy for this scenario

> Every requirement in `requirements/` currently points its single
> `manual` validation entry at this document. This section is what that
> reference means. Replace those entries with real code-typed refs as each
> domain lands — at least two automated layers for every P0 and P1
> requirement.

This scenario's tests carry an unusual burden: **its product is refusal.**
A suite that only proves the happy path proves nothing that matters here,
because every valuable behavior is a gate closing on bad input.

### The controlling rule — test the refusal, not just the pass

For every gate, the primary test is a deliberately broken fixture that
must fail, with the *specific* finding named. A test asserting that a good
package passes is necessary but secondary: a gate that always returns
`pass` satisfies it.

Fixture corpus lives at `api/internal/testdata/fixtures/` and is shared
across domains. Each fixture is minimal and breaks exactly one thing:

| Fixture | Breaks | Must be caught by |
|---|---|---|
| `drift-removed-command/` | A `SKILL.md` documents a command absent from the pinned manifest | `PLG-CONF-DRIFT` |
| `drift-renamed-flag/` | A documented flag no longer exists | `PLG-CONF-DRIFT` |
| `frontmatter-unknown-key/` | A key outside the permitted set | `PLG-CONF-FRONTMATTER` |
| `frontmatter-name-mismatch/` | `name` does not match its folder | `PLG-CONF-FRONTMATTER` |
| `frontmatter-angle-bracket/` | An angle bracket in frontmatter | `PLG-CONF-ANGLE` |
| `body-zero-width/` | A zero-width character in the body | `PLG-CONF-UNICODE` |
| `body-bidi-mark/` | A bidirectional control mark | `PLG-CONF-UNICODE` |
| `body-non-nfc/` | Non-NFC normalized text | `PLG-CONF-UNICODE` |
| `tools-unrestricted-shell/` | `allowed-tools` grants unrestricted shell | `PLG-CONF-TOOLS` |
| `install-mutable-ref/` | A download pinned to a branch | `PLG-CONF-INSTALL-PIN` |
| `install-no-checksum/` | A download with no checksum verification | `PLG-CONF-INSTALL-SUM` |
| `install-elevation/` | An install script requesting elevation | `PLG-CONF-INSTALL-PRIV` |
| `secret-literal/` | A credential literal in the tree | `PLG-ATTEST-NO-SECRETS` |
| `install-not-idempotent/` | Second install run is not a no-op | `PLG-REHEARSE-IDEMPOTENT` |
| `install-stealth-acquire/` | Acquires an undeclared resource | `PLG-REHEARSE-NO-STEALTH` |
| `clean/` | Nothing — the control fixture | Must pass every gate |

Adding a gate means adding a fixture. A gate with no failing fixture is
untested regardless of its line coverage.

### Ordering tests are correctness tests

The pipeline chain is enforced in the domains, so it must be tested there:

- Attestation must refuse to sign when the conformance record is absent or
  failing (`PLG-ATTEST-ORDER`). Assert the refusal, and assert that **no
  signature was produced** — not merely that an error was returned.
- Distribution must refuse to publish without a release decision bound to
  the same source commit (`PLG-DIST-GATE`). Assert that **no channel call
  was made**, using a channel fake that records invocations.
- A stage must never advance from a non-terminal predecessor state. Cover
  every illegal transition in `FLOWS.md` §State Machines.

### `unavailable` is not `failed`

A timed-out or infrastructure-blocked stage must record `unavailable`, not
`failed`. Test both paths explicitly: a package that was never judged must
not be attributed a failure. This is the single most likely correctness
regression in this scenario, because the two look alike from the outside.

### Side-effect teardown

Rehearsal allocates a real sandbox. Assert teardown on **every** exit path
— success, failure, timeout, cancellation, and panic. Follow the
side-effect cleanup pattern used by the example domain's flow tests. A
leaked sandbox is the one failure here that consumes real host resources.

### Never test against the network

Composition, conformance, and attestation ordering must be provable
offline. Scanners, Sigstore, and registries are seams with fakes
(`api/internal/*/mocks/`). A test that reaches the network is a defect in
the test, not a stronger test.

### Redaction tests

Assert that no credential literal, no full skill body, and no unredacted
command output appears in logs, findings, verdicts, or attestations. A
skill body under test may be adversarial by construction, so echoing it
into a log is a real vulnerability rather than a cosmetic issue.

### Layers per domain

| Domain | API | CLI | UI | E2E (BAS) |
|---|---|---|---|---|
| declaration | loader, readiness, prerequisite evaluation | `readiness`, `declaration show` | readiness board, blocking-reason display | readiness board shows the blocking prerequisite |
| composition | tree, manifest, component independence, mcp | `package build`, `package show` | package detail, manifest preview | — |
| conformance | every rule, against every fixture | `check run`, `check show` | finding list, drift detail with pinned revision | drift blocks a package end-to-end |
| attestation | ordering, scan verdict, redaction, sign, provenance, sbom | `attest run/show/verify` | scan verdict, provenance, SBOM summary | secret literal blocks publish before any network call |
| rehearsal | sandbox lifecycle, idempotency, disclosure, evidence emission | `rehearse run/show` | journey manifest, command results | command exercise evidence; stealth acquisition blocked |
| distribution | verdict, gate, oci, confirm, revoke, partial revoke | `publish`, `revoke`, `channels list` | publish summary, channel state, revocation state | publish blocked without a gate decision; revocation fans out |
