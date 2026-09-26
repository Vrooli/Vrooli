package corpus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	documentpb "github.com/vrooli/vrooli/packages/proto/gen/go/document-manager/v1/shared"
)

type Collection struct {
	ID                  string
	Name                string
	DefaultPrivacyClass documentpb.PrivacyClass
	Federated           bool
	CreatedAt           time.Time
}

type Membership struct {
	CollectionID string
	DocumentHash string
	PrivacyClass documentpb.PrivacyClass
	CreatedAt    time.Time
}

type Archive struct {
	SchemaVersion int               `json:"schema_version"`
	Format        string            `json:"format"`
	ExportedAt    time.Time         `json:"exported_at"`
	Collection    Collection        `json:"collection"`
	Documents     []Membership      `json:"documents"`
	Artifacts     []ArchiveArtifact `json:"artifacts"`
	AnchorURIs    []string          `json:"anchor_uris"`
	Custody       []json.RawMessage `json:"custody"`
}

type ArchiveArtifact struct {
	Kind        string `json:"kind"`
	ContentHash string `json:"content_hash"`
	BytesBase64 string `json:"bytes_base64,omitempty"`
}

type Repository interface {
	CreateCollection(context.Context, Collection) error
	GetCollection(context.Context, string) (Collection, error)
	ListCollections(context.Context, int) ([]Collection, error)
	AddDocument(context.Context, Membership) error
	ListDocuments(context.Context, string, int) ([]Membership, error)
	ListAnchors(context.Context, string) ([]string, error)
	AddAnchor(context.Context, string, string) error
	CanRead(context.Context, string, documentpb.PrivacyClass) (bool, error)
}

func EncodeArchive(a Archive) ([]byte, error) {
	if a.SchemaVersion == 0 {
		a.SchemaVersion = 1
	}
	if a.Format == "" {
		a.Format = "vrooli-document-corpus+json;version=1"
	}
	return json.MarshalIndent(a, "", "  ")
}

func DecodeArchive(data []byte) (Archive, error) {
	var a Archive
	if err := json.Unmarshal(data, &a); err != nil {
		return Archive{}, fmt.Errorf("decode corpus archive: %w", err)
	}
	if a.SchemaVersion != 1 || a.Format != "vrooli-document-corpus+json;version=1" {
		return Archive{}, fmt.Errorf("unsupported corpus archive format")
	}
	return a, nil
}

func ValidatePrivacyInheritance(collection, document documentpb.PrivacyClass) error {
	if collection == documentpb.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED || document == documentpb.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED {
		return fmt.Errorf("privacy class is required")
	}
	if document < collection {
		return fmt.Errorf("document privacy class %s is less restrictive than collection default %s", document, collection)
	}
	return nil
}
