package sceneprep

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/api-core/blobstore"
	preparationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	ErrInvalid  = errors.New("invalid photographic preparation")
	ErrNotFound = errors.New("photographic preparation not found")
)

// Repository owns only immutable preparation metadata. Blobs are independently
// content addressed and retained as data, outside disposable inference outputs.
type Repository interface {
	Put(context.Context, *preparationv1.StoredPhotograph) error
	Get(context.Context, string) (*preparationv1.StoredPhotograph, error)
	List(context.Context, int) ([]*preparationv1.StoredPhotograph, error)
	ReviewOperation(context.Context, string) (string, *preparationv1.StoredPhotograph, error)
	PutReviewOperation(context.Context, string, string, *preparationv1.StoredPhotograph) (*preparationv1.StoredPhotograph, error)
}

type Service struct {
	Repository Repository
	Blobs      blobstore.BlobStore
}

func (s *Service) Create(ctx context.Context, req *preparationv1.CreatePhotographRequest) (*preparationv1.StoredPhotograph, error) {
	if req == nil || strings.TrimSpace(req.Label) == "" || len(req.Label) > 160 {
		return nil, fmt.Errorf("%w: a label of 1..160 bytes is required", ErrInvalid)
	}
	if req.Manifest == nil || req.Manifest.GetReview().GetDecision() != "pending" || req.Manifest.GetCameraEnvelope().GetOverscanReviewed() {
		return nil, fmt.Errorf("%w: new source preparation requires pending review and unreviewed overscan", ErrInvalid)
	}
	artifacts, err := Validate(req.Manifest)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if len(req.Artifacts) != len(artifacts) {
		return nil, fmt.Errorf("%w: bind every declared artifact exactly once", ErrInvalid)
	}
	bindings := map[string]string{}
	inline := map[string]string{}
	for _, b := range req.Artifacts {
		if b == nil || artifacts[b.ArtifactPath] == nil || (bindings[b.ArtifactPath] != "" || inline[b.ArtifactPath] != "") || ((b.StoredRef == "") == (b.InlineJson == "")) {
			return nil, fmt.Errorf("%w: invalid or duplicate artifact binding", ErrInvalid)
		}
		if b.InlineJson != "" {
			if req.Manifest.Background.RepairProvenance == nil || b.ArtifactPath != req.Manifest.Background.RepairProvenance.Path || artifacts[b.ArtifactPath].MediaType != "application/json" || len(b.InlineJson) > 64<<10 || !json.Valid([]byte(b.InlineJson)) {
				return nil, fmt.Errorf("%w: inline data is restricted to bounded declared repair-provenance JSON", ErrInvalid)
			}
			inline[b.ArtifactPath] = b.InlineJson
		} else {
			if !ownedRef(b.StoredRef) {
				return nil, fmt.Errorf("%w: artifact is not an owned reference", ErrInvalid)
			}
			bindings[b.ArtifactPath] = b.StoredRef
		}
	}
	raw, err := build(req.Manifest, func(name string) (io.ReadCloser, error) {
		if raw, ok := inline[name]; ok {
			return io.NopCloser(strings.NewReader(raw)), nil
		}
		r, _, e := s.Blobs.Get(ctx, bindings[name])
		return r, e
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return s.retain(ctx, raw, req.Label, "")
}

func (s *Service) Get(ctx context.Context, id string) (*preparationv1.StoredPhotograph, error) {
	if !validSHA(id) {
		return nil, fmt.Errorf("%w: invalid preparation identity", ErrInvalid)
	}
	return s.Repository.Get(ctx, id)
}

func (s *Service) List(ctx context.Context, limit uint32) ([]*preparationv1.StoredPhotograph, error) {
	if limit == 0 {
		limit = 30
	}
	if limit > 100 {
		return nil, fmt.Errorf("%w: limit must be 1..100", ErrInvalid)
	}
	return s.Repository.List(ctx, int(limit))
}

// Review creates a child content revision, preserving the original decision
// and bytes. It never claims automatic scene/render qualification.
func (s *Service) Review(ctx context.Context, req *preparationv1.ReviewPhotographRequest) (*preparationv1.StoredPhotograph, error) {
	if req == nil || req.Review == nil || req.Review.Decision == "pending" || !validOperationID(req.OperationId) {
		return nil, fmt.Errorf("%w: an explicit completed review and stable operation_id UUID are required", ErrInvalid)
	}
	fingerprint, err := reviewFingerprint(req)
	if err != nil {
		return nil, err
	}
	if priorFingerprint, existing, err := s.Repository.ReviewOperation(ctx, req.OperationId); err != nil {
		return nil, err
	} else if existing != nil {
		if priorFingerprint != fingerprint {
			return nil, fmt.Errorf("%w: operation_id was already used for a different review request", ErrInvalid)
		}
		return existing, nil
	}
	prior, err := s.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	r, _, err := s.Blobs.Get(ctx, prior.BundleRef)
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(r, MaxBundleBytes+1))
	closeErr := r.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if len(raw) > int(MaxBundleBytes) {
		return nil, fmt.Errorf("stored preparation exceeds limit")
	}
	priorHash := sha256.Sum256(raw)
	if hex.EncodeToString(priorHash[:]) != req.Id {
		return nil, fmt.Errorf("stored preparation identity changed")
	}
	bundle, err := Open(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}
	m := proto.Clone(bundle.Manifest).(*preparationv1.PreparedPhotograph)
	m.Review = proto.Clone(req.Review).(*preparationv1.VisualReview)
	// This value is assigned by the owner and included in the immutable child
	// manifest so downstream publication retries keep the same review digest.
	ownerTimestamp := time.Now().UTC().UnixMilli()
	m.Review.ReviewedAtUnixMs = ownerTimestamp
	m.CameraEnvelope.OverscanReviewed = req.OverscanReviewed
	updated, err := build(m, bundle.OpenArtifact)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	return s.retainReview(ctx, updated, prior.Label, prior.Id, req.OperationId, fingerprint)
}

func (s *Service) retain(ctx context.Context, raw []byte, label, parent string) (*preparationv1.StoredPhotograph, error) {
	stored, err := s.buildStored(ctx, raw, label, parent)
	if err != nil {
		return nil, err
	}
	if err := s.Repository.Put(ctx, stored); err != nil {
		return nil, err
	}
	return s.Repository.Get(ctx, stored.Id)
}

func (s *Service) retainReview(ctx context.Context, raw []byte, label, parent, operationID, fingerprint string) (*preparationv1.StoredPhotograph, error) {
	stored, err := s.buildStored(ctx, raw, label, parent)
	if err != nil {
		return nil, err
	}
	return s.Repository.PutReviewOperation(ctx, operationID, fingerprint, stored)
}

func (s *Service) buildStored(ctx context.Context, raw []byte, label, parent string) (*preparationv1.StoredPhotograph, error) {
	bundle, err := Open(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}
	assessment, err := assessBundle(bundle)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	hash := sha256.Sum256(raw)
	id := hex.EncodeToString(hash[:])
	ref := "scene-preparations/" + id + ".zip"
	if err := s.Blobs.Put(ctx, ref, bytes.NewReader(raw), "application/zip"); err != nil {
		return nil, err
	}
	previewRefs := map[string]string{}
	for _, a := range []*preparationv1.Artifact{bundle.Manifest.Source.Oriented, bundle.Manifest.Background.CleanPlate} {
		r, e := bundle.OpenArtifact(a.Path)
		if e != nil {
			return nil, e
		}
		ref := "scene-preparation-assets/" + a.Sha256
		e = s.Blobs.Put(ctx, ref, r, a.MediaType)
		closeErr := r.Close()
		if e != nil {
			return nil, e
		}
		if closeErr != nil {
			return nil, closeErr
		}
		previewRefs[a.Path] = ref
	}
	createdAt := time.Now().UTC().UnixMilli()
	if bundle.Manifest.Review.ReviewedAtUnixMs > 0 {
		createdAt = bundle.Manifest.Review.ReviewedAtUnixMs
	}
	stored := &preparationv1.StoredPhotograph{RepairAssessment: assessment, SourcePreviewRef: previewRefs[bundle.Manifest.Source.Oriented.Path], CleanPlatePreviewRef: previewRefs[bundle.Manifest.Background.CleanPlate.Path], Id: id, Label: label, BundleRef: ref, Manifest: bundle.Manifest, ParentId: parent, CreatedAtUnixMs: createdAt}
	return stored, nil
}

func validOperationID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, r := range id {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if r != '-' {
				return false
			}
			continue
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func reviewFingerprint(req *preparationv1.ReviewPhotographRequest) (string, error) {
	copy := proto.Clone(req).(*preparationv1.ReviewPhotographRequest)
	copy.OperationId = ""
	copy.Review.ReviewedAtUnixMs = 0
	raw, err := protojson.Marshal(copy)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func ownedRef(ref string) bool {
	if len(ref) > 256 || strings.ContainsAny(ref, "\\:\x00") || strings.HasPrefix(ref, "/") || strings.Contains(ref, "..") {
		return false
	}
	for _, prefix := range []string{"input/", "inputs/", "mask/", "masks/", "out/", "scene-preparations/"} {
		if strings.HasPrefix(ref, prefix) && len(ref) > len(prefix) {
			return true
		}
	}
	return false
}

// build streams bounded owned inputs into a deterministic ZIP, then validates
// actual bytes and digests. A caller cannot substitute aliases or extra files.
func build(m *preparationv1.PreparedPhotograph, open func(string) (io.ReadCloser, error)) ([]byte, error) {
	artifacts, err := Validate(m)
	if err != nil {
		return nil, err
	}
	raw, err := protojson.Marshal(m)
	if err != nil {
		return nil, err
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, raw); err != nil {
		return nil, err
	}
	raw = compact.Bytes()
	if int64(len(raw)) > MaxManifestBytes {
		return nil, fmt.Errorf("manifest exceeds limit")
	}
	var total int64
	names := make([]string, 0, len(artifacts))
	for name, a := range artifacts {
		total += a.SizeBytes
		names = append(names, name)
	}
	if total > int64(MaxExpandedBytes) {
		return nil, fmt.Errorf("artifact bytes exceed preparation limit")
	}
	sort.Strings(names)
	var output bytes.Buffer
	zw := zip.NewWriter(&output)
	w, err := zw.Create("manifest.json")
	if err != nil {
		return nil, err
	}
	if _, err = w.Write(raw); err != nil {
		return nil, err
	}
	for _, name := range names {
		r, err := open(name)
		if err != nil {
			return nil, fmt.Errorf("artifact %q: %w", name, err)
		}
		w, err := zw.Create(name)
		if err != nil {
			r.Close()
			return nil, err
		}
		n, copyErr := io.Copy(w, io.LimitReader(r, artifacts[name].SizeBytes+1))
		closeErr := r.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if n != artifacts[name].SizeBytes {
			return nil, fmt.Errorf("artifact %q actual size differs", name)
		}
	}
	if err = zw.Close(); err != nil {
		return nil, err
	}
	if _, err = Open(bytes.NewReader(output.Bytes()), int64(output.Len())); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
