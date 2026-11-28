# Handoff — template shelf ownership

> Historical handoff. Superseded on 2026-11-20: curated templates are workspace
> aggregates in `library/workspace-seeds.json`; the legacy API/store and browser
> authoring shelf described below are retired. The migration imports old rows,
> then drops their tables transactionally. Only a read-only browser-cache reader
> remains for unsynced component-context bookmarks during the URL compatibility
> window.

The studio's saved templates moved from `localStorage`-only to the scenario API.
`ui/src/workbench/templates/local.ts` is still the only reader/writer of the
shelf; the browser cache is now a first-paint cache plus an honest offline
fallback, not the system of record.

## What landed

| Area | File |
|---|---|
| Domain + repository | `api/internal/templates/{schema.sql,schema.go,types.go,sqlite.go,migrate.go}` |
| Domain tests | `api/internal/templates/{sqlite_test.go,migrate_test.go}` |
| Browser REST surface | `api/handlers/templates/{module.go,handler_test.go}` |
| Boot wiring | `api/main.go` (templates import, `templatesInternal.Migrate` next to `designsInternal.Migrate`, `templatesH.Module(primaryDB)` in `server.New`) |
| Schema registry | `api/internal/modules/registry.go` (`templatesH.Schema`, `templatesH.Endpoints`) |
| Generated surface | `.vrooli/endpoints.json` (`make endpoints`) |
| Client | `ui/src/workbench/templates/local.ts` (+ `local.test.ts`) |

`ui/` callers were not edited. `DesignStudioPage.tsx`, `useStudio.ts` and
`studio/**` still call the same six exports.

## Endpoints

All responses are JSON with the designs-style envelope. Errors are
`{"error":{"code":"invalid_argument|not_found|conflict|internal","message":"…"}}`.

| Method | Path | Body | Success |
|---|---|---|---|
| GET | `/api/templates` | — | `200 {"templates":[Template]}` newest-`updatedAt` first |
| POST | `/api/templates` | `Template` input (below) | `201 {"template":Template}` |
| PUT | `/api/templates/{id}` | same input; path id authoritative, body id ignored | `200 {"template":Template}` |
| DELETE | `/api/templates/{id}` | — | `204` (idempotent) |

`Template` on the wire is:

```json
{
  "id": "aperture-study-abc",
  "name": "Aperture study",
  "tagline": "Analytics workspace",
  "category": "analytics",
  "palette": { "bg": "#0c0e12", "accent": "#e0a45e" },
  "display": "serif",
  "radius": 16,
  "spec": { "id": "aperture.page", "type": "page", "props": {}, "children": [] },
  "slotHint": "Primary action in the workspace header",
  "idea": "One number leads.",
  "createdAt": "2026-11-19T12:00:00.000Z",
  "updatedAt": "2026-11-19T12:00:00.000Z"
}
```

`id`, `createdAt` and `updatedAt` are server-owned. Timestamps use
`2006-01-02T15:04:05.000Z` UTC, the same fixed-width shape as `designs`.

POST with an existing id returns `409 {"error":{"code":"conflict",…}}`. PUT on a
missing id returns `404`. DELETE is a no-op success for a missing id.

### Validation

Bodies are decoded with `DisallowUnknownFields`; an unknown key returns
`400 invalid_argument` with `unknown field "<name>"`. A body over 2 MiB
(`spec` + `palette` + 1 MiB slack) is also `400` and names `body`.

| Field | Rule |
|---|---|
| `id` | required on POST; `^[a-z0-9][a-z0-9._-]{0,127}$` |
| `name` | required, trimmed, 1–120 chars |
| `tagline` | optional, trimmed, ≤240 chars, defaults `""` |
| `category` | optional, trimmed, ≤80 chars, defaults `""` |
| `display` | `sans` \| `serif`; absent/empty defaults `sans` |
| `radius` | integer 0–48; absent defaults 14 |
| `palette` | JSON object, ≤8 KiB after trimming; absent/`null` → `{}` |
| `spec` | JSON object, ≤1 MiB after trimming; absent/`null` → `{}` |
| `slotHint`, `idea` | optional strings, stored with the spec document |

Every failure names the field in `message` (for example `radius must be between
0 and 48`, `spec must be at most 1048576 bytes`).

PUT is a replace: omitted optional fields fall back to the defaults above, so a
partial PUT clears `tagline`/`category`/`slotHint`/`idea` and resets `palette`
and `spec` to `{}`. The client always sends the whole document.

### Storage note

`studio_templates` has the columns in the contract. It has no columns for
`slotHint`/`idea`, so `spec_json` stores the authored document
`{"spec":<TemplateNode>,"slotHint":"…","idea":"…"}`. The server never
re-models the spec tree; a bare legacy value (a TemplateNode without the
envelope) is still readable and is treated as the spec. `palette_json` is the
palette object verbatim.

## Migration

- `internal/templates.Migrate(ctx, *sql.DB)` re-applies the idempotent
  `CREATE TABLE/INDEX IF NOT EXISTS`. The table is new, so there is no
  `ALTER TABLE` step; a second boot is a no-op (covered by
  `TestMigrateIsIdempotent`). `main.go` calls it immediately after
  `designsInternal.Migrate`.
- The client performs a **one-time** data migration: on the first successful
  `GET /api/templates`, templates that exist only in the cache are POSTed to the
  API. `rcl.workbench.templates.migrated.v1 = "1"` is set only when every
  upload succeeded. A failed upload stays in the cache, is added to
  `rcl.workbench.templates.pending.v1`, and is retried on the next successful
  list/mount.
- Templates present on both sides: the server copy wins (it is the durable
  store). The exception is a template with a pending local write — the cache
  holds the newer unsynced edit and is re-uploaded instead of being clobbered.

## Fallback, queue and cross-tab semantics

- `readLocalTemplates()` stays synchronous and reads only the cache, so first
  paint is unchanged. Server templates are also returned with `local: true`:
  `local` still means "authoring draft", never "published library asset".
- `upsertLocalTemplate(template)` and `removeLocalTemplate(id)` are
  synchronous and return `void`. They update the cache and dispatch
  `rcl:workbench-templates` immediately, then enqueue the API write.
- Writes are serialized per template id (`Map<id, Promise>`), so two saves for
  the same id cannot race. The queue reads the latest cache value when it runs,
  so rapid rename/save bursts coalesce.
- Upsert chooses POST when the id was never confirmed on the server and PUT
  afterwards. A PUT `404` recreates with POST; a POST `409` retries as PUT.
- A failed write keeps the cache entry, records the id in the pending key, and
  dispatches `window` event **`rcl:workbench-templates-error`** with
  `detail.message` bounded to 200 chars. It does not claim success.
- A failed DELETE restores the entry (unless something re-added it) and marks
  it pending, so the shelf never silently forgets a template.
- `templatesUsingFallback()` is true when the last list/write found the API
  unreachable (fetch threw) or `GET /api/templates` answered `404`. A `5xx`/`4xx`
  from an existing route is surfaced as an error event, not silently relabelled
  as fallback.
- `useLocalTemplates()` runs the list/migration once per page load (shared by
  every mount), listens for `storage`, `rcl:workbench-templates` and the
  dependency-free `BroadcastChannel("rcl.templates")` `templates-changed`
  message, and refreshes from the API when another tab reports a change.
- `flushLocalTemplateWrites()` awaits the queue (tests and a page-hide flush).

## Page copy that must change

`ui/src/workbench/pages/DesignStudioPage.tsx` (coordinator-owned):

1. Line ~524, `workbench.studio.savedTemplate`:
   - now: `"Saved {{name}} as a browser-local template."`
   - change to: `"Saved {{name}} to the template shelf."`
2. Line ~545, `workbench.studio.deletedTemplate`:
   - now: `"Deleted the local template."`
   - change to: `"Deleted the template."`
3. Line ~742, the shelf option suffix:
   - now: `{entry.local ? " · local" : ""} — {entry.tagline}`
   - change to: `{entry.local ? " · draft" : ""} — {entry.tagline}`
     (every saved template, server or cache-only, now carries `local: true`;
     "draft" is still exactly true, "local" no longer distinguishes source).
4. Optional honesty line: show `templatesUsingFallback()` as
   "Template shelf is in this browser only — API unavailable" and listen for
   `TEMPLATE_ERROR_EVENT` (`rcl:workbench-templates-error`) to flash
   `event.detail.message`. `TEMPLATE_ERROR_EVENT` is exported by
   `templates/local.ts`.

## Verification

- `cd api && go test ./internal/templates/... ./handlers/templates/...` green;
  `go vet` clean.
- `cd api && go test ./...` retains exactly the three pre-existing failures:
  `internal/reconcile TestCatalogSelectorResolvesThroughRealImportedSource`,
  `internal/gates TestLiveUtilityClassAllowlistIsExactAndShrinkOnly`,
  `internal/gates TestLiveReleaseProvenanceCoversEveryMaterializedRelease`.
- `make endpoints` regenerated `.vrooli/endpoints.json`; `templates_list`,
  `templates_create`, `templates_update`, `templates_delete` are present.
- `cd ui && pnpm exec vitest run src/workbench/templates/local.test.ts` 11/11.
- `cd ui && pnpm exec eslint src/workbench/templates/local.ts src/workbench/templates/local.test.ts`
  clean.
- `pnpm exec tsc --noEmit` is **not** clean, but none of the errors are in the
  template files. Pre-existing errors remain in `designs.ts`, `studio/patch.ts`,
  `StudioInspector.tsx` and `StudioAssist.tsx`; no template-file error is
  emitted.
