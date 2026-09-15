# Native Signal / Studio renderer checkpoint

This directory is the bounded, unmounted LPBS renderer slice. The Aquila branding
checkpoint is separate:
`/home/matthalloran8/.vrooli/plan-artifacts/efforts/aquila-configurable-product-presentation/findings/web-console-ui-branding.md`.

## Integration boundary

```tsx
import { PresentationPage, actionKey } from './presentation';

<PresentationPage
  presentation={decodedPresentation}
  display={configuredDisplay}
  resolvedActions={resolvedActions}
/>
```

The exported `PresentationPageProps` is the current boundary. `presentation`
uses Go's snake_case ResolveResult/Page/Block/Content shapes, including
`single_app` and `app_detail`. The parent shared-API decoder owns protobuf
oneof unwrapping through generated descriptors; this module does not decode,
fetch, select eligible apps, infer routes, sort membership, or access providers.

`resolvedActions` is a map keyed by `actionKey(action)` (kind, app_key,
plan_ref, target). Values are `{status:'ready', href}`,
`{status:'ready', onActivate}`, or `{status:'unavailable', reason}`.
Only explicit owner joins enable transactional actions. Configured anchors and
app-detail routes are local navigation. Unresolved transactions are disabled with
a visible, associated configured reason. No fake download/purchase modal exists.

## Canonical contract review and remaining differences

Read Go `api/internal/presentation/types.go` and `validation.go` after the typed
workspace/backdrop/workflow, artifact and voice additions (2026-09-15).
`resolvedResources.ts` mirrors their serialized fixture/asset fields.
Canonical `presentation.fixtures[]` and `presentation.assets[]` are the only
demo-content / asset-URL sources. No duplicate content sidecar remains.
`resources` is now an internal index, not an integration prop.

The remaining **display-only configuration** is explicit in `resources.ts`.
It has no brand defaults, fixture data, asset URLs, or page-name heuristics.
Parent/core coordination is still required before the response alone can render
the complete mockup. Exact required typed fields:

| Canonical destination | Fields still needed |
| --- | --- |
| Page shell / Navigation / Footer | brand_name, brand_mark, brand_target, brand_subtitle, skip_label, menu_label, footer_brand_name, footer_brand_mark, footer_brand_target, footer_tagline, copyright, footer_note, unavailable_reason, preview_label, header_action (Action) |
| WorkspaceFixture | mark (finite Mark), avatar, time, tabs_label, terminal_label, messages_label, file_changes (filename to string label; current files remain string[]) |
| BackdropFixture | mark (finite Mark); all actual exhibit content already canonical |
| ResolvedAsset or configured use-site | alt; optional sizes |
| Block display | optional eyebrow, description, note, accessibility_label, badge, mark, formats[], anchors[capability_id], heading_breaks[] |
| HeroItem | fixture_ref (optional, exact fixture ID); current visual_ref is validated as an asset |
| DeviceStoryContent | fixture_ref (optional, exact workspace fixture ID); current visual_ref is validated as an asset |
| ResolvedAppSpotlight | fixture_ref or visual_ref, mark, tone (amber/sage), detail_label |

For now these fields are passed as `display: PresentationDisplay`.
`display.blocks[id].hero_fixture_refs[app_key]` and
`display.blocks[id].fixture_ref` are explicit joins for the two missing canonical
references. They never change selected apps. `display.apps[app_key]` supplies
catalog visuals; it does not decide catalog membership.

Go's valid hero exhibit names are used: artwork/screenshot/product-view/visual.
Workspace/backdrop/workflow are fixture discriminants, not invented exhibit names.
Voice waveform data uses canonical normalized [0,1] samples. Headline content has
no embedded line breaks in the fixture: explicit character offsets select editorial
line breaks without duplicating copy. Go currently disallows newlines in text.

The fixtures are **renderer review inputs**, not publishable Go Documents or
verified release records. The native workspace/phone display joins currently
override an illustrative asset reference; replace these joins with the canonical
fixture_ref additions, not screenshot substitutions. Review asset hashes are real
hashes of copied bytes; their review release/provenance IDs are not production
asset receipts. Studio's illustrative membership does not publish Backdrop Studio.

## Implemented behavior and finite bounds

- Signal centered product hero and Studio editorial bundle hero.
- One explicit visual group per selected hero app; artwork studies stay inside
  that app group. No cap/sort/slice inference in the renderer.
- Native workspace session/sidebar, file changes, review summary, terminal and
  conversation controls; catalog exhibits contain no nested interactive controls.
- Keyboard artifact tabs: arrows, Home/End, roving focus, associated hidden panels.
  Plan/flow/checks, image composition, structured HTML composition, video storyboard;
  audio/code/PDF use safe configured text excerpts. No iframe, executable HTML,
  injected CSS, fake microphone, or fake playback controls.
- Configured voice feature rows/waveform/transcript/summary/qualification; native
  phone/session view; roadmap status labels and constraints, story, catalog,
  closing/footer, simple pricing-action and FAQ blocks.
- Native workflow fixture renderer exists; no browser-automation product is
  enabled or added to these review pages.
- `contract.ts` lists the implemented visual variants. Unsupported themes,
  versions, variants and invalid hero/artifact selections throw explicitly.
  This is not an implementation of every Go-permitted visual variant.
- CSS is scoped to `.presentation-page`; artwork is bounded away from narrative
  headings. These tests make no release copy-over-art legibility claim.

## Source inventory

| Files | Responsibility |
| --- | --- |
| PresentationPage.tsx, index.ts | Pure entry component and exports |
| types.ts, contract.ts | Wire-shaped unions, actions, finite supported variants |
| resources.ts, resolvedResources.ts | Explicit remaining display vocabulary; canonical fixture/asset indexes |
| registry.tsx | Finite block dispatch |
| exhibits.tsx, ArtifactExplorer.tsx, primitives.tsx, links.ts | Native demos, keyboard behavior, safe navigation, configured media |
| presentation.css | Scoped Signal/Studio design and responsive/accessibility corrections |
| fixtures/signal.json, fixtures/studio.json | Selected, translated mockup review configuration; not whole mockup runtime imports |
| preview.html, preview.tsx | Isolated noindex review entry; not imported by production |
| PresentationPage.test.tsx, tsconfig.json, visual-check.mjs | Focused component, strict type and browser validation |
| ui/public/presentation/*.png | Copied survey-relief, pale-moon, tidal-halftone |
| ui/public/presentation/*.ttf, *-OFL.txt | Copied Archivo, IBM Plex Sans/Mono with licenses |

No PublicLanding.tsx, App.tsx, router, shared API, admin, backend, dependency,
deployment, team or sibling-worker edits were made by this slice.

## Focused evidence

Run from the LPBS `ui/` directory:

```sh
pnpm exec vitest run src/surfaces/public-landing/presentation/PresentationPage.test.tsx
pnpm exec tsc --noEmit -p src/surfaces/public-landing/presentation/tsconfig.json
pnpm exec eslint src/surfaces/public-landing/presentation
PRESENTATION_PLAYWRIGHT_MODULE=/home/matthalloran8/Vrooli/node_modules/.pnpm/playwright@1.62.1/node_modules/playwright node src/surfaces/public-landing/presentation/visual-check.mjs
```

Latest focused output:

```text
PresentationPage.test.tsx (17 tests) 324ms
Test Files  1 passed (1)
Tests       17 passed (17)
Duration    860ms
Focused strict TypeScript: exit 0
Scoped ESLint: exit 0, no findings
Standalone Vite build: 37 modules, 592ms
```

Browser evidence:
`/tmp/lpbs-presentation-review-8ZIc2z/report.json` plus viewport, full-page and
section PNGs in that directory. Both designs at 320/390/768/1440: no overflow,
missing images or browser exceptions. Eight axe contexts passed with zero
violations: Signal and Studio at 390/1440, plus all four Signal artifact panels
with the conversation view at 390. Native mobile menu/Escape, conversation toggle,
artifact End/Home passed. Final desktop heroes and mobile HTML artifact screenshots
were inspected, in addition to earlier reference and rendered desktop/mobile views.

Whole UI `pnpm run type-check` fails outside this directory:
`src/shared/api/landing.ts:415` PricingPayload.creditTopups missing; lines 416/417
implicit-any callbacks. Parent has been informed; these files were not edited.

Scoped Test Genie unit run: `20260915-064546-a951a9d2`.
Single waiter returned FAIL after 114 seconds (06:45:46–06:47:40 UTC):
4 errors, 74 warnings. Execution failures include the API Go test command;
the framework-policy check flags direct render imports while its configured
`ui/src/test-utils/renderWithProviders.tsx` does not exist in this scenario.
The isolated component test explicitly documents its provider-free boundary;
this slice does not create shared provider infrastructure. Coverage/architecture
findings are broad advisory evidence, not attribution to this unmounted renderer.
The parent owns cross-scenario/schema integration and follow-up triage.
No production mount, release validation, API integration, or complete scenario
certification is claimed.
