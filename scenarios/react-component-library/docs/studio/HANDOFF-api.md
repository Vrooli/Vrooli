# Handoff — designs persistence + brand (API stream)

## What landed

- **Designs domain** (`api/internal/designs`, `api/handlers/designs`): SQLite-backed
  list/get/create/save/delete, optimistic concurrency, optional design ↔ page binding,
  and design-scoped asset storage. Frames and theme remain opaque JSON — the server
  never re-models the TemplateNode tree.
- **Partial-PUT fixed.** `name`, `frames` and `theme` are required on both POST and PUT.
  Absence is detected separately from explicit `null`: `frames:null` still means `[]`
  and `theme:null` still means `{}`, but omitting either returns
  `400 {"error":{"code":"invalid_argument","message":"frames is required"}}`.
- **Optimistic concurrency.** PUT accepts an optional `revision`. When present, the
  UPDATE is conditional in SQL (`WHERE id=? AND revision=?`); a mismatch writes nothing
  and returns `409` with the current document. When absent, saves stay last-write-wins.
- **Binding.** Nullable `scenario`/`page` columns (empty text = unbound). Both are
  optional on create/save, returned as `string|null`, and filter exact matches through
  `GET /api/designs?scenario=…&page=…`.
- **Assets.** New `studio_design_assets` table stores bytes in SQLite (≤8 MiB each).
  Metadata lists, upload, raw-byte serving with `ETag`/`304`, delete, and an
  all-designs metadata list.
- **Migration.** `internal/designs.Migrate(ctx, *sql.DB)` re-applies the idempotent
  schema (adding the new asset table on old databases), inspects
  `pragma_table_info('studio_designs')` and `ALTER TABLE … ADD COLUMN` for `scenario`
  and `page`. It never drops or rewrites rows and is safe to run on every boot.
  `api/main.go` calls it once immediately after `EnsureSchemas`.
- **Brand domain** (`api/internal/brand`, `api/handlers/brand`): `GET /api/brand`
  resolves the RCL assignment from **Brand Manager's supported surface** (Connect
  `AssignmentsService.GetScenarioStatus` + `BrandsService.GetBrand`, plus its own
  mark-bytes route). No Brand Manager record is created; no colour is hardcoded.
- **Client** `ui/src/workbench/designs.ts` keeps every existing export and adds the
  conflict error, binding fields, asset helpers and a fallback flag.
- Endpoint descriptors regenerated with `make endpoints`; all ten designs routes and
  brand are in `.vrooli/endpoints.json`.

## Endpoint contract

All JSON responses are `application/json`. Every error is:

```json
{"error":{"code":"invalid_argument|not_found|conflict|internal","message":"..."}}
```

The one exception is the asset byte route, which serves the raw image; its errors are
still the JSON envelope.

| Method | Path | Body | Success |
|---|---|---|---|
| GET | `/api/designs` | —; optional `?scenario=&page=` exact filters | `200 {"designs":[DesignSummary]}` newest `updatedAt` first |
| POST | `/api/designs` | `{name,frames,theme,scenario?,page?}` | `201 {"design":DesignDocument}` |
| GET | `/api/designs/{id}` | — | `200 {"design":DesignDocument}` |
| PUT | `/api/designs/{id}` | `{name,frames,theme,scenario?,page?,revision?}` | `200 {"design":DesignDocument}` |
| DELETE | `/api/designs/{id}` | — | `204` (idempotent; deletes the design's assets in the same transaction) |
| GET | `/api/designs/{id}/assets` | — | `200 {"assets":[AssetMeta]}` oldest first |
| POST | `/api/designs/{id}/assets` | `{name,mime,dataBase64,fidelity?}` | `201 {"asset":AssetMeta}` |
| GET | `/api/designs/{id}/assets/{assetId}` | — | `200` raw bytes, stored `Content-Type`, `Cache-Control: private, max-age=3600`, `ETag` = asset id; `304` on `If-None-Match` |
| DELETE | `/api/designs/{id}/assets/{assetId}` | — | `204` (idempotent; scoped to that design) |
| GET | `/api/design-assets` | — | `200 {"assets":[AssetMeta]}` across designs, newest first |
| GET | `/api/brand` | — | `200 {"assigned":bool,"name":string,"mark":string\|null,"accent":string\|null,"wordmark":string\|null}` |

`DesignDocument = {id,name,updatedAt,revision,frames,theme,scenario,page}`;
`DesignSummary = {id,name,updatedAt,frameCount,scenario,page}`;
`AssetMeta = {id,designId,name,mime,bytes,width,height,fidelity,addedAt,url}` where
`url` is `/api/designs/{designId}/assets/{assetId}`.

Server-owned fields: `id`, `revision` (1 on create, +1 per successful save),
`updatedAt` (fixed-width UTC `2006-01-02T15:04:05.000Z`), `addedAt`. Client-sent
`id`/`updatedAt` are ignored.

### Required fields and conflict envelope

- Missing `name`, `frames` or `theme` → `400` naming the field, on POST and PUT.
- Explicit `frames:null` → `[]`; explicit `theme:null` → `{}`; `scenario`/`page`
  absent, `null` or empty → unbound.
- PUT with `revision` not equal to the stored revision:

```json
{
  "error": {"code": "conflict", "message": "design was modified elsewhere"},
  "design": { "id": "...", "name": "...", "revision": 7, "frames": [], "theme": {}, "scenario": null, "page": null, "updatedAt": "..." }
}
```

  The conflicting write changes nothing. The losing writer can adopt `design` or
  retry against it. Omitting `revision` keeps the documented last-write-wins path.

### Asset limits

- Allowed mime (case/parameters normalised): `image/png`, `image/jpeg`, `image/webp`,
  `image/gif`, `image/svg+xml`.
- Decoded size ≤ **8 MiB** (`designs.MaxAssetBytes`); empty name, empty data, invalid
  base64, non-image mime and bad `fidelity` → `400 invalid_argument`.
- `fidelity` defaults to `reference`; the other accepted value is `captured`.
- Width/height come from the PNG/GIF/JPEG/WebP header with no codec; SVG and unknown
  payloads report `0,0`.
- `dataBase64` accepts either bare base64 or a `data:…;base64,` data URL.
- Assets are only readable/deletable through their owning design. Deleting through
  another design is a 204 no-op and leaves the asset intact.

## Schema

`studio_designs(id PK, name, revision, frame_count, frames_json, theme_json, scenario,
page, created_at, updated_at)` and
`studio_design_assets(id PK, design_id, name, mime, bytes, width, height, fidelity, data
BLOB, created_at)` — `internal/designs/schema.sql`, applied by `EnsureSchemas`;
`Migrate` handles the additive columns and is idempotent.

## Client surface (`ui/src/workbench/designs.ts`)

Existing exports and signatures still work:

```ts
listDesigns(filter?: { scenario?: string; page?: string }): Promise<DesignSummary[]>
getDesign(id: string): Promise<DesignDocument>
saveDesign(doc: DesignDocument): Promise<DesignDocument>
createDesign(input: CreateDesignInput): Promise<DesignDocument>
deleteDesign(id: string): Promise<void>
```

Additions:

```ts
type DesignDocument = { …; scenario?: string | null; page?: string | null }
type DesignSummary  = { …; scenario?: string | null; page?: string | null }

class DesignConflictError extends Error {
  readonly current: DesignDocument;   // the document another tab wrote
}

type DesignAsset = {
  id: string; designId: string; name: string; mime: string;
  bytes: number; width: number; height: number;
  fidelity: "reference" | "captured"; addedAt: string; url: string;
};

listDesignAssets(designId: string): Promise<DesignAsset[]>
listAllDesignAssets(): Promise<DesignAsset[]>
uploadDesignAsset(designId, file: File, fidelity?: "reference" | "captured"): Promise<DesignAsset>
deleteDesignAsset(designId, assetId): Promise<void>
designAssetUrl(asset: DesignAsset): string   // absolute against API_BASE
designsUsingFallback(): boolean              // true when localStorage answered
```

`saveDesign` always sends the document's `revision`, so the studio gets 409 protection
by default; it throws `DesignConflictError` carrying `current`. Fallback honesty: a
local draft is used **only** when `fetch` throws or the collection route returns 404.
A 4xx/5xx on a route that exists now throws, and `designsUsingFallback()` lets the
studio surface "local draft — API unavailable" instead of pretending the draft is
durable.

## Run / verify

```bash
cd scenarios/react-component-library && make endpoints
cd api && go test ./...
cd ui && pnpm exec vitest run src/workbench/designs.test.ts
cd ui && pnpm exec tsc --noEmit
```

The running scenario must be restarted (`vrooli scenario restart
react-component-library`) to load the new routes and run `Migrate`. Until then the UI
falls back to local drafts and the brand falls back to the neutral monogram — no crash.

## Test results

- `internal/designs`: create/get/list/save/delete, required-field document validation,
  binding round-trip and filter, stale-revision rejection, two concurrent saves from one
  base revision (exactly one success + one `ErrConflict`), asset metadata/bytes,
  header-derived dimensions, mime/size/fidelity limits, cross-design delete isolation,
  cascade delete, and migration from the old schema.
- `handlers/designs`: 10 endpoint descriptors, required-field 400s on POST and PUT,
  binding filter, 409 envelope including the current design, asset lifecycle
  (201/list/raw bytes/ETag/304/204/404), cross-design isolation, upload limits, and the
  error envelope on every failure.
- `go test ./...`: only the three known pre-existing failures remain —
  `internal/reconcile TestCatalogSelectorResolvesThroughRealImportedSource`,
  `internal/gates TestLiveUtilityClassAllowlistIsExactAndShrinkOnly`,
  `internal/gates TestLiveReleaseProvenanceCoversEveryMaterializedRelease`.
- `go vet ./internal/designs ./handlers/designs .`: clean.
- `make endpoints`: regenerated; `.vrooli/endpoints.json` lists all ten designs routes
  (and the previously missing brand route).
- `ui`: `vitest run src/workbench/designs.test.ts` green (9 tests); those tests cover
  required-field PUT payload, 400 surfacing without fallback, 409 →
  `DesignConflictError.current`, network/404 fallback vs 5xx/member-404 errors, and
  asset URL building. `tsc --noEmit` reports only three pre-existing errors in the
  uncommitted `ui/src/components/SidebarShell.test.tsx` (another stream's file);
  `designs.ts` and `designs.test.ts` contribute none.

## For studio / in-context streams

- Import from `ui/src/workbench/designs.ts`; do not call `/api/designs` directly.
- Autosave should send the whole document. On success adopt the server's
  `revision`/`updatedAt`/`scenario`/`page`. On `DesignConflictError`, adopt
  `error.current` or show a conflict prompt; do not retry the stale revision blindly.
- `frames[].spec` is passed through verbatim, including unknown keys.
- Wire the page selector to `listDesigns({ scenario, page })` and set
  `scenario`/`page` on create/save (DEBT #3 server half is done).
- The Assets tab can move from in-memory to `listDesignAssets` / `uploadDesignAsset` /
  `deleteDesignAsset`; `designAssetUrl` returns a ready `<img src>`.
- Show `designsUsingFallback()` in the status bar so a local draft is never mistaken for
  a durable one.
- Brand: `useBrand()` already matches this contract; `mark` is a data URI.

## Follow-up: DEBT #11 — sketch error classification

`api/handlers/sketch/connect_handler.go` now defaults unmapped errors to
`connect.CodeInternal` instead of `CodeInvalidArgument`; existing explicit mappings
(AlreadyExists, NotFound, FailedPrecondition, Aborted) are unchanged, and an
already-coded `*connect.Error` is preserved so call-site validation still reaches the
client as `InvalidArgument`. Added
`api/handlers/sketch/connect_error_test.go`, a table test covering the whole
taxonomy (unexpected disk/permission/parse → Internal; explicit validation →
InvalidArgument; conflicts/locks/missing rows as before).
`go test ./handlers/sketch/...` passes; `go test ./...` shows only the pre-existing
`internal/reconcile` and `internal/gates` failures noted above.

## Wave-3 additions

### `POST /api/designs/{id}/assist`

The AI patch route. It is documented fully in [`HANDOFF-ai.md`](./HANDOFF-ai.md): request/response,
the `409` stale-revision envelope, `422 invalid_proposal`, `502 ai_failed`, `503 ai_unavailable`, and
the patch vocabulary. Two server-side properties matter to any caller:

- the design is **never written** by this route;
- the operation is durable and idempotent through `internal/designinference`, shared with the
  sketch inference surface (`design_inference_operations`).

### `putDesign` / `saveDesign`

`ui/src/workbench/designs.ts` now exposes:

```ts
interface DesignWriteResult { design: DesignDocument; local: boolean }
putDesign(doc): Promise<DesignWriteResult>
saveDesign(doc): Promise<DesignDocument>   // compatibility wrapper
```

`local: true` means the API was unreachable and localStorage took the write. A caller that ignores
it and reports "saved" is lying. The studio uses it to:

- render **Local draft · revision N** instead of **Saved**;
- skip the cross-tab broadcast (a browser-only revision is not a server revision);
- keep the design dirty for reconciliation, so a remote save cannot silently replace the only copy;
- retry the write on `online`.

Local writes no longer bump `revision`: the server still owns that number, so the retry can PUT
against the revision the document actually has.

### `GET /api/templates` and friends

Template shelf CRUD is a separate module (`handlers/templates`, `internal/templates`); see
[`HANDOFF-templates.md`](./HANDOFF-templates.md). The collection is `GET/POST /api/templates`, a
member is `PUT/DELETE /api/templates/{id}`. `POST` to a member is a `405` — a live-verification bug
that is now covered by a route-asserting test.
