# swarm-manager Interoperability Audit

## Last Updated
2026-02-05

## Proto Adoption Status
- [ ] All API request/response types use generated protos
- [ ] All UI↔API communication uses fromJson/toJsonString
- [ ] Protovalidate enforced at API ingress
- [ ] No unsafe type assertions

## Issues Found
1. `scenarios/swarm-manager/ui/src/services/agent-manager-service.ts` still consumes ad-hoc JSON for agent-manager status (no proto contract defined yet).
2. `packages/proto/schemas/swarm-manager/v1/domain/scenario.proto` and `packages/proto/schemas/swarm-manager/v1/domain/backlog.proto` encode lifecycle states as strings; enums would be safer but require a deprecation/migration plan.

## Priority Fixes
1. Add proto contract + protovalidate enforcement for agent-manager status endpoints and switch UI/API to proto parsing.
2. Plan a string-to-enum migration for scenario/backlog statuses with deprecation window and UI updates.

## Notes
- Recommendations, settings, and backlog research endpoints now use proto contracts with protovalidate at ingress.
- UI domain types are now derived from generated proto types to reduce drift risk.
- Backlog/scenario API handlers validate proto payloads at ingress; remaining non-proto endpoints still rely on custom validation.
- Bufbuild JSON parsing accepts proto field names; ensure UI continues to serialize using proto field names for snake_case compatibility.
