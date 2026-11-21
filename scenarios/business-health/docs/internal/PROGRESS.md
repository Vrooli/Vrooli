# Progress — Business Health

Lifecycle log for meaningful scenario changes. Future agents read this
file to understand what changed without reconstructing history from git.

This file ships empty in newly generated scenarios. Append entries when
work lands, not while work is still speculative.

## Progress Log

| Date | Author | Status | Summary |
|---|---|---|---|
| _No progress entries yet._ |  |  |  |

## Entry Template

Use this table shape when appending entries.

```markdown
| YYYY-MM-DD | author | done | Concise summary of the completed change |
```

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md) — known issues, tech debt, and deferred work
- [`DECISIONS.md`](DECISIONS.md) — durable decisions and tradeoffs
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system map

## 2026-07-02 — Phase 3 (plan: business-health-provider): contract checks + parity report

The full check set landed, mapped 1:1 onto the frozen `.vrooli/maturity.json` vocabulary (now 23 codes: `business_registry_unparseable` was added for unloadable registry files). Template-section extraction and checks live in `packages/intent-go` (`prdtemplate.go`, `registry.go`) per the single-parser ratchet; business-health composes them in `internal/checks`.

### Parity vs the native test-genie business phase (Phase 3 gate)

| Scenario | Native findings | business-health findings | Verdict |
|---|---|---|---|
| image-tools | none | none | identical |
| test-genie | `intent.ref_missing:TESTGENIE-ORCH-P0` (error) | same code, severity, and message | identical + 1 explained addition |
| go-code-graph | none | 21 findings | all explained (below) |

Explained differences (every one is intentional new coverage or a native defect, never a lost signal):

1. **`requirements_readme` (test-genie, go-code-graph)** — absorbed from prd-control-tower's `requirements_readme` rule, previously reachable only through the deprecated standards three-hop chain. The native business phase never carried it.
2. **`business_invalid_status` ×15 (go-code-graph, status "draft")** — the native rule is unreachable dead code: `NormalizeDeclaredStatus` maps any unknown status to `pending` during parsing, so `InvalidStatusRule` can never see an invalid value. business-health reads the raw declared status and restores the intended signal. Not filed as a bug: the native pipeline is deleted at plan phase 7.
3. **`business_starter_template` (go-code-graph)** — the starter module `01-foundation/module.json` is not declared in `index.json` imports; the native parser only reads declared imports and never sees the residue. business-health walks every `module.json` under `requirements/` by design (orphaned starter modules are real debt).
4. **`intent.ot_orphan` (go-code-graph P2 targets)** — new coverage: business-health is the first caller of `intent.CheckOrphanOutcome` (the OT→requirement direction previously lived only in prd-control-tower's `prd_operational_target_linkage`). P0 orphans escalate to error, preserving the legacy tier signal.

Cutover note for phase 7: scenarios with vocabulary drift (e.g. "draft" statuses) or undeclared starter modules will gain advisory warnings when delegation lands. The dimension stays advisory (severity cap at ERROR, never BLOCKER), so no suite starts failing on these.

Afid note (per the plan's risk table): `intent.ref_missing` / `intent.prd_ref_unmatched` were emitted under the ARCHITECTURE finding source by the native phase; through the delegated provider every business finding carries FINDING_SOURCE_BUSINESS, so those two codes take a one-time afid churn at cutover.

## 2026-07-02 — Phase 5 (plan: business-health-provider): the deterministic wizard

`internal/wizard` landed: the question model derives from `intent.DefaultPRDTemplate()` (validator and scaffolder share one source — required content anchors are enforced at answer time, so an accepted interview cannot fail validation), sessions persist under `data/wizard-sessions/` and resume across processes, Scaffold is a pure diff renderer and Apply is the only writer (refusing while required questions are open). Round-trip property test: for a corpus of generated answer sets, wizard output validates with ZERO findings.

Live smoke (recorded here per the phase gate):
- RPC flow against a scratch target: StartSession → SubmitAnswers (7 answers) → PreviewScaffold (6 files) → ApplyScaffold → `residualFindings: null`; standalone `ValidateScenario` on the scaffolded tree: PASSED, 0 findings.
- TTY flow: `business-health wizard start <s> --interactive` prompts each remaining question with the validator's anchor help, accepts multi-line answers (`.` terminator) and `Title :: description` target lines, re-asks on invalid answers.
- Non-interactive: `wizard answer <s> --answers file.json` (the `--answers` mode of the plan).

Notes: every OT tier requires ≥1 target (the canonical template keeps all three tiers populated; the legacy validator warned on empty tiers too). The dedup hook ships behind the `Hinter` seam (default silent no-op; the search leaf wires it in plan phase 8). Manifest gotcha: boolean flags need `"bool": true` or the parser demands a value.

## 2026-07-02 — Phase 7 (plan: business-health-provider): delegated takeover of the business phase

test-genie's `business` phase is now delegated to business-health (catalog `delegatedSpec`, phase name + FINDING_SOURCE_BUSINESS + advisory posture preserved). Deleted: `phase_business.go`, `phase_business_findings.go`, `internal/business/`, and the now-unconsumed contract-side internals (`internal/requirements/{validation,reporting,refresolver}`, `phase_inspect.go`, Service.Validate/Report/GetSummary) — the evidence syncer path (Service.Sync + discovery/parsing/evidence/enrichment/snapshot/sync) kept intact, compiler-verified. `cli/requirements` slimmed to `sync` (moved verbs answer with a pointer to business-health). The root facade (`vrooli scenario requirements`) routes contract verbs to the business-health CLI with `--auto-start` (validate/lint-prd → `validate scenario`; report → `matrix show --format summary`; phase → `matrix show --phase all`; drift → `drift show`; init → `fix apply --rules prd_missing_requirements`; manual-log → `manual-log add` with `--validated-by`→`--by`; expiry flags rejected — policy-owned now); sync/snapshot unchanged.

Gates: `provider-contract check business-health business-health` ok (L3, all capabilities clean); fleet `provider-contract scan`: `business → business-health adoption=100% reach/contract/identity/spec/metrics = yes`. Delegated phase executed on image-tools / test-genie / go-code-graph — findings match the Phase 3 parity report plus the evidence coverage (stale snapshots, unproven claims). `test-genie fix go-code-graph --deterministic` lists business-health candidates (status normalization). Full `vrooli scenario test test-genie`: phase verdicts IDENTICAL to the §9 anchor after a `go mod tidy` in api/cli (pruned packages left stale requires — the one transient delta, fixed). test-genie's `.vrooli/testing.json` inert `business` block removed.

## 2026-07-02 — Phases 8 + 9 (plan: business-health-provider): the intent search leaf + the four UI surfaces

**Phase 8 —** `business-health.intent` is a live, ACTIVE search-hub provider (BUCKET_STATE, type `requirements`): fleet-wide corpus (one doc per PRD purpose / operational target / requirement — 3,760 docs live), dense + task-prefix + rerank-blend tuning from the `.vrooli/search.json` SSOT, per-provider control token gating the shared reindex/config plane. The `product-manager-agent.requirements` capability-gap stub is fully retired (live registry, seed corpus, test pins, requirements entry) and answer-space cells #22/#34 flipped to ACTIVE/DERIVED. Wizard dedup hook live (regime-aware Weak filtering — never absolute score floors: RRF-blend scores sit ~0.03).

**Eval record (the BH-SRCH-002 gate, run 2026-07-02):** `recall@5 = 1.000 (12/12), target 0.80; stale=0; excluded_candidates=1 (the generated candidate, held out by design); gibberish negatives rejected.` Federated smoke: `search-hub query` routes to the leaf with correct top-1 at 386ms. Three tuning lessons recorded for posterity: (1) the eval grader matches `SearchResult.ID` — a `Projector` must rewrite the point UUID to the natural record id or every case grades STALE; (2) purpose docs need the OT-title capability inventory folded into their embed body or interrogative capability queries can't retrieve them; (3) capability cases' `expect_ids` accept any strong intent doc of the right scenario — cell #34's answer is the *scenario*, not one specific doc.

**Phase 9 —** all four surfaces shipped by a build agent and verified: Matrix (flagship: OT-grouped grid, evidence chips, unproven emphasis, requirement drill-in drawer with attestation), Fleet (worst-first + filters + row→matrix), Wizard (resumable interview, live preview, per-file diff, gated apply), Findings (grouped findings, remediation-doc links, PreviewFix→ApplyFix). i18n en/ja/ar, selectors manifest, 3 BAS cases (explicit scenario refs — `@scenario/self` hits an unexpanded-token resolver bug, filed). 228 vitest green; coverage branches 88.51% ≥ 85 (stable, not threshold-riding); tsc/eslint clean.
