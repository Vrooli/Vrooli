# swarm-manager Interoperability Audit

## Last Updated
2026-02-05

## Proto Adoption Status
- [ ] All API request/response types use generated protos
- [ ] All UI↔API communication uses fromJson/toJsonString
- [ ] Protovalidate enforced at API ingress
- [ ] No unsafe type assertions

## Issues Found
1. `packages/proto/schemas/swarm-manager/v1/domain/scenario.proto` and `packages/proto/schemas/swarm-manager/v1/domain/backlog.proto` encode lifecycle states as strings; enums would be safer but require a deprecation/migration plan.

## Priority Fixes
1. Plan a string-to-enum migration for scenario/backlog statuses with deprecation window and UI updates.
2. Audit remaining non-proto endpoints (e.g., file content responses) to confirm they should stay raw/streamed vs. adopt proto wrappers.

## Notes
- Recommendations, settings, backlog, scenarios, and agent-manager status endpoints now use proto contracts with protovalidate at ingress or parsing.
- UI domain types are derived from generated proto types to reduce drift risk.
- Backlog/scenario API handlers validate proto payloads at ingress; remaining non-proto endpoints still rely on custom validation.
- UI proto parsing accepts both proto field names and JSON field names via `fromJson` mapping.
- 2026-02-05: Revalidated settings flow uses proto parsing + normalization before UI consumption; no unsafe response casting found.
