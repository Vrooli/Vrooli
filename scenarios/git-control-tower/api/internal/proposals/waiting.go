package proposals

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"time"
)

// WaitingItem is one open proposal in the operator's waiting digest.
type WaitingItem struct {
	ID           string    `json:"id"`
	RepositoryID string    `json:"repository_id"`
	EffortRef    string    `json:"effort_ref,omitempty"`
	Epoch        string    `json:"epoch,omitempty"`
	Subject      string    `json:"subject"`
	Files        int       `json:"files"`
	CreatedAt    time.Time `json:"created_at"`
}

// Waiting lists open proposals across the given repositories created at or
// before cutoff, oldest first.
func (s *Service) Waiting(ctx context.Context, repositoryIDs []string, cutoff time.Time) ([]WaitingItem, error) {
	var items []WaitingItem
	for _, repositoryID := range repositoryIDs {
		open, err := s.store.List(ctx, ListFilter{RepositoryID: repositoryID, Limit: 200})
		if err != nil {
			return nil, err
		}
		for _, proposal := range open {
			if proposal.CreatedAt.After(cutoff) {
				continue
			}
			items = append(items, WaitingItem{
				ID: proposal.ID, RepositoryID: proposal.RepositoryID, EffortRef: proposal.Work.EffortRef, Epoch: proposal.Work.Epoch,
				Subject: proposal.Message.Subject, Files: len(proposal.Files), CreatedAt: proposal.CreatedAt,
			})
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

// DigestSent reports whether the waiting digest for a UTC day ("2006-01-02")
// was already sent.
func (s *Store) DigestSent(ctx context.Context, day string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT proposal_count FROM gct_proposal_digests WHERE day = ?`, day).Scan(&count)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// RecordDigest marks the digest for a UTC day as sent.
func (s *Store) RecordDigest(ctx context.Context, day string, count int, sentAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO gct_proposal_digests (day, proposal_count, sent_at) VALUES (?,?,?)`, day, count, sentAt.UTC().Format(time.RFC3339Nano))
	return err
}

// DigestSent reports whether the waiting digest for a UTC day was sent.
func (s *Service) DigestSent(ctx context.Context, day string) (bool, error) {
	return s.store.DigestSent(ctx, day)
}

// RecordDigest marks the waiting digest for a UTC day as sent.
func (s *Service) RecordDigest(ctx context.Context, day string, count int) error {
	return s.store.RecordDigest(ctx, day, count, s.now())
}
