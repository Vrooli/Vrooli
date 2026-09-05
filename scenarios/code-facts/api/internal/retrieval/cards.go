package retrieval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

type CardPolicy struct {
	Version       string
	Model         string
	Dimensions    int
	MaxTextBytes  int
	EligibleRoles map[string]bool
	EligibleKinds map[string]bool
	Storage       string
}

func DefaultCardPolicy() CardPolicy {
	return CardPolicy{
		Version: "code-card-v1", Model: "nomic-embed-text:latest", Dimensions: 768,
		MaxTextBytes: 4096, Storage: "qdrant-on-disk-scalar-quantized",
		EligibleRoles: map[string]bool{"implementation": true, "contract": true, "generated_alias": true},
		EligibleKinds: map[string]bool{"file": true, "symbol": true, "contract": true, "route": true, "persistence": true, "relationship": true},
	}
}

type Card struct {
	ID            string
	DocumentID    string
	SourceFileID  string
	SourceHash    string
	Generation    string
	Kind          string
	Role          string
	Scope         string
	Path          string
	DisplayText   string
	EmbeddingText string
	EmbeddingHash string
	Policy        string
	Model         string
	Dimensions    int
}

type CardStats struct {
	Total              int
	ByRole             map[string]int
	ByKind             map[string]int
	ByLanguage         map[string]int
	ByScope            map[string]int
	EstimatedBytes     int64
	RejectedIneligible int
}

type CardExtractor struct{ Policy CardPolicy }

func (extractor CardExtractor) Extract(generation string, documents []Document) ([]Card, CardStats, error) {
	policy := extractor.Policy
	if strings.TrimSpace(policy.Version) == "" || strings.TrimSpace(policy.Model) == "" || policy.Dimensions <= 0 || policy.MaxTextBytes <= 0 {
		return nil, CardStats{}, fmt.Errorf("card policy requires version, model, dimensions, and max text bytes")
	}
	stats := CardStats{ByRole: map[string]int{}, ByKind: map[string]int{}, ByLanguage: map[string]int{}, ByScope: map[string]int{}}
	cards := make([]Card, 0, len(documents))
	for _, document := range documents {
		if !policy.EligibleRoles[document.Role] || !policy.EligibleKinds[document.Kind] {
			stats.RejectedIneligible++
			continue
		}
		display := boundedText(document.Body, policy.MaxTextBytes)
		embedding := boundedText(strings.Join([]string{
			"kind: " + document.Kind,
			"title: " + document.Title,
			"path: " + document.Path,
			"scope: " + document.Scope,
			"contract: " + document.ContractText,
			"aliases: " + strings.Join(document.Aliases, " "),
			"evidence: " + document.Body,
		}, "\n"), policy.MaxTextBytes)
		hash := sha256.Sum256([]byte(strings.Join([]string{policy.Version, policy.Model, document.SourceHash, embedding}, "\x00")))
		card := Card{
			ID: "card:" + document.ID, DocumentID: document.ID, SourceFileID: document.SourceFileID,
			SourceHash: document.SourceHash, Generation: generation, Kind: document.Kind, Role: document.Role,
			Scope: document.Scope, Path: document.Path, DisplayText: display, EmbeddingText: embedding,
			EmbeddingHash: "sha256:" + hex.EncodeToString(hash[:]), Policy: policy.Version,
			Model: policy.Model, Dimensions: policy.Dimensions,
		}
		cards = append(cards, card)
		stats.Total++
		stats.ByRole[document.Role]++
		stats.ByKind[document.Kind]++
		stats.ByLanguage[document.Language]++
		stats.ByScope[document.Scope]++
		stats.EstimatedBytes += int64(policy.Dimensions*4 + len(card.EmbeddingText) + len(card.DisplayText) + 256)
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].ID < cards[j].ID })
	return cards, stats, nil
}

func boundedText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return strings.TrimSpace(value[:limit])
}

type CardIndexer struct {
	Embedder  Embedder
	Store     VectorStore
	Admission Admission
	BatchSize int
}

func (indexer CardIndexer) Index(ctx context.Context, cards []Card) (int, error) {
	if indexer.Embedder == nil || indexer.Store == nil {
		return 0, fmt.Errorf("card indexer requires embedder and vector store")
	}
	batchSize := indexer.BatchSize
	if batchSize <= 0 {
		batchSize = 32
	}
	if batchSize > 128 {
		batchSize = 128
	}
	indexed := 0
	for offset := 0; offset < len(cards); offset += batchSize {
		end := offset + batchSize
		if end > len(cards) {
			end = len(cards)
		}
		batch := cards[offset:end]
		release := func() {}
		if indexer.Admission != nil {
			var err error
			release, err = indexer.Admission.Acquire(ctx, "embedding", len(batch))
			if err != nil {
				return indexed, err
			}
		}
		texts := make([]string, len(batch))
		for index, card := range batch {
			texts[index] = card.EmbeddingText
		}
		vectors, err := indexer.Embedder.Embed(ctx, texts)
		release()
		if err != nil {
			return indexed, fmt.Errorf("embed card batch: %w", err)
		}
		if len(vectors) != len(batch) {
			return indexed, fmt.Errorf("embedder returned %d vectors for %d cards", len(vectors), len(batch))
		}
		records := make([]VectorRecord, len(batch))
		for index, card := range batch {
			if len(vectors[index]) != card.Dimensions {
				return indexed, fmt.Errorf("card %q vector dimensions %d, want %d", card.ID, len(vectors[index]), card.Dimensions)
			}
			records[index] = VectorRecord{
				ID: card.ID, Vector: vectors[index], SourceHash: card.SourceHash, Generation: card.Generation,
				Payload: map[string]string{"document_id": card.DocumentID, "path": card.Path, "role": card.Role, "scope": card.Scope, "kind": card.Kind, "embedding_hash": card.EmbeddingHash, "policy": card.Policy, "model": card.Model, "display_text": card.DisplayText},
			}
		}
		if err := indexer.Store.Upsert(ctx, records); err != nil {
			return indexed, fmt.Errorf("upsert card batch: %w", err)
		}
		indexed += len(batch)
	}
	return indexed, nil
}
