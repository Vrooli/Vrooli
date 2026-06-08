# Problems — Measures Health

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

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

### 2026-06-08 — Phase 4 remaining deliverables (handoff)

**Symptom:** The scenario enforces + grades measure adoption (the `validation`
domain + the test-genie→EM producer) but does not yet *federate/answer*
analytical questions, has no fleet UI, and still ships the `notes` example.

**Root cause:** Phase 4 was landed in dependency order; the federation half and
UI remain.

**Status update 2026-06-08 (continuation):** **Deliverable 3 (the `index`
domain) is DONE and green** — `SearchService.Search/Status`
(`packages/proto/schemas/measures-health/v1/search/`), `internal/measureindex/`
(harvest → `LexicalMatcher` → `measures-go` Engine with HTTPExecutor +
LLMExtractor + θ), `handlers/search/`, CLI `measures-health search query|status`,
and `.vrooli/search.json` self-registered via `searchregister.Register` at boot.
33 new tests (matcher/harvest/provider gate/handler **wire-shape**/search.json
contract) + gofumpt + golangci-lint clean; endpoints.json regenerated. **One
deliberate, documented deviation** (see DECISIONS.md): the Matcher is a
deterministic **lexical** index over `questions[]`, NOT the plan's aisearch-go
qdrant hybrid index — because **no scenario declares a `measure` block yet**
(Phase 5/6), so a vector index would index an empty corpus and be unexercisable.
`measures.Matcher` is the seam; the aisearch hybrid index drops in once a corpus
exists. **Remaining for Phase 4: Deliverable 5 (fleet UI) + Gate 7 + close-out
(below).**

**Follow-up (Phase 6+) — wire the aisearch hybrid index:** once ≥1 scenario
declares a measure, implement `measures.Matcher` with an aisearch-go hybrid index
(embed `questions[]` via `measures.MeasureComposer`, store the serialized decl as
`SourceDoc.Body`, qdrant retrieval — mirror `cli-health/internal/aisearch`) and
layer it over / swap it for `LexicalMatcher` in `measureindex.NewProvider`. This
is the only piece of the original Deliverable-3 text deferred, and it is deferred
precisely so the live retrieval path can actually be validated (against a real
corpus) rather than shipped unexercised.

**Remaining work (precise):**

1. ~~**Deliverable 3 — central index + search-hub provider (the `index`
   domain).**~~ **DONE 2026-06-08 (see status update above).** Original spec, for
   reference / the aisearch follow-up:
   Mirror `cli-health` exactly (it is a working search provider):
   - New proto domain `packages/proto/schemas/measures-health/v1/search/` with a
     provider query `SearchService.Search(query, limit) -> {results:[{score, measure:{…}}]}`.
     The `measure` object is snake_case and maps to `SearchHit.measure`
     (Phase 3's `routing.proto` carrier); the registration descriptor sets
     `result_mapping.measure_field: "measure"`.
   - Harvest **all** scenarios' manifest measure blocks (reuse
     `internal/measurescan` — `Parse` + `Assemble`, same as `validation`), build
     an `aisearch-go` hybrid index embedding the `questions[]` via the
     **measures-go `MeasureComposer`** (`measures.MetaQuestions`/`MetaIntent`
     meta keys; store the serialized declaration as `SourceDoc.Body`).
   - Serve the query via the shared **measures-go `Engine`** (`NewEngine(matcher,
     WithExecutor(HTTPExecutor), WithExtractor(ollama Completer), WithThreshold(θ))`)
     — do NOT re-author matching/gate/exec. The `Matcher` is the aisearch index;
     the `Executor` is `measures.HTTPExecutor` over an api-core discovery
     `BaseURLResolver`; the `Completer` is a `resource-ollama gateway generate`
     shell-out (copy `search-hub`'s `classifier_ollama.go`).
   - Register once at boot via `searchregister.Register()` reading a new
     `.vrooli/search.json` (descriptor `provider_id: measures-health.measures`,
     `type: "measure"`, `bucket: BUCKET_STATE`). Pattern: cli-health
     `api/main.go:199` + `packages/searchregister-go`.
   - θ lever: `measures.DefaultConfidenceThreshold` (0.8), exposed via
     `MEASURES_HEALTH_CONFIDENCE_THRESHOLD`.
   - Also serve the shared token-gated `search-hub.v1.control.SearchControlService`
     (Reindex/WriteConfig) like cli-health's `handlers/searchcontrol/` if the
     control plane is wanted (optional for the acceptance criterion).
   - Acceptance: `search-hub query "how many backlog items closed this week"`
     returns a measure hit end-to-end (needs an adopter with a live measure —
     swarm-manager Phase 6 — or a fixture provider for the test).

2. ~~**Deliverable 5 — fleet-view UI**~~ **DONE 2026-06-08.** `ui/src/features/fleet/`
   (`FleetTable` over `ListFleetCoverage`, `ScenarioCoverageCard` drill-down over
   `ValidateScenario`, `FleetView` composer, `coverage.ts` enum presentation),
   `ui/src/pages/FleetPage.tsx` at `/fleet`, nav/route/selectors/i18n wired. 16
   fleet tests; full UI suite 175/175 green; new code 0 lint violations. Built
   **beside** notes (scaffold's prescribed "real domain beside notes, then remove
   notes" flow) — so Gate 7 is the clean next step. Along the way fixed
   pre-existing template debt: RR-v6 future-flags (router opt-in in
   `app/routes.tsx` + `test-utils/renderWithProviders.tsx`), duplicate
   nav-landmark a11y (distinct `layout.bottomNavLabel`), AppShell hardcoded
   `aria-label`→`layout.mainLabel`. KEPT (genuine false positives, same as
   security-health): ThemeProvider `!window.matchMedia` guards — jsdom needs them
   though lib.dom types call them "always falsy"; reverting them breaks
   ThemeProvider/App tests. Remaining UI lint = 5 unused-key + 2 ThemeProvider +
   2 react-refresh, ALL universal template debt (security-health carries the same
   class; standards baseline tolerates).

3. ~~**Gate 7 — remove the `notes` example domain**~~ **DONE 2026-06-08.**
   Removed api/cli/ui/proto/gen/seed/i18n/selectors + the `notes` group in
   `cli/manifest.json` (a file the map below missed) + the
   `api/cmd/gen-endpoints/main_test.go` notes fixture rows (also missed).
   `make endpoints` + `pnpm strings:gen` regenerated. Residue search across
   `api/ cli/ ui/src/ .vrooli/` + proto schema dir is CLEAN (only the
   legitimate waiver/tier `Note` vocabulary in `internal/validation` remains).
   Re-pointed all in-code worked-example doc-comments (module/errors/httpc/
   pinger/repokit/testutil) off `notes`. `vrooli scenario orient` passes
   `example-domain-removed`. Deeper internal-doc drift (SEAMS/ARCHITECTURE/
   reference) deliberately left + tracked in the Architecture Drift table above
   (matches shipped security-health; Gate 7 scope = product code). Domains are
   now `health`, `validation`, `search`. Original precise file map (for the
   record):
   - **Delete dirs/files:** `api/internal/notes/`, `api/handlers/notes/`,
     `cli/domains/notes/`, `packages/proto/schemas/measures-health/v1/notes/`
     (notes.proto + attachments.proto), `ui/src/features/notes/`,
     `ui/src/api/notes.ts`, `ui/src/api/notes.test.ts`, `ui/src/pages/NotesPage.tsx`.
   - **Edit wiring (must still compile):** `api/internal/modules/registry.go`
     (drop `notesH` + `notesv1` imports, `out = append(out, notesH.Endpoints...)`,
     and the `{Module:"notes", File:notesv1...}` descriptor row), `api/main.go`
     (notes module mount), `cli/domains/domains.go` (notes register). Fix the
     notes-coupled tests that will then break: `api/internal/module/module_test.go`,
     `api/internal/modules/registry_test.go`, `api/internal/server/server_test.go`,
     `api/internal/database/system_test.go`.
   - **UI coordination:** remove notes from `ui/src/app/routes.tsx` (import+route),
     `ui/src/layout/navItems.ts` (item + key union), `ui/src/consts/selectors.ts`
     (notes literal block, `pages.notes`, `"notes"` from the 2 nav-key enums),
     en/ja/ar.json (`layout.nav.notes`, `pages.notes`, the `notes` block), then
     `pnpm strings:gen`. Update tests: `ui/src/app/routes.test.tsx` (drop /notes
     case), `ui/src/layout/AppShell.test.tsx` (drop `"notes"` from the nav loop).
   - **Generic doc-comments referencing `internal/notes` as the worked example**
     (NOT domain code — they don't block compile): `api/internal/database/pinger.go`,
     `api/internal/httpc/doer.go`, `api/internal/httpx/errors.go`,
     `api/internal/testutil/repokit/{doc.go,slicerepo.go}`,
     `api/internal/testutil/db/sqlite.go`, `api/internal/module/module.go`.
     Re-point these to the real domain (`internal/validation` /
     `internal/measureindex`) to clear residue per Gate 7's "no product residue"
     search.
   - **Seed:** remove notes rows from `api/cmd/gen-endpoints/cli_commands_seed.json`,
     then `make endpoints`. Expect proto-gen churn (deleting notes proto regens
     `packages/proto/gen/**` — consistent with the existing regen note below).

4. **Close-out:** capture `git-control-tower baseline snapshot --scenario
   measures-health --name measures-p4` was NOT taken at phase start (the
   scenario did not exist yet); take a fresh baseline now as the new floor.
   Also: test-genie + ecosystem-manager baselines `measures-p4` WERE snapshotted
   at phase start — run `baseline diff` for both at close-out. Outside-scenario
   allowlist this phase touched: `packages/proto/schemas/{architecture,measures-health}/**`
   (+ the broad `packages/proto/gen/**` regen — see the regen note below).

**Workaround / state today:** the `validation` half is fully green and usable
(`measures-health validate scenario <name> [--probe] --json`,
`measures-health validate coverage`). The producer feeds the EM `measures`
dimension at soft R4.

**Owner:** next Phase 4 continuation agent.

**Refs:** `api/internal/validation/`, `api/internal/measurescan/`,
`api/handlers/validation/`, `cli/domains/validate/`,
`packages/proto/schemas/measures-health/v1/validation/`; producer:
`scenarios/test-genie/api/internal/orchestrator/phases/phase_measures.go`,
`scenarios/ecosystem-manager/api/pkg/{dimensions,ladder}/`.

### 2026-06-08 — broad `packages/proto/gen` regen churn

**Symptom:** `make generate` (run to emit the measures-health validation proto +
the `FINDING_SOURCE_MEASURES` enum) also rewrote ~78 unrelated generated files.

**Root cause:** the committed `packages/proto/gen` was stale relative to several
already-uncommitted source protos on `agi` (Phase 3's 4 search-hub files +
cli-health `reindex` deletion) AND a cosmetic buf-version import-reorder on
~70 untouched scenarios. My regen produced the canonical, consistent
`buf generate` output for the current source.

**Workaround:** kept as-is — reverting generated files would only re-introduce
source/gen drift, and the memory hard-ban rules out `git checkout/restore`. The
search-hub `MeasureHit` gen (Phase 3) is now correctly present.

**Real fix:** commit the gen as the canonical output, or have a future
proto-owning change re-run `make generate` cleanly on a synced tree.

**Owner:** unassigned (cross-cutting).

**Refs:** `packages/proto/gen/**`; `git status --short packages/proto`.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| `docs/internal/SEAMS.md` + `ARCHITECTURE.md` + reference pages (`api-endpoints.md`, `cli-commands.md`, `ui-manifest.md`, `DATA.md`, `FLOWS.md`, …) | Still document the template `notes`/attachments worked-example domain (4 full seam entries in SEAMS.md; RESTReasonMultipartUpload "notes attachments" worked-example in ARCHITECTURE.md) that **Gate 7 deleted from code/proto/.vrooli on 2026-06-08**. Product code, `.vrooli`, and the proto schema dir are clean; only docs retain the examples. | Low — cosmetic/teaching drift, not load-bearing for the live `validation`/`search` domains. Matches the shipped `security-health` reference, which carries identical doc residue post-Gate-7 (41 notes refs in its SEAMS.md across 20 doc files) and still passed DoD. Gate 7's defined residue scope is `api/cli/ui/.vrooli/proto`, not docs. | Rewrite SEAMS.md's seam entries to document the real `validation`/`measureindex`/`search` seams and re-point the ARCHITECTURE/reference worked-examples. Best done as a dedicated docs pass (it is a Phase-4 *documentation* deliverable, not Gate-7 product-code removal). |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
