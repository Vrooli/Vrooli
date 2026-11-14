# Problems — Treasury

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

Append entries as they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-08-18 — Generated scenario cannot complete Gate 0 (scaffold health)

**Symptom:** `make setup` / `vrooli scenario start treasury` fails at the
`build-ui` step with exit code 2. `tsc --noEmit` rejects the generated file
`ui/src/test-utils/renderWithProviders.test.tsx` with TS2322: two different
`QueryClient` types, one from `@tanstack/query-core` 5.100.9 and one from
5.59.0.

**Root cause:** a version-range mismatch between two governed sources. The
`react-vite` template pins `"@tanstack/react-query": "^5.59.0"` in
`templates/scenarios/react-vite/ui/package.json`, which resolves to 5.100.9
and brings `query-core` 5.100.9. The shared package `packages/api-base`
pins `@tanstack/react-query` at exactly 5.59.0, bringing `query-core`
5.59.0. Both land in the generated scenario's lockfile and the generated
test file mixes them. **This is inherited template debt, not specific to
this scenario** — any scenario generated from `react-vite` today hits it.
`money-ledger` and `offer-desk` carry no tanstack dependency, which is why
the fleet has not surfaced it.

**Workaround:** none for the UI build. The Go API builds and the
documentation gates are unaffected, which is why gates 1-5b completed while
gate 0 did not.

**Real fix:** align the template range with the `api-base` pin, or widen
the `api-base` pin — through `scenario-dependency-analyzer`, never by hand.
Not attempted here: shared-package version changes ripple across the whole
fleet and are outside the scope of authoring one scenario's documentation.

**Owner:** unassigned; filed to scenario-qa.

**Refs:** scenario-qa bug `knw-1787080554065889915`
(`bug-inbox/code-defect/every-newly-generated-react-vite-scenario-fails-build`);
runtime evidence is available through `vrooli scenario logs treasury`.

**Resolution:** On 2026-08-18, the scenario-local React Query version was aligned
to 5.59.0 through Scenario Dependency Analyzer. Treasury now builds and Test
Genie executes all 22 phases. The fleet-level generator defect remains tracked
by the referenced scenario-qa bug.

### 2026-08-18 — Experience pages have no BAS observer cases

**Symptom:** `experience-manager spec validate treasury` reports four
`experience.route_unspecced` warnings for `approvals`, `budgets`,
`mandates`, `activity` and `rails`.

**Root cause:** the experience contract now describes the intended product
routes, but no UI exists yet, so there are no selectors for a BAS case to
wait on. The generated `dashboard` observer smoke was retargeted to
`approvals` (which owns `/`), and its selector reference
`@selector/pages.approvals` is a placeholder until the page is built.

**Workaround:** none needed. The warnings are accurate.

**Real fix:** add one observer smoke per page as each page is implemented,
with real selectors. Also raise the page claims from `aspirational` to
`machine` tier once stable roles and `data-testid` selectors exist — the
claims are currently stated intent that the validator cannot check.

**Owner:** whoever implements the UI.

**Refs:** `experience/pages/`, `bas/cases/experience-spec/approvals.json`.

### 2026-08-18 — Every security mitigation is designed, not verified

**Symptom:** `docs/internal/SECURITY.md` describes a complete threat model
in which every mitigation carries status `designed`.

**Root cause:** no implementation exists. The structural claims — that the
agent-facing service declares no policy-mutating method, that identity
fails closed, that the schema refuses a third-party beneficiary — are
architectural intentions until a test earns them.

**Workaround:** treat the security posture as a specification to build
against, not as a description of the system. Do not repeat any claim from
that document as fact in a plan, report, or external statement until its
requirement's validation is earned by a passing test.

**Real fix:** implement, and let requirement sync flip the statuses from
evidence rather than assertion.

**Owner:** whoever implements the P0 spine.

**Refs:** `docs/internal/SECURITY.md`, `requirements/01-must-ship/module.json`.

**Progress:** Phase 2 now verifies the mandate's named grant constraints,
signed immutable representation, read-time expiry, and the schema-level
single-beneficiary boundary. Phase 3 now verifies live fail-closed identity,
server-side policy evaluation, named refusals, derived headroom and concurrent
pending holds. Phase 4 now verifies the exact agent-facing descriptor and the
operator-realm boundary on every admin method. Phase 5 now verifies the local
approval queue, optional relay failure isolation, and release on decline or
expiry. Phase 6 verifies rail uniformity, the no-mandate refusal, manual-rail
parity, mandate-derived instrument scope, and reference-only credential
storage. Phase 7 now verifies exactly-once settlement, durable ambiguity,
query-only failure resolution, cancellation-safe outcome recording, and live
identity rebinding. Complete attempt evidence and ledger emission remain
designed until Phase 8.

**Resolution:** Phases 8–16 now verify complete immutable evidence, durable
Money Ledger emission, operator controls, scoped-card issuance, and the final
security/comprehensive suites. The implementation gap described here is
closed; operator-dependent live rail proofs remain separate entries below.

### 2026-08-18 — Storage Manager mistakes a domain Query result for sql.Rows

**Symptom:** Test Genie run `20260819-020951-3b1f6edd` failed Storage with
`DB_ROWS_NOT_CLOSED` at the assignment from `rail.Adapter.Query`, although the
method returned a value-typed `rail.Result` and no database cursor existed.

**Workaround:** the rail contract now uses the clearer name `QueryOutcome`.
Focused `storage-manager validate scenario treasury --json` passes with no
errors after that rename.

**Real fix:** make the rows-close analyzer resolve the call's return type before
classifying a method named `Query` as a database cursor.

**Owner:** Storage Manager / scenario-qa `knw-1787106202657008986`.

### 2026-08-18 — governed install and tidy validation disagree

**Symptom:** Test Genie run `20260819-014547-256280ab` fails only the
dependency package-readiness check added in Phase 6, while dependency
governance itself passes.

**Root cause:** `scenario-dependency-analyzer deps install` successfully adds
the approved in-repo credential-client dependency but leaves its owning module
annotated `// indirect`. The same analyzer's health provider runs
`GOWORK=off go mod tidy -diff`, which requires moving that module into the
direct block. Re-running the governed root and subpackage installs does not
reach the enforced tidy fixpoint; `deps reconcile` owns missing replaces only.

**Workaround:** none inside Treasury. The repository rule correctly forbids a
raw package-manager command or hand edit of `go.mod`.

**Real fix:** make the governed install gateway finish at the tidy fixpoint its
own validator enforces, then rerun the Treasury comprehensive suite.

**Owner:** Scenario Dependency Analyzer.

**Refs:** scenario-qa `knw-1787104257680826857`, run
`20260819-014547-256280ab`, `scenarios/treasury/api/go.mod`.

**Resolution:** The governed dependency state now passes the dependency phase
in comprehensive run `20260819-063957-7b748ec4`. Keep the upstream bug for its
installer/validator consistency work; Treasury no longer needs a workaround.

### 2026-08-19 — live recurring transaction needs operator-owned facts

**Symptom:** The P1-001 recurring payment and P1-002 operator-use targets have
no real settlement, evidence ID, or downstream Money Ledger posting.

**Root cause:** The repository cannot choose or invent a real obligation. It
also cannot authorize an Agent Manager run on the operator's behalf.

**Workaround:** Use automated and attended local fixtures for implementation
validation. Do not present them as a real transaction.

**Real fix:** Supply the merchant, amount, currency, payment time, non-sensitive
receipt reference, confirmation that payment is due, and an authorized agent
identity. Then settle against Money Ledger book
`87f1fc53-97c9-48ed-b4c8-6c09499e36a2` and account
`99e87b30-e093-47e3-8f45-4b15838ae798`.

**Owner:** operator.

**Refs:** `requirements/02-post-launch/module.json`,
`docs/operations/RUNBOOK.md`.

### 2026-08-19 — live x402 proof needs a funded signer

**Symptom:** P1-001 has no cent-scale outbound and inbound x402 settlement on
a real network.

**Root cause:** No wallet, signer, network selection, or funds were supplied.

**Workaround:** Contract, adapter, policy, and failure behavior are covered by
local tests. They do not prove value moved.

**Real fix:** Provision an approved signer and funded test account through the
governed credential path, select the network, and run both directions against
the self-hosted facilitator.

**Owner:** operator.

**Refs:** `api/internal/rail/x402/`, `handlers/x402gate/`,
`requirements/02-post-launch/module.json`.

### 2026-08-19 — scoped-card enforcement needs a Lithic sandbox account

**Symptom:** P1-003 has no live provider response proving that an amount or
merchant outside the mandate is declined by the issuer.

**Root cause:** Credential identity `vrooli/treasury/lithic` is not configured,
and the sandbox account/API key must be created by an operator.

**Workaround:** The local fake provider proves the exact issued scope and
provider-decline behavior. The live test remains gated by
`TREASURY_LITHIC_SANDBOX_INTEGRATION=1` and resolves its key through Credential
Authority.

**Real fix:** Create a Lithic sandbox account, provision its API key at the
documented credential identity, and run the operator-gated integration test.

**Owner:** operator.

**Refs:** `api/internal/rail/card/lithic/adapter_test.go`,
`docs/operations/RUNBOOK.md`, `requirements/02-post-launch/module.json`.

### 2026-08-19 — requirement sync promotes planned manual validations

**Symptom:** `test-genie requirements sync` changes planned manual-only P2
validations to implemented after the named suite phase passes, then checks off
unimplemented PRD targets. Business Health immediately reports them as
unattested manual claims.

**Root cause:** Test Genie's sync writer conflates a passing phase with a manual
attestation.

**Workaround:** Run the required sync once, then restore unimplemented manual
requirements and PRD targets to planned. Business Health returns clean L3.

**Real fix:** Require manual-log evidence before promoting a manual validation.

**Owner:** Test Genie / scenario-qa `knw-1787121761545564738`.

**Refs:** `requirements/03-future/module.json`, `PRD.md`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## Work ladder

- Rung: W3 / R0
- Evidence: W0 comparison still matches goal `treasury-full-implementation` to every P0/P1 operational target; W1 Business Health is clean L3; W2 `vrooli scenario requirements validate treasury` passes; and W3 Test Genie run `20260819-065815-5977126a` passes all 22 phases. A 2026-08-19 live-state recheck found only `phase8-*` fixture rows in Treasury, no x402 prices or admissions, no non-fixture Treasury posting in Money Ledger, `vrooli/treasury/lithic:value` configured=false, and facilitator support `{kinds:[], extensions:[], signers:{}}` with empty chains/schemes.
- Blocker: R1 remains unearned until the operator supplies (1) a real recurring obligation's merchant, amount, currency, payment time, non-sensitive receipt reference, confirmation it is due, and authorization to execute it; (2) an approved funded x402 signer, network, asset, and live counterparty; and (3) an operator-created Lithic sandbox credential. P1-001, P1-002, and P1-003 remain planned; no fixture or mock is represented as live financial evidence.
- Measured: 2026-08-19
