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
- Making validations commit-aware (`git_commit_hash` as required field on new validations; nullable in DB for legacy rows)
- Transaction-safe bridging via shared `*sql.Tx` context using DBTX interface pattern
- UI approval status badge on VideoReviewPanel
- CLI `--commit` flag on validation commands
- Tests for the bridging logic
- Removing orphaned `ApprovalID` field from `validation.Record` struct
- Defaulting reviewer identity to `'operator'` when not provided

### Out of scope
- Changes to LPBS artifact publishing
- Post-publish version sync (separate backlog item)
- Rollback API (separate backlog item)
- Auto-deploy after all approvals pass (separate backlog item)
- Changes outside `scenarios/deployment-manager/**`
- Proper reviewer identity / auth integration (follow-up item)

## 5. Current Technical Context

### Key files
| File | Role |
|------|------|
| `api/validation/handler.go` | Visual validation CRUD + review submission (lines 105-130: SubmitReview) |
| `api/validation/repository.go` | SQL repository for `visual_validations` — all methods use `r.db` directly, no `*sql.Tx` support |
| `api/validation/types.go` | Request/Record/ReviewRequest structs — Record has orphaned `ApprovalID` field (line 30, to be removed); no `git_commit_hash` field |
| `api/deployments/approvals_handler.go` | Approval CRUD + decide + release-gate |
| `api/deployments/approvals_repository.go` | SQL repository — `Create()` calls `MarkStale()` non-atomically; only `SetRequiredPlatforms()` uses internal tx |
| `api/deployments/approvals_types.go` | DeploymentApproval struct has `ValidationID` field; CreateApprovalRequest accepts optional `ValidationID` |
| `api/deployments/orchestrator.go` | Checks release gate before deploy (HTTP 412) |
| `api/server/server.go` | Handler wiring — ValidationHandler and ApprovalsHandler constructed independently (lines 138-139) |
| `api/server/routes.go` | Route registration |
| `ui/src/components/VideoReviewPanel.tsx` | Video player + review form — no reviewer identity sent, no approval status shown |
| `cli/validations/commands.go` | CLI: run, status, video, review, list — no --commit flag |
| `initialization/postgres/003_add_visual_validations.sql` | Visual validations schema — no git_commit_hash column |
| `api/migrations/003_add_deployment_approvals.sql` | Approvals schema — has validation_id column, UNIQUE on (profile_id, git_commit_hash, platform) |

### Database relationship
- `visual_validations`: has `profile_id`, `platform`, `review_decision` (approved/rejected) — **no `git_commit_hash`**
- `deployment_approvals`: has `profile_id`, `git_commit_hash`, `platform`, `validation_id` (currently unused), `status` (pending/approved/rejected/stale)
- The `validation_id` FK on approvals is the designed join point

### Current operator workflow (disconnected)
1. Create visual validation → record video → review (approve/reject)
2. Separately: create deployment approval → decide (approve/reject)
3. Check release gate → deploy

### Target operator workflow (integrated)
1. Create visual validation **with commit hash** for a specific platform
2. Record video, review evidence
3. Approve/reject from review — **automatically creates/updates deployment approval** (atomic tx)
4. Check release gate (now reflects validation-driven approvals) → deploy

## 6. Target End State

- Approving a visual validation review automatically creates/updates a `deployment_approval` for the same `(profile_id, git_commit_hash, platform)` with `validation_id` set
- Rejecting a validation review creates a rejected approval record (or updates existing to rejected)
- The release gate check reflects validation-driven approvals seamlessly
- UI shows approval status badge on VideoReviewPanel after review submission, using data from the enriched SubmitReview response
- CLI supports `--commit` flag on validation creation and review output shows resulting approval state
- Manual approval (without validation) remains supported for backwards compatibility
- Legacy validations (NULL `git_commit_hash`) continue to work but do not trigger approval bridging
- Reviewer identity defaults to `'operator'` when not explicitly provided

## 7. Implementation Strategy

### Phase 1: Schema + migration + type cleanup
Add `git_commit_hash TEXT` (nullable) to `visual_validations` table via new migration file. Column is nullable so legacy rows get NULL. Application layer enforces non-null on new creates via request validation. Remove orphaned `ApprovalID` field from `validation.Record`.

**Files changed:**
- New: `api/migrations/004_add_validation_commit_hash.sql`
  - `ALTER TABLE visual_validations ADD COLUMN git_commit_hash TEXT;`
- Modified: `api/validation/repository.go` — add `git_commit_hash` to all INSERT/SELECT statements
- Modified: `api/validation/types.go`:
  - Add `GitCommitHash string` to `Record` (JSON: `git_commit_hash`) and `Request` structs
  - Remove orphaned `ApprovalID string` field from `Record`
  - Add `ApprovalID string` and `ApprovalStatus string` fields to `ReviewResponse` (or create it if absent)

### Phase 2: Transaction infrastructure (DBTX pattern)
Add a `DBTX` interface (`QueryContext`, `ExecContext`, `QueryRowContext`) that both `*sql.DB` and `*sql.Tx` satisfy. Refactor both repository types to use `DBTX` internally. Add `WithTx(tx *sql.Tx)` method that returns a new repository instance backed by the tx.

**Side-effect fix:** `ApprovalsRepository.Create()` currently calls `MarkStale()` outside any transaction. After the DBTX refactor, when called via `WithTx()`, both `MarkStale` and `INSERT` run within the caller's transaction, fixing the pre-existing atomicity bug.

**Files changed:**
- Modified: `api/validation/repository.go` — refactor `SQLRepository` to use `DBTX` interface; add `WithTx(tx *sql.Tx) *SQLRepository`
- Modified: `api/deployments/approvals_repository.go` — refactor `SQLApprovalsRepository` to use `DBTX` interface; add `WithTx(tx *sql.Tx) *SQLApprovalsRepository`

### Phase 3: Bridging logic in SubmitReview
Inject `ApprovalsRepository` and `*sql.DB` into `ValidationHandler`. When `SubmitReview` is called:
1. Begin transaction on `*sql.DB`
2. Call `repo.WithTx(tx).UpdateReview(...)` to update validation record
3. If validation has a non-NULL `git_commit_hash`:
   a. Default `reviewed_by` to `"operator"` if empty
   b. Call `approvalsRepo.WithTx(tx).Create(...)` to create/update deployment approval with `validation_id` set, mapping review decision → approval status (`approved` → `approved`, `rejected` → `rejected`)
4. Commit (or rollback on error)
5. Return enriched response with `approval_id` and `approval_status` (empty if no bridging occurred)

**Files changed:**
- Modified: `api/validation/handler.go` — add `approvalsRepo` and `db` fields to `Handler`; update `SubmitReview` with bridging logic and enriched response
- Modified: `api/server/server.go` — pass `approvalsRepo` and `db` to `NewHandler`

### Phase 4: UI enhancement
Extend `VideoReviewPanel.tsx` to:
1. After review submission, read `approval_id` and `approval_status` from the enriched SubmitReview response
2. Display approval status badge (approved/rejected) alongside validation status
3. On mount, if the validation is already reviewed and has `git_commit_hash`, fetch approval status via existing approvals endpoint

**Files changed:**
- Modified: `ui/src/components/VideoReviewPanel.tsx` — add approval status badge rendering using SubmitReview response data

### Phase 5: CLI updates
- Add `--commit` flag to `validations run` command (required for new validations)
- Update `validations review` output to show resulting approval state from enriched response
- Add `--commit` flag to `validations list` for filtering

**Files changed:**
- Modified: `cli/validations/commands.go` — add `--commit` flags, update review output to show approval state

## 8. Contract Decisions

### API changes
| Endpoint | Change |
|----------|--------|
| `POST /api/v1/validations` | Add required `git_commit_hash` field to request body (app-layer validation; column is nullable for legacy) |
| `POST /api/v1/validations/{id}/review` | Response enriched: now includes `approval_id` and `approval_status` fields (empty strings if no bridging) |
| `GET /api/v1/validations/{id}` | Response now includes `git_commit_hash` field (null for legacy rows) |

### Data model changes
| Table | Change |
|-------|--------|
| `visual_validations` | Add `git_commit_hash TEXT` column (nullable; NULL for legacy, enforced non-null by app on new creates) |
| `deployment_approvals.validation_id` | Populated automatically when review triggers approval bridging |

### Type changes
| Struct | Change |
|--------|--------|
| `validation.Record` | Remove orphaned `ApprovalID` field; add `GitCommitHash string` |
| `validation.Request` | Add `GitCommitHash string` (required at app layer) |
| `validation.ReviewResponse` | Add `ApprovalID string` and `ApprovalStatus string` |

### Reviewer identity
- `reviewed_by` defaults to `"operator"` when not provided by caller
- Both validation review and bridged approval record use this value
- Proper identity propagation is deferred to a follow-up item

### Transaction boundary
- `SubmitReview` wraps validation update + approval create/update in a single `*sql.Tx`
- If approval write fails, validation review also rolls back
- `ApprovalsRepository.Create` via `WithTx()` runs `MarkStale` + `INSERT` within the same tx (fixes pre-existing atomicity bug)

## 9. Testing Plan

### Unit tests
- Bridging logic: validation approve with git_commit_hash → approval created with status=approved, validation_id set
- Bridging logic: validation reject with git_commit_hash → approval created with status=rejected, validation_id set
- Bridging logic: validation without git_commit_hash (legacy) → no approval created
- Bridging logic: second validation for same (profile, commit, platform) → old approval marked stale within tx
- Transaction rollback: if approval create fails, validation review is also rolled back
- git_commit_hash required validation on create endpoint (app layer, not DB constraint)
- Reviewer defaults to 'operator' when not provided
- SubmitReview response includes approval_id and approval_status when bridging fires
- SubmitReview response has empty approval fields when no git_commit_hash

### Integration tests (testcontainers)
- Full flow: create validation with commit → submit review → verify approval exists in DB with correct validation_id
- Release gate: create validation, approve → check release gate returns ready for that platform
- Manual approval still works: create approval directly, decide → release gate reflects it
- Stale handling: approve validation, then approve new validation for same commit+platform → first approval stale
- Legacy validation (no commit hash): review does not create approval
- Migration: apply migration on DB with existing visual_validations rows → git_commit_hash is NULL, no errors

### UI tests (if test infra exists)
- VideoReviewPanel renders approval badge after review submission
- VideoReviewPanel shows approval status on mount when already reviewed

## 10. Rollout/Validation Checklist

- [ ] Migration applies cleanly on fresh DB
- [ ] Migration applies cleanly on existing DB with data (git_commit_hash → NULL for existing rows)
- [ ] Run existing test suite — no regressions
- [ ] Run new bridging tests — all pass
- [ ] Manually test: create validation with commit, approve, verify approval created
- [ ] Manually test: reject validation, verify rejection propagated
- [ ] Manually test: check release gate reflects validation-driven approval
- [ ] Manually test: create approval directly (no validation) — still works
- [ ] Manually test: review legacy validation (no commit hash) — no approval created, no error
- [ ] Verify UI badge renders correctly after approve and reject

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Required `git_commit_hash` breaks existing callers | Medium | Medium | Column is nullable in DB; only app layer enforces on new creates. Existing code paths that don't provide it will get a validation error with clear message. CLI error message explains the new required flag. |
| Transaction boundary complexity | Low | High | Use simple `DBTX` interface pattern (well-established Go idiom); test rollback explicitly |
| UI approval fetch adds extra API call on mount for reviewed validations | Low | Low | Only needed when validation is already reviewed on mount; primary flow uses enriched SubmitReview response (zero extra calls) |
| Existing data in `visual_validations` without commit hash | Low | Low | Column is nullable; NULL means 'legacy, no bridging'. Bridging logic has explicit NULL check. No collision risk. |
| `MarkStale` + `INSERT` atomicity (pre-existing bug) | Medium | Medium | Fixed as side-effect of DBTX refactor — both run in caller's tx when using `WithTx()` |
| Reviewer identity is a placeholder | Low | Low | Default `'operator'` is clearly a placeholder; follow-up item tracked for proper identity |

## 12. Non-goals / Prohibited Patterns

- Do not modify LPBS or scenario-to-desktop code
- Do not add auto-deploy after approval (separate item)
- Do not break existing manual approval flow
- Do not add compatibility shims — extend existing types directly
- Do not introduce an event bus or message queue for this bridging
- Do not add bidirectional approval↔validation references (approval→validation via validation_id is sufficient)
- Do not solve reviewer identity/auth in this item

## 13. Definition of Done

- [ ] Validation review (approve/reject) automatically creates/updates deployment approval with `validation_id` set when `git_commit_hash` is present
- [ ] `git_commit_hash` required on all new visual validations (app-layer enforcement)
- [ ] Legacy validations (NULL `git_commit_hash`) continue working without bridging
- [ ] Release gate correctly reflects validation-driven approvals
- [ ] SubmitReview response includes `approval_id` and `approval_status`
- [ ] UI shows approval status badge on VideoReviewPanel
- [ ] CLI supports `--commit` flag on validation creation
- [ ] All bridging logic wrapped in `*sql.Tx` for atomicity
- [ ] All new logic covered by automated tests (unit + integration)
- [ ] Manual approval flow continues working unchanged
- [ ] Orphaned `ApprovalID` field removed from `validation.Record`
- [ ] Reviewer defaults to `'operator'` when not provided
- [ ] Migration handles existing data gracefully (nullable column, no data loss)
