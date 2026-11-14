# Authenticated presentation editor checkpoint

## Explicit local legacy import — 2026-09-15

`LegacyImportPanel` imports a pasted or local UTF-8 legacy snapshot into the
current dirty document. Select an existing disabled/private/draft app and its
exact page locale explicitly. Conversion appends conservative recovery blocks;
it never saves, publishes, changes eligibility or activates imported links.
The normal read-only preview and generation-guarded Save remain separate.

The version1 adapter in `shared/lib/presentationLegacyImport.ts` retains exact
source bytes and field dispositions in administrative base64 chunks. Receipts
use RFC6901 pointers into source and canonical generated document JSON. Reimport
verifies the byte/chunk/receipt ledger before reporting idempotence. Invalid
input, corrupt storage and async results from an obsolete document, selection,
route or authentication context do not overwrite current edits. File decoding
is fatal UTF-8 and does not silently replace invalid bytes.

Parent validation:55 focused panel/editor tests and15 adapter unit cases pass,
plus whole-UI TypeScript and scoped ESLint. An opt-in bridge exported the actual
canonical generated seed, ran the real adapter, then passed authoritative Go
SaveDraft/GetPresentation validation under races for synthetic edge cases and
the preserved agency BAS variant. The latter retained8631bytes/157fields in an
isolated unpublished draft. This is not a live authenticated browser write or
a public migration. Exact artifacts and limitations are in the effort's
`handoffs/IMPLEMENTATION-CHECKPOINT.md`.

## Mounted unsaved preview terminal — 2026-09-15

The editor now defaults to a 300ms read-only local document preview through the
generated Preview.document field 5. It hides obsolete results immediately and
aborts superseded requests; invalid syntax, server validation, stale identities
and auth loss fail closed without saving source. RevisionPreview uses the same
native renderer in a responsive editor column. Explicit retained-revision reads,
draft Save, confirmed Publish/Rollback and CAS protection remain separate.
No backend writes were performed. Whole-UI typecheck and the focused 389-test run
pass. Exact commands, complete phase inventory, browser evidence and limitations:
[phase-four terminal](../../public-landing/presentation/README.md#phase-four-terminal--external-playback-and-mounted-local-preview-2026-09-15).

## Current mounting checkpoint — 2026-09-15

`PresentationAdminPage` now mounts at protected `/admin/presentation` and
`/admin/presentation/:variantSlug` using existing AdminLayout and mandatory auth.
Variant selection is explicit. The wrapper wires `onDirtyChange` to guarded
in-app navigation, browser back/forward and logout; the editor retains beforeunload.
Private preview uses the same renderer with basename-aware local navigation and
disabled transactions, without public config/assignment or public metrics.
No real server Save/Publish/Rollback was performed.

Additional files: PresentationAdminPage.tsx and its five-test component suite;
PresentationEditorRoute.tsx, RevisionPreview.tsx and its four-test suite add the
optional linkBase integration. Parent-authorized mounting also changes
app/routes/adminRoutes.tsx, admin-portal/components/AdminLayout.tsx and
admin-portal/config/navigation.ts. App and auth-provider composition are unchanged.
Complete current source inventory, command/output and bootstrap exposure caveats:
[public integration checkpoint](../../public-landing/presentation/README.md).
Final run: 12 files / 124 tests passed in 4.78s; whole-UI typecheck and scoped lint
exit 0. Live server/publication/release and assignment accounting remain deferred.

## Prior editor/decoder checkpoints (historical scope/evidence)

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
  onDirtyChange={parentDirtyNavigationGuard}
/>
```

Exports are in `index.ts`. The route is **not mounted** by this slice.
The parent owns App/router/navigation, auth-provider placement, dirty in-app
navigation integration, and real server/release validation.
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
  exact resolved revision before decoding the generated response.
- The same native PresentationPage renders the response using mandatory page.display.
  The built-in generated decoder is shared with public integration. Transactions
  are not supplied to preview; missing display/schema failures show a bounded
  message, never demonstration content.
- Existing slate admin utilities and shared Button/Input/Textarea/Dialog components
  provide layout and confirmation behavior. No new dependencies or styles outside
  the owned directory.

## Canonical display status

See [DISPLAY-FIELD-HANDOFF.md](./DISPLAY-FIELD-HANDOFF.md). Required typed
Page.display is generated and installed. The editor uses the generated document
descriptor and preserves display on Save. decodeProductPresentation performs
exhaustive oneof unwrapping and reference checks. PresentationPage now takes
presentation and optional resolvedActions only; no display prop or mapPreview
callback remains. No ambient shims or demo fallbacks were added.

No real server-authenticated preview, Save, Publish or Rollback was performed.
Those live checks require the parent's mounted integration and owner-qualified
configuration. In-app dirty navigation must be wired through `onDirtyChange`;
this route adds only the browser beforeunload guard itself.

## Complete source inventory for this slice

| File | Responsibility |
| --- | --- |
| ../../../shared/api/productPresentation.ts | Typed generated client, session/no-store transport, strict JSON codec, decoder export |
| PresentationEditorRoute.tsx | Auth-gated route export, draft/publication/revision controls, confirmations |
| usePresentationEditor.ts | Dirty state, generation guards, request lifecycle, conflict/privacy handling |
| DocumentEditor.tsx | Whole-document editing and explicit ordering |
| RevisionPreview.tsx | Same generated decoder/native renderer, sanitized private failure UI |
| index.ts | Parent-facing exports |
| testFixtures.ts | Generated, synthetic test data and typed client fakes; no production import |
| PresentationEditorRoute.test.tsx | 18 editor interaction and safety regressions |
| RevisionPreview.test.tsx | 3 native preview boundary regressions |
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

Prior admin-only checkpoint output (2026-09-15, 03:07:40 local):

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


## Latest decoder integration evidence

Current focused run at 03:24:39 local: five files, 53 tests passed in 2.21s.
Includes 18 editor, 3 private preview, 4 client, 17 renderer and 11 decoder/closure
regressions. Both focused TypeScript projects and scoped ESLint exit 0.
The dirty-document test also verifies canonical Page.display survives SaveDraft.
Complete commands/output and isolated eight-viewport/eight-axe-context evidence
are in ../../public-landing/presentation/README.md. No live server writes were made.
