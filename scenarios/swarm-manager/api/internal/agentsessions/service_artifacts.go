package agentsessions

import (
	"context"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/idgen"
)

func (s *Service) AttachArtifact(_ context.Context, artifact Artifact) (Artifact, error) {
	if artifact.ID == "" {
		artifact.ID = "art_" + idgen.Generate()
	}
	if artifact.CreatedAt == "" {
		artifact.CreatedAt = nowRFC3339()
	}
	if err := s.store.AppendArtifact(artifact.SessionID, artifact); err != nil {
		return Artifact{}, err
	}
	s.emitArtifactLinked(artifact)
	return artifact, nil
}

func (s *Service) AttachArtifacts(_ context.Context, artifacts []Artifact) ([]Artifact, error) {
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
	}
	if err := s.store.AppendArtifacts(artifacts[0].SessionID, artifacts); err != nil {
		return nil, err
	}
	for _, artifact := range artifacts {
		s.emitArtifactLinked(artifact)
	}
	return artifacts, nil
}

func (s *Service) ListArtifacts(_ context.Context, sessionID string) ([]Artifact, error) {
	artifacts, err := s.store.ListArtifacts(strings.TrimSpace(sessionID))
	if err != nil {
		return nil, mapStoreError(err)
	}
	return artifacts, nil
}

func (s *Service) ListArtifactsByEntity(_ context.Context, artifactType ArtifactType, entityRef string) ([]Artifact, error) {
	if !IsKnownArtifactType(artifactType) {
		return nil, apierr.BadRequest("artifact_type is invalid")
	}
	if strings.TrimSpace(entityRef) == "" {
		return nil, apierr.BadRequest("entity_ref is required")
	}
	return s.store.ListArtifactsByEntity(artifactType, entityRef)
}
