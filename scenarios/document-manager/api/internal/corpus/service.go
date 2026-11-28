package corpus

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

type Service struct{ Repo Repository }

func NewService(repo Repository) Service { return Service{Repo: repo} }

func (s Service) CreateCollection(ctx context.Context, name string, class documentpb.PrivacyClass, federated bool) (Collection, error) {
	if strings.TrimSpace(name) == "" {
		return Collection{}, fmt.Errorf("collection name is required")
	}
	if class == documentpb.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED {
		class = documentpb.PrivacyClass_PRIVACY_CLASS_INTERNAL
	}
	c := Collection{ID: uuid.NewString(), Name: strings.TrimSpace(name), DefaultPrivacyClass: class, Federated: federated, CreatedAt: time.Now().UTC()}
	if err := s.Repo.CreateCollection(ctx, c); err != nil {
		return Collection{}, err
	}
	return c, nil
}

func (s Service) GetCollection(ctx context.Context, id string) (Collection, error) {
	return s.Repo.GetCollection(ctx, id)
}

func (s Service) ListCollections(ctx context.Context, limit int) ([]Collection, error) {
	return s.Repo.ListCollections(ctx, limit)
}

func (s Service) AddDocument(ctx context.Context, collectionID, hash string, class documentpb.PrivacyClass) (Membership, error) {
	if strings.TrimSpace(hash) == "" {
		return Membership{}, fmt.Errorf("document hash is required")
	}
	c, err := s.Repo.GetCollection(ctx, collectionID)
	if err != nil {
		return Membership{}, err
	}
	if class == documentpb.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED {
		class = c.DefaultPrivacyClass
	}
	m := Membership{CollectionID: collectionID, DocumentHash: strings.ToLower(strings.TrimSpace(hash)), PrivacyClass: class, CreatedAt: time.Now().UTC()}
	if err := s.Repo.AddDocument(ctx, m); err != nil {
		return Membership{}, err
	}
	return m, nil
}

func (s Service) ListDocuments(ctx context.Context, collectionID string, limit int) ([]Membership, error) {
	return s.Repo.ListDocuments(ctx, collectionID, limit)
}

func (s Service) Export(ctx context.Context, collectionID string) ([]byte, error) {
	c, err := s.Repo.GetCollection(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	docs, err := s.Repo.ListDocuments(ctx, collectionID, 0)
	if err != nil {
		return nil, err
	}
	anchors, err := s.Repo.ListAnchors(ctx, collectionID)
	if err != nil {
		return nil, err
	}
	return EncodeArchive(Archive{SchemaVersion: 1, Format: "vrooli-document-corpus+json;version=1", ExportedAt: time.Now().UTC(), Collection: c, Documents: docs, Artifacts: []ArchiveArtifact{}, AnchorURIs: anchors, Custody: []json.RawMessage{}})
}

func (s Service) Import(ctx context.Context, data []byte) (Collection, int, error) {
	a, err := DecodeArchive(data)
	if err != nil {
		return Collection{}, 0, err
	}
	if a.Collection.ID == "" {
		a.Collection.ID = uuid.NewString()
	}
	if err := s.Repo.CreateCollection(ctx, a.Collection); err != nil {
		return Collection{}, 0, err
	}
	for _, document := range a.Documents {
		document.CollectionID = a.Collection.ID
		if err := s.Repo.AddDocument(ctx, document); err != nil {
			return Collection{}, 0, err
		}
	}
	for _, uri := range a.AnchorURIs {
		if err := s.Repo.AddAnchor(ctx, a.Collection.ID, uri); err != nil {
			return Collection{}, 0, err
		}
	}
	return a.Collection, len(a.Documents), nil
}

func (s Service) Prune(_ context.Context, dryRun bool) (bool, []string, int64) {
	// Raw bytes and custody are intentionally not prune candidates. Derivation
	// and index eviction are delegated to their owning stores; this operation
	// reports the safe no-op until those stores are registered.
	return dryRun, []string{}, 0
}
