# Presentation editor / renderer handoff

The requested native model override is `gpt-6-astra`, according to the parent.
Runtime model attestation is not independently verified from this thread.
Capability searches do not establish native model availability. The earlier
unavailability inference in the branding report has been corrected.

This is a bounded W3 implementation slice under the operator's editor assignment.
Parent-owned contract and integration work remains concurrent. The existing
renderer inventory is `../../public-landing/presentation/README.md`.

## Canonical display decision and projection handoff

The parent has decided to add required typed `Page.display`, mirroring the finite
`PresentationDisplay` shape below. It is locale/page-owned document configuration.
Resolved pages will contain sanitized, reference-closed display. Generation and
installation are parent-owned; this editor adds no ambient protocol declarations.
The JSON editor uses the generated document descriptor, so newly generated display
fields become editable without a parallel editor-specific schema. The parent's
preview decoder must project that page-owned display to `PresentationPageProps`.

The generated admin service supplies a typed document and sanitized resolved
preview. These are the agreed display fields to verify after regeneration:

- Shell: configured brand/mark/target, subtitle, skip/menu labels, footer identity,
  tagline/copyright/note, unavailable reason, preview label, header action.
- Workspace: mark/avatar/time, tab-group/terminal/conversation labels, file-change labels.
- Asset use sites: alt text; optional responsive sizes. Backdrop: mark.
- Blocks: editorial heading breaks, extra eyebrow/body/note/accessibility text,
  artifact badge/formats, capability anchors.
- Hero/device/catalog: explicit fixture references; catalog mark/tone/detail label.

The complete field types remain in the renderer's `resources.ts`. Do not derive
these from app names, IDs, public mode, array positions, or demo fixtures.

`PresentationEditorRoute` accepts a `mapPreview` function supplied by the parent.
It receives only the authorized, sanitized generated `ResolvedProductPresentation`
message and returns `PresentationPageProps`. The parent decoder owns generated
oneof unwrapping and the rich display mapping. The same `PresentationPage` renders
the result. Without a mapper, the editor reports the explicit integration gap.
It never substitutes demonstration content. Transaction handlers are not passed
to the embedded revision preview.

## Acceptance boundary

The editor keeps the complete generated document editable as JSON, with focused
page metadata controls and explicit page/block reordering. Array order is retained.
SaveDraft does not publish. Publish and Rollback each require a separate explicit
confirmation and the exact loaded generation. Conflicts retain local text and
lock mutations until an explicit reload. Dirty reload requires discard confirmation.
Revision previews go only through authenticated Preview RPC, remain in memory,
and require preview/noindex/no-store diagnostics for the requested revision.

No router, navigation, production landing, existing shared API, backend, payment,
deployment, team, dependency or descendant changes are part of this slice.
Validation is focused component/client tests, scoped lint and strict TypeScript;
the operator explicitly excludes broad suites for this task.
