// Package intake owns restartable admission of externally authored structured
// articles. It deliberately stores the submitted JSON verbatim alongside the
// interpreted fields so unsupported or rejected source data remains recoverable.
package intake

import (
	"context"
	"encoding/json"
	"time"

	"content-desk/internal/artifacts"
)

type Source struct {
	ArtifactID string `json:"artifact_id"`
	RevisionID string `json:"revision_id"`
	Producer   string `json:"producer"`
	Intent     string `json:"import_intent"`
}

// Variant is an independently useful derivative of the source article.
// RawFields carries producer-specific fields that this version does not know.
type Variant struct {
	ID        string                     `json:"id"`
	Name      string                     `json:"name"`
	Channel   string                     `json:"channel"`
	Title     string                     `json:"title,omitempty"`
	Body      string                     `json:"body"`
	RawFields map[string]json.RawMessage `json:"raw_fields,omitempty"`
}

// Item keeps both the structured interpretation and original source payload.
// RejectedFields and UnresolvedFields are explicit caller/system annotations;
// they are never used to remove data from RawPayload.
type Item struct {
	ExternalID      string                     `json:"external_id"`
	Kind            string                     `json:"kind"`
	Title           string                     `json:"title"`
	Body            string                     `json:"body"`
	SourceURL       string                     `json:"source_url,omitempty"`
	MediaPlacements []json.RawMessage          `json:"media_placements,omitempty"`
	Variants        []Variant                  `json:"variants,omitempty"`
	RawFields       map[string]json.RawMessage `json:"raw_fields,omitempty"`
	RejectedFields  []string                   `json:"rejected_fields,omitempty"`
	RawPayload      json.RawMessage            `json:"-"`
}

// UnmarshalJSON keeps producer-specific top-level fields byte-for-byte in the
// raw payload while still exposing the known structured fields to admission.
func (i *Item) UnmarshalJSON(data []byte) error {
	type itemAlias Item
	var decoded itemAlias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*i = Item(decoded)
	i.RawPayload = append(json.RawMessage(nil), data...)
	return nil
}

type Request struct {
	Source         Source `json:"source"`
	IdempotencyKey string `json:"idempotency_key"`
	Items          []Item `json:"items"`
}

type Assessment struct {
	ExternalID       string   `json:"external_id"`
	Status           string   `json:"status"` // admissible, unresolved, rejected
	UnresolvedFields []string `json:"unresolved_fields,omitempty"`
	RejectedFields   []string `json:"rejected_fields,omitempty"`
	Reason           string   `json:"reason,omitempty"`
}

type Preview struct {
	Source      Source       `json:"source"`
	Assessments []Assessment `json:"assessments"`
}

type Receipt struct {
	ID               string          `json:"id"`
	BatchID          string          `json:"batch_id"`
	ExternalID       string          `json:"external_id"`
	Attempt          int             `json:"attempt"`
	Status           string          `json:"status"`
	UnresolvedFields []string        `json:"unresolved_fields,omitempty"`
	RejectedFields   []string        `json:"rejected_fields,omitempty"`
	Reason           string          `json:"reason,omitempty"`
	PayloadSHA256    string          `json:"payload_sha256"`
	RawPayload       json.RawMessage `json:"raw_payload"`
	CreatedAt        time.Time       `json:"created_at"`
}

type StoredItem struct {
	BatchID    string     `json:"batch_id"`
	Source     Source     `json:"source"`
	Item       Item       `json:"item"`
	Assessment Assessment `json:"assessment"`
	Receipt    Receipt    `json:"receipt"`
	Receipts   []Receipt  `json:"receipts"`
}

// PromoteRequest selects one immutable admitted item receipt and the exact
// named variants to materialize as ordinary, unapproved Content Desk drafts.
type PromoteRequest struct {
	BatchID           string   `json:"batch_id"`
	ExternalID        string   `json:"external_id"`
	ExpectedReceiptID string   `json:"expected_receipt_id"`
	VariantIDs        []string `json:"variant_ids,omitempty"`
	Channel           string   `json:"channel,omitempty"` // required for the source body when no variants exist
	Format            string   `json:"format"`
	PostTypeID        string   `json:"post_type_id"`
	CampaignID        string   `json:"campaign_id,omitempty"`
	ProjectID         string   `json:"project_id,omitempty"`
}

// PromotionRecord provides source and receipt lineage for each created draft.
type PromotionRecord struct {
	DraftID           string    `json:"draft_id"`
	BatchID           string    `json:"batch_id"`
	ExternalID        string    `json:"external_id"`
	ReceiptID         string    `json:"receipt_id"`
	SourceArtifactID  string    `json:"source_artifact_id"`
	SourceRevisionID  string    `json:"source_revision_id"`
	VariantID         string    `json:"variant_id"`
	VariantName       string    `json:"variant_name"`
	SourcePayloadHash string    `json:"source_payload_sha256"`
	CreatedAt         time.Time `json:"created_at"`
}

type PromotedDraft struct {
	Draft  artifacts.Draft `json:"draft"`
	Source PromotionRecord `json:"source"`
}

type Service interface {
	Preview(context.Context, Request) (Preview, error)
	Admit(context.Context, Request) ([]StoredItem, error)
	List(context.Context, string) ([]StoredItem, error)
	Reconcile(context.Context, string, Request) ([]StoredItem, error)
	Promote(context.Context, PromoteRequest) ([]PromotedDraft, error)
}
