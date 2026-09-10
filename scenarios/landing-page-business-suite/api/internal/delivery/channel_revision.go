package delivery

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrChannelRevisionConflict = errors.New("channel revision conflict")
	ErrChannelHalted           = errors.New("channel is halted")
)

// ChannelPromotionRequest identifies the complete immutable artifact set that
// should become visible for one channel. Artifact IDs are keyed by the exact
// update platform name used by the electron updater contract.
type ChannelPromotionRequest struct {
	BundleKey              string           `json:"bundle_key,omitempty"`
	AppKey                 string           `json:"app_key"`
	VariantKey             string           `json:"variant_key,omitempty"`
	ExpectedRevision       int64            `json:"expected_revision"`
	ArtifactIDs            map[string]int64 `json:"artifact_ids"`
	ReleaseID              string           `json:"release_id,omitempty"`
	ArtifactManifestDigest string           `json:"artifact_manifest_digest,omitempty"`
	CandidateID            string           `json:"candidate_id,omitempty"`
	DestinationRevisionID  string           `json:"destination_revision_id,omitempty"`
	AuthorizationEpoch     uint64           `json:"authorization_epoch,omitempty"`
	ReadinessReviewKey     string           `json:"readiness_review_key,omitempty"`
}

type ChannelRevision struct {
	BundleKey              string           `json:"bundle_key"`
	AppKey                 string           `json:"app_key"`
	VariantKey             string           `json:"variant_key"`
	Revision               int64            `json:"revision"`
	PredecessorRevision    int64            `json:"predecessor_revision"`
	ArtifactIDs            map[string]int64 `json:"artifact_ids"`
	ReleaseID              string           `json:"release_id,omitempty"`
	ArtifactManifestDigest string           `json:"artifact_manifest_digest,omitempty"`
	CandidateID            string           `json:"candidate_id,omitempty"`
	DestinationRevisionID  string           `json:"destination_revision_id,omitempty"`
	AuthorizationEpoch     uint64           `json:"authorization_epoch,omitempty"`
	ReadinessReviewKey     string           `json:"readiness_review_key,omitempty"`
	CreatedAt              time.Time        `json:"created_at"`
}

type ChannelHead struct {
	BundleKey   string           `json:"bundle_key"`
	AppKey      string           `json:"app_key"`
	VariantKey  string           `json:"variant_key"`
	Revision    int64            `json:"revision"`
	ArtifactIDs map[string]int64 `json:"artifact_ids"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// ChannelHaltRequest changes the mutable offer gate for one exact channel
// head. The expected revision prevents an operator from halting or resuming a
// channel after its destination has changed underneath the request.
type ChannelHaltRequest struct {
	BundleKey        string `json:"bundle_key"`
	AppKey           string `json:"app_key"`
	VariantKey       string `json:"variant_key"`
	ExpectedRevision int64  `json:"expected_revision"`
	Halted           bool   `json:"halted"`
	Reason           string `json:"reason"`
	DryRun           bool   `json:"dry_run,omitempty"`
}

type ChannelHalt struct {
	BundleKey       string    `json:"bundle_key"`
	AppKey          string    `json:"app_key"`
	VariantKey      string    `json:"variant_key"`
	Revision        int64     `json:"revision"`
	Halted          bool      `json:"halted"`
	Reason          string    `json:"reason,omitempty"`
	UpdatedAt       time.Time `json:"updated_at"`
	ObservedAt      time.Time `json:"observed_at"`
	Outcome         string    `json:"outcome"`
	Health          string    `json:"health"`
	ExternalReceipt string    `json:"external_receipt,omitempty"`
	DryRun          bool      `json:"dry_run"`
}

// ChannelRecoveryRequest is an owner-routed recovery intent. Recovery never
// mutates an existing immutable revision: rollback creates a new head that
// points at the exact predecessor artifact set, while forward repair creates a
// new head from an explicitly supplied, compatibility-qualified artifact set.
type ChannelRecoveryRequest struct {
	BundleKey             string           `json:"bundle_key"`
	AppKey                string           `json:"app_key"`
	VariantKey            string           `json:"variant_key"`
	ExpectedRevision      int64            `json:"expected_revision"`
	ExpectedPredecessor   int64            `json:"expected_predecessor_revision"`
	Action                string           `json:"action"`
	DataCompatibility     string           `json:"data_compatibility"`
	ArtifactIDs           map[string]int64 `json:"artifact_ids,omitempty"`
	Reason                string           `json:"reason"`
	CandidateID           string           `json:"candidate_id,omitempty"`
	DestinationRevisionID string           `json:"destination_revision_id,omitempty"`
	DryRun                bool             `json:"dry_run,omitempty"`
}

type ChannelRecovery struct {
	BundleKey             string    `json:"bundle_key"`
	AppKey                string    `json:"app_key"`
	VariantKey            string    `json:"variant_key"`
	Revision              int64     `json:"revision"`
	PredecessorRevision   int64     `json:"predecessor_revision"`
	Action                string    `json:"action"`
	Outcome               string    `json:"outcome"`
	Health                string    `json:"health"`
	ExternalReceipt       string    `json:"external_receipt,omitempty"`
	CandidateID           string    `json:"candidate_id,omitempty"`
	DestinationRevisionID string    `json:"destination_revision_id,omitempty"`
	ObservedAt            time.Time `json:"observed_at"`
	DryRun                bool      `json:"dry_run"`
}

// RecoverChannel applies the owner-specific safe recovery vocabulary. A
// rollback is accepted only when the current revision names the requested
// predecessor and the caller has explicitly qualified data compatibility.
// Forward repair has the same compatibility gate and requires a complete
// replacement artifact set.
func (s *CatalogService) RecoverChannel(ctx context.Context, req ChannelRecoveryRequest) (*ChannelRecovery, error) {
	req.BundleKey, req.AppKey, req.VariantKey = strings.TrimSpace(req.BundleKey), strings.TrimSpace(req.AppKey), strings.TrimSpace(req.VariantKey)
	if req.VariantKey == "" {
		req.VariantKey = "default"
	}
	req.Action, req.DataCompatibility = strings.TrimSpace(req.Action), strings.TrimSpace(req.DataCompatibility)
	if req.BundleKey == "" || req.AppKey == "" || req.ExpectedRevision < 0 || req.Action == "" {
		return nil, fmt.Errorf("bundle_key, app_key, expected_revision, and action are required")
	}
	if req.Action != "withdraw" && req.Action != "rollback" && req.Action != "forward_repair" {
		return nil, fmt.Errorf("unsupported channel recovery action %q", req.Action)
	}
	if req.Action != "withdraw" && req.DataCompatibility != "compatible" {
		return nil, fmt.Errorf("%s requires explicit data_compatibility=compatible; unsafe binary rollback is refused", req.Action)
	}
	if req.Action == "forward_repair" && len(req.ArtifactIDs) == 0 {
		return nil, fmt.Errorf("forward_repair requires a replacement artifact set")
	}

	head, err := s.GetChannelHead(req.BundleKey, req.AppKey, req.VariantKey)
	if err != nil {
		return nil, fmt.Errorf("read channel head for recovery: %w", err)
	}
	current := int64(0)
	if head != nil {
		current = head.Revision
	}
	if current != req.ExpectedRevision {
		return nil, fmt.Errorf("%w: expected %d, current %d", ErrChannelRevisionConflict, req.ExpectedRevision, current)
	}

	predecessor := int64(0)
	if req.Action == "rollback" {
		if req.ExpectedPredecessor <= 0 {
			return nil, fmt.Errorf("rollback requires a positive expected_predecessor_revision")
		}
		row := s.db.QueryRow(`SELECT predecessor_revision FROM download_channel_revisions WHERE bundle_key = $1 AND app_key = $2 AND variant_key = $3 AND revision = $4`, req.BundleKey, req.AppKey, req.VariantKey, current)
		if err := row.Scan(&predecessor); err != nil {
			return nil, fmt.Errorf("read rollback predecessor: %w", err)
		}
		if predecessor != req.ExpectedPredecessor {
			return nil, fmt.Errorf("rollback predecessor conflict: expected %d, recorded %d", req.ExpectedPredecessor, predecessor)
		}
	}
	now := time.Now().UTC()
	if req.DryRun {
		return &ChannelRecovery{BundleKey: req.BundleKey, AppKey: req.AppKey, VariantKey: req.VariantKey, Revision: current, PredecessorRevision: req.ExpectedPredecessor, Action: req.Action, Outcome: "preview", Health: "unknown", CandidateID: req.CandidateID, DestinationRevisionID: req.DestinationRevisionID, ObservedAt: now, DryRun: true}, nil
	}
	if req.Action == "withdraw" {
		halt, err := s.SetChannelHalt(ctx, ChannelHaltRequest{BundleKey: req.BundleKey, AppKey: req.AppKey, VariantKey: req.VariantKey, ExpectedRevision: current, Halted: true, Reason: req.Reason})
		if err != nil {
			return nil, err
		}
		return &ChannelRecovery{BundleKey: req.BundleKey, AppKey: req.AppKey, VariantKey: req.VariantKey, Revision: halt.Revision, Action: req.Action, Outcome: "withdrawn", Health: "stopped", ExternalReceipt: "lpbs:channel-withdrawal:" + halt.ExternalReceipt, CandidateID: req.CandidateID, DestinationRevisionID: req.DestinationRevisionID, ObservedAt: halt.ObservedAt}, nil
	}

	artifactIDs := req.ArtifactIDs
	if req.Action == "rollback" {
		row := s.db.QueryRow(`SELECT artifact_set FROM download_channel_revisions WHERE bundle_key = $1 AND app_key = $2 AND variant_key = $3 AND revision = $4`, req.BundleKey, req.AppKey, req.VariantKey, predecessor)
		var encoded []byte
		if err := row.Scan(&encoded); err != nil {
			return nil, fmt.Errorf("read rollback artifact set: %w", err)
		}
		if err := json.Unmarshal(encoded, &artifactIDs); err != nil {
			return nil, fmt.Errorf("decode rollback artifact set: %w", err)
		}
	}
	revision, err := s.PromoteChannel(ctx, ChannelPromotionRequest{BundleKey: req.BundleKey, AppKey: req.AppKey, VariantKey: req.VariantKey, ExpectedRevision: current, ArtifactIDs: artifactIDs})
	if err != nil {
		return nil, err
	}
	outcome, health := "forward_repaired", "channel_updated"
	if req.Action == "rollback" {
		outcome, health = "rolled_back", "channel_restored"
	}
	return &ChannelRecovery{BundleKey: req.BundleKey, AppKey: req.AppKey, VariantKey: req.VariantKey, Revision: revision.Revision, PredecessorRevision: revision.PredecessorRevision, Action: req.Action, Outcome: outcome, Health: health, ExternalReceipt: fmt.Sprintf("lpbs:channel-recovery:%s:%s:%s:%d", req.Action, req.AppKey, req.VariantKey, revision.Revision), CandidateID: req.CandidateID, DestinationRevisionID: req.DestinationRevisionID, ObservedAt: revision.CreatedAt}, nil
}

func (s *CatalogService) SetChannelHalt(ctx context.Context, req ChannelHaltRequest) (*ChannelHalt, error) {
	req.BundleKey, req.AppKey, req.VariantKey = strings.TrimSpace(req.BundleKey), strings.TrimSpace(req.AppKey), strings.TrimSpace(req.VariantKey)
	if req.VariantKey == "" {
		req.VariantKey = "default"
	}
	if req.BundleKey == "" || req.AppKey == "" || req.ExpectedRevision < 0 {
		return nil, fmt.Errorf("bundle_key, app_key, and non-negative expected_revision are required")
	}
	if req.Halted && strings.TrimSpace(req.Reason) == "" {
		return nil, fmt.Errorf("reason is required when halting a channel")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin channel halt: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var revision int64
	err = tx.QueryRowContext(ctx, `
		SELECT current_revision FROM download_channel_heads
		WHERE bundle_key = $1 AND app_key = $2 AND variant_key = $3 FOR UPDATE
	`, req.BundleKey, req.AppKey, req.VariantKey).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		revision = 0
	} else if err != nil {
		return nil, fmt.Errorf("read channel head for halt: %w", err)
	}
	if revision != req.ExpectedRevision {
		return nil, fmt.Errorf("%w: expected %d, current %d", ErrChannelRevisionConflict, req.ExpectedRevision, revision)
	}
	if req.DryRun {
		now := time.Now().UTC()
		return &ChannelHalt{BundleKey: req.BundleKey, AppKey: req.AppKey, VariantKey: req.VariantKey, Revision: revision, Halted: req.Halted, Reason: strings.TrimSpace(req.Reason), UpdatedAt: now, ObservedAt: now, Outcome: "preview", Health: "unknown", DryRun: true}, nil
	}
	var updatedAt time.Time
	err = tx.QueryRowContext(ctx, `
		INSERT INTO download_channel_halts (bundle_key, app_key, variant_key, revision, halted, reason)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (bundle_key, app_key, variant_key) DO UPDATE SET
			revision = EXCLUDED.revision, halted = EXCLUDED.halted, reason = EXCLUDED.reason, updated_at = NOW()
		RETURNING updated_at
	`, req.BundleKey, req.AppKey, req.VariantKey, revision, req.Halted, strings.TrimSpace(req.Reason)).Scan(&updatedAt)
	if err != nil {
		return nil, fmt.Errorf("persist channel halt: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit channel halt: %w", err)
	}
	return &ChannelHalt{BundleKey: req.BundleKey, AppKey: req.AppKey, VariantKey: req.VariantKey, Revision: revision, Halted: req.Halted, Reason: strings.TrimSpace(req.Reason), UpdatedAt: updatedAt.UTC(), ObservedAt: updatedAt.UTC(), Outcome: "halted", Health: "stopped", ExternalReceipt: fmt.Sprintf("lpbs:channel-halt:%s:%s:%s:%d:%d", req.BundleKey, req.AppKey, req.VariantKey, revision, updatedAt.UnixNano())}, nil
}

func (s *CatalogService) IsChannelHalted(bundleKey, appKey, variantKey string) (bool, error) {
	if strings.TrimSpace(variantKey) == "" {
		variantKey = "default"
	}
	var halted bool
	err := s.db.QueryRow(`
		SELECT halted FROM download_channel_halts
		WHERE bundle_key = $1 AND app_key = $2 AND variant_key = $3
	`, bundleKey, appKey, variantKey).Scan(&halted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return halted, err
}

// PromoteChannel records a new immutable artifact set and advances the
// destination head only when ExpectedRevision is still current.
func (s *CatalogService) PromoteChannel(ctx context.Context, req ChannelPromotionRequest) (*ChannelRevision, error) {
	req.BundleKey = strings.TrimSpace(req.BundleKey)
	req.AppKey = strings.TrimSpace(req.AppKey)
	req.VariantKey = strings.TrimSpace(req.VariantKey)
	req.ReleaseID = strings.TrimSpace(req.ReleaseID)
	req.ArtifactManifestDigest = strings.TrimSpace(req.ArtifactManifestDigest)
	req.CandidateID = strings.TrimSpace(req.CandidateID)
	req.DestinationRevisionID = strings.TrimSpace(req.DestinationRevisionID)
	req.ReadinessReviewKey = strings.TrimSpace(req.ReadinessReviewKey)
	if req.VariantKey == "" {
		req.VariantKey = "default"
	}
	if req.BundleKey == "" || req.AppKey == "" || len(req.ArtifactIDs) == 0 {
		return nil, fmt.Errorf("bundle_key, app_key, and artifact_ids are required")
	}
	releaseBound := strings.TrimSpace(req.ReleaseID) != ""
	if releaseBound && (strings.TrimSpace(req.ArtifactManifestDigest) == "" || strings.TrimSpace(req.CandidateID) == "" || strings.TrimSpace(req.DestinationRevisionID) == "" || req.AuthorizationEpoch == 0 || strings.TrimSpace(req.ReadinessReviewKey) == "") {
		return nil, fmt.Errorf("release-bound channel promotion requires release, manifest, candidate, destination, authorization epoch, and readiness review identity")
	}

	artifactSet := make(map[string]int64, len(req.ArtifactIDs))
	for platform, id := range req.ArtifactIDs {
		platform = strings.TrimSpace(platform)
		if platform == "" || id <= 0 {
			return nil, fmt.Errorf("artifact_ids must contain positive IDs keyed by platform")
		}
		artifactSet[platform] = id
	}
	encoded, err := json.Marshal(artifactSet)
	if err != nil {
		return nil, fmt.Errorf("marshal artifact set: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin channel promotion: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var currentRevision int64
	var currentSet []byte
	err = tx.QueryRowContext(ctx, `
		SELECT current_revision, artifact_set
		FROM download_channel_heads
		WHERE bundle_key = $1 AND app_key = $2 AND variant_key = $3
		FOR UPDATE
	`, req.BundleKey, req.AppKey, req.VariantKey).Scan(&currentRevision, &currentSet)
	if errors.Is(err, sql.ErrNoRows) {
		currentRevision = 0
	} else if err != nil {
		return nil, fmt.Errorf("read channel head: %w", err)
	}
	if currentRevision != req.ExpectedRevision {
		return nil, fmt.Errorf("%w: expected %d, current %d", ErrChannelRevisionConflict, req.ExpectedRevision, currentRevision)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT platform FROM download_assets
		WHERE bundle_key = $1 AND app_key = $2 AND variant_key = $3
	`, req.BundleKey, req.AppKey, req.VariantKey)
	if err != nil {
		return nil, fmt.Errorf("read channel target set: %w", err)
	}
	configuredPlatforms := make(map[string]struct{})
	for rows.Next() {
		var platform string
		if err := rows.Scan(&platform); err != nil {
			rows.Close()
			return nil, fmt.Errorf("read channel target: %w", err)
		}
		configuredPlatforms[strings.TrimSpace(platform)] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("read channel target set: %w", err)
	}
	rows.Close()
	if len(configuredPlatforms) > 0 && len(configuredPlatforms) != len(artifactSet) {
		return nil, fmt.Errorf("channel promotion requires the complete configured target set")
	}
	for platform := range configuredPlatforms {
		if _, ok := artifactSet[platform]; !ok {
			return nil, fmt.Errorf("channel promotion is missing configured platform %q", platform)
		}
	}

	for platform, artifactID := range artifactSet {
		var storedAppKey, storedPlatform, storedReleaseID string
		var releaseVersion, sha256, sha512 string
		var metadata []byte
		if err := tx.QueryRowContext(ctx, `
			SELECT COALESCE(app_key, ''), platform, COALESCE(release_id, ''), COALESCE(release_version, ''), COALESCE(sha256, ''), COALESCE(sha512, ''), COALESCE(metadata, '{}'::jsonb)
			FROM download_artifacts WHERE bundle_key = $1 AND id = $2
		`, req.BundleKey, artifactID).Scan(&storedAppKey, &storedPlatform, &storedReleaseID, &releaseVersion, &sha256, &sha512, &metadata); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("artifact %d is not owned by bundle %q", artifactID, req.BundleKey)
			}
			return nil, fmt.Errorf("validate artifact %d: %w", artifactID, err)
		}
		if strings.TrimSpace(storedPlatform) != platform {
			return nil, fmt.Errorf("artifact %d platform %q does not match %q", artifactID, storedPlatform, platform)
		}
		if releaseBound && strings.TrimSpace(storedAppKey) != req.AppKey {
			return nil, fmt.Errorf("artifact %d app_key %q does not match %q", artifactID, storedAppKey, req.AppKey)
		}
		if releaseBound && strings.TrimSpace(storedReleaseID) != req.ReleaseID {
			return nil, fmt.Errorf("artifact %d release_id %q does not match %q", artifactID, storedReleaseID, req.ReleaseID)
		}
		if strings.TrimSpace(releaseVersion) == "" {
			return nil, fmt.Errorf("artifact %d has no release version", artifactID)
		}
		if strings.TrimSpace(sha512) == "" {
			return nil, fmt.Errorf("artifact %d has no sha512 identity", artifactID)
		}
		if len(metadata) == 0 {
			metadata = []byte(`{}`)
		}
		if releaseBound {
			if err := validateReleaseIdentity(metadata, req); err != nil {
				return nil, fmt.Errorf("artifact %d release identity: %w", artifactID, err)
			}
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO download_assets
				(bundle_key, app_key, platform, variant_key, artifact_source, artifact_id, release_version, release_notes, checksum, metadata, updated_at)
			VALUES ($1,$2,$3,$4,'managed',$5,$6,'',$7,$8,NOW())
			ON CONFLICT (bundle_key, app_key, platform, variant_key) DO UPDATE SET
				artifact_source = 'managed', artifact_id = EXCLUDED.artifact_id,
				release_version = EXCLUDED.release_version, checksum = EXCLUDED.checksum,
				metadata = EXCLUDED.metadata, updated_at = NOW()
		`, req.BundleKey, req.AppKey, platform, req.VariantKey, artifactID, releaseVersion, sha256, metadata); err != nil {
			return nil, fmt.Errorf("stage channel asset %q: %w", platform, err)
		}
	}

	nextRevision := currentRevision + 1
	var createdAt time.Time
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO download_channel_revisions
			(bundle_key, app_key, variant_key, revision, predecessor_revision, artifact_set)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING created_at
	`, req.BundleKey, req.AppKey, req.VariantKey, nextRevision, currentRevision, encoded).Scan(&createdAt); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			return nil, fmt.Errorf("%w: channel revision was created concurrently", ErrChannelRevisionConflict)
		}
		return nil, fmt.Errorf("record channel revision: %w", err)
	}

	if currentRevision == 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO download_channel_heads
				(bundle_key, app_key, variant_key, current_revision, artifact_set)
			VALUES ($1,$2,$3,$4,$5)
		`, req.BundleKey, req.AppKey, req.VariantKey, nextRevision, encoded); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				return nil, fmt.Errorf("%w: channel head was created concurrently", ErrChannelRevisionConflict)
			}
			return nil, fmt.Errorf("create channel head: %w", err)
		}
	} else if result, err := tx.ExecContext(ctx, `
		UPDATE download_channel_heads
		SET current_revision = $4, artifact_set = $5, updated_at = NOW()
		WHERE bundle_key = $1 AND app_key = $2 AND variant_key = $3 AND current_revision = $6
	`, req.BundleKey, req.AppKey, req.VariantKey, nextRevision, encoded, currentRevision); err != nil {
		return nil, fmt.Errorf("advance channel head: %w", err)
	} else if affected, err := result.RowsAffected(); err != nil || affected != 1 {
		return nil, fmt.Errorf("%w: channel head predecessor changed", ErrChannelRevisionConflict)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit channel promotion: %w", err)
	}
	return &ChannelRevision{
		BundleKey: req.BundleKey, AppKey: req.AppKey, VariantKey: req.VariantKey,
		Revision: nextRevision, PredecessorRevision: currentRevision,
		ArtifactIDs: artifactSet, ReleaseID: req.ReleaseID, ArtifactManifestDigest: req.ArtifactManifestDigest,
		CandidateID: req.CandidateID, DestinationRevisionID: req.DestinationRevisionID,
		AuthorizationEpoch: req.AuthorizationEpoch, ReadinessReviewKey: req.ReadinessReviewKey,
		CreatedAt: createdAt.UTC(),
	}, nil
}

func validateReleaseIdentity(metadata []byte, req ChannelPromotionRequest) error {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(metadata, &values); err != nil {
		return fmt.Errorf("decode metadata: %w", err)
	}
	stringValue := func(key string) string {
		var value string
		if err := json.Unmarshal(values[key], &value); err != nil {
			return ""
		}
		return strings.TrimSpace(value)
	}
	if stringValue("bundle_key") != strings.TrimSpace(req.BundleKey) {
		return fmt.Errorf("bundle_key mismatch")
	}
	if stringValue("app_key") != strings.TrimSpace(req.AppKey) {
		return fmt.Errorf("app_key mismatch")
	}
	if stringValue("release_id") != strings.TrimSpace(req.ReleaseID) {
		return fmt.Errorf("release_id mismatch")
	}
	if stringValue("artifact_manifest_digest") != strings.TrimSpace(req.ArtifactManifestDigest) {
		return fmt.Errorf("artifact_manifest_digest mismatch")
	}
	if stringValue("candidate_id") != strings.TrimSpace(req.CandidateID) {
		return fmt.Errorf("candidate_id mismatch")
	}
	if stringValue("destination_revision_id") != strings.TrimSpace(req.DestinationRevisionID) {
		return fmt.Errorf("destination_revision_id mismatch")
	}
	if stringValue("readiness_review_key") != strings.TrimSpace(req.ReadinessReviewKey) {
		return fmt.Errorf("readiness_review_key mismatch")
	}
	var authEpoch uint64
	if err := json.Unmarshal(values["authorization_epoch"], &authEpoch); err != nil || authEpoch != req.AuthorizationEpoch {
		return fmt.Errorf("authorization_epoch mismatch")
	}
	return nil
}

func (s *CatalogService) GetChannelHead(bundleKey, appKey, variantKey string) (*ChannelHead, error) {
	var revision int64
	var artifactSet []byte
	var updatedAt time.Time
	err := s.db.QueryRow(`
		SELECT current_revision, artifact_set, updated_at
		FROM download_channel_heads
		WHERE bundle_key = $1 AND app_key = $2 AND variant_key = $3
	`, bundleKey, appKey, variantKey).Scan(&revision, &artifactSet, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids map[string]int64
	if err := json.Unmarshal(artifactSet, &ids); err != nil {
		return nil, fmt.Errorf("decode channel head: %w", err)
	}
	return &ChannelHead{BundleKey: bundleKey, AppKey: appKey, VariantKey: variantKey, Revision: revision, ArtifactIDs: ids, UpdatedAt: updatedAt.UTC()}, nil
}

func (s *CatalogService) currentChannelArtifactID(bundleKey, appKey, variantKey, platform string) (int64, bool, error) {
	var artifactSet []byte
	err := s.db.QueryRow(`
		SELECT artifact_set FROM download_channel_heads
		WHERE bundle_key = $1 AND app_key = $2 AND variant_key = $3
	`, bundleKey, appKey, variantKey).Scan(&artifactSet)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	var ids map[string]int64
	if err := json.Unmarshal(artifactSet, &ids); err != nil {
		return 0, true, fmt.Errorf("decode channel artifact set: %w", err)
	}
	id := ids[platform]
	return id, true, nil
}
