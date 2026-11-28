package intake

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"content-desk/internal/artifacts"
	"github.com/google/uuid"
)

type selectedVariant struct {
	id, name, channel, body string
}

// Promote materializes only the named variants of the exact admitted item and
// current receipt. It creates requested drafts and immutable lineage rows;
// it never runs a lifecycle transition, review, approval or release operation.
func (s *service) Promote(ctx context.Context, req PromoteRequest) ([]PromotedDraft, error) {
	if s.drafts == nil {
		return nil, errors.New("draft creation is not configured for article intake")
	}
	req.BatchID = strings.TrimSpace(req.BatchID)
	req.ExternalID = strings.TrimSpace(req.ExternalID)
	req.ExpectedReceiptID = strings.TrimSpace(req.ExpectedReceiptID)
	req.Channel = strings.TrimSpace(req.Channel)
	req.Format = strings.TrimSpace(req.Format)
	req.PostTypeID = strings.TrimSpace(req.PostTypeID)
	if req.BatchID == "" || req.ExternalID == "" || req.ExpectedReceiptID == "" {
		return nil, errors.New("batch id, external id, and expected receipt id are required")
	}
	if req.Format == "" || req.PostTypeID == "" {
		return nil, errors.New("format and post type id are required")
	}
	items, err := s.List(ctx, req.BatchID)
	if err != nil {
		return nil, err
	}
	var selected *StoredItem
	for i := range items {
		if items[i].Item.ExternalID == req.ExternalID {
			selected = &items[i]
			break
		}
	}
	if selected == nil {
		return nil, errors.New("selected article item is not in the admitted batch")
	}
	if selected.Assessment.Status != "admissible" {
		return nil, fmt.Errorf("article item status %q cannot be promoted", selected.Assessment.Status)
	}
	if selected.Receipt.ID != req.ExpectedReceiptID {
		return nil, errors.New("expected receipt does not match the selected item's current admitted receipt")
	}
	if selected.Receipt.Status != "admissible" {
		return nil, errors.New("selected receipt is not admissible")
	}

	variants := make([]selectedVariant, 0, len(selected.Item.Variants))
	if len(selected.Item.Variants) == 0 {
		if len(req.VariantIDs) != 0 || req.Channel == "" {
			return nil, errors.New("a source-only article requires a channel and no variant ids")
		}
		variants = append(variants, selectedVariant{id: "source", name: "Source", channel: req.Channel, body: selected.Item.Body})
	} else {
		if req.Channel != "" || len(req.VariantIDs) == 0 {
			return nil, errors.New("items with variants require explicit variant ids and do not accept a source channel")
		}
		byID := make(map[string]Variant, len(selected.Item.Variants))
		for _, v := range selected.Item.Variants {
			if strings.TrimSpace(v.ID) == "" || strings.TrimSpace(v.Name) == "" || strings.TrimSpace(v.Channel) == "" || strings.TrimSpace(v.Body) == "" {
				return nil, errors.New("every promotable variant requires id, name, channel and body")
			}
			if _, duplicate := byID[v.ID]; duplicate {
				return nil, fmt.Errorf("admitted item contains duplicate variant id %q", v.ID)
			}
			byID[v.ID] = v
		}
		seen := map[string]bool{}
		for _, id := range req.VariantIDs {
			id = strings.TrimSpace(id)
			if id == "" || seen[id] {
				return nil, errors.New("variant ids must be non-empty and unique")
			}
			seen[id] = true
			v, ok := byID[id]
			if !ok {
				return nil, fmt.Errorf("variant %q is not named on the admitted item", id)
			}
			variants = append(variants, selectedVariant{id: v.ID, name: v.Name, channel: v.Channel, body: v.Body})
		}
	}

	// Stable order makes retries and returned records deterministic.
	sort.Slice(variants, func(i, j int) bool { return variants[i].id < variants[j].id })
	config, err := json.Marshal(struct {
		Format, PostTypeID, CampaignID, ProjectID string
	}{req.Format, req.PostTypeID, req.CampaignID, req.ProjectID})
	if err != nil {
		return nil, err
	}
	configHashBytes := sha256.Sum256(config)
	configHash := hex.EncodeToString(configHashBytes[:])
	now := s.now().UTC()
	created := make([]PromotedDraft, 0, len(variants))

	for _, variant := range variants {
		var prior PromotionRecord
		var createdAt string
		err := s.db.QueryRowContext(ctx, `SELECT draft_id,batch_id,external_id,receipt_id,source_artifact_id,source_revision_id,variant_id,variant_name,source_payload_sha256,config_sha256,created_at FROM article_intake_promotions WHERE batch_id=? AND external_id=? AND variant_id=?`, req.BatchID, req.ExternalID, variant.id).Scan(&prior.DraftID, &prior.BatchID, &prior.ExternalID, &prior.ReceiptID, &prior.SourceArtifactID, &prior.SourceRevisionID, &prior.VariantID, &prior.VariantName, &prior.SourcePayloadHash, &configHash, &createdAt)
		if err == nil {
			if configHash != hex.EncodeToString(configHashBytes[:]) || prior.ReceiptID != req.ExpectedReceiptID {
				return nil, fmt.Errorf("variant %q was already promoted with different settings or receipt", variant.id)
			}
			prior.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
			if err != nil {
				return nil, err
			}
			draft, err := s.drafts.Get(ctx, prior.DraftID)
			if err != nil {
				return nil, err
			}
			created = append(created, PromotedDraft{Draft: draft, Source: prior})
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		draft, err := s.drafts.Create(ctx, artifacts.Draft{
			ID: uuid.NewString(), CampaignID: req.CampaignID, ProjectID: req.ProjectID,
			PostTypeID: req.PostTypeID, Body: variant.body, Channel: variant.channel,
			Format: req.Format, Status: artifacts.DraftRequested,
		})
		if err != nil {
			return nil, fmt.Errorf("create requested draft for variant %q: %w", variant.id, err)
		}
		record := PromotionRecord{
			DraftID: draft.ID, BatchID: req.BatchID, ExternalID: req.ExternalID,
			ReceiptID: selected.Receipt.ID, SourceArtifactID: selected.Source.ArtifactID,
			SourceRevisionID: selected.Source.RevisionID, VariantID: variant.id,
			VariantName: variant.name, SourcePayloadHash: selected.Receipt.PayloadSHA256,
			CreatedAt: now,
		}
		_, err = s.db.ExecContext(ctx, `INSERT INTO article_intake_promotions(draft_id,batch_id,external_id,receipt_id,source_artifact_id,source_revision_id,variant_id,variant_name,source_payload_sha256,config_sha256,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, record.DraftID, record.BatchID, record.ExternalID, record.ReceiptID, record.SourceArtifactID, record.SourceRevisionID, record.VariantID, record.VariantName, record.SourcePayloadHash, hex.EncodeToString(configHashBytes[:]), now.Format(time.RFC3339Nano))
		if err != nil {
			return nil, fmt.Errorf("record source lineage for draft %q: %w", draft.ID, err)
		}
		created = append(created, PromotedDraft{Draft: draft, Source: record})
	}
	return created, nil
}
