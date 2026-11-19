# Testing — React Component Library

## Shared guidance

Choose focused regressions and scoped Test Genie phases using [docs/TESTING.md](../../../../docs/TESTING.md). A failing advisory suite is retained with its finding disposition; it does not erase narrower evidence. A green build alone is not visual proof.

## Focused commands

From the scenario root, run token and route checks with `node ui/scripts/design-token-generate.mjs --check` and `node ui/scripts/experience-routes-check.mjs`. Run Go package tests from `api/`; run selected Vitest files from `ui/`.

Use `react-component-library components test page:CoveragePage --version workspace --json` for application-page behavior, or the exact library ID/version for asset stories. BAS owns browser execution; the shared evaluator interprets interactions and expectations. No second ad hoc browser runner is required.

## Behavior claim register

Only the specific current-behavior claims mapped below are asserted as checked. Other documentation prose is explicitly **design intent** or a **dated observation**, not evidence that the current corpus implements it. A test names its enforcement boundary; consult its result before claiming readiness. Paths below are scenario-relative unless prefixed `packages/`.

| Current behavior | Enforcing check |
| --- | --- |
| Shared token values derive from BaseStyles; app values cannot collide | `ui/scripts/design-token-generate.mjs --check`; `ui/scripts/design-token-generate.test.mjs` |
| Logically absent Presence subtrees relinquish input; callback updates preserve exit deadlines | `library/tests/remaining-press-lifecycle.test.tsx`; native hit testing in `docs/internal/PROBLEMS.md` |
| Sidebar primary presses dismiss; closed mobile panels are inert; visible opening panels take focus once; desktop navigation retains input | `library/tests/remaining-press-lifecycle.test.tsx`; served Web Console browser evidence in `docs/internal/PROBLEMS.md` |
| Focus returns only after a real close; closed mounts and ref identity changes do not steal focus | `library/tests/remaining-press-lifecycle.test.tsx`; Chromium and WebKitGTK fixtures in `docs/internal/PROBLEMS.md` |
| Overlay swipes survive content rerenders; viewport changes and closing abort; cancelled drags preserve later pointer and keyboard activation | `library/tests/overlay-gesture-lifecycle.test.tsx` |
| Exiting drawers, sheets, responsive dialogs, dialogs, alerts and command palettes relinquish input immediately and restore it on reopen | `library/tests/overlay-exit-composition.test.tsx`; live Web Console pointer-hit evidence in `docs/internal/PROBLEMS.md` |
| Canonical library ID stamps retain separate legacy source-slot metadata | `ui/scripts/vite-plugin-asset-stamp.test.mjs` |
| Consumer major/exact imports resolve through one implementation | `packages/react-component-library/tooling/resolve-specifier.test.mjs` |
| A BAS `ux-capture.v1` outline and identity artifact join to consumer source; unstamped nodes remain unattributed | `api/internal/pageinspect/service_test.go` |
| Literal selector/testid forwarding is proven conservatively | `api/internal/reconcile/resolver_test.go`; `packages/react-component-library/tooling/dom-bindings.test.mjs` |
| Local source can resolve without a sketch adoption; zero proven bindings cannot pass | `api/internal/reconcile/verdict_test.go` |
| Required region defaults match the schema | `TestRegionRequiredDefaultsToSchemaContract` in resolver tests |
| Page routes are concrete and per-story overrides reach the browser | `api/internal/components/page_stories_test.go`; `api/internal/componenttests/bas_executor_test.go` |
| Page fixtures precede mount and block undeclared mutation methods | `ui/src/page-stories.test.tsx` |
| Every declared URL pattern has an experience document and page story | `ui/scripts/experience-routes-check.mjs` |
| Semantic catalog kind is filtered before truncation | `api/handlers/components/catalog_projection_test.go` |
| Components preserves kind tabs, search, filters and views | `ui/src/workbench/pages/ComponentsPage.story.json`; served browser sweep in `docs/concepts/WORKBENCH-REDESIGN.md` |
| Coverage never renders zero while unavailable or preparing; it retries and exposes refresh | `ui/src/workbench/pages/CoveragePage.story.json`; `api/handlers/catalog/report_cache_test.go` |
| Fixtures drag by pointer and keyboard without activating the specimen | `ui/src/workbench/data.test.ts`; `ui/src/workbench/templates/TemplateRenderer.test.tsx`; served browser sweep |
| In-context move/resize persists per component and template and resets to the authored slot | `ui/src/workbench/templates/spec.test.ts`; `ui/src/workbench/templates/TemplateRenderer.test.tsx`; served browser sweep |
| The studio edits text, layers and template presentation, and browser-local templates round-trip | `ui/src/workbench/templates/local.test.ts`; `ui/src/workbench/templates/spec.test.ts`; served browser sweep |
| A cold coverage pass is persisted, coalesced, and reported as preparing instead of timing out | `api/handlers/catalog/report_cache_test.go`; `docs/internal/PROBLEMS.md` (2026-11-19) |
| Design ranks declared pages and retains access to empty scenarios | `ui/src/workbench/pages/DesignStudioPage.story.json`; `ui/src/workbench/data.test.ts` |
| Preview source errors expose a retry control | PreviewPopoutPage page story; `ui/src/features/components/ComponentEditor.test.tsx` |
| Settings, capability readiness and unknown-asset boundaries render | Their source-adjacent page stories |
| Package boundary and pinned exports survive compilation | `packages/react-component-library/tooling/build-boundary.test.mjs`; `ui/scripts/library-pins-check.mjs` |
| Wrong/unresolved ui-health routes are not replaced with root captures | `scenarios/ui-health/cli/domains/capture` package tests |

## Document claim coverage

This inventory covers the scenario README, design intent and every Markdown document under docs. The narrow checked claim column does not promote surrounding recommendations into implementation facts. Historical numerical results keep their dates; current counts must come from their owner.

| Document | Checked claim or status | Check / evidence boundary |
| --- | --- | --- |
| [README.md](../../README.md) | Catalog/source observations, page stories and the served-bundle edit loop | The checks below and guides/asset-update-flow.md |
| [DESIGN.md](../../DESIGN.md) | Design intent only; runtime token values are not authored here | No blanket visual/readiness guarantee |
| [docs/concepts/WORKBENCH-REDESIGN.md](../concepts/WORKBENCH-REDESIGN.md) | D · Nocturne implementation record, page map, data contracts, acceptance path and open capture limits | `ui/scripts/experience-routes-check.mjs`; the workbench checks in the register above; served browser sweep. Governed capture evidence remains blocked. |
| [experience/README.md](../../experience/README.md) | Draft Review/Activity/Create contracts; aspirational claims; planned pages | Experience schema validation and route guard; draft pages reconcile advisory only |
| [docs/RESEARCH.md](../RESEARCH.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/business/GO-TO-MARKET.md](../business/GO-TO-MARKET.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/business/MONETIZATION.md](../business/MONETIZATION.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/concepts/ARCHITECTURE.md](../concepts/ARCHITECTURE.md) | The explicit boundary table only | Checks named in its Runtime observability table |
| [docs/concepts/ASSET-DERIVATION.md](../concepts/ASSET-DERIVATION.md) | The explicit ownership table only | Checks named in its Current and target ownership table |
| [docs/concepts/COLLECTIONS.md](../concepts/COLLECTIONS.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/concepts/DATA.md](../concepts/DATA.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/concepts/DOMAINS.md](../concepts/DOMAINS.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/concepts/FLOWS.md](../concepts/FLOWS.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/concepts/GESTURES.md](../concepts/GESTURES.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/concepts/INTEGRATIONS.md](../concepts/INTEGRATIONS.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/concepts/STORY-CONTRACT.md](../concepts/STORY-CONTRACT.md) | Parsed v5 grammar, argument validation, concrete page routes and API fixtures | api/internal/components/story_contract_test.go; api/internal/components/page_stories_test.go; ui/src/page-stories.test.tsx |
| [docs/concepts/UI-SPEC-RECONCILIATION.md](../concepts/UI-SPEC-RECONCILIATION.md) | Proven local source, separate coverage axes, no zero-coverage pass | api/internal/reconcile/resolver_test.go; api/internal/reconcile/verdict_test.go |
| [docs/concepts/one-asset-one-verdict.md](../concepts/one-asset-one-verdict.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/guides/asset-preview-composition.md](../guides/asset-preview-composition.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/guides/asset-update-flow.md](../guides/asset-update-flow.md) | Package exports require compilation, UI bundle requires rebuild; token probe changes the captured field width | packages/react-component-library/tooling/build-boundary.test.mjs; dated 2026-09-08 rebuild/capture receipts |
| [docs/guides/troubleshooting.md](../guides/troubleshooting.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/internal/DECISIONS.md](DECISIONS.md) | 2026-09-08 token authority and canonical stamp decisions | Token generator check; ui/scripts/vite-plugin-asset-stamp.test.mjs |
| [docs/internal/ERROR-HANDLING.md](ERROR-HANDLING.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/internal/PERFORMANCE.md](PERFORMANCE.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/internal/PROBLEMS.md](PROBLEMS.md) | 2026-09-08 closure and limitations are dated observations | Focused checks and producer receipts named in that entry |
| [docs/internal/PROGRESS.md](PROGRESS.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/internal/SEAMS.md](SEAMS.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/internal/SECURITY.md](SECURITY.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/internal/TESTING.md](TESTING.md) | This is the claim/check register, not a green-suite declaration | The named executable checks; latest receipts remain producer-owned |
| [docs/operations/DEPLOYMENT.md](../operations/DEPLOYMENT.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/operations/OBSERVABILITY.md](../operations/OBSERVABILITY.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/operations/RUNBOOK.md](../operations/RUNBOOK.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/reference/api-endpoints.md](../reference/api-endpoints.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/reference/cli-commands.md](../reference/cli-commands.md) | Page inspect returns bounded screenshot/DOM attribution and rejects nonconcrete routes | api/internal/pageinspect/service_test.go; cli/domains/page/page_test.go |
| [docs/reference/configuration.md](../reference/configuration.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/reference/sizing-contract.md](../reference/sizing-contract.md) | Design intent; dated observations retain their dates | No claim of current implementation unless a specific executable check is cited |
| [docs/reference/style-ownership.md](../reference/style-ownership.md) | Named stylesheet, utility and token fallback gate policies | api/internal/gates/ tests; no corpus-wide compliance assertion |
| [docs/reference/token-contract.md](../reference/token-contract.md) | Generated token parity and disjoint app-local values | ui/scripts/design-token-generate.mjs --check; ui/scripts/design-token-generate.test.mjs |

## Scenario-specific testing

The D · Nocturne redesign exposes Home, Components, Component, Design, Review, Coverage, Capabilities, Activity, Settings, Create and Preview from one shell. Every served route has an experience contract and an executable page story; Home, Review, Activity and Create are active contracts. They are still not production-certified: governed capture evidence is blocked by the capture service and full phone journeys are not captured. See [experience/README.md](../../experience/README.md) and [WORKBENCH-REDESIGN.md](../concepts/WORKBENCH-REDESIGN.md).

The workbench's own focused tests are:

- `ui/src/workbench/data.test.ts` — findings projection, activity projection, timestamp normalization, coverage warming detection.
- `ui/src/workbench/templates/spec.test.ts` — template shelf integrity, spec operations, slot placement, layer move/duplicate.
- `ui/src/workbench/templates/TemplateRenderer.test.tsx` — every template renders a page root, slot and selection state; placed slots float above the page and respond to keyboard move/resize.
- `ui/src/workbench/templates/local.test.ts` — browser-local template persistence, duplication, blank starter and malformed-storage recovery.
- `ui/src/workbench/preferences.test.ts` — browser-local workbench preferences.
- `api/handlers/catalog/report_cache_test.go` — persisted coverage, stale-while-refresh, single-flight cold compute and the preparing state.
- `ui/src/workbench/preferences.test.ts` — local preference round-trip and document application.
- `ui/src/App.test.tsx` — route resolution and one-shell continuity.

Page stories under `ui/src/workbench/pages/*.story.json` are the executable route contracts. The ordinary suite owner is `vrooli scenario test react-component-library --phases <relevant-phases>`. For a background run, block once with `test-genie runs wait --json react-component-library <run-id>`. Do not poll or confuse canceling a local tool with aborting the server-owned run.

## Cross-references

[Edit, rebuild and look](../guides/asset-update-flow.md#edit-rebuild-and-look), [known problems](PROBLEMS.md), [story contract](../concepts/STORY-CONTRACT.md).
