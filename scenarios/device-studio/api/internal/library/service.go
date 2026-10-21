package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/blobstore"
)

type Service struct {
	repo  Repository
	store blobstore.BlobStore
	now   func() time.Time
	newID func() string
}

func NewService(repo Repository, store blobstore.BlobStore, now func() time.Time, newID func() string) (Service, error) {
	if repo == nil || store == nil {
		return Service{}, errors.New("library repository and blobstore are required")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = uuid.NewString
	}
	return Service{repo: repo, store: store, now: now, newID: newID}, nil
}

// Publish accepts a trusted internal candidate and a review bound to its
// canonical manifest. Uploaded bytes remain unselectable until one atomic
// immutable metadata transaction commits. Failed/ambiguous writes retain their
// uniquely staged objects; this owner has no implicit garbage collector.
func (s Service) Publish(ctx context.Context, input PublishInput) (BundleRevision, error) {
	manifest, manifestJSON, manifestSHA, err := canonicalManifest(input.Manifest)
	if err != nil {
		return BundleRevision{}, err
	}
	review, reviewJSON, reviewSHA, err := canonicalReview(input.Review, manifestSHA)
	if err != nil {
		return BundleRevision{}, err
	}
	if current, getErr := s.repo.GetRevision(ctx, manifest.BundleID, manifest.Revision); getErr == nil {
		current, err = validateStored(current, manifest.BundleID, manifest.Revision)
		if err != nil {
			return BundleRevision{}, err
		}
		if current.ManifestSHA256 == manifestSHA && current.ReviewSHA256 == reviewSHA {
			return current, nil
		}
		return BundleRevision{}, ErrRevisionConflict
	} else if !errors.Is(getErr, ErrRevisionNotFound) {
		return BundleRevision{}, fmt.Errorf("check existing library revision: %w", getErr)
	}
	if len(input.Sources) != len(manifest.Artifacts) {
		return BundleRevision{}, errors.New("one source stream is required for each reviewed artifact")
	}
	keys := make(map[string]string, len(manifest.Artifacts))
	for _, artifact := range manifest.Artifacts {
		source := input.Sources[artifact.ID]
		if source == nil {
			return BundleRevision{}, fmt.Errorf("artifact %q source stream is required", artifact.ID)
		}
		token := strings.TrimSpace(s.newID())
		if !safeID.MatchString(token) {
			return BundleRevision{}, errors.New("library staging id source returned an invalid id")
		}
		key := fmt.Sprintf("bundles/%s/revisions/%d/staged/%s/%s", manifest.BundleID, manifest.Revision, token, artifact.ID)
		stream := &uploadVerifier{ctx: ctx, reader: source, expectedSize: artifact.SizeBytes, maxSize: roleLimit(artifact.Role), expectedSHA: artifact.SHA256, hash: sha256.New()}
		if err := s.store.Put(ctx, key, stream, artifact.MIME); err != nil {
			return BundleRevision{}, fmt.Errorf("stage library artifact %q: %w", artifact.ID, err)
		}
		if !stream.verified() {
			return BundleRevision{}, fmt.Errorf("stage library artifact %q: %w", artifact.ID, stream.failure())
		}
		if err := verifyBlob(ctx, s.store, key, artifact); err != nil {
			return BundleRevision{}, fmt.Errorf("verify staged library artifact %q: %w", artifact.ID, err)
		}
		keys[artifact.ID] = key
	}
	created := BundleRevision{
		Manifest: manifest, ManifestSHA256: manifestSHA, Review: review,
		ReviewSHA256: reviewSHA, PublishedAt: s.now().UTC(), artifactKeys: keys,
		manifestJSON: manifestJSON, reviewJSON: reviewJSON,
	}
	if err := s.repo.CreateRevision(ctx, created); err == nil {
		return created, nil
	} else {
		// A commit may succeed while its acknowledgement is lost. Readback is
		// the authority; staged objects are never deleted on this path.
		current, readErr := s.repo.GetRevision(ctx, manifest.BundleID, manifest.Revision)
		if readErr == nil {
			current, validateErr := validateStored(current, manifest.BundleID, manifest.Revision)
			if validateErr != nil {
				return BundleRevision{}, errors.Join(fmt.Errorf("publish library revision: %w", err), validateErr)
			}
			if current.ManifestSHA256 == manifestSHA && current.ReviewSHA256 == reviewSHA {
				return current, nil
			}
			return BundleRevision{}, ErrRevisionConflict
		}
		return BundleRevision{}, errors.Join(fmt.Errorf("publish library revision: %w", err), fmt.Errorf("confirm publication readback: %w", readErr))
	}
}

func (s Service) GetRevision(ctx context.Context, bundleID string, revision uint32) (BundleRevision, error) {
	if !safeID.MatchString(bundleID) || revision == 0 {
		return BundleRevision{}, ErrRevisionNotFound
	}
	stored, err := s.repo.GetRevision(ctx, bundleID, revision)
	if err != nil {
		return BundleRevision{}, err
	}
	return validateStored(stored, bundleID, revision)
}

// ListRevisions returns bounded, validated immutable metadata in stable
// bundle/revision order. Artifact storage keys and bytes remain private.
func (s Service) ListRevisions(ctx context.Context, afterBundleID string, afterRevision uint32, limit int) ([]BundleRevision, error) {
	if limit < 1 || limit > 100 || (afterBundleID == "" && afterRevision != 0) {
		return nil, errors.New("library revision list requires a valid cursor and page size between 1 and 100")
	}
	rows, err := s.repo.ListRevisions(ctx, afterBundleID, afterRevision, limit)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		var identity BundleManifest
		if err := decodeCanonical(rows[i].manifestJSON, &identity); err != nil {
			return nil, fmt.Errorf("decode listed library identity: %w", err)
		}
		rows[i], err = validateStored(rows[i], identity.BundleID, identity.Revision)
		if err != nil {
			return nil, fmt.Errorf("validate listed library revision: %w", err)
		}
	}
	return rows, nil
}

// OpenArtifact resolves an opaque artifact identity and returns a streaming
// reader that withholds the last byte until size, EOF, and SHA-256 agree.
func (s Service) OpenArtifact(ctx context.Context, bundleID string, revision uint32, artifactID string, use Use) (ResolvedArtifact, error) {
	if !validUse(use) || use == UseExternalAIUpload {
		return ResolvedArtifact{}, ErrUseDenied
	}
	stored, err := s.GetRevision(ctx, bundleID, revision)
	if err != nil {
		return ResolvedArtifact{}, err
	}
	if !containsUse(stored.Review.AllowedUses, use) {
		return ResolvedArtifact{}, ErrUseDenied
	}
	for _, artifact := range stored.Manifest.Artifacts {
		if artifact.ID != artifactID {
			continue
		}
		if !containsUse(artifact.Rights, use) {
			return ResolvedArtifact{}, ErrUseDenied
		}
		key := stored.artifactKeys[artifact.ID]
		if key == "" {
			return ResolvedArtifact{}, errors.New("library artifact reference is missing")
		}
		reader, _, err := s.store.Get(ctx, key)
		if err != nil {
			return ResolvedArtifact{}, fmt.Errorf("open library artifact bytes: %w", err)
		}
		return ResolvedArtifact{BundleID: bundleID, Revision: revision, Artifact: artifact,
			Reader: &verifiedReader{ctx: ctx, reader: reader, expectedSize: artifact.SizeBytes, expectedSHA: artifact.SHA256, maxSize: roleLimit(artifact.Role), hash: sha256.New()}}, nil
	}
	return ResolvedArtifact{}, ErrArtifactNotFound
}

func validateStored(stored BundleRevision, bundleID string, revision uint32) (BundleRevision, error) {
	if len(stored.manifestJSON) == 0 || len(stored.manifestJSON) > MaxStoredManifestBytes || len(stored.reviewJSON) == 0 || len(stored.reviewJSON) > MaxStoredReviewBytes {
		return BundleRevision{}, errors.New("stored library metadata exceeds its bounded size")
	}
	var manifest BundleManifest
	if err := decodeCanonical(stored.manifestJSON, &manifest); err != nil {
		return BundleRevision{}, fmt.Errorf("stored library manifest is invalid: %w", err)
	}
	manifest, canonical, manifestSHA, err := canonicalManifest(manifest)
	if err != nil || !bytes.Equal(canonical, stored.manifestJSON) || manifestSHA != stored.ManifestSHA256 || manifest.BundleID != bundleID || manifest.Revision != revision {
		return BundleRevision{}, errors.New("stored library manifest identity or digest is inconsistent")
	}
	var review ReviewBinding
	if err := decodeCanonical(stored.reviewJSON, &review); err != nil {
		return BundleRevision{}, fmt.Errorf("stored library review is invalid: %w", err)
	}
	review, reviewCanonical, reviewSHA, err := canonicalReview(review, manifestSHA)
	if err != nil || !bytes.Equal(reviewCanonical, stored.reviewJSON) || reviewSHA != stored.ReviewSHA256 {
		return BundleRevision{}, errors.New("stored library review identity or digest is inconsistent")
	}
	if stored.PublishedAt.IsZero() || len(stored.artifactKeys) != len(manifest.Artifacts) {
		return BundleRevision{}, errors.New("stored library artifact references are incomplete")
	}
	for _, artifact := range manifest.Artifacts {
		if stored.artifactKeys[artifact.ID] == "" {
			return BundleRevision{}, fmt.Errorf("stored library reference for artifact %q is missing", artifact.ID)
		}
	}
	stored.Manifest, stored.Review = manifest, review
	return stored, nil
}

func decodeCanonical(data []byte, target any) error {
	if len(data) == 0 || len(data) > MaxStoredManifestBytes {
		return fmt.Errorf("stored metadata exceeds its bounded size")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("stored metadata contains trailing JSON")
	}
	return nil
}

func containsUse(uses []Use, target Use) bool {
	for _, use := range uses {
		if use == target {
			return true
		}
	}
	return false
}

func roleLimit(role ArtifactRole) int64 {
	if role == RoleGeometryGLB {
		return MaxGeometryBytes
	}
	return MaxTextureBytes
}

func verifyBlob(ctx context.Context, store blobstore.BlobStore, key string, artifact Artifact) error {
	reader, _, err := store.Get(ctx, key)
	if err != nil {
		return err
	}
	verified := &verifiedReader{ctx: ctx, reader: reader, expectedSize: artifact.SizeBytes, expectedSHA: artifact.SHA256, maxSize: roleLimit(artifact.Role), hash: sha256.New()}
	_, readErr := io.Copy(io.Discard, verified)
	closeErr := verified.Close()
	return errors.Join(readErr, closeErr)
}

type uploadVerifier struct {
	ctx          context.Context
	reader       io.Reader
	expectedSize int64
	maxSize      int64
	expectedSHA  string
	hash         hashWriter
	count        int64
	eof          bool
	err          error
}

func (r *uploadVerifier) Read(p []byte) (int, error) {
	if r.ctx != nil {
		if err := r.ctx.Err(); err != nil {
			r.err = err
			return 0, err
		}
	}
	if r.err != nil {
		return 0, r.err
	}
	if len(p) == 0 {
		return 0, nil
	}
	remaining := r.expectedSize + 1 - r.count
	if remaining <= 0 || r.count >= r.maxSize+1 {
		r.err = errors.New("artifact stream exceeds its declared or role limit")
		return 0, r.err
	}
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err := r.reader.Read(p)
	if n > 0 {
		r.count += int64(n)
		_, _ = r.hash.Write(p[:n])
		if r.count > r.expectedSize || r.count > r.maxSize {
			r.err = errors.New("artifact stream exceeds its declared or role limit")
			return n, r.err
		}
	}
	if errors.Is(err, io.EOF) {
		r.eof = true
		if r.count != r.expectedSize || hex.EncodeToString(r.hash.Sum(nil)) != r.expectedSHA {
			r.err = errors.New("artifact bytes differ from the reviewed size or digest")
			return n, r.err
		}
	}
	if err != nil && !errors.Is(err, io.EOF) {
		r.err = err
	}
	return n, err
}

func (r *uploadVerifier) verified() bool { return r.eof && r.err == nil && r.count == r.expectedSize }
func (r *uploadVerifier) failure() error {
	if r.err != nil {
		return r.err
	}
	return errors.New("blobstore did not read the complete artifact stream")
}

type hashWriter interface {
	Write([]byte) (int, error)
	Sum([]byte) []byte
}

type verifiedReader struct {
	ctx          context.Context
	reader       io.ReadCloser
	expectedSize int64
	expectedSHA  string
	maxSize      int64
	count        int64
	hash         hashWriter
	finalByte    byte
	finalRead    bool
	checked      bool
	verifyErr    error
}

func (r *verifiedReader) Read(p []byte) (int, error) {
	if r.ctx != nil {
		if err := r.ctx.Err(); err != nil {
			r.verifyErr = err
			return 0, err
		}
	}
	if len(p) == 0 {
		return 0, nil
	}
	if r.verifyErr != nil {
		return 0, r.verifyErr
	}
	if r.checked {
		return 0, io.EOF
	}
	if !r.finalRead {
		remaining := r.expectedSize - r.count
		if remaining <= 0 {
			r.verifyErr = errors.New("artifact size accounting is invalid")
			return 0, r.verifyErr
		}
		if remaining > 1 {
			if int64(len(p)) > remaining-1 {
				p = p[:remaining-1]
			}
			n, err := r.reader.Read(p)
			if n > 0 {
				r.count += int64(n)
				_, _ = r.hash.Write(p[:n])
			}
			if errors.Is(err, io.EOF) {
				r.verifyErr = errors.New("artifact ended before its declared size")
				return n, r.verifyErr
			}
			if err != nil {
				r.verifyErr = err
			}
			return n, err
		}
		var last [1]byte
		n, err := r.reader.Read(last[:])
		if n == 0 {
			if errors.Is(err, io.EOF) {
				r.verifyErr = errors.New("artifact ended before its declared size")
			} else if err != nil {
				r.verifyErr = err
			}
			return 0, r.errorOr(err)
		}
		r.finalByte, r.finalRead = last[0], true
		r.count++
		_, _ = r.hash.Write(last[:])
		if err != nil && !errors.Is(err, io.EOF) {
			r.verifyErr = err
			return 0, err
		}
		if errors.Is(err, io.EOF) {
			return r.publishFinal(p)
		}
	}
	var extra [1]byte
	n, err := r.reader.Read(extra[:])
	if n > 0 {
		r.verifyErr = errors.New("artifact exceeds its declared size")
		return 0, r.verifyErr
	}
	if err != nil && !errors.Is(err, io.EOF) {
		r.verifyErr = err
		return 0, err
	}
	if !errors.Is(err, io.EOF) {
		return 0, nil
	}
	return r.publishFinal(p)
}

func (r *verifiedReader) publishFinal(p []byte) (int, error) {
	if r.count != r.expectedSize || r.count > r.maxSize || hex.EncodeToString(r.hash.Sum(nil)) != r.expectedSHA {
		r.verifyErr = errors.New("artifact bytes differ from stored size or digest")
		return 0, r.verifyErr
	}
	r.checked = true
	p[0] = r.finalByte
	return 1, nil
}

func (r *verifiedReader) errorOr(fallback error) error {
	if r.verifyErr != nil {
		return r.verifyErr
	}
	return fallback
}
func (r *verifiedReader) Close() error {
	if !r.checked && r.verifyErr == nil {
		r.verifyErr = ErrArtifactUnverified
	}
	return errors.Join(r.verifyErr, r.reader.Close())
}
