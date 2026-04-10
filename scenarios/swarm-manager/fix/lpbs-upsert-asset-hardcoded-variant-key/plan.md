# Implementation Plan: Remove Hardcoded Variant Key In LPBS UpsertAsset

## 1. Purpose

Parameterize the hardcoded `variant_key = 'default'` in `UpsertAsset` so callers can specify the variant, enabling multi-channel desktop releases.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

No additional domain skills discovered — this is a straightforward Go/SQL parameterization fix.

## 3. Problem Statement

`UpsertAsset` (download_service.go:596) hardcodes `'default'` as the `variant_key` in the INSERT VALUES clause. The `ON CONFLICT` key includes `variant_key`, so non-default variants can never be upserted. This blocks multi-channel (beta, nightly, etc.) desktop release publishing.

Additionally, after the upsert, line 613 calls `s.GetAsset(bundleKey, appKey, platform)` which does NOT filter by `variant_key`, meaning it could return the wrong row once multiple variants exist for the same app/platform.

## 4. Scope

**In scope:**
- Add `VariantKey string` field to `DownloadAsset` struct
- Parameterize `variant_key` in `UpsertAsset` SQL (defaulting to `"default"` when empty)
- Update `UpsertAsset` return call to use `GetAssetByVariant` instead of `GetAsset`
- Update both call sites in `download_hosting_handlers.go` to pass variant key
- Add/update tests

**Out of scope:**
- Changing `GetAsset` behavior globally
- Adding new API endpoints or channels
- Modifying `channelToVariantKey` logic
- Database schema changes (column already exists)

## 5. Current Technical Context

| File | Lines | Role |
|------|-------|------|
| `api/download_service.go:34-51` | `DownloadAsset` struct — no `VariantKey` field |
| `api/download_service.go:551-614` | `UpsertAsset` — hardcodes `'default'` at line 596 |
| `api/download_service.go:506-527` | `GetAsset` — no variant filter |
| `api/download_service.go:529-547` | `GetAssetByVariant` — already exists with variant filter |
| `api/download_hosting_handlers.go:256` | Call site 1 — `handleAdminPublishArtifact` |
| `api/download_hosting_handlers.go:347` | Call site 2 — `handleAdminSetArtifactAsCurrent` |
| `api/update_handlers.go:50-55` | `channelToVariantKey()` — maps channel→variant_key |

## 6. Target End State

- `DownloadAsset.VariantKey` field exists
- `UpsertAsset` uses `asset.VariantKey` (defaulting to `"default"` if empty)
- `UpsertAsset` returns via `GetAssetByVariant` for correctness
- Both call sites pass variant key (currently `"default"` to maintain existing behavior; callers pass channel-derived values when multi-channel is wired up)
- Tests validate both default and non-default variant keys

## 7. Implementation Strategy

### Phase 1: Struct + UpsertAsset (single commit)

1. Add `VariantKey string \`json:"variant_key"\`` to `DownloadAsset` struct after `Metadata`
2. In `UpsertAsset`:
   - Add `variantKey := strings.TrimSpace(asset.VariantKey); if variantKey == "" { variantKey = "default" }`
   - Replace `'default'` literal in SQL with `$12` parameter, shift metadata to `$13`... wait, need to recount params
   - Actually: current params are $1-$11 for the 11 fields, then `'default'` is a literal. Add `variantKey` as `$12`, shift nothing — just replace the literal with `$12` and add the param
   - Change return from `s.GetAsset(...)` to `s.GetAssetByVariant(asset.BundleKey, asset.AppKey, asset.Platform, variantKey)`

### Phase 2: Call sites

3. Both call sites in `download_hosting_handlers.go` — add `VariantKey: "default"` to the struct literal (preserves current behavior, makes it explicit)

### Phase 3: Tests

4. Update existing `UpsertAsset` tests (if any) and add test cases:
   - Upsert with empty VariantKey → defaults to "default"
   - Upsert with explicit VariantKey "beta" → stored correctly
   - Two upserts with different variant keys for same app/platform → both rows exist

## 8. Contract Decisions

- `VariantKey` defaults to `"default"` when empty string — backwards compatible
- JSON field name: `variant_key`
- No API schema change required (field is optional, defaults safely)

## 9. Testing Plan

- Unit test `UpsertAsset` with empty variant key (assert defaults to "default")
- Unit test `UpsertAsset` with explicit variant key "beta"
- Unit test two upserts with same (bundle, app, platform) but different variant keys produce two distinct rows
- Run full test suite: `cd scenarios/landing-page-business-suite/api && go test ./... -timeout 300s`

## 10. Rollout/Validation Checklist

- [ ] `go build ./...` passes
- [ ] `go test ./... -timeout 300s` passes
- [ ] `gofumpt -w .` applied

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Existing callers break | Low | Medium | Default to "default" when VariantKey is empty — fully backwards compatible |
| GetAsset return ambiguity | Already exists | Medium | Fix by switching to GetAssetByVariant in UpsertAsset |

## 12. Non-goals / Prohibited Patterns

- Do NOT modify `channelToVariantKey` logic
- Do NOT add migration (column already exists)
- Do NOT change `GetAsset` signature or behavior
- Do NOT add compatibility shims

## 13. Definition of Done

- `DownloadAsset` struct includes `VariantKey` field
- `UpsertAsset` SQL uses parameterized variant_key
- `UpsertAsset` returns correct row via `GetAssetByVariant`
- Both call sites pass explicit variant key
- All tests pass including new variant-key-specific tests
- Code formatted with `gofumpt`
