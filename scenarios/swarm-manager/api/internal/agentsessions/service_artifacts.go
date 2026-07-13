package agentsessions

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/evidence"
	"swarm-manager/internal/idgen"
)

const sessionArtifactMigrationKey = "agent-session-artifacts/v1"

func (s *Service) AttachArtifact(ctx context.Context, artifact Artifact) (Artifact, error) {
	artifacts, err := s.AttachArtifacts(ctx, []Artifact{artifact})
	if err != nil {
		return Artifact{}, err
	}
	return artifacts[0], nil
}

func (s *Service) AttachArtifacts(ctx context.Context, artifacts []Artifact) ([]Artifact, error) {
	if len(artifacts) == 0 {
		return []Artifact{}, nil
	}
	for i := range artifacts {
		if artifacts[i].ID == "" {
			artifacts[i].ID = "art_" + idgen.Generate()
		}
		if artifacts[i].CreatedAt == "" {
			artifacts[i].CreatedAt = nowRFC3339()
		}
		if i > 0 && artifacts[i].SessionID != artifacts[0].SessionID {
			return nil, apierr.BadRequest("all artifacts must belong to the same session")
		}
		if err := artifacts[i].Validate(); err != nil {
			return nil, err
		}
	}
	if s == nil || s.evidenceService == nil {
		return nil, apierr.Unavailable("canonical evidence service is unavailable")
	}
	session, err := s.store.LoadSession(artifacts[0].SessionID)
	if err != nil {
		return nil, mapStoreError(err)
	}
	observations := make([]evidence.Observation, 0, len(artifacts))
	for _, artifact := range artifacts {
		observations = append(observations, artifactObservation("swarm-manager.session-artifact", artifact, artifactRunID(session, artifact), artifactConfidence(artifact), artifactVerification(artifact)))
	}
	if _, err := s.evidenceService.IngestForOwnerBatch(ctx, evidence.Owner{Kind: evidence.OwnerAgentSession, ID: session.ID}, observations); err != nil {
		return nil, fmt.Errorf("record canonical session artifacts: %w", err)
	}
	for _, artifact := range artifacts {
		s.emitArtifactLinked(artifact)
	}
	return artifacts, nil
}

func artifactRunID(session Session, artifact Artifact) string {
	runID := strings.TrimSpace(artifact.RunID)
	if runID == "" {
		if artifact.Attribution != nil {
			runID = strings.TrimSpace(artifact.Attribution.RunID)
		}
	}
	if runID == "" {
		runID = strings.TrimSpace(session.RunID)
	}
	if runID == "" {
		return "session-artifact:" + session.ID
	}
	return runID
}

func artifactConfidence(artifact Artifact) evidence.Confidence {
	confidence := evidence.ConfidenceReported
	if artifact.Attribution != nil && artifact.Attribution.Type == AttributionAgent && strings.TrimSpace(artifact.MutationSource) != "" {
		confidence = evidence.ConfidenceAuthoritative
	}
	return confidence
}

func artifactVerification(artifact Artifact) evidence.Verification {
	verification := evidence.VerificationUnverified
	if artifact.Attribution == nil {
		return verification
	}
	if artifact.Attribution.Type == AttributionAgent {
		verification = evidence.VerificationVerified
	}
	return verification
}

func artifactObservation(source string, artifact Artifact, runID string, confidence evidence.Confidence, verification evidence.Verification) evidence.Observation {
	createdAt, _ := time.Parse(time.RFC3339Nano, artifact.CreatedAt)
	return evidence.Observation{
		SourceSystem:  source,
		SourceEventID: artifact.ID,
		RunID:         runID,
		Subject:       evidence.Subject{Kind: string(artifact.ArtifactType), ID: artifact.EntityRef},
		Action:        string(artifact.Action),
		Confidence:    confidence,
		Verification:  verification,
		Metadata: map[string]string{
			"session_id":          artifact.SessionID,
			"artifact_id":         artifact.ID,
			"title":               artifact.Title,
			"proposal_id":         artifact.ProposalID,
			"activity_id":         artifact.ActivityID,
			"mutation_source":     artifact.MutationSource,
			"attribution_type":    stringAttributionType(artifact),
			"attribution_task":    attributionTask(artifact),
			"attribution_profile": attributionProfile(artifact),
		},
		ObservedAt: createdAt,
	}
}

func stringAttributionType(artifact Artifact) string {
	if artifact.Attribution == nil {
		return ""
	}
	return string(artifact.Attribution.Type)
}

func attributionTask(artifact Artifact) string {
	if artifact.Attribution == nil {
		return ""
	}
	return artifact.Attribution.TaskID
}

func attributionProfile(artifact Artifact) string {
	if artifact.Attribution == nil {
		return ""
	}
	return artifact.Attribution.ProfileKey
}

// MigrateArtifactEvidence imports every JSONL artifact as an explicit legacy
// source. Replays are idempotent, and legacy records without a run remain
// unverified rather than being falsely bound to an Agent Manager run.
func (s *Service) MigrateArtifactEvidence(ctx context.Context) error {
	if s == nil || s.evidenceService == nil {
		return nil
	}
	sessions, err := s.store.ListSessions(ListFilters{})
	if err != nil {
		return err
	}
	legacyKeys := make([]string, 0)
	for _, session := range sessions {
		artifacts, err := s.store.ListArtifacts(session.ID)
		if err != nil {
			return err
		}
		for _, artifact := range artifacts {
			legacyKeys = append(legacyKeys, session.ID+"/"+artifact.ID)
			runID := strings.TrimSpace(artifact.RunID)
			if runID == "" && artifact.Attribution != nil {
				runID = strings.TrimSpace(artifact.Attribution.RunID)
			}
			if runID == "" {
				runID = "legacy-session:" + session.ID
			}
			source := "legacy-session-artifact"
			confidence, verification := evidence.ConfidenceReported, evidence.VerificationUnverified
			if artifact.Attribution != nil && artifact.Attribution.Type == AttributionAgent && runID != "" {
				source, verification = "swarm-manager.session-artifact", evidence.VerificationVerified
				if strings.TrimSpace(artifact.MutationSource) != "" {
					confidence = evidence.ConfidenceAuthoritative
				}
			}
			if _, err := s.evidenceService.IngestForOwner(ctx, evidence.Owner{Kind: evidence.OwnerAgentSession, ID: session.ID}, artifactObservation(source, artifact, runID, confidence, verification)); err != nil {
				return err
			}
		}
	}
	projectedKeys, err := s.projectedArtifactKeys(ctx, sessions, legacyKeys)
	if err != nil {
		return err
	}
	sourceDigest := artifactKeyDigest(legacyKeys)
	projectedDigest := artifactKeyDigest(projectedKeys)
	if len(legacyKeys) != len(projectedKeys) || sourceDigest != projectedDigest {
		return fmt.Errorf("session artifact migration parity failed: source=%d projected=%d", len(legacyKeys), len(projectedKeys))
	}
	return s.evidenceService.SaveMigrationAudit(ctx, evidence.MigrationAudit{MigrationKey: sessionArtifactMigrationKey, SourceCount: len(legacyKeys), ProjectedCount: len(projectedKeys), SourceDigest: sourceDigest, ProjectedDigest: projectedDigest})
}

func (s *Service) projectedArtifactKeys(ctx context.Context, sessions []Session, legacyKeys []string) ([]string, error) {
	expected := make(map[string]struct{}, len(legacyKeys))
	for _, key := range legacyKeys {
		expected[key] = struct{}{}
	}
	projected := make([]string, 0, len(legacyKeys))
	for _, session := range sessions {
		records, err := s.evidenceService.ListByOwner(ctx, evidence.Owner{Kind: evidence.OwnerAgentSession, ID: session.ID})
		if err != nil {
			return nil, err
		}
		for _, record := range records {
			artifactID := strings.TrimSpace(record.Observation.Metadata["artifact_id"])
			key := session.ID + "/" + artifactID
			if _, ok := expected[key]; ok {
				projected = append(projected, key)
			}
		}
	}
	return projected, nil
}

func artifactKeyDigest(keys []string) string {
	sorted := append([]string(nil), keys...)
	sort.Strings(sorted)
	hash := sha256.Sum256([]byte(strings.Join(sorted, "\n")))
	return fmt.Sprintf("sha256:%x", hash[:])
}

func (s *Service) ListArtifacts(ctx context.Context, sessionID string) ([]Artifact, error) {
	if s.evidenceProjection && s.evidenceService != nil {
		records, err := s.evidenceService.ListByOwner(ctx, evidence.Owner{Kind: evidence.OwnerAgentSession, ID: strings.TrimSpace(sessionID)})
		if err != nil {
			return nil, err
		}
		artifacts := make([]Artifact, 0, len(records))
		for _, record := range records {
			if record.Observation.SourceSystem != "legacy-session-artifact" && record.Observation.SourceSystem != "swarm-manager.session-artifact" {
				continue
			}
			artifacts = append(artifacts, artifactFromEvidence(record))
		}
		return artifacts, nil
	}
	artifacts, err := s.store.ListArtifacts(strings.TrimSpace(sessionID))
	if err != nil {
		return nil, mapStoreError(err)
	}
	return artifacts, nil
}

func artifactFromEvidence(record evidence.Record) Artifact {
	metadata := record.Observation.Metadata
	artifact := Artifact{ID: metadata["artifact_id"], SessionID: metadata["session_id"], ArtifactType: ArtifactType(record.Observation.Subject.Kind), Action: ArtifactAction(record.Observation.Action), EntityRef: record.Observation.Subject.ID, Title: metadata["title"], ProposalID: metadata["proposal_id"], ActivityID: metadata["activity_id"], RunID: record.Observation.RunID, MutationSource: metadata["mutation_source"], CreatedAt: record.Observation.ObservedAt.UTC().Format(time.RFC3339Nano)}
	if artifact.RunID == "legacy-session:"+artifact.SessionID || artifact.RunID == "session-artifact:"+artifact.SessionID {
		artifact.RunID = ""
	}
	if kind := metadata["attribution_type"]; kind != "" {
		artifact.Attribution = &Attribution{Type: AttributionType(kind), RunID: artifact.RunID, TaskID: metadata["attribution_task"], ProfileKey: metadata["attribution_profile"]}
	}
	return artifact
}

func (s *Service) ListArtifactsByEntity(ctx context.Context, artifactType ArtifactType, entityRef string) ([]Artifact, error) {
	if !IsKnownArtifactType(artifactType) {
		return nil, apierr.BadRequest("artifact_type is invalid")
	}
	if strings.TrimSpace(entityRef) == "" {
		return nil, apierr.BadRequest("entity_ref is required")
	}
	if !s.evidenceProjection || s.evidenceService == nil {
		return s.store.ListArtifactsByEntity(artifactType, entityRef)
	}
	sessions, err := s.List(ctx, ListFilters{})
	if err != nil {
		return nil, err
	}
	artifacts := []Artifact{}
	for _, session := range sessions {
		for _, artifact := range session.Artifacts {
			if artifact.ArtifactType == artifactType && strings.TrimSpace(artifact.EntityRef) == strings.TrimSpace(entityRef) {
				artifacts = append(artifacts, artifact)
			}
		}
	}
	return artifacts, nil
}
