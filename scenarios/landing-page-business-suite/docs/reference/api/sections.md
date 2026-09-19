---
title: "Presentation and Legacy Section APIs"
description: "Typed product publication and private snapshot recovery"
category: "reference"
order: 4
audience: ["developers"]
---

# Presentation and legacy section APIs

Public marketing content is served by the generated
`LandingConfigService.GetLandingConfig` procedure. Its route-aware response
contains one immutable, validated product presentation. Section snapshots are
private recovery data, not an alternate public renderer or publication mechanism.

## Typed editing and publication

All `ProductPresentationAdminService` methods require administrator authentication:

| Method | Purpose |
|---|---|
| `GetPresentation` | Read the current draft or a retained immutable revision |
| `SaveDraft` | Validate and save a document with an expected-generation guard |
| `Preview` | Resolve a private preview without public exposure tracking |
| `Publish` | Verify and activate a saved revision with an expected-generation guard |
| `Rollback` | Activate a retained validated publication with the same guard |

Preview accepts exactly one of `revision` or `document`. The latter validates
the unsaved document and returns a content-derived identity without a storage
write. Both require an existing variant and a valid request-scoped storage lease
in test mode. Responses are private/no-store/noindex; an ephemeral identity is
not a published revision or permission to activate it.

The canonical request and response definitions live in
`packages/proto/schemas/landing-page-business-suite/v1/product_presentation.proto`.
Use generated clients and proto-name JSON. Do not infer wire shapes from the old
section schemas. The configurable block vocabulary and publication rules are in
[Configurable product presentation](../../concepts/PRODUCT-PRESENTATION.md).

Changing a draft does not change the public page. Publish and rollback are
explicit administrator operations. Blocks are ordered by their document arrays;
app membership, visibility and publication determine which page is resolved.
Commerce prices, entitlements and installer availability remain owner facts.

## Private legacy snapshot recovery

`GET /api/v1/variants/{variant_slug}/sections` requires an admin session and
returns `{"sections": [...]}`, including disabled sections. Each legacy entry
retains its ID, type, content, order and enabled flag. Existing authenticated
variant snapshot export/import operations remain available for recovery.

The old `/api/v1/public/variants/{slug}/sections` and
`/api/v1/public/variants/{slug}` routes are retired and return 404.
No anonymous reader may retrieve old section copy merely because its
`enabled` field is true. Mutating a legacy snapshot does not publish a typed
presentation. Standalone `/sections` CRUD is not a current registered API;
use the endpoint inventory rather than historical examples.

Browser Automation Studio recovery bytes, metadata and source digests are
retained separately from improved draft copy. See
[BAS preservation](../../internal/BAS-PRESERVATION.md). Retirement of the public
legacy reader does not remove that recovery material.

## See also

- [API overview](OVERVIEW.md)
- [Variants](variants.md)
- [Admin guide](../../guides/ADMIN_GUIDE.md)
- Generated endpoint inventory: `.vrooli/endpoints.json`
