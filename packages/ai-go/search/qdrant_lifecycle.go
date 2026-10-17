package aisearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// GenerationLifecycleState is the durable state of a physical generation.
// Unknown collections are intentionally represented as quarantined: absence
// of catalog evidence is not permission to delete data.
type GenerationLifecycleState string

const (
	GenerationStateBuilding    GenerationLifecycleState = "building"
	GenerationStateCandidate   GenerationLifecycleState = "candidate"
	GenerationStateActive      GenerationLifecycleState = "active"
	GenerationStateRetired     GenerationLifecycleState = "retired"
	GenerationStateProtected   GenerationLifecycleState = "protected"
	GenerationStateExpired     GenerationLifecycleState = "expired"
	GenerationStateFailed      GenerationLifecycleState = "failed"
	GenerationStateQuarantined GenerationLifecycleState = "quarantined"
	GenerationStateDeleted     GenerationLifecycleState = "deleted"
)

// ValidGenerationTransition is kept small and explicit so catalog adapters
// cannot silently turn a failed or unknown collection into active data.
func ValidGenerationTransition(from, to GenerationLifecycleState) bool {
	if from == to {
		return true
	}
	switch from {
	case GenerationStateBuilding:
		return to == GenerationStateCandidate || to == GenerationStateFailed
	case GenerationStateCandidate:
		return to == GenerationStateActive || to == GenerationStateFailed || to == GenerationStateRetired
	case GenerationStateActive:
		return to == GenerationStateRetired || to == GenerationStateProtected
	case GenerationStateRetired:
		return to == GenerationStateExpired || to == GenerationStateProtected || to == GenerationStateDeleted
	case GenerationStateFailed:
		return to == GenerationStateExpired || to == GenerationStateDeleted
	case GenerationStateExpired:
		return to == GenerationStateDeleted
	case GenerationStateProtected:
		return to == GenerationStateRetired || to == GenerationStateDeleted
	case GenerationStateQuarantined:
		return to == GenerationStateProtected || to == GenerationStateRetired
	case GenerationStateDeleted:
		return false
	default:
		return false
	}
}

type GenerationLease struct {
	ID        string    `json:"id,omitempty"`
	Holder    string    `json:"holder,omitempty"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
}

func (l GenerationLease) Active(now time.Time) bool {
	return strings.TrimSpace(l.ID) != "" && l.ExpiresAt.After(now)
}

// GenerationRecord is the catalog projection owned by the scenario using the
// shared Qdrant substrate. Bytes are optional because Qdrant's collection API
// exposes logical counts, not physical filesystem allocation.
type GenerationRecord struct {
	Metadata       GenerationMetadata       `json:"metadata"`
	CollectionName string                   `json:"collectionName"`
	State          GenerationLifecycleState `json:"state"`
	Lease          GenerationLease          `json:"lease,omitempty"`
	Points         int                      `json:"points,omitempty"`
	Bytes          int64                    `json:"bytes,omitempty"`
	CleanupOutcome string                   `json:"cleanupOutcome,omitempty"`
	UpdatedAt      time.Time                `json:"updatedAt"`
}

// GenerationCatalog is deliberately an adapter seam. Durable ownership stays
// beside the domain schema; the shared package only defines the contract.
type GenerationCatalog interface {
	ListGenerationRecords(context.Context, string, string) ([]GenerationRecord, error)
	SaveGenerationRecord(context.Context, GenerationRecord) error
}

type GenerationCleanupReceiptStore interface {
	LoadGenerationCleanupReceipt(context.Context, string) (GenerationCleanupReceipt, bool, error)
	SaveGenerationCleanupReceipt(context.Context, GenerationCleanupReceipt) error
}

type GenerationRetentionPolicy struct {
	PolicyRevision     string        `json:"policyRevision"`
	KeepActive         int           `json:"keepActive"`
	KeepRollback       int           `json:"keepRollback"`
	MaxGenerationCount int           `json:"maxGenerationCount"`
	RetiredMaxAge      time.Duration `json:"retiredMaxAge"`
	FailedMaxAge       time.Duration `json:"failedMaxAge"`
	MaxPhysicalBytes   int64         `json:"maxPhysicalBytes"`
	UnknownState       string        `json:"unknownState"`
	DeletionBatchSize  int           `json:"deletionBatchSize"`
}

func DefaultGenerationRetentionPolicy() GenerationRetentionPolicy {
	return GenerationRetentionPolicy{
		PolicyRevision: "agent-manager-qdrant-v1", KeepActive: 1, KeepRollback: 2,
		MaxGenerationCount: 6, RetiredMaxAge: 72 * time.Hour, FailedMaxAge: time.Hour,
		MaxPhysicalBytes: 8 * 1024 * 1024 * 1024, UnknownState: "quarantine", DeletionBatchSize: 2,
	}
}

type GenerationInspection struct {
	Owner              string             `json:"owner"`
	Namespace          string             `json:"namespace"`
	Alias              string             `json:"alias"`
	ActiveCollection   string             `json:"activeCollection"`
	SnapshotIdentity   string             `json:"snapshotIdentity"`
	ObservedAt         time.Time          `json:"observedAt"`
	Generations        []GenerationRecord `json:"generations"`
	Protected          []string           `json:"protected"`
	Quarantined        []string           `json:"quarantined"`
	Eligible           []string           `json:"eligible"`
	LogicalBytes       int64              `json:"logicalBytes"`
	PhysicalBytesKnown bool               `json:"physicalBytesKnown"`
}

type GenerationCleanupCandidate struct {
	GenerationID   string                   `json:"generationId"`
	CollectionName string                   `json:"collectionName"`
	State          GenerationLifecycleState `json:"state"`
	Bytes          int64                    `json:"bytes,omitempty"`
	Reason         string                   `json:"reason"`
}

type GenerationCleanupPlan struct {
	PlanIdentity     string                       `json:"planIdentity"`
	IdempotencyKey   string                       `json:"idempotencyKey"`
	PolicyRevision   string                       `json:"policyRevision"`
	Policy           GenerationRetentionPolicy    `json:"policy"`
	SnapshotIdentity string                       `json:"snapshotIdentity"`
	Namespace        string                       `json:"namespace"`
	Alias            string                       `json:"alias"`
	Candidates       []GenerationCleanupCandidate `json:"candidates"`
	Protected        []string                     `json:"protected"`
	Quarantined      []string                     `json:"quarantined"`
	Approved         bool                         `json:"approved"`
	CreatedAt        time.Time                    `json:"createdAt"`
}

type GenerationCleanupReceipt struct {
	PlanIdentity           string    `json:"planIdentity"`
	IdempotencyKey         string    `json:"idempotencyKey"`
	PolicyRevision         string    `json:"policyRevision"`
	ActiveCollection       string    `json:"activeCollection"`
	DeletedCollections     []string  `json:"deletedCollections"`
	SkippedCollections     []string  `json:"skippedCollections"`
	QuarantinedCollections []string  `json:"quarantinedCollections"`
	LogicalReclaimedBytes  int64     `json:"logicalReclaimedBytes"`
	PhysicalReclaimedBytes int64     `json:"physicalReclaimedBytes"`
	PhysicalBytesKnown     bool      `json:"physicalBytesKnown"`
	CompletedAt            time.Time `json:"completedAt"`
}

type GenerationLifecycleMetrics struct {
	GenerationCount  int       `json:"generationCount"`
	RetiredCount     int       `json:"retiredCount"`
	RetiredBytes     int64     `json:"retiredBytes"`
	OldestGeneration time.Time `json:"oldestGeneration,omitempty"`
	CleanupErrors    uint64    `json:"cleanupErrors"`
	QuarantineCount  int       `json:"quarantineCount"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type GenerationLifecycleOwner interface {
	InspectGenerationLifecycle(context.Context, GenerationRetentionPolicy) (GenerationInspection, error)
	PreviewGenerationCleanup(context.Context, GenerationRetentionPolicy, string) (GenerationCleanupPlan, error)
	ApplyGenerationCleanup(context.Context, GenerationCleanupPlan) (GenerationCleanupReceipt, error)
}

func cleanupSnapshotIdentity(namespace, alias, revision string, records []GenerationRecord) string {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.CollectionName+":"+string(record.State)+":"+record.UpdatedAt.UTC().Format(time.RFC3339Nano))
	}
	sort.Strings(ids)
	sum := sha256.Sum256([]byte(strings.Join([]string{namespace, alias, revision, strings.Join(ids, "|")}, "|")))
	return hex.EncodeToString(sum[:])
}

func validateCleanupPolicy(policy GenerationRetentionPolicy) error {
	if strings.TrimSpace(policy.PolicyRevision) == "" {
		return errors.New("generation cleanup policy revision is required")
	}
	if policy.KeepActive < 1 || policy.KeepRollback < 0 || policy.MaxGenerationCount < policy.KeepActive+policy.KeepRollback {
		return fmt.Errorf("invalid generation retention counts")
	}
	if policy.RetiredMaxAge <= 0 || policy.FailedMaxAge <= 0 || policy.DeletionBatchSize <= 0 {
		return errors.New("generation cleanup ages and batch size must be positive")
	}
	if policy.UnknownState != "quarantine" {
		return fmt.Errorf("unknown generation state action must be quarantine")
	}
	return nil
}

func lifecycleFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func generationIDFromCollection(collection string) string {
	marker := generationNameSeparator
	if index := strings.Index(collection, marker); index >= 0 {
		return collection[index+len(marker):]
	}
	return collection
}
