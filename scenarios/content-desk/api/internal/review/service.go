package review

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	OutcomePassed  = "passed"
	OutcomeBlocked = "blocked"
)

type (
	Verdict struct {
		Mode     string
		Passed   bool
		Evidence string
		Finding  string
	}
	Run struct {
		ID, DraftID, Outcome string
		Verdicts             []Verdict
	}
	EditorialPackage struct {
		ID, DraftID, SourceRevisionID, Scope, Body, BodySHA256, PackageSHA256, ManifestJSON string
		CreatedAt                                                                           time.Time
	}
	RevisionSnapshot struct {
		DraftID    string
		Body       string
		RevisionID string
	}
	RevisionSource interface {
		CurrentRevision(context.Context, string) (RevisionSnapshot, error)
	}
	EditorialContext struct {
		Channel      string                `json:"channel"`
		Format       string                `json:"format"`
		CampaignID   string                `json:"campaign_id"`
		ProjectID    string                `json:"project_id"`
		PostTypeID   string                `json:"post_type_id"`
		Lane         string                `json:"lane"`
		SKU          string                `json:"sku"`
		ScenarioName string                `json:"scenario_name"`
		Attachments  []EditorialAttachment `json:"attachments"`
	}
	RevisionProvenance struct {
		ActorKind string `json:"actor_kind"`
		Capacity  string `json:"capacity"`
		CreatedAt string `json:"created_at"`
	}
	EditorialAttachment struct {
		ID          string `json:"id"`
		AssetID     string `json:"asset_id"`
		Role        string `json:"role"`
		AspectRatio string `json:"aspect_ratio"`
		AltText     string `json:"alt_text"`
		Position    int    `json:"position"`
	}
	EditorialSnapshot struct {
		RevisionSnapshot
		Context            EditorialContext
		RevisionProvenance *RevisionProvenance
	}
	EditorialSnapshotSource interface {
		CurrentEditorialSnapshotTx(context.Context, *sql.Tx, string) (EditorialSnapshot, error)
	}
	SQLExecutor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	Service interface {
		Record(context.Context, string, []Verdict) (Run, error)
		List(context.Context) ([]Run, error)
		CreateEditorialPackage(context.Context, string, string, string, string) (EditorialPackage, error)
		GetEditorialPackage(context.Context, string) (EditorialPackage, error)
	}
	service struct {
		db        SQLExecutor
		revisions RevisionSource
	}
)

const EditorialPackageScope = "destination_unbound_wording"

func NewService(db SQLExecutor, revisions RevisionSource) Service {
	return &service{db: db, revisions: revisions}
}

func (s *service) CreateEditorialPackage(ctx context.Context, draftID, expectedRevisionID, expectedBody, idempotencyKey string) (EditorialPackage, error) {
	draftID = strings.TrimSpace(draftID)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if draftID == "" {
		return EditorialPackage{}, fmt.Errorf("draft id is required")
	}
	if idempotencyKey == "" {
		return EditorialPackage{}, fmt.Errorf("idempotency key is required")
	}
	if strings.TrimSpace(expectedBody) == "" {
		return EditorialPackage{}, fmt.Errorf("a wording package requires a non-empty expected body")
	}
	bodyHash := digest([]byte(expectedBody))
	fingerprintBasis, err := json.Marshal(struct{ DraftID, RevisionID, BodySHA256 string }{draftID, expectedRevisionID, bodyHash})
	if err != nil {
		return EditorialPackage{}, err
	}
	fingerprint := digest(fingerprintBasis)
	if prior, found, lookupErr := s.packageOperation(ctx, idempotencyKey); lookupErr != nil {
		return EditorialPackage{}, lookupErr
	} else if found {
		if prior.fingerprint != fingerprint {
			return EditorialPackage{}, fmt.Errorf("idempotency key was reused with a different wording-package request")
		}
		return s.GetEditorialPackage(ctx, prior.packageID)
	}
	if s.revisions == nil {
		return EditorialPackage{}, fmt.Errorf("current revision source is unavailable")
	}
	current, err := s.revisions.CurrentRevision(ctx, draftID)
	if err != nil {
		return EditorialPackage{}, fmt.Errorf("read current draft revision: %w", err)
	}
	if current.RevisionID != expectedRevisionID {
		return EditorialPackage{}, fmt.Errorf("draft revision changed; refresh the exact wording before creating a package")
	}
	if current.Body != expectedBody {
		return EditorialPackage{}, fmt.Errorf("draft wording changed since the queue loaded; refresh before creating a package")
	}
	txBeginner, ok := s.db.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return EditorialPackage{}, fmt.Errorf("transactional storage is required for idempotent package admission")
	}
	tx, err := txBeginner.BeginTx(ctx, nil)
	if err != nil {
		return s.recoverPackageAdmission(ctx, idempotencyKey, fingerprint, err)
	}
	defer tx.Rollback()
	var priorFingerprint, priorPackageID string
	err = tx.QueryRowContext(ctx, `SELECT request_fingerprint,package_id FROM editorial_review_package_operations WHERE idempotency_key = ?`, idempotencyKey).Scan(&priorFingerprint, &priorPackageID)
	if err == nil {
		if priorFingerprint != fingerprint {
			return EditorialPackage{}, fmt.Errorf("idempotency key was reused with a different wording-package request")
		}
		if err := tx.Commit(); err != nil {
			return EditorialPackage{}, err
		}
		return s.GetEditorialPackage(ctx, priorPackageID)
	}
	if err != sql.ErrNoRows {
		return EditorialPackage{}, err
	}
	snapshotSource, ok := s.revisions.(EditorialSnapshotSource)
	if !ok {
		_ = tx.Rollback()
		return EditorialPackage{}, fmt.Errorf("transactional editorial context source is required for package admission")
	}
	snapshot, err := snapshotSource.CurrentEditorialSnapshotTx(ctx, tx, draftID)
	if err != nil {
		_ = tx.Rollback()
		return EditorialPackage{}, fmt.Errorf("read transactional editorial snapshot: %w", err)
	}
	if snapshot.RevisionID != expectedRevisionID || snapshot.Body != expectedBody {
		_ = tx.Rollback()
		return EditorialPackage{}, fmt.Errorf("draft revision or wording changed before package admission; refresh before creating a package")
	}
	manifestJSON, err := json.Marshal(struct {
		SchemaVersion     string              `json:"schema_version"`
		DraftID           string              `json:"draft_id"`
		SourceRevisionID  string              `json:"source_revision_id"`
		RevisionAuthoring *RevisionProvenance `json:"revision_authoring,omitempty"`
		BodySHA256        string              `json:"body_sha256"`
		Scope             string              `json:"scope"`
		Context           EditorialContext    `json:"context"`
	}{"content-desk-editorial-snapshot/v2", draftID, snapshot.RevisionID, snapshot.RevisionProvenance, bodyHash, EditorialPackageScope, snapshot.Context})
	if err != nil {
		_ = tx.Rollback()
		return EditorialPackage{}, err
	}
	packageBasis, err := json.Marshal(struct {
		DraftID        string `json:"draft_id"`
		RevisionID     string `json:"source_revision_id"`
		Scope          string `json:"scope"`
		BodySHA256     string `json:"body_sha256"`
		ManifestSHA256 string `json:"manifest_sha256"`
	}{draftID, snapshot.RevisionID, EditorialPackageScope, bodyHash, digest(manifestJSON)})
	if err != nil {
		_ = tx.Rollback()
		return EditorialPackage{}, err
	}
	created := time.Now().UTC()
	item := EditorialPackage{ID: uuid.NewSHA1(uuid.NameSpaceURL, []byte("content-desk/editorial-package/"+idempotencyKey)).String(), DraftID: draftID, SourceRevisionID: snapshot.RevisionID, Scope: EditorialPackageScope, Body: snapshot.Body, BodySHA256: bodyHash, PackageSHA256: digest(packageBasis), ManifestJSON: string(manifestJSON), CreatedAt: created}
	_, err = tx.ExecContext(ctx, `INSERT INTO editorial_review_packages (id,draft_id,source_revision_id,scope,body,body_sha256,package_sha256,created_at) VALUES (?,?,?,?,?,?,?,?)`, item.ID, item.DraftID, item.SourceRevisionID, item.Scope, item.Body, item.BodySHA256, item.PackageSHA256, item.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		_ = tx.Rollback()
		return s.recoverPackageAdmission(ctx, idempotencyKey, fingerprint, err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO editorial_review_package_manifests (package_id,manifest_json) VALUES (?,?)`, item.ID, item.ManifestJSON); err != nil {
		_ = tx.Rollback()
		return s.recoverPackageAdmission(ctx, idempotencyKey, fingerprint, err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO editorial_review_package_operations (idempotency_key,request_fingerprint,package_id,created_at) VALUES (?,?,?,?)`, idempotencyKey, fingerprint, item.ID, item.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		_ = tx.Rollback()
		return s.recoverPackageAdmission(ctx, idempotencyKey, fingerprint, err)
	}
	if err := tx.Commit(); err != nil {
		_ = tx.Rollback()
		return s.recoverPackageAdmission(ctx, idempotencyKey, fingerprint, err)
	}
	return item, nil
}

func (s *service) recoverPackageAdmission(ctx context.Context, key, fingerprint string, cause error) (EditorialPackage, error) {
	prior, found, err := s.packageOperation(ctx, key)
	if err != nil {
		return EditorialPackage{}, err
	}
	if !found {
		return EditorialPackage{}, cause
	}
	if prior.fingerprint != fingerprint {
		return EditorialPackage{}, fmt.Errorf("idempotency key was reused with a different wording-package request")
	}
	return s.GetEditorialPackage(ctx, prior.packageID)
}

type packageOperation struct{ fingerprint, packageID string }

func (s *service) packageOperation(ctx context.Context, key string) (packageOperation, bool, error) {
	var found packageOperation
	err := s.db.QueryRowContext(ctx, `SELECT request_fingerprint,package_id FROM editorial_review_package_operations WHERE idempotency_key = ?`, key).Scan(&found.fingerprint, &found.packageID)
	if err == sql.ErrNoRows {
		return packageOperation{}, false, nil
	}
	if err != nil {
		return packageOperation{}, false, err
	}
	return found, true, nil
}

func (s *service) GetEditorialPackage(ctx context.Context, id string) (EditorialPackage, error) {
	var item EditorialPackage
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT id,draft_id,source_revision_id,scope,body,body_sha256,package_sha256,created_at FROM editorial_review_packages WHERE id = ?`, strings.TrimSpace(id)).Scan(&item.ID, &item.DraftID, &item.SourceRevisionID, &item.Scope, &item.Body, &item.BodySHA256, &item.PackageSHA256, &created)
	if err != nil {
		return EditorialPackage{}, err
	}
	item.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return EditorialPackage{}, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT manifest_json FROM editorial_review_package_manifests WHERE package_id = ?`, item.ID).Scan(&item.ManifestJSON)
	if err == sql.ErrNoRows {
		// Legacy wording-only packages retain their original digest basis.
		legacyBasis, marshalErr := json.Marshal(struct {
			DraftID    string `json:"draft_id"`
			RevisionID string `json:"source_revision_id"`
			Scope      string `json:"scope"`
			BodySHA256 string `json:"body_sha256"`
		}{item.DraftID, item.SourceRevisionID, item.Scope, digest([]byte(item.Body))})
		if marshalErr != nil {
			return EditorialPackage{}, marshalErr
		}
		if item.BodySHA256 != digest([]byte(item.Body)) || item.PackageSHA256 != digest(legacyBasis) {
			return EditorialPackage{}, fmt.Errorf("editorial package %q failed its content digest check", item.ID)
		}
		return item, nil
	}
	if err != nil {
		return EditorialPackage{}, err
	}
	manifestBasis, err := json.Marshal(struct {
		DraftID        string `json:"draft_id"`
		RevisionID     string `json:"source_revision_id"`
		Scope          string `json:"scope"`
		BodySHA256     string `json:"body_sha256"`
		ManifestSHA256 string `json:"manifest_sha256"`
	}{item.DraftID, item.SourceRevisionID, item.Scope, digest([]byte(item.Body)), digest([]byte(item.ManifestJSON))})
	if err != nil {
		return EditorialPackage{}, err
	}
	if item.BodySHA256 != digest([]byte(item.Body)) || item.PackageSHA256 != digest(manifestBasis) {
		return EditorialPackage{}, fmt.Errorf("editorial package %q failed its content digest check", item.ID)
	}
	return item, nil
}

func digest(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}
func (s *service) Record(ctx context.Context, draftID string, verdicts []Verdict) (Run, error) {
	if len(verdicts) == 0 {
		return Run{}, fmt.Errorf("review run requires declared failure-mode verdicts")
	}
	if err := s.validateDeclaredModes(ctx, draftID, verdicts); err != nil {
		return Run{}, err
	}
	priorRunIDs, err := s.unsupersededRunIDs(ctx, draftID)
	if err != nil {
		return Run{}, err
	}
	run := Run{ID: uuid.NewString(), DraftID: draftID, Outcome: OutcomePassed, Verdicts: verdicts}
	for _, v := range verdicts {
		if !v.Passed {
			run.Outcome = OutcomeBlocked
		}
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO review_runs (id,draft_id,outcome,created_at) VALUES (?,?,?,?)`, run.ID, run.DraftID, run.Outcome, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return Run{}, err
	}
	for _, v := range verdicts {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO review_verdicts (run_id,mode,passed,evidence,finding) VALUES (?,?,?,?,?)`, run.ID, v.Mode, boolInt(v.Passed), v.Evidence, v.Finding); err != nil {
			return Run{}, err
		}
	}
	for _, priorRunID := range priorRunIDs {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO review_supersessions (superseded_run_id, superseding_run_id) VALUES (?, ?)`, priorRunID, run.ID); err != nil {
			return Run{}, err
		}
	}
	return run, nil
}

func (s *service) unsupersededRunIDs(ctx context.Context, draftID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM review_runs WHERE draft_id = ? AND id NOT IN (SELECT superseded_run_id FROM review_supersessions)`, draftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *service) validateDeclaredModes(ctx context.Context, draftID string, verdicts []Verdict) error {
	var postTypeID string
	if err := s.db.QueryRowContext(ctx, `SELECT post_type_id FROM drafts WHERE id = ?`, draftID).Scan(&postTypeID); err != nil {
		return fmt.Errorf("load draft post type: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT failure_mode FROM post_type_failure_modes WHERE post_type_id = ? UNION SELECT mode FROM review_policy_failure_modes ORDER BY 1`, postTypeID)
	if err != nil {
		return err
	}
	defer rows.Close()
	expected := map[string]struct{}{}
	for rows.Next() {
		var mode string
		if err := rows.Scan(&mode); err != nil {
			return err
		}
		expected[mode] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(expected) == 0 {
		return fmt.Errorf("post type %q has no declared failure modes", postTypeID)
	}
	seen := map[string]struct{}{}
	for _, verdict := range verdicts {
		if verdict.Mode == "" {
			return fmt.Errorf("review verdict mode is required")
		}
		if _, duplicate := seen[verdict.Mode]; duplicate {
			return fmt.Errorf("duplicate review verdict mode %q", verdict.Mode)
		}
		seen[verdict.Mode] = struct{}{}
		if _, declared := expected[verdict.Mode]; !declared {
			return fmt.Errorf("review verdict mode %q is not declared by post type %q", verdict.Mode, postTypeID)
		}
	}
	for mode := range expected {
		if _, present := seen[mode]; !present {
			return fmt.Errorf("review is missing declared failure mode %q", mode)
		}
	}
	return nil
}

func (s *service) List(ctx context.Context) ([]Run, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,draft_id,outcome FROM review_runs ORDER BY created_at DESC,id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		var run Run
		if err := rows.Scan(&run.ID, &run.DraftID, &run.Outcome); err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// A SQLite :memory: database exposes one connection in tests. Close the
	// parent cursor before issuing child queries so List cannot self-deadlock.
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for i := range out {
		verdicts, err := s.verdicts(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Verdicts = verdicts
	}
	return out, nil
}

func (s *service) verdicts(ctx context.Context, id string) ([]Verdict, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT mode,passed,evidence,finding FROM review_verdicts WHERE run_id=? ORDER BY passed ASC, mode`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Verdict
	for rows.Next() {
		var v Verdict
		var passed int
		if err := rows.Scan(&v.Mode, &passed, &v.Evidence, &v.Finding); err != nil {
			return nil, err
		}
		v.Passed = passed != 0
		out = append(out, v)
	}
	return out, rows.Err()
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
