# Authenticated presentation editor checkpoint

Bounded W3 implementation under the operator's 2026-09-15 assignment. The
scenario-work-ladder scoped route and writing-standards placement guidance keep
this handoff explicit about implemented behavior versus live integration evidence.
No broader W0–W2 or scenario certification is claimed.

## Entry component

```tsx
import { PresentationEditorRoute } from './presentation';

// Inside the existing AdminAuthProvider and admin layout:
<PresentationEditorRoute
  variantSlug={explicitVariantSlug}
  mapPreview={parentGeneratedPreviewMapper}
  onDirtyChange={parentDirtyNavigationGuard}
/>
```

Exports are in `index.ts`. The route is **not mounted** by this slice.
The parent owns App/router/navigation, auth-provider placement, dirty in-app
navigation integration, and the canonical display/preview mapper.
The optional `client` prop is a typed test seam, not an alternate HTTP protocol.

## Implemented

- Complete generated document JSON editing: bundle/apps/pages/blocks/fixtures/
  assets/capabilities/localized strings, including private administrative fields.
  Generated protobuf descriptors parse and serialize the entire document.
  Unknown fields and oversized JSON are rejected rather than discarded.
- Focused page-title/description controls and accessible move-up/down controls
  for pages and blocks. Arrays preserve their explicit order; no silent sorting.
  Add/remove/configure any document field through the complete JSON editor.
- Draft Save only invokes SaveDraft. Publish and Rollback each require explicit
  confirmation showing the exact immutable revision and loaded generation.
- Every mutation sends the exact uint64 generation as a bigint. No numeric
  rounding, generation inference, automatic publication, or automatic retry.
- CAS conflicts retain the edited text and lock further writes until reload.
  Dirty reload requires discard confirmation; cancellation retains local edits.
  Unknown network outcomes also lock writes pending explicit reload.
- A synchronous request lock prevents duplicate clicks; requests have a 30-second
  deadline and are aborted on unmount/auth-context changes. Late responses cannot
  repopulate logged-out state.
- No fetch before authentication. Authorization failure hides and clears private
  source/preview. Private data stays in component memory, not URL parameters,
  browser storage, analytics, a public config request, or a share link.
- Preview uses authenticated Preview RPC for the loaded saved revision and the
  explicit route/locale. It verifies preview/noindex/no-store diagnostics and the
  exact resolved revision before calling the mapper.
- The same native PresentationPage renders mapped responses; transaction handlers
  are not supplied to the embedded preview. Mapping failures show a bounded
  message. Missing mapping shows an explicit pending-integration state, not demos.
- Existing slate admin utilities and shared Button/Input/Textarea/Dialog components
  provide layout and confirmation behavior. No new dependencies or styles outside
  the owned directory.

## Canonical display status

See [DISPLAY-FIELD-HANDOFF.md](./DISPLAY-FIELD-HANDOFF.md). Parent decided that
required typed `Page.display` owns the finite display configuration and that
resolved pages carry sanitized/reference-closed display. At the last installed
descriptor inspection during this checkpoint, that new field was not yet present.
The editor will recognize it automatically through the regenerated document
descriptor; the parent mapper still owns protobuf oneof unwrapping and the
`PresentationPage({presentation, display})` boundary. No ambient shims or demo
fallbacks were added.

No real server-authenticated preview, Save, Publish or Rollback was performed.
Those live checks require the parent's mounted integration and owner-qualified
configuration. In-app dirty navigation must be wired through `onDirtyChange`;
this route adds only the browser beforeunload guard itself.

## Complete source inventory for this slice

| File | Responsibility |
| --- | --- |
| ../../../shared/api/productPresentation.ts | New typed generated service client, session/no-store transport, strict JSON codec |
| PresentationEditorRoute.tsx | Auth-gated route export, draft/publication/revision controls, confirmations |
| usePresentationEditor.ts | Dirty state, generation guards, request lifecycle, conflict/privacy handling |
| DocumentEditor.tsx | Whole-document editing and explicit ordering |
| RevisionPreview.tsx | Same native renderer, parent mapper boundary and sanitized failure UI |
| index.ts | Parent-facing exports |
| testFixtures.ts | Generated, synthetic test data and typed client fakes; no production import |
| PresentationEditorRoute.test.tsx | 18 editor interaction and safety regressions |
| RevisionPreview.test.tsx | 2 native preview boundary regressions |
| productPresentation.test.ts | 4 generated codec/transport regressions |
| tsconfig.json | Focused strict compilation with inherited scenario rules |
| DISPLAY-FIELD-HANDOFF.md, README.md | Canonical field decision, integration and evidence handoff |

The separately authorized report correction is in
`/home/matthalloran8/.vrooli/plan-artifacts/efforts/aquila-configurable-product-presentation/findings/web-console-ui-branding.md`.
It now records the parent-requested native `gpt-6-astra` override and unverified
runtime attestation. Capability discovery cannot establish native model availability.

## Focused validation

From the scenario `ui/` directory:

```sh
pnpm exec vitest run src/surfaces/admin-portal/presentation
pnpm exec tsc --noEmit -p src/surfaces/admin-portal/presentation/tsconfig.json
pnpm exec eslint src/surfaces/admin-portal/presentation src/shared/api/productPresentation.ts
```

Final focused output (2026-09-15, 03:07:40 local):

```text
RevisionPreview.test.tsx          2 tests   56ms
productPresentation.test.ts       4 tests   16ms
PresentationEditorRoute.test.tsx 18 tests  599ms
Test Files  3 passed (3)
Tests      24 passed (24)
Duration   1.46s
Focused strict TypeScript: exit 0
Scoped ESLint: exit 0, no findings
```

The client test sends/decodes an actual generated Connect request through an
injected fetch implementation, verifying administrator credentials, no-store and
generation 9007199254740993 without rounding. Component tests use the repository
renderWithProviders helper and explicit auth/client seams.

No broad suites, new dependencies, descendants, PM-team activation, deployment,
payment, backend, PublicLanding, App/router/nav, existing landing.ts or ambient
declaration edits were made. Parent-owned worktree changes were preserved.

