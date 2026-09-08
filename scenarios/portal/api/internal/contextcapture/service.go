package contextcapture

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/blobstore"
)

type Service struct {
	repo  Repository
	blobs blobstore.BlobStore
	now   func() time.Time
}

func NewService(repo Repository, blobs blobstore.BlobStore, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, blobs: blobs, now: now}
}
func imageKey(id string) string { return "context/" + id + "/original.png" }

func (s *Service) Import(ctx context.Context, owner string, input Import, retention time.Duration) (Document, error) {
	if ctx.Err() != nil {
		return Document{}, ctx.Err()
	}
	prepareAt := s.now()
	if input.RequestID != "" {
		previous, err := s.repo.LookupIntent(ctx, owner, input.RequestID)
		if err == nil {
			if previous.DocumentID == "" {
				return Document{}, ErrUnavailable
			}
			prepareAt = previous.CreatedAt
		} else if !errors.Is(err, ErrUnavailable) {
			return Document{}, err
		}
	}
	doc, pixels, err := Prepare(owner, input, prepareAt, retention)
	if err != nil {
		return Document{}, err
	}
	if !s.now().Before(doc.ExpiresAt) {
		return Document{}, ErrUnavailable
	}
	if err = s.repo.Reserve(ctx, doc, int64(len(pixels))); err != nil {
		if errors.Is(err, ErrIntentExists) {
			previous, lookupErr := s.repo.LookupIntent(ctx, owner, doc.RequestID)
			if lookupErr != nil {
				return Document{}, lookupErr
			}
			if previous.DocumentID == "" {
				return Document{}, ErrUnavailable
			}
			if previous.Digest != doc.ImportDigest {
				return Document{}, ErrConflict
			}
			existing, _, readErr := s.Read(ctx, owner, previous.DocumentID)
			return existing, readErr
		}
		return Document{}, err
	}
	err = s.repo.WithRecord(ctx, owner, doc.ID, func(record Record) (Action, error) {
		if record.State != Staging || !s.now().Before(record.Document.ExpiresAt) {
			return Keep, ErrUnavailable
		}
		if err := s.blobs.Put(ctx, imageKey(doc.ID), bytes.NewReader(pixels), "image/png"); err != nil {
			return Keep, err
		}
		if ctx.Err() != nil {
			return Keep, ctx.Err()
		}
		if !s.now().Before(record.Document.ExpiresAt) {
			return Keep, ErrUnavailable
		}
		return Publish, nil
	})
	if err != nil {
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		// Cleanup failure preserves the reservation so the expiry worker can retry.
		return Document{}, errors.Join(err, s.Delete(cleanup, owner, doc.ID))
	}
	return doc, nil
}

func (s *Service) Read(ctx context.Context, owner, id string) (Document, []byte, error) {
	var doc Document
	var pixels []byte
	err := s.repo.WithRecord(ctx, owner, id, func(record Record) (Action, error) {
		if record.State != Ready || !s.now().Before(record.Document.ExpiresAt) {
			return Keep, ErrUnavailable
		}
		reader, mime, err := s.blobs.Get(ctx, imageKey(id))
		if err != nil {
			return Keep, err
		}
		defer reader.Close()
		if mime != "image/png" {
			return Keep, ErrUnavailable
		}
		pixels, err = io.ReadAll(io.LimitReader(reader, MaxImageBytes+1))
		if err != nil {
			return Keep, err
		}
		digest := sha256.Sum256(pixels)
		if int64(len(pixels)) != record.Size || hex.EncodeToString(digest[:]) != record.Document.OriginalSHA256 || !s.now().Before(record.Document.ExpiresAt) {
			return Keep, ErrUnavailable
		}
		if ctx.Err() != nil {
			return Keep, ctx.Err()
		}
		doc = record.Document
		return Keep, nil
	})
	if err != nil {
		return Document{}, nil, err
	}
	return doc, pixels, nil
}

// ValidateReference authorizes an opaque document ID without returning image
// bytes. Delivery performs a fresh Render later, so an expired or deleted
// document cannot be used just because it was once attached to a message.
func (s *Service) ValidateReference(ctx context.Context, owner, id string) error {
	if owner == "" || id == "" {
		return ErrUnavailable
	}
	return s.repo.WithRecord(ctx, owner, id, func(record Record) (Action, error) {
		if record.State != Ready || !s.now().Before(record.Document.ExpiresAt) {
			return Keep, ErrUnavailable
		}
		return Keep, nil
	})
}

func (s *Service) Delete(ctx context.Context, owner, id string) error {
	return s.repo.WithRecord(ctx, owner, id, func(Record) (Action, error) {
		if err := s.blobs.Delete(ctx, imageKey(id)); err != nil {
			return Keep, err
		}
		return Remove, nil
	})
}

// Reap removes expired staging and published blobs in bounded batches. Access
// expires on every read even if a cleanup attempt fails or the worker is late.
func (s *Service) Reap(ctx context.Context, limit int) error {
	ids, err := s.repo.Expired(ctx, s.now(), limit)
	if err != nil {
		return err
	}
	var failures []error
	for _, id := range ids {
		err = s.repo.WithRecord(ctx, id.Owner, id.ID, func(record Record) (Action, error) {
			if s.now().Before(record.Document.ExpiresAt) {
				return Keep, nil
			}
			if err := s.blobs.Delete(ctx, imageKey(id.ID)); err != nil {
				return Keep, err
			}
			return Remove, nil
		})
		if err != nil && !errors.Is(err, ErrUnavailable) {
			failures = append(failures, err)
		}
	}
	failures = append(failures, s.repo.PruneIntents(ctx, s.now()))
	return errors.Join(failures...)
}

// ImportStatus reports durable publication state without replaying an import or
// accessing image bytes. Read still verifies blob integrity before pixel use.
type ImportStatus struct {
	State    string
	Document *Document
}

func (s *Service) ReconcileImport(ctx context.Context, owner, requestID string) (ImportStatus, error) {
	id, err := uuid.Parse(requestID)
	if owner == "" || err != nil || id == uuid.Nil || id.String() != requestID {
		return ImportStatus{}, ErrInvalid
	}
	intent, err := s.repo.LookupIntent(ctx, owner, requestID)
	if errors.Is(err, ErrUnavailable) {
		return ImportStatus{State: "absent"}, nil
	}
	if err != nil {
		return ImportStatus{}, err
	}
	result := ImportStatus{State: "unavailable"}
	err = s.repo.WithRecord(ctx, owner, intent.DocumentID, func(record Record) (Action, error) {
		if record.Document.RequestID != requestID || record.Document.ImportDigest != intent.Digest {
			return Keep, ErrUnavailable
		}
		if !s.now().Before(record.Document.ExpiresAt) {
			return Keep, nil
		}
		if record.State == Staging {
			result.State = "staging"
			return Keep, nil
		}
		if record.State == Ready {
			result.State = "ready"
			doc := record.Document
			result.Document = &doc
		}
		return Keep, nil
	})
	if errors.Is(err, ErrUnavailable) {
		return ImportStatus{State: "unavailable"}, nil
	}
	if err != nil {
		return ImportStatus{}, err
	}
	return result, nil
}

// CancelImport commits a receipt even if the original request has not arrived.
// It removes any existing blob under the same exclusion as publication.
func (s *Service) CancelImport(ctx context.Context, owner, requestID string) error {
	return s.repo.CancelIntent(ctx, owner, requestID, s.now(), func(record Record) error {
		return s.blobs.Delete(ctx, imageKey(record.Document.ID))
	})
}
