# Canonical presentation decoder / editor handoff

## Mounting update — 2026-09-15

Public root/detail and authenticated admin routes are now mounted in source.
Public owner-action observations are joined in publicIntegration.ts; private
preview continues to ignore transactional joins. Both surfaces retain the same
mandatory Page.display decoder/renderer boundary. Proxy-aware preview receives
only linkBase, not an alternate display or protocol shim.

Current focused evidence: 124 passing tests, UI typecheck and scoped lint clean.
See the public README for complete mounting inventory and output. Initial matching
bootstrap stays pinned without another config request; owner exposure accounting
and cross-route revision pinning require the parent's typed seam and are not
claimed complete. Download platform workflow and live release/authentication
validation also remain explicit. Earlier mounting ownership notes below describe
the prior decoder checkpoint, not the present source state.

The parent explicitly requested native gpt-6-astra. Runtime attestation is not
independently verified here; capability discovery cannot establish model availability.
The earlier branding report's availability inference was corrected separately.

## Integrated contract

Page.display is now generated, installed, mandatory and locale/page-owned.
The renderer projects the installed generated response through
decodeProductPresentation(value: ResolvedProductPresentation): Presentation.
Export: ui/src/shared/api/productPresentation.ts (also the public renderer index).
The native entry is PresentationPage({ presentation, resolvedActions? }).
There is no display prop, runtime display sidecar, or route-name inference.

PresentationEditorRoute no longer accepts mapPreview. Its authorized revision
response goes through the same decoder and native page. Preview still requires
preview/noindex/no-store and the requested saved revision, stays in memory, and
never passes live transaction joins. Missing display or schema/reference failures
produce sanitized failure UI, never demonstration content.

The complete JSON editor uses generated descriptors, including inline display;
draft-save tests verify display survives with private fields and exact uint64
generation. No ambient shim or alternate protocol is present.

Installed/repository shared descriptor hash:
e84c668f29c8407c922b472ab588e6a3f4d2a2f46418d21117189d0419302627.
Parent source digest: 54092391239066175c121427d8664ffcc98c5c3981abf68100ff9819b19e86d3.
SDA refresh is complete; no further refresh is needed for this checkpoint.

## Projection requirements for parent review

Hero groups use eligible_app_keys plus resolved spotlight profiles, independently
of selected_app_keys. Display apps must include any referenced hero app even at
k=0. Device/hero fixtures are exactly-one alternatives to released visual refs.
All returned display reference keys, assets, fixture images, capability refs and
responsive alternatives must be closed. Workspace interactive labels and asset
label entries are required; decorative alt may deliberately be empty.
No hidden app details may enter any of these projected resources.

The decoder deliberately does not interpret wire actions as authorization.
Public owner-action integration remains parent-owned; private preview ignores
wire ready-action observations. Cross-aspect-ratio responsive art needs explicit
art-direction policy, not an inferred breakpoint.

## Acceptance and remaining integration

Full JSON editing, explicit order, guarded Save/Publish/Rollback/CAS, dirty-state,
privacy and native revision preview are focused-tested. See both README files
for complete source inventories and final test output (53 passing tests).
Public/App/router mounting, real authenticated server integration and release
qualification remain parent-owned. Existing PM team stays disabled.
