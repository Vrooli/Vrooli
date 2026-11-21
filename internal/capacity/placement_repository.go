package capacity

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxPlacementArtifactBytes = 8 << 20

// RecordPlacementQualification verifies the linked immutable run artifact,
// computes a content ID and appends the measurement. Replaying the same
// receipt is idempotent; changing any receipt field produces a different ID.
func (s *SQLiteStore) RecordPlacementQualification(ctx context.Context, q PlacementQualification) (PlacementQualification, error) {
	if err := validatePlacementQualification(q, s.now()); err != nil {
		return PlacementQualification{}, err
	}
	if err := verifyPlacementArtifact(q); err != nil {
		return PlacementQualification{}, err
	}
	q.ReceiptID = ""
	canonical, err := json.Marshal(q)
	if err != nil {
		return PlacementQualification{}, fmt.Errorf("encode placement qualification: %w", err)
	}
	sum := sha256.Sum256(canonical)
	q.ReceiptID = hex.EncodeToString(sum[:])
	payload, err := json.Marshal(q)
	if err != nil {
		return PlacementQualification{}, fmt.Errorf("encode identified placement qualification: %w", err)
	}
	err = s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		claim, err := getClaimTx(ctx, tx, q.ClaimID)
		if err != nil {
			return fmt.Errorf("verify placement receipt claim: %w", err)
		}
		if claim.Generation != q.ClaimGeneration || !IsActiveClaimStatus(claim.Status) {
			return ErrStaleGeneration
		}
		if claim.DegradeProfile == nil || claim.DegradeProfile.Role != q.Role || claim.DegradeProfile.Model != q.Model {
			return errors.New("placement receipt role/model do not match the current claim profile")
		}
		if _, ok := findStep(claim.DegradeProfile, q.Rung); !ok {
			return fmt.Errorf("placement receipt rung %q is not declared by the current claim profile", q.Rung)
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO capacity_placement_qualifications
(receipt_id, claim_id, claim_generation, role, model_digest, rung, device, suite_digest, sampled_at, qualification_json, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(receipt_id) DO NOTHING`, q.ReceiptID, q.ClaimID, q.ClaimGeneration, q.Role, q.Model.Digest, q.Rung, q.Device, q.SuiteDigest, formatTime(q.SampledAt), string(payload), formatTime(s.now()))
		if err != nil {
			return fmt.Errorf("append placement qualification: %w", err)
		}
		var stored string
		if err := tx.QueryRowContext(ctx, `SELECT qualification_json FROM capacity_placement_qualifications WHERE receipt_id=?`, q.ReceiptID).Scan(&stored); err != nil {
			return fmt.Errorf("verify stored placement qualification: %w", err)
		}
		if stored != string(payload) {
			return errors.New("content-addressed placement receipt ID has conflicting payload")
		}
		return nil
	})
	if err != nil {
		return PlacementQualification{}, err
	}
	return q, nil
}

func (s *SQLiteStore) ListPlacementQualifications(ctx context.Context, filter PlacementQualificationFilter) ([]PlacementQualification, error) {
	query := `SELECT receipt_id, qualification_json FROM capacity_placement_qualifications WHERE 1=1`
	args := make([]any, 0, 6)
	if v := strings.TrimSpace(filter.ClaimID); v != "" {
		query += ` AND claim_id=?`
		args = append(args, v)
	}
	if filter.ClaimGeneration > 0 {
		query += ` AND claim_generation=?`
		args = append(args, filter.ClaimGeneration)
	}
	if len(filter.ClaimIDs) > 0 {
		if len(filter.ClaimIDs) > 500 {
			return nil, errors.New("placement qualification claim filter exceeds 500 identities")
		}
		ids := append([]string(nil), filter.ClaimIDs...)
		sort.Strings(ids)
		query += ` AND claim_id IN (` + strings.TrimRight(strings.Repeat("?,", len(ids)), ",") + `)`
		for _, id := range ids {
			args = append(args, strings.TrimSpace(id))
		}
	}
	if v := strings.TrimSpace(filter.Role); v != "" {
		query += ` AND role=?`
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.ModelDigest); v != "" {
		query += ` AND model_digest=?`
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.Rung); v != "" {
		query += ` AND rung=?`
		args = append(args, v)
	}
	if v := strings.TrimSpace(filter.Device); v != "" {
		query += ` AND device=?`
		args = append(args, v)
	}
	if filter.Since != nil {
		query += ` AND sampled_at>=?`
		args = append(args, formatTime(*filter.Since))
	}
	query += ` ORDER BY sampled_at DESC, receipt_id ASC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list placement qualifications: %w", err)
	}
	defer rows.Close()
	var out []PlacementQualification
	for rows.Next() {
		var id, payload string
		if err := rows.Scan(&id, &payload); err != nil {
			return nil, err
		}
		var q PlacementQualification
		if err := json.Unmarshal([]byte(payload), &q); err != nil {
			return nil, fmt.Errorf("decode stored placement qualification: %w", err)
		}
		if q.ReceiptID != id {
			return nil, fmt.Errorf("stored placement receipt identity mismatch: row=%s payload=%s", id, q.ReceiptID)
		}
		if err := verifyPlacementArtifact(q); err != nil {
			return nil, fmt.Errorf("stored placement receipt %s artifact is unavailable or changed: %w", id, err)
		}
		out = append(out, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func validatePlacementQualification(q PlacementQualification, now time.Time) error {
	if strings.TrimSpace(q.ClaimID) == "" || q.ClaimGeneration <= 0 || strings.TrimSpace(q.Role) == "" || !q.Model.Complete() || strings.TrimSpace(q.Rung) == "" {
		return errors.New("placement qualification requires exact claim generation, role, model, and rung identity")
	}
	if strings.TrimSpace(q.Suite) == "" || !isSHA256(q.SuiteDigest) || q.Cases < 20 {
		return errors.New("placement qualification requires a suite digest and at least 20 cases")
	}
	if q.SampledAt.IsZero() || q.SampledAt.After(now) {
		return errors.New("placement qualification sample time is missing or in the future")
	}
	if !validResidencyBackend(q.Device) || q.P95LatencyMS <= 0 || q.MaxLatencyMS <= 0 || q.ObservedVRAM < 0 {
		return errors.New("placement qualification has invalid device, latency, or VRAM measurement")
	}
	if q.Device == ResidencyBackendCPU || q.Device == ResidencyBackendUnloaded {
		if q.ObservedVRAM != 0 {
			return errors.New("CPU/unloaded placement receipt must measure zero VRAM")
		}
	} else if q.ObservedVRAM <= 0 {
		return errors.New("accelerator placement receipt must measure positive VRAM")
	}
	if strings.TrimSpace(q.RunID) == "" || strings.TrimSpace(q.ArtifactURI) == "" || !isSHA256(q.ArtifactSHA) {
		return errors.New("placement qualification requires run ID and SHA-256-linked artifact")
	}
	return nil
}

func isSHA256(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func verifyPlacementArtifact(q PlacementQualification) error {
	path := filepath.Clean(strings.TrimSpace(q.ArtifactURI))
	if !filepath.IsAbs(path) || path != q.ArtifactURI {
		return errors.New("placement artifact URI must be a clean absolute local path")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("read placement suite artifact: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() < 1 || info.Size() > maxPlacementArtifactBytes {
		return fmt.Errorf("placement artifact must be a regular file between 1 and %d bytes", maxPlacementArtifactBytes)
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open placement suite artifact: %w", err)
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, io.LimitReader(f, maxPlacementArtifactBytes+1)); err != nil {
		return fmt.Errorf("hash placement suite artifact: %w", err)
	}
	if hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(q.ArtifactSHA) {
		return errors.New("placement suite artifact SHA-256 does not match receipt")
	}
	return nil
}
