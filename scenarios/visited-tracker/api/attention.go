package main

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrAttentionInput = errors.New("invalid attention input")
	ErrClaimConflict  = errors.New("claim conflicts with current state")
	ErrClaimExpired   = errors.New("claim expired")
	ErrClaimCapacity  = errors.New("campaign claim history is full; create a new campaign")
)

const (
	// maxActiveClaims bounds live coordination state. Terminal claims move to
	// the bounded archive so a long-running campaign does not become unable to
	// accept work merely because old requests accumulated.
	maxActiveClaims   = 10000
	maxArchivedClaims = 10000
)

// ReviewClaim preserves request identity after expiry and completion. Expiry
// releases selection eligibility, never proof that the subject was reviewed.
type ReviewClaim struct {
	ID          string     `json:"id"`
	RequestID   string     `json:"request_id"`
	Worker      string     `json:"worker"`
	FileID      uuid.UUID  `json:"file_id"`
	FilePath    string     `json:"file_path"`
	Revision    string     `json:"revision"`
	TTLSeconds  int        `json:"ttl_seconds"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Outcome     string     `json:"outcome,omitempty"`
	Evidence    string     `json:"evidence,omitempty"`
	ArchivedAt  *time.Time `json:"archived_at,omitempty"`
}

func claimTerminal(claim ReviewClaim, now time.Time) bool {
	return claim.CompletedAt != nil || !claim.ExpiresAt.After(now)
}

// archiveTerminalClaims keeps replay records while removing expired and
// completed claims from the active selection set. The archive is bounded; a
// replay older than the retained history is treated as a new request.
func archiveTerminalClaims(c *Campaign, now time.Time) {
	active := c.Claims[:0]
	for _, claim := range c.Claims {
		if !claimTerminal(claim, now) {
			active = append(active, claim)
			continue
		}
		if claim.ArchivedAt == nil {
			archivedAt := now.UTC()
			claim.ArchivedAt = &archivedAt
		}
		c.ArchivedClaims = append(c.ArchivedClaims, claim)
	}
	c.Claims = active
	if len(c.ArchivedClaims) <= maxArchivedClaims {
		return
	}
	sort.SliceStable(c.ArchivedClaims, func(i, j int) bool {
		left, right := time.Time{}, time.Time{}
		if c.ArchivedClaims[i].ArchivedAt != nil {
			left = *c.ArchivedClaims[i].ArchivedAt
		}
		if c.ArchivedClaims[j].ArchivedAt != nil {
			right = *c.ArchivedClaims[j].ArchivedAt
		}
		return left.Before(right)
	})
	c.ArchivedClaims = c.ArchivedClaims[len(c.ArchivedClaims)-maxArchivedClaims:]
}

func findClaimByRequest(c *Campaign, requestID string) *ReviewClaim {
	for i := range c.Claims {
		if c.Claims[i].RequestID == requestID {
			return &c.Claims[i]
		}
	}
	for i := range c.ArchivedClaims {
		if c.ArchivedClaims[i].RequestID == requestID {
			return &c.ArchivedClaims[i]
		}
	}
	return nil
}

func priority(file TrackedFile) float64 {
	if file.PriorityWeight <= 0 || math.IsNaN(file.PriorityWeight) || math.IsInf(file.PriorityWeight, 0) {
		return 1
	}
	return math.Min(file.PriorityWeight, 100)
}

// Attention is revision-specific. Lifetime visit counts cannot permanently
// suppress a changed file. The score ranks priority slots; selectAttention
// reserves every fourth claim for the least recently selected eligible file.
func attentionScore(file TrackedFile, now time.Time) float64 {
	if file.Deleted || file.Excluded {
		return 0
	}
	if file.ContentHash == nil || file.ReviewedRevision != *file.ContentHash || file.LastReviewed == nil {
		age := math.Max(0, now.Sub(file.FirstSeen).Hours()/24)
		return priority(file) * (10 + math.Min(age, 365))
	}
	age := math.Max(0, now.Sub(*file.LastReviewed).Hours()/24)
	return priority(file) * (1 + math.Min(age, 365)) / (1 + math.Log1p(float64(max(0, file.ReviewCount))))
}

func selectAttention(c *Campaign, limit int, now time.Time) []TrackedFile {
	held := map[uuid.UUID]bool{}
	for _, claim := range c.Claims {
		if claim.CompletedAt == nil && claim.ExpiresAt.After(now) {
			held[claim.FileID] = true
		}
	}
	files := []TrackedFile{}
	for _, file := range c.TrackedFiles {
		if file.Deleted || file.Excluded || held[file.ID] {
			continue
		}
		file.AttentionScore = attentionScore(file, now)
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].AttentionScore != files[j].AttentionScore {
			return files[i].AttentionScore > files[j].AttentionScore
		}
		if !files[i].FirstSeen.Equal(files[j].FirstSeen) {
			return files[i].FirstSeen.Before(files[j].FirstSeen)
		}
		return files[i].FilePath < files[j].FilePath
	})
	// A persisted exploration slot prevents repeated incomplete high-priority
	// work from starving lower-priority files. Sequence, not wall time or
	// review credit, orders attempts; concurrent claims serialize the cursor.
	// For a fixed eligible set of N files this gives each an attempt within
	// 4*N successful new claims. It does not promise completion or wall time.
	if c.AttentionSequence%4 == 3 && len(files) > 1 {
		oldest := 0
		for i := 1; i < len(files); i++ {
			left, right := files[i], files[oldest]
			if left.LastAttentionSequence < right.LastAttentionSequence ||
				(left.LastAttentionSequence == right.LastAttentionSequence &&
					(left.FirstSeen.Before(right.FirstSeen) || (left.FirstSeen.Equal(right.FirstSeen) && left.FilePath < right.FilePath))) {
				oldest = i
			}
		}
		selected := files[oldest]
		copy(files[1:oldest+1], files[:oldest])
		files[0] = selected
	}
	if limit > 0 && len(files) > limit {
		files = files[:limit]
	}
	return files
}

func activeClaimCount(c *Campaign, now time.Time) int {
	count := 0
	for _, claim := range c.Claims {
		if claim.CompletedAt == nil && claim.ExpiresAt.After(now) {
			count++
		}
	}
	return count
}

func oldestEligibleAge(files []TrackedFile, now time.Time) time.Duration {
	oldest := time.Duration(0)
	for _, file := range files {
		at := file.FirstSeen
		if file.LastReviewed != nil && file.ContentHash != nil && file.ReviewedRevision == *file.ContentHash {
			at = *file.LastReviewed
		}
		age := now.Sub(at)
		if age > oldest {
			oldest = age
		}
	}
	return oldest
}

func oldestActiveClaimAge(c *Campaign, now time.Time) time.Duration {
	oldest := time.Duration(0)
	for _, claim := range c.Claims {
		if claim.CompletedAt != nil || !claim.ExpiresAt.After(now) {
			continue
		}
		age := now.Sub(claim.CreatedAt)
		if age > oldest {
			oldest = age
		}
	}
	return oldest
}

func validIdentity(value string) bool { return strings.TrimSpace(value) != "" && len(value) <= 200 }

func claimAttention(ctx context.Context, id uuid.UUID, requestID, worker string, ttlSeconds int, now time.Time) (*ReviewClaim, error) {
	if !validIdentity(requestID) || !validIdentity(worker) || ttlSeconds < 1 || ttlSeconds > 3600 {
		return nil, fmt.Errorf("%w: request_id and worker are required; ttl_seconds must be 1..3600", ErrAttentionInput)
	}
	var result *ReviewClaim
	_, err := mutateCampaign(ctx, id, func(c *Campaign) error {
		archiveTerminalClaims(c, now)
		if claim := findClaimByRequest(c, requestID); claim != nil {
			if claim.Worker != worker || claim.TTLSeconds != ttlSeconds {
				return ErrClaimConflict
			}
			copy := *claim
			result = &copy
			return nil
		}
		if len(c.Claims) >= maxActiveClaims {
			return ErrClaimCapacity
		}
		if _, err := syncCampaignFiles(c, c.Patterns); err != nil {
			return err
		}
		files := selectAttention(c, 1, now)
		if len(files) == 0 {
			return nil
		}
		file := files[0]
		claim := ReviewClaim{
			ID: uuid.NewString(), RequestID: requestID, Worker: worker, FileID: file.ID,
			FilePath: file.FilePath, Revision: *file.ContentHash, TTLSeconds: ttlSeconds,
			CreatedAt: now.UTC(), ExpiresAt: now.Add(time.Duration(ttlSeconds) * time.Second).UTC(),
		}
		c.Claims = append(c.Claims, claim)
		c.AttentionSequence++
		for i := range c.TrackedFiles {
			if c.TrackedFiles[i].ID == file.ID {
				c.TrackedFiles[i].LastAttentionSequence = c.AttentionSequence
				break
			}
		}
		result = &claim
		return nil
	})
	return result, err
}

func completeAttention(ctx context.Context, id uuid.UUID, claimID, worker, outcome, evidence string, now time.Time) (*ReviewClaim, error) {
	if !validIdentity(claimID) || !validIdentity(worker) || len(evidence) > 4096 ||
		(outcome != "reviewed" && outcome != "incomplete" && outcome != "skipped") ||
		(outcome == "reviewed" && strings.TrimSpace(evidence) == "") {
		return nil, fmt.Errorf("%w: require a claim, worker, supported outcome and evidence for reviewed work", ErrAttentionInput)
	}
	var result *ReviewClaim
	_, err := mutateCampaign(ctx, id, func(c *Campaign) error {
		archiveTerminalClaims(c, now)
		index := -1
		for i := range c.Claims {
			if c.Claims[i].ID == claimID {
				index = i
				break
			}
		}
		if index < 0 {
			for i := range c.ArchivedClaims {
				if c.ArchivedClaims[i].ID != claimID {
					continue
				}
				claim := &c.ArchivedClaims[i]
				if claim.Worker != worker {
					return ErrClaimConflict
				}
				if claim.CompletedAt != nil && claim.Outcome == outcome && claim.Evidence == evidence {
					copy := *claim
					result = &copy
					return nil
				}
				return ErrClaimExpired
			}
			return ErrClaimConflict
		}
		claim := &c.Claims[index]
		if claim.Worker != worker {
			return ErrClaimConflict
		}
		if claim.CompletedAt != nil {
			if claim.Outcome != outcome || claim.Evidence != evidence {
				return ErrClaimConflict
			}
			copy := *claim
			result = &copy
			return nil
		}
		if !claim.ExpiresAt.After(now) {
			return ErrClaimExpired
		}
		if _, err := syncCampaignFiles(c, c.Patterns); err != nil {
			return err
		}
		// sync can move files but does not mutate the claims slice.
		var file *TrackedFile
		for i := range c.TrackedFiles {
			if c.TrackedFiles[i].ID == claim.FileID {
				file = &c.TrackedFiles[i]
				break
			}
		}
		if outcome == "reviewed" {
			if file == nil || file.Deleted || file.Excluded || file.ContentHash == nil || *file.ContentHash != claim.Revision {
				return ErrClaimConflict
			}
			file.ReviewedRevision = claim.Revision
			timestamp := now.UTC()
			file.LastReviewed = &timestamp
			file.ReviewCount++
		}
		timestamp := now.UTC()
		claim.CompletedAt = &timestamp
		claim.Outcome, claim.Evidence = outcome, evidence
		copy := *claim
		result = &copy
		return nil
	})
	return result, err
}
