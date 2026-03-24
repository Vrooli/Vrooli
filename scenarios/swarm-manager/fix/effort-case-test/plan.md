# Implementation Plan: Effort Case Test

## Purpose
Ensure the swarm-manager backlog API correctly normalizes the `effort` field (T-shirt sizes: XS, S, M, L, XL) to uppercase on both create and update operations, and rejects invalid values with a clear error message.

## Problem Statement
The backlog system recently added an `effort` field to `BacklogItem` for T-shirt sizing estimates. When users or agents submit effort values in mixed or lowercase (e.g., "xl", "m"), the API must normalize them to uppercase ("XL", "M") before persisting. Additionally, invalid values (e.g., "HUGE", "XXXL") must be rejected with a 400 error. The fix ensures this normalization and validation logic is correct and tested end-to-end.

## Scope

### In Scope
- Case normalization of `effort` field on create (`POST /api/v1/backlog`)
- Case normalization of `effort` field on update (`PUT /api/v1/backlog/{kind}/{name}`)
- Validation that only XS, S, M, L, XL are accepted
- Effort field is optional (empty/omitted is valid)
- Proto-level normalization before validation
- Unit tests covering all effort edge cases

### Out of Scope
- Changing the set of valid effort values (XS/S/M/L/XL is fixed)
- CLI-side effort validation (handled by API)
- UI for effort selection
- Effort-based sorting or filtering in list endpoints

## Current Technical Context
Key files:
- `scenarios/swarm-manager/api/internal/backlog/types.go` — `BacklogItem` struct with `Effort` field, `validateEffort()` function
- `scenarios/swarm-manager/api/internal/backlog/handler.go` — Create/Update handlers with effort normalization logic
- `scenarios/swarm-manager/api/internal/backlog/handler_test.go` — Test cases: `TestCreate_WithEffort`, `TestCreate_EffortNormalizesCase`, `TestCreate_InvalidEffort`, `TestCreate_EffortOptional`, `TestUpdate_WithEffort`, `TestUpdate_InvalidEffort`
- `scenarios/swarm-manager/api/internal/backlog/store.go` — Persistence logic for effort field

## Target End State
- All effort values are normalized to uppercase before storage
- Invalid effort values return 400 with descriptive error
- Empty/omitted effort is preserved as empty string
- Full test coverage for create and update paths
- Proto serialization handles effort correctly

## Implementation Strategy
1. Add `Effort` field to `BacklogItem` struct in types.go
2. Implement `validateEffort()` function with T-shirt size validation
3. Add normalization logic in Create handler (uppercase + trim before validation)
4. Add normalization logic in Update handler (same pattern)
5. Add proto mapping for effort field in store.go
6. Write comprehensive tests covering: valid effort, case normalization, invalid effort, optional effort, update with effort, update with invalid effort

## Contract Decisions
- Effort field is a string, not an enum, to keep JSON simple
- Normalization: `strings.ToUpper(strings.TrimSpace(effort))` before validation
- Empty string after normalization means "no effort set" (treated as omitted)
- Valid values after normalization: XS, S, M, L, XL
- Error message for invalid: "effort must be XS, S, M, L, or XL"

## Testing Plan
| Test Case | Input | Expected |
|-----------|-------|----------|
| `TestCreate_WithEffort` | effort: "L" | Stored as "L" |
| `TestCreate_EffortNormalizesCase` | effort: "xl" | Stored as "XL" |
| `TestCreate_InvalidEffort` | effort: "HUGE" | 400 error |
| `TestCreate_EffortOptional` | (no effort) | Stored as "" |
| `TestUpdate_WithEffort` | effort: "M" | Updated to "M" |
| `TestUpdate_InvalidEffort` | effort: "XXXL" | 400 error |

## Rollout / Validation Checklist
- [ ] All effort-related tests pass: `go test ./internal/backlog/ -run Effort -v`
- [ ] Full test suite passes: `go test ./...`
- [ ] Manual verification: create item with lowercase effort via CLI, confirm uppercase in spec.json

## Risks + Mitigations
| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Proto schema mismatch | Low | Medium | Ensure proto `Effort` field matches Go struct |
| Existing items with non-normalized effort | Low | Low | Normalization happens on read/write, not migration |

## Non-goals / Prohibited Patterns
- Do not add effort to list filtering (separate feature)
- Do not change effort from string to enum type
- Do not add effort validation in CLI (API is authoritative)

## Definition of Done
- [ ] `validateEffort()` correctly accepts XS, S, M, L, XL and rejects all others
- [ ] Create handler normalizes effort to uppercase before validation and storage
- [ ] Update handler normalizes effort to uppercase before validation and storage
- [ ] All 6 effort test cases pass
- [ ] Full `go test ./...` passes in swarm-manager API
