package release

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/legibility"

	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

var releaseIDPattern = regexp.MustCompile(`^backdrop-[A-Za-z0-9._-]+$`)

var pendingSequence atomic.Uint64

const releasedBackdropDir = "released-backdrops"

// CandidateEvidence is the release decision's view of one render result. It
// comes from the render owner, never from the release request. Keeping this
// internal avoids making the public wire contract carry a second, caller-
// controlled copy of bytes and qualification measurements.
type CandidateEvidence struct {
	ID                string
	JobID             string
	ImagePNG          []byte
	Width, Height     int
	StyleID           string
	Strategy          string
	SurfaceID         string
	Placement         string
	Regions           []catalog.Region
	ContrastThreshold float64
}

// CandidateSource resolves the bytes and qualification inputs that actually
// came from a render. A release cannot be created without it: a metadata-only
// candidate is not a deliverable asset.
type CandidateSource interface {
	CandidateEvidence(candidateID string) (CandidateEvidence, bool)
}

type Request struct {
	CandidateID, StyleID, Strategy, SurfaceID, Placement, AltText string
	Width, Height, ExpectedWidth, ExpectedHeight                  int
	Decorative, AIGeneratedSet, AIGenerated, LegibilityPasses     bool
	ContrastRatio, ContrastThreshold                              float64
	Regions                                                       []catalog.Region
	ImagePNG                                                      []byte
}
type Backdrop struct {
	ID, CandidateID, JobID, StyleID, SurfaceID, Placement, AltText string
	Width, Height                                                  int
	Decorative, AIGenerated                                        bool
	ContrastRatio, ContrastThreshold                               float64
	Regions                                                        []catalog.Region
	ImagePNG                                                       []byte
	MIMEType, ContentHash                                          string
	AssetStudioRef                                                 string
}

// AssetPublisher hands a released backdrop to the authority that owns
// provenance and disclosure records. The provenance argument is separate from
// the request because the request is what the caller asked for and the
// provenance is what actually happened — see provenance.go.
type AssetPublisher interface {
	Publish(ctx context.Context, r Request, p Provenance) (string, error)
}
type Store struct {
	mu         sync.RWMutex
	publisher  AssetPublisher
	provenance ProvenanceSource
	candidates CandidateSource
	roots      *filerouting.RoutedRoots
}

// NewStore is retained for validation-only unit tests. A release store with no
// routed roots cannot persist a release and therefore cannot accidentally make
// process memory the production source of truth.
func NewStore() *Store { return &Store{} }
func NewStoreWithPublisher(publisher AssetPublisher, provenance ProvenanceSource, candidates ...CandidateSource) *Store {
	var source CandidateSource
	if len(candidates) > 0 {
		source = candidates[0]
	}
	return &Store{publisher: publisher, provenance: provenance, candidates: source}
}

// NewStoreWithPublisherAndRoots wires the release domain to the scenario's
// routed data roots. Roots are selected for each operation so test-mode calls
// cannot read or write the live release tree.
func NewStoreWithPublisherAndRoots(publisher AssetPublisher, provenance ProvenanceSource, roots *filerouting.RoutedRoots, candidates ...CandidateSource) *Store {
	store := NewStoreWithPublisher(publisher, provenance, candidates...)
	store.roots = roots
	return store
}

func (s *Store) Release(r Request) (Backdrop, error) {
	return s.ReleaseContext(context.Background(), r)
}

func (s *Store) ReleaseContext(ctx context.Context, r Request) (Backdrop, error) {
	if err := ctx.Err(); err != nil {
		return Backdrop{}, err
	}
	if r.CandidateID == "" || r.StyleID == "" {
		return Backdrop{}, fmt.Errorf("release: candidate_id and style_id are required")
	}
	if r.AIGeneratedSet {
		return Backdrop{}, fmt.Errorf("release: ai_generated is derived and cannot be set directly")
	}
	if !r.Decorative && r.AltText == "" {
		return Backdrop{}, fmt.Errorf("release: alt_text is required, or set decorative=true")
	}
	if s.candidates == nil {
		return Backdrop{}, fmt.Errorf("release: candidate evidence source is unavailable")
	}
	candidate, known := s.candidates.CandidateEvidence(r.CandidateID)
	if !known {
		return Backdrop{}, fmt.Errorf("release: candidate %q has no recorded candidate evidence", r.CandidateID)
	}
	if candidate.ID != r.CandidateID {
		return Backdrop{}, fmt.Errorf("release: candidate evidence id %q does not match requested candidate %q", candidate.ID, r.CandidateID)
	}
	if candidate.JobID == "" {
		return Backdrop{}, fmt.Errorf("release: candidate evidence is missing job id")
	}
	if len(candidate.ImagePNG) == 0 {
		return Backdrop{}, fmt.Errorf("release: candidate bytes are missing")
	}
	actualWidth, actualHeight, format, err := pngMetadata(candidate.ImagePNG)
	if err != nil {
		return Backdrop{}, fmt.Errorf("release: candidate bytes are invalid: %w", err)
	}
	if format != "png" {
		return Backdrop{}, fmt.Errorf("release: candidate MIME must be image/png, got image/%s", format)
	}
	if candidate.Width <= 0 || candidate.Height <= 0 || candidate.Width != actualWidth || candidate.Height != actualHeight {
		return Backdrop{}, fmt.Errorf("release: candidate dimensions do not match bytes: recorded %dx%d, actual %dx%d", candidate.Width, candidate.Height, actualWidth, actualHeight)
	}
	if candidate.StyleID == "" || candidate.SurfaceID == "" || candidate.Placement == "" || candidate.Strategy == "" {
		return Backdrop{}, fmt.Errorf("release: candidate evidence is missing style, strategy, surface, or placement provenance")
	}
	if r.StyleID != "" && r.StyleID != candidate.StyleID {
		return Backdrop{}, fmt.Errorf("release: style_id %q does not match candidate evidence %q", r.StyleID, candidate.StyleID)
	}
	if r.Strategy != "" && r.Strategy != candidate.Strategy {
		return Backdrop{}, fmt.Errorf("release: strategy %q does not match candidate evidence %q", r.Strategy, candidate.Strategy)
	}
	if r.SurfaceID != "" && r.SurfaceID != candidate.SurfaceID {
		return Backdrop{}, fmt.Errorf("release: surface_id %q does not match candidate evidence %q", r.SurfaceID, candidate.SurfaceID)
	}
	if r.Placement != "" && r.Placement != candidate.Placement {
		return Backdrop{}, fmt.Errorf("release: placement %q does not match candidate evidence %q", r.Placement, candidate.Placement)
	}
	if r.Width > 0 && (r.Width != actualWidth || r.Height != actualHeight) {
		return Backdrop{}, fmt.Errorf("release: supplied dimensions %dx%d do not match candidate bytes %dx%d", r.Width, r.Height, actualWidth, actualHeight)
	}
	if r.ExpectedWidth > 0 && (actualWidth != r.ExpectedWidth || actualHeight != r.ExpectedHeight) {
		return Backdrop{}, fmt.Errorf("release: dimensions mismatch: expected %dx%d, got %dx%d", r.ExpectedWidth, r.ExpectedHeight, actualWidth, actualHeight)
	}

	threshold := candidate.ContrastThreshold
	if threshold <= 0 {
		threshold = 4.5
	}
	regions := make([]legibility.Region, 0, len(candidate.Regions))
	for _, region := range candidate.Regions {
		regions = append(regions, legibility.Region{X: region.X, Y: region.Y, Width: region.Width, Height: region.Height, Kind: region.Kind, TextColor: region.TextColor})
	}
	if !r.Decorative && len(regions) == 0 {
		return Backdrop{}, fmt.Errorf("release: candidate legibility regions are missing")
	}
	verdict, err := legibility.Measure(candidate.ImagePNG, regions, threshold, candidate.Placement)
	if err != nil {
		return Backdrop{}, fmt.Errorf("release: measure candidate legibility: %w", err)
	}
	if !verdict.Passes {
		return Backdrop{}, fmt.Errorf("release: candidate legibility failed with measured ratio %.3f below threshold %.3f", verdict.MinimumRatio, verdict.Threshold)
	}

	// Replace caller-controlled qualification fields with the evidence that was
	// just verified. This is also the exact payload handed to Asset Studio.
	r.StyleID = candidate.StyleID
	r.Strategy = candidate.Strategy
	r.SurfaceID = candidate.SurfaceID
	r.Placement = candidate.Placement
	r.Width, r.Height = actualWidth, actualHeight
	r.ContrastRatio, r.ContrastThreshold = verdict.MinimumRatio, verdict.Threshold
	r.Regions = append([]catalog.Region(nil), candidate.Regions...)
	r.ImagePNG = append([]byte(nil), candidate.ImagePNG...)
	hash := sha256.Sum256(candidate.ImagePNG)
	contentHash := hex.EncodeToString(hash[:])
	aig := r.Strategy == "guided" || r.Strategy == "synthesized"
	assetRef := ""
	if s.provenance == nil {
		return Backdrop{}, fmt.Errorf("release: candidate provenance source is unavailable")
	}
	p, known := s.provenance.CandidateProvenance(r.CandidateID)
	if !known {
		return Backdrop{}, fmt.Errorf("release: candidate %q has no recorded provenance", r.CandidateID)
	}
	expectedModelBacked := r.Strategy == "guided" || r.Strategy == "synthesized"
	if p.Strategy != "" && p.Strategy != r.Strategy {
		return Backdrop{}, fmt.Errorf("release: provenance strategy %q does not match candidate strategy %q", p.Strategy, r.Strategy)
	}
	if p.ModelBacked != expectedModelBacked {
		return Backdrop{}, fmt.Errorf("release: provenance disclosure state does not match candidate strategy %q", r.Strategy)
	}
	if aig {
		// The refusal below is the behaviour that must survive this capability
		// landing. When Asset Studio is absent, a model-backed backdrop is not
		// released at all — it is never quietly downgraded to a procedural
		// candidate, and its provenance is never invented locally, because both
		// would produce a synthetic image wearing an honest-looking label.
		if s.publisher == nil {
			return Backdrop{}, fmt.Errorf("release: model-backed candidate requires the asset-studio publisher capability, which is unavailable")
		}
		var err error
		assetRef, err = s.publisher.Publish(ctx, r, p)
		if err != nil {
			return Backdrop{}, fmt.Errorf("release: asset-studio handoff: %w", err)
		}
	}
	id := fmt.Sprintf("backdrop-%s", r.CandidateID)
	if !releaseIDPattern.MatchString(id) {
		return Backdrop{}, fmt.Errorf("release: candidate id is not a safe release identifier")
	}
	b := Backdrop{ID: id, CandidateID: r.CandidateID, JobID: candidate.JobID, StyleID: r.StyleID, SurfaceID: r.SurfaceID, Placement: r.Placement, AltText: r.AltText, Width: r.Width, Height: r.Height, Decorative: r.Decorative, AIGenerated: aig, ContrastRatio: r.ContrastRatio, ContrastThreshold: r.ContrastThreshold, Regions: append([]catalog.Region(nil), r.Regions...), ImagePNG: append([]byte(nil), r.ImagePNG...), MIMEType: "image/png", ContentHash: contentHash, AssetStudioRef: assetRef}
	if s.roots == nil {
		return Backdrop{}, fmt.Errorf("release: durable storage is unavailable")
	}
	return s.persist(ctx, b)
}

func (s *Store) Get(id string) (Backdrop, error) {
	return s.GetContext(context.Background(), id)
}

func (s *Store) GetContext(ctx context.Context, id string) (Backdrop, error) {
	if s.roots == nil {
		return Backdrop{}, fmt.Errorf("release: durable storage is unavailable")
	}
	if !releaseIDPattern.MatchString(id) {
		return Backdrop{}, fmt.Errorf("release: backdrop %q not found", id)
	}
	dir, err := s.roots.PickRequired(ctx, storage.ClassData)
	if err != nil {
		return Backdrop{}, err
	}
	root, err := os.OpenRoot(dir)
	if errors.Is(err, os.ErrNotExist) {
		return Backdrop{}, fmt.Errorf("release: backdrop %q not found", id)
	}
	if err != nil {
		return Backdrop{}, err
	}
	defer root.Close()
	return readDurableBackdrop(root, id)
}

type persistedBackdrop struct {
	ID                string           `json:"id"`
	CandidateID       string           `json:"candidate_id"`
	JobID             string           `json:"job_id"`
	StyleID           string           `json:"style_id"`
	SurfaceID         string           `json:"surface_id"`
	Placement         string           `json:"placement"`
	AltText           string           `json:"alt_text"`
	Width             int              `json:"width"`
	Height            int              `json:"height"`
	Decorative        bool             `json:"decorative"`
	AIGenerated       bool             `json:"ai_generated"`
	ContrastRatio     float64          `json:"contrast_ratio"`
	ContrastThreshold float64          `json:"contrast_threshold"`
	Regions           []catalog.Region `json:"reserved_regions"`
	MIMEType          string           `json:"mime_type"`
	ContentHash       string           `json:"content_hash"`
	AssetStudioRef    string           `json:"asset_studio_ref,omitempty"`
	ByteLength        int              `json:"byte_length"`
}

func (b Backdrop) persisted() persistedBackdrop {
	return persistedBackdrop{ID: b.ID, CandidateID: b.CandidateID, JobID: b.JobID, StyleID: b.StyleID, SurfaceID: b.SurfaceID, Placement: b.Placement, AltText: b.AltText, Width: b.Width, Height: b.Height, Decorative: b.Decorative, AIGenerated: b.AIGenerated, ContrastRatio: b.ContrastRatio, ContrastThreshold: b.ContrastThreshold, Regions: append([]catalog.Region(nil), b.Regions...), MIMEType: b.MIMEType, ContentHash: b.ContentHash, AssetStudioRef: b.AssetStudioRef, ByteLength: len(b.ImagePNG)}
}

func (p persistedBackdrop) backdrop(imagePNG []byte) Backdrop {
	return Backdrop{ID: p.ID, CandidateID: p.CandidateID, JobID: p.JobID, StyleID: p.StyleID, SurfaceID: p.SurfaceID, Placement: p.Placement, AltText: p.AltText, Width: p.Width, Height: p.Height, Decorative: p.Decorative, AIGenerated: p.AIGenerated, ContrastRatio: p.ContrastRatio, ContrastThreshold: p.ContrastThreshold, Regions: append([]catalog.Region(nil), p.Regions...), ImagePNG: append([]byte(nil), imagePNG...), MIMEType: p.MIMEType, ContentHash: p.ContentHash, AssetStudioRef: p.AssetStudioRef}
}

func (s *Store) persist(ctx context.Context, b Backdrop) (Backdrop, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dir, err := s.roots.PickRequired(ctx, storage.ClassData)
	if err != nil {
		return Backdrop{}, err
	}
	if err := storage.EnsureDirectory(dir, 0700); err != nil {
		return Backdrop{}, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return Backdrop{}, err
	}
	defer root.Close()
	if err := root.MkdirAll(releasedBackdropDir, 0700); err != nil {
		return Backdrop{}, err
	}
	finalPath := filepath.Join(releasedBackdropDir, b.ID)
	if existing, err := readDurableBackdrop(root, b.ID); err == nil {
		if existing.ContentHash != b.ContentHash || existing.CandidateID != b.CandidateID || existing.JobID != b.JobID || existing.MIMEType != b.MIMEType || existing.Width != b.Width || existing.Height != b.Height {
			return Backdrop{}, fmt.Errorf("release: immutable backdrop %q conflicts with existing content", b.ID)
		}
		return existing, nil
	} else if !errors.Is(err, errDurableNotFound) && !strings.Contains(err.Error(), "not found") {
		return Backdrop{}, err
	}

	metadata, err := json.Marshal(b.persisted())
	if err != nil {
		return Backdrop{}, fmt.Errorf("release: encode metadata: %w", err)
	}
	metadataHash := sha256.Sum256(metadata)
	metadataHashText := hex.EncodeToString(metadataHash[:])
	temporary := filepath.Join(releasedBackdropDir, fmt.Sprintf(".pending-%s-%d", b.ID, pendingSequence.Add(1)))
	if err := root.RemoveAll(temporary); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Backdrop{}, err
	}
	if err := root.Mkdir(temporary, 0700); err != nil {
		return Backdrop{}, err
	}
	cleanup := func() { _ = root.RemoveAll(temporary) }
	defer cleanup()
	if err := storage.WriteFileAtomicInRoot(root, filepath.Join(temporary, "asset.png"), b.ImagePNG, 0600); err != nil {
		return Backdrop{}, fmt.Errorf("release: write immutable asset: %w", err)
	}
	if err := storage.WriteFileAtomicInRoot(root, filepath.Join(temporary, "metadata.json"), metadata, 0600); err != nil {
		return Backdrop{}, fmt.Errorf("release: write immutable metadata: %w", err)
	}
	if err := storage.WriteFileAtomicInRoot(root, filepath.Join(temporary, "metadata.sha256"), []byte(metadataHashText+"\n"), 0600); err != nil {
		return Backdrop{}, fmt.Errorf("release: write metadata integrity record: %w", err)
	}
	if err := root.Rename(temporary, finalPath); err != nil {
		if existing, readErr := readDurableBackdrop(root, b.ID); readErr == nil && existing.ContentHash == b.ContentHash && existing.JobID == b.JobID && existing.MIMEType == b.MIMEType && existing.Width == b.Width && existing.Height == b.Height {
			return existing, nil
		}
		return Backdrop{}, fmt.Errorf("release: commit immutable backdrop: %w", err)
	}
	s.roots.RecordWrite(ctx)
	return b, nil
}

var errDurableNotFound = errors.New("durable backdrop not found")

func readDurableBackdrop(root *os.Root, id string) (Backdrop, error) {
	base := filepath.Join(releasedBackdropDir, id)
	metadata, err := readRootFile(root, filepath.Join(base, "metadata.json"))
	if errors.Is(err, os.ErrNotExist) {
		return Backdrop{}, errDurableNotFound
	}
	if err != nil {
		return Backdrop{}, err
	}
	checksum, err := readRootFile(root, filepath.Join(base, "metadata.sha256"))
	if err != nil {
		return Backdrop{}, fmt.Errorf("release: metadata integrity record unavailable: %w", err)
	}
	expected := strings.TrimSpace(string(checksum))
	digest := sha256.Sum256(metadata)
	if expected != hex.EncodeToString(digest[:]) {
		return Backdrop{}, fmt.Errorf("release: metadata integrity check failed")
	}
	var persisted persistedBackdrop
	decoder := json.NewDecoder(bytes.NewReader(metadata))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&persisted); err != nil {
		return Backdrop{}, fmt.Errorf("release: decode durable metadata: %w", err)
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Backdrop{}, fmt.Errorf("release: durable metadata contains multiple values")
	}
	if persisted.ID != id || persisted.JobID == "" || persisted.MIMEType != "image/png" || persisted.ByteLength <= 0 || persisted.ContentHash == "" {
		return Backdrop{}, fmt.Errorf("release: durable metadata is incomplete")
	}
	imagePNG, err := readRootFile(root, filepath.Join(base, "asset.png"))
	if err != nil {
		return Backdrop{}, fmt.Errorf("release: released asset unavailable: %w", err)
	}
	if len(imagePNG) != persisted.ByteLength {
		return Backdrop{}, fmt.Errorf("release: released asset integrity check failed: length mismatch")
	}
	assetHash := sha256.Sum256(imagePNG)
	if persisted.ContentHash != hex.EncodeToString(assetHash[:]) {
		return Backdrop{}, fmt.Errorf("release: released asset integrity check failed")
	}
	width, height, format, err := pngMetadata(imagePNG)
	if err != nil || format != "png" || width != persisted.Width || height != persisted.Height {
		return Backdrop{}, fmt.Errorf("release: released asset integrity check failed: metadata mismatch")
	}
	return persisted.backdrop(imagePNG), nil
}

func readRootFile(root *os.Root, path string) ([]byte, error) {
	file, err := root.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func pngMetadata(encoded []byte) (int, int, string, error) {
	config, format, err := image.DecodeConfig(bytes.NewReader(encoded))
	if err != nil {
		return 0, 0, "", err
	}
	return config.Width, config.Height, format, nil
}
