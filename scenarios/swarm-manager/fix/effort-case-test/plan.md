# Effort Case Test — Implementation Plan

## 1. Purpose

Fix an issue related to effort field case normalization in the swarm-manager backlog API. The exact problem needs clarification (description is empty), but the title and codebase context point to the effort t-shirt sizing field (`XS`, `S`, `M`, `L`, `XL`) and its case-handling behavior.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
```

## 3. Problem Statement

The effort field in the swarm-manager backlog API accepts t-shirt sizes (`XS`, `S`, `M`, `L`, `XL`). The proto schema enforces uppercase-only values via `buf.validate`:

```protobuf
optional string effort = 11 [(buf.validate.field).string = { in: ["XS", "S", "M", "L", "XL"] }];
```

The Go API normalizes effort to uppercase **before** proto validation in three places:
1. `normalizeCreateBacklogItemRequest()` — proto-based create path (line 121-128 of handler.go)
2. `Create()` handler — calls `validateEffort()` which also uppercases (line 339-347)
3. `Update()` handler — normalizes before proto validation (line 516-524)

Current tests (`TestCreate_EffortNormalizesCase`, `TestBatchCreate_WithEffort`, etc.) all pass. The exact bug or gap to fix needs to be identified through the workshop.

### Current State Assessment

- **All existing effort tests pass** (create, update, batch create, invalid, optional)
- **Normalization is redundant** in the create path: `normalizeCreateBacklogItemRequest` uppercases first, then `validateEffort` uppercases again
- **CLI and UI** send effort as-is; normalization happens server-side
- **Potential gaps**: No update-path case normalization test, no batch update effort test, possible edge cases with whitespace/mixed case

## 4. Scope

### In Scope

<!-- TBD — depends on what the actual fix target is -->

### Out of Scope

<!-- TBD -->

## 5. Current Technical Context

### Key Files

| File | Role |
|------|------|
| `scenarios/swarm-manager/api/internal/backlog/handler.go` | Create/Update handlers with effort normalization |
| `scenarios/swarm-manager/api/internal/backlog/handler_test.go` | Tests for effort on create/update |
| `scenarios/swarm-manager/api/internal/backlog/batch_handler.go` | Batch create with effort support |
| `scenarios/swarm-manager/api/internal/backlog/batch_handler_test.go` | Batch effort tests |
| `scenarios/swarm-manager/api/internal/backlog/types.go` | `validateEffort()` function |
| `packages/proto/schemas/swarm-manager/v1/api/backlog.proto` | Proto effort field with `in` constraint |
| `packages/proto/schemas/swarm-manager/v1/domain/backlog.proto` | Domain proto effort field |

### Existing Effort Tests

| Test | What It Covers |
|------|---------------|
| `TestCreate_WithEffort` | Create with uppercase "L" |
| `TestCreate_EffortNormalizesCase` | Create with lowercase "xl" → "XL" |
| `TestCreate_InvalidEffort` | Create with invalid "HUGE" rejected |
| `TestCreate_EffortOptional` | Create without effort field |
| `TestBatchCreate_WithEffort` | Batch create with "S" and "xl" |
| `TestBatchCreate_InvalidEffort` | Batch create with "HUGE" rejected |
| `TestBatchCreate_EffortOptional` | Batch create without effort |
| `TestUpdate_WithEffort` | Update with effort |
| `TestUpdate_InvalidEffort` | Update with invalid effort |

## 6. Target End State

<!-- TBD — depends on workshop decisions -->

## 7. Implementation Strategy

<!-- TBD — will be phased once the approach is decided in workshop -->

## 8. Contract Decisions

- Effort field accepts case-insensitive input, always normalizes to uppercase before storage and proto validation
- Valid values: `XS`, `S`, `M`, `L`, `XL` (or empty/omitted)

## 9. Testing Plan

<!-- TBD — depends on scope -->

## 10. Rollout / Validation Checklist

- [ ] All existing effort tests continue to pass
- [ ] New tests cover the identified gap
- [ ] `go test ./internal/backlog/ -run Effort -v` passes

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Empty description leads to solving wrong problem | High | Medium | Clarify in workshop round 1 |
| Fix is trivial and over-planned | Medium | Low | Keep scope minimal |

## 12. Non-goals / Prohibited Patterns

- Don't change the valid effort values (XS/S/M/L/XL)
- Don't change the normalization behavior (always uppercase)
- Don't add backwards-compatibility shims

## 13. Definition of Done

<!-- TBD — will be defined after workshop decisions clarify the target fix -->
