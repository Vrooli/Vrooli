# Plan: Turn Visual Validation Into A Real Approval Flow

## 1. Purpose

Connect the existing visual-validation subsystem to the deployment-approval subsystem so that validation review outcomes (approve/reject) automatically drive approval state, replacing the current disconnected flow where validation and approval are independent operations.

## 2. Required Reading

```bash
prompt-manager skill read implementation-plan-authoring seam-discovery-and-enforcement test
swarm-manager backlog file-get --kind research --name desktop-release-control-plane-audit --path conclusion.md
```

**Key context:** The research audit (desktop-release-control-plane-audit) maps the full 4-system relay and identifies that `deployment_approvals.validation_id` already exists but is never populated automatically.

## 3. Problem Statement

Visual validation and deployment approval are two working subsystems that operate independently:

- **Visual validation** (`visual_validations` table, `validation/handler.go`): Operators can create validations, record video, and submit review decisions (approve/reject) — but the review outcome stays local to the validation record.
- **Deployment approval** (`deployment_approvals` table, `deployments/approvals_handler.go`): Operators can create approvals, decide on them, and the release gate checks all required platforms — but approvals must be created and decided manually.

The `deployment_approvals` table already has a `validation_id TEXT` column, hinting at the intended linkage, but nothing populates it. The result: operators must perform validation review AND separately create/approve a deployment approval, with no enforcement that one feeds the other.

## 4. Scope

### In scope
- Linking validation review outcomes to deployment approval state changes
- API endpoint(s) to bridge validation → approval
- UI approval workflow (review panel + approval status)
- CLI commands for the integrated flow
- Tests for the bridging logic

### Out of scope
- Changes to LPBS artifact publishing
- Post-publish version sync (separate backlog item)
- Rollback API (separate backlog item)
- Auto-deploy after all approvals pass (separate backlog item)
- Changes outside `scenarios/deployment-manager/**`

## 5. Current Technical Context

### Key files
| File | Role |
|------|------|
| `api/validation/handler.go` | Visual validation CRUD + review submission |
| `api/validation/repository.go` | SQL repository for `visual_validations` |
| `api/validation/types.go` | Request/Record/ReviewRequest structs |
| `api/deployments/approvals_handler.go` | Approval CRUD + decide + release-gate |
| `api/deployments/approvals_repository.go` | SQL repository for `deployment_approvals` |
| `api/deployments/approvals_types.go` | Approval structs, gate status types |
| `api/deployments/orchestrator.go` | Checks release gate before deploy (HTTP 412) |
| `api/server/routes.go` | Route registration |
| `api/server/server.go` | Handler wiring and dependency injection |
| `ui/src/components/VideoReviewPanel.tsx` | Video player + review form (approve/reject) |
| `cli/validations/commands.go` | CLI: run, status, video, review, list |
| `initialization/postgres/003_add_visual_validations.sql` | Visual validations schema |
| `api/migrations/003_add_deployment_approvals.sql` | Approvals schema |

### Database relationship
- `visual_validations`: has `profile_id`, `platform`, `review_decision` (approved/rejected)
- `deployment_approvals`: has `profile_id`, `git_commit_hash`, `platform`, `validation_id` (currently unused), `status` (pending/approved/rejected/stale)
- The `validation_id` FK on approvals is the designed join point

### Current operator workflow (disconnected)
1. Create visual validation → record video → review (approve/reject)
2. Separately: create deployment approval → decide (approve/reject)
3. Check release gate → deploy

### Target operator workflow (integrated)
1. Create visual validation for a specific commit + platform
2. Record video, review evidence
3. Approve/reject from review — this automatically creates or updates the corresponding deployment approval
4. Check release gate (now reflects validation-driven approvals) → deploy

## 6. Target End State

- Approving a visual validation review automatically creates/updates a `deployment_approval` for the same `(profile_id, git_commit_hash, platform)` with `validation_id` set
- Rejecting a validation review creates a rejected approval record (or updates existing to rejected)
- The release gate check reflects validation-driven approvals seamlessly
- UI shows the unified flow: validation evidence + approval status in one view
- CLI supports the integrated flow
- Manual approval (without validation) remains supported for backwards compatibility

## 7. Implementation Strategy

<!-- TBD — pending decisions on bridging approach and UI integration -->

### Phase 1: API bridging layer
- Modify `SubmitReview()` in validation handler to also create/update deployment approval
- OR create a new bridging endpoint that accepts validation ID + commit hash and drives both
- Ensure `validation_id` is populated on the approval record

### Phase 2: Schema changes (if needed)
- Add `git_commit_hash` to `visual_validations` table if not present (currently missing — validations aren't commit-aware)
- Migration to add the column

### Phase 3: UI integration
- Extend VideoReviewPanel or create new ApprovalWorkflowPanel
- Show approval state alongside validation evidence
- Require commit hash context when launching validation from approval flow

### Phase 4: CLI updates
- Add `validations run --commit <hash>` flag
- Update `validations review` to show resulting approval state

## 8. Contract Decisions

<!-- TBD — pending workshop decisions -->

### API changes
- Validation create: accept optional `git_commit_hash`
- Validation review: optionally auto-create/update approval
- New endpoint TBD vs. extending existing

### Data model changes
- `visual_validations` table: add `git_commit_hash TEXT` column
- `deployment_approvals.validation_id`: populate automatically on review

## 9. Testing Plan

<!-- TBD — pending approach decisions -->

- Unit tests for bridging logic (validation review → approval creation)
- Integration tests with real DB (testcontainers pattern)
- Test: approve validation → approval created with correct validation_id
- Test: reject validation → approval rejected
- Test: new validation for same commit+platform stales old approval
- Test: release gate reflects validation-driven approvals
- Test: manual approval still works without validation

## 10. Rollout/Validation Checklist

<!-- TBD -->

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| `git_commit_hash` on validations breaks existing callers | Medium | Medium | Make column nullable, optional in API |
| Transaction boundary: review + approval must be atomic | Medium | High | Use DB transaction wrapping both operations |
| UI complexity: showing both validation and approval state | Low | Medium | Progressive disclosure — show approval status as badge on validation |

## 12. Non-goals / Prohibited Patterns

- Do not modify LPBS or scenario-to-desktop code
- Do not add auto-deploy after approval (separate item)
- Do not break existing manual approval flow
- Do not add compatibility shims — extend existing types directly

## 13. Definition of Done

- [ ] Validation review (approve/reject) automatically creates/updates deployment approval with `validation_id` set
- [ ] `git_commit_hash` supported on visual validations
- [ ] Release gate correctly reflects validation-driven approvals
- [ ] UI shows unified validation + approval workflow
- [ ] CLI supports `--commit` flag on validation commands
- [ ] All new logic covered by automated tests
- [ ] Manual approval flow continues working unchanged
