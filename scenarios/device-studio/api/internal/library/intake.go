package library

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"hash"
	"io"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"device-studio/internal/revisionstore"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/blobstore"
)

const (
	MaxIntakeFilenameBytes = 200
	MaxIntakeClaimBytes    = 1024
	MaxIntakePageSize      = 100
)

var (
	ErrIntakeNotFound = errors.New("intake candidate not found")
	ErrIntakeConflict = errors.New("intake candidate id already has different content or claims")
	ErrInvalidIntake  = errors.New("invalid local geometry intake")
	ErrIntakeTooLarge = errors.New("intake GLB exceeds the geometry size limit")
)

// IntakeClaims are user statements. They are never treated as verified rights.
type IntakeClaims struct {
	CandidateID    string   `json:"candidate_id"`
	DisplayName    string   `json:"display_name"`
	SourceLabel    string   `json:"source_label"`
	LicenseClaim   string   `json:"license_claim"`
	Attribution    string   `json:"attribution"`
	DeclaredUses   []string `json:"declared_uses,omitempty"`
	ParentID       string   `json:"parent_candidate_id,omitempty"`
	ParentRevision uint32   `json:"parent_revision,omitempty"`
	ParentSHA256   string   `json:"parent_sha256,omitempty"`
}

type IntakeInspection struct {
	Profile        string `json:"profile"`
	ProfileVersion uint32 `json:"profile_version"`
	Supported      bool   `json:"supported"`
	NodeCount      uint32 `json:"node_count,omitempty"`
	MeshCount      uint32 `json:"mesh_count,omitempty"`
	TriangleCount  uint64 `json:"triangle_count,omitempty"`
	Finding        string `json:"finding,omitempty"`
}

type IntakeInspector func(context.Context, []byte) (IntakeInspection, error)

type IntakeCandidate struct {
	ID             string           `json:"id"`
	Revision       uint32           `json:"revision"`
	RevisionSHA256 string           `json:"revision_sha256"`
	DisplayName    string           `json:"display_name"`
	OriginalName   string           `json:"original_filename"`
	OriginalSHA256 string           `json:"original_sha256"`
	SizeBytes      int64            `json:"size_bytes"`
	MIME           string           `json:"mime"`
	ImportedAt     time.Time        `json:"imported_at"`
	ImportMethod   string           `json:"import_method"`
	SourceLabel    string           `json:"source_label"`
	LicenseClaim   string           `json:"license_claim"`
	Attribution    string           `json:"attribution"`
	DeclaredUses   []string         `json:"declared_uses,omitempty"`
	ParentID       string           `json:"parent_candidate_id,omitempty"`
	ParentRevision uint32           `json:"parent_revision,omitempty"`
	ParentSHA256   string           `json:"parent_sha256,omitempty"`
	Inspection     IntakeInspection `json:"inspection"`
	ReviewStatus   string           `json:"review_status"`
	RenderStatus   string           `json:"render_status"`
	Creation       *CreationLineage `json:"creation_lineage,omitempty"`
}

// CreationLineage records model-generated inputs without granting review,
// rights, or publication authority to the generator.
type CreationLineage struct {
	JobID                  string                      `json:"job_id"`
	RecipeSHA256           string                      `json:"recipe_sha256"`
	ReferenceSHA256        []string                    `json:"reference_sha256"`
	Provider               string                      `json:"provider,omitempty"`
	Model                  string                      `json:"model,omitempty"`
	ProviderCalls          int32                       `json:"provider_calls_made"`
	InputTokens            int64                       `json:"input_tokens,omitempty"`
	OutputTokens           int64                       `json:"output_tokens,omitempty"`
	CostMicros             int64                       `json:"cost_micros,omitempty"`
	InputTokensProvenance  inferencev1.UsageProvenance `json:"input_tokens_provenance"`
	OutputTokensProvenance inferencev1.UsageProvenance `json:"output_tokens_provenance"`
	CostProvenance         inferencev1.UsageProvenance `json:"cost_provenance"`
	AppliedSettings        *sharedv1.AppliedSettings   `json:"applied_settings"`
	InferenceOutcomeSHA256 string                      `json:"inference_outcome_sha256"`
}

type intakeRecord struct {
	Candidate IntakeCandidate `json:"candidate"`
	BlobKey   string          `json:"blob_key"`
}

type intakeRevisionRepository interface {
	Save(context.Context, revisionstore.Record[intakeRecord], uint32) (revisionstore.Record[intakeRecord], error)
	Get(context.Context, string, uint32) (revisionstore.Record[intakeRecord], error)
	ListHeads(context.Context, string, int) ([]revisionstore.Record[intakeRecord], error)
}

type IntakeService struct {
	repo    intakeRevisionRepository
	store   blobstore.BlobStore
	inspect IntakeInspector
	now     func() time.Time
	newID   func() string
}

func NewIntakeService(db revisionstore.Database, store blobstore.BlobStore, inspect IntakeInspector, now func() time.Time, newID func() string) (*IntakeService, error) {
	if db == nil || store == nil || inspect == nil {
		return nil, errors.New("intake requires routed SQL, blob storage, and a bounded GLB inspector")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = uuid.NewString
	}
	repo, err := revisionstore.NewSQLite(db, revisionstore.LibraryIntakeRevisions, revisionstore.Codec[intakeRecord]{ID: func(v intakeRecord) string { return v.Candidate.ID }, Decode: decodeIntakeRecord})
	if err != nil {
		return nil, err
	}
	return &IntakeService{repo: repo, store: store, inspect: inspect, now: now, newID: newID}, nil
}

type StagedIntakeImport interface {
	Candidate() IntakeCandidate
	Commit(context.Context) (IntakeCandidate, error)
	Abort(context.Context) error
}

func (s *IntakeService) BeginImport(ctx context.Context, claims IntakeClaims, filename string, declaredSize int64, body io.Reader) (StagedIntakeImport, error) {
	claims, filename, err := validateIntakeClaims(claims, filename)
	if err != nil {
		return nil, err
	}
	if claims.ParentID != "" {
		parent, parentErr := s.repo.Get(ctx, claims.ParentID, claims.ParentRevision)
		if errors.Is(parentErr, revisionstore.ErrNotFound) {
			return nil, fmt.Errorf("%w: parent candidate revision or digest does not resolve", ErrInvalidIntake)
		}
		if parentErr != nil {
			return nil, fmt.Errorf("resolve intake parent revision: %w", parentErr)
		}
		if parent.Value.Candidate.RevisionSHA256 != claims.ParentSHA256 {
			return nil, fmt.Errorf("%w: parent candidate revision or digest does not resolve", ErrInvalidIntake)
		}
	}
	if declaredSize > MaxGeometryBytes {
		return nil, fmt.Errorf("%w: maximum is %d bytes", ErrIntakeTooLarge, MaxGeometryBytes)
	}
	if body == nil || declaredSize == 0 || declaredSize < -1 {
		return nil, fmt.Errorf("%w: file is empty or declared size is invalid", ErrInvalidIntake)
	}
	keyToken := strings.TrimSpace(s.newID())
	if !safeID.MatchString(keyToken) {
		return nil, errors.New("intake staging id source returned an invalid id")
	}
	key := "intake/candidates/" + claims.CandidateID + "/staged/" + keyToken + ".glb"
	stream := &intakeUploadReader{ctx: ctx, reader: body, max: MaxGeometryBytes, hash: sha256.New()}
	if err := s.store.Put(ctx, key, stream, "model/gltf-binary"); err != nil {
		return nil, errors.Join(fmt.Errorf("stage GLB bytes: %w", err), s.store.Delete(ctx, key))
	}
	if !stream.eof || stream.count == 0 || (declaredSize >= 0 && stream.count != declaredSize) {
		return nil, errors.Join(fmt.Errorf("%w: uploaded size did not match the complete file", ErrInvalidIntake), s.store.Delete(ctx, key))
	}
	data, err := readVerifiedIntakeBlob(ctx, s.store, key, stream.count, hex.EncodeToString(stream.hash.Sum(nil)))
	if err != nil {
		return nil, errors.Join(fmt.Errorf("verify staged GLB: %w", err), s.store.Delete(ctx, key))
	}
	if !validGLBContainer(data) {
		return nil, errors.Join(fmt.Errorf("%w: file is not a complete glTF binary 2.0 container", ErrInvalidIntake), s.store.Delete(ctx, key))
	}
	inspection, inspectErr := s.inspect(ctx, data)
	if inspectErr != nil {
		if errors.Is(inspectErr, context.Canceled) || errors.Is(inspectErr, context.DeadlineExceeded) {
			return nil, errors.Join(inspectErr, s.store.Delete(ctx, key))
		}
		inspection = IntakeInspection{Profile: "device-studio-physical-glb-v1", ProfileVersion: 1, Supported: false, Finding: "unsupported_by_current_geometry_inspector"}
	}
	inspection = normalizeIntakeInspection(inspection)
	candidate := IntakeCandidate{
		ID: claims.CandidateID, Revision: 1, DisplayName: claims.DisplayName, OriginalName: filename,
		OriginalSHA256: hex.EncodeToString(stream.hash.Sum(nil)), SizeBytes: stream.count, MIME: "model/gltf-binary",
		ImportedAt: s.now().UTC(), ImportMethod: "local_upload", SourceLabel: claims.SourceLabel,
		LicenseClaim: claims.LicenseClaim, Attribution: claims.Attribution, DeclaredUses: claims.DeclaredUses,
		ParentID: claims.ParentID, ParentRevision: claims.ParentRevision, ParentSHA256: claims.ParentSHA256,
		Inspection: inspection, ReviewStatus: "unreviewed", RenderStatus: "not_reviewed",
	}
	candidate.RevisionSHA256, err = intakeCandidateDigest(candidate)
	if err != nil {
		return nil, errors.Join(err, s.store.Delete(ctx, key))
	}
	return &stagedIntake{service: s, record: intakeRecord{Candidate: candidate, BlobKey: key}}, nil
}

func (s *IntakeService) Get(ctx context.Context, id string, revision uint32) (IntakeCandidate, error) {
	if !safeID.MatchString(id) || revision == 0 {
		return IntakeCandidate{}, ErrIntakeNotFound
	}
	row, err := s.repo.Get(ctx, id, revision)
	if errors.Is(err, revisionstore.ErrNotFound) {
		return IntakeCandidate{}, ErrIntakeNotFound
	}
	if err != nil {
		return IntakeCandidate{}, err
	}
	return row.Value.Candidate, nil
}

func (s *IntakeService) List(ctx context.Context, afterID string, limit int) ([]IntakeCandidate, error) {
	if (afterID != "" && !safeID.MatchString(afterID)) || limit < 1 || limit > MaxIntakePageSize {
		return nil, fmt.Errorf("%w: invalid cursor or page size", ErrInvalidIntake)
	}
	rows, err := s.repo.ListHeads(ctx, afterID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]IntakeCandidate, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Value.Candidate)
	}
	return out, nil
}

func (s *IntakeService) Open(ctx context.Context, id string, revision uint32) (IntakeCandidate, io.ReadCloser, error) {
	if !safeID.MatchString(id) || revision == 0 {
		return IntakeCandidate{}, nil, ErrIntakeNotFound
	}
	row, err := s.repo.Get(ctx, id, revision)
	if errors.Is(err, revisionstore.ErrNotFound) {
		return IntakeCandidate{}, nil, ErrIntakeNotFound
	}
	if err != nil {
		return IntakeCandidate{}, nil, err
	}
	r, _, err := s.store.Get(ctx, row.Value.BlobKey)
	if err != nil {
		return IntakeCandidate{}, nil, err
	}
	c := row.Value.Candidate
	return c, &verifiedReader{ctx: ctx, reader: r, expectedSize: c.SizeBytes, expectedSHA: c.OriginalSHA256, maxSize: MaxGeometryBytes, hash: sha256.New()}, nil
}

type stagedIntake struct {
	service *IntakeService
	record  intakeRecord
	done    bool
}

func (s *stagedIntake) Candidate() IntakeCandidate { return s.record.Candidate }
func (s *stagedIntake) Abort(ctx context.Context) error {
	if s.done {
		return nil
	}
	s.done = true
	return s.service.store.Delete(ctx, s.record.BlobKey)
}
func (s *stagedIntake) Commit(ctx context.Context) (IntakeCandidate, error) {
	if s.done {
		return IntakeCandidate{}, errors.New("staged intake import is already finalized")
	}
	s.done = true
	canonical, digest, err := encodeIntakeRecord(s.record)
	if err != nil {
		return IntakeCandidate{}, errors.Join(err, s.service.store.Delete(ctx, s.record.BlobKey))
	}
	row, err := s.service.repo.Save(ctx, revisionstore.Record[intakeRecord]{ID: s.record.Candidate.ID, SHA256: digest, Value: s.record, Canonical: canonical, CreatedAt: s.record.Candidate.ImportedAt}, 0)
	if err == nil {
		return row.Value.Candidate, nil
	}
	// A competing exact import may have committed with another unique staged key.
	// Readback decides whether our object is unreferenced; uncertain reads retain it.
	readCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	prior, readErr := s.service.repo.Get(readCtx, s.record.Candidate.ID, 1)
	if readErr == nil {
		if intakeCandidatesEqual(prior.Value.Candidate, s.record.Candidate) {
			if prior.Value.BlobKey == s.record.BlobKey {
				return prior.Value.Candidate, nil
			}
			cleanupErr := s.service.store.Delete(ctx, s.record.BlobKey)
			return prior.Value.Candidate, cleanupErr
		}
		return IntakeCandidate{}, errors.Join(ErrIntakeConflict, s.service.store.Delete(ctx, s.record.BlobKey))
	}
	return IntakeCandidate{}, errors.Join(fmt.Errorf("commit intake metadata; staged bytes retained because outcome is uncertain: %w", err), readErr)
}

type intakeUploadReader struct {
	ctx        context.Context
	reader     io.Reader
	max, count int64
	hash       hash.Hash
	eof        bool
}

func (r *intakeUploadReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	if r.count > r.max {
		return 0, ErrIntakeTooLarge
	}
	if int64(len(p)) > r.max+1-r.count {
		p = p[:r.max+1-r.count]
	}
	n, err := r.reader.Read(p)
	if n > 0 {
		r.count += int64(n)
		_, _ = r.hash.Write(p[:n])
	}
	if r.count > r.max {
		return n, ErrIntakeTooLarge
	}
	if errors.Is(err, io.EOF) {
		r.eof = true
	}
	return n, err
}

func validateIntakeClaims(in IntakeClaims, filename string) (IntakeClaims, string, error) {
	// Normalize a private copy: callers may safely reuse claims across retries.
	in.DeclaredUses = append([]string(nil), in.DeclaredUses...)
	in.CandidateID = strings.TrimSpace(in.CandidateID)
	if !safeID.MatchString(in.CandidateID) {
		return in, "", fmt.Errorf("%w: candidate_id must be a stable idempotency key", ErrInvalidIntake)
	}
	filename = filepath.Base(strings.TrimSpace(filename))
	if filename == "." || filename == "" || len(filename) > MaxIntakeFilenameBytes || strings.ContainsAny(filename, "/\\") {
		return in, "", fmt.Errorf("%w: invalid original filename", ErrInvalidIntake)
	}
	for _, r := range filename {
		if unicode.IsControl(r) {
			return in, "", fmt.Errorf("%w: invalid original filename", ErrInvalidIntake)
		}
	}
	for _, field := range []string{in.DisplayName, in.SourceLabel, in.LicenseClaim, in.Attribution} {
		if len(field) > MaxIntakeClaimBytes || strings.IndexFunc(field, unicode.IsControl) >= 0 {
			return in, "", fmt.Errorf("%w: provenance claim is invalid or too long", ErrInvalidIntake)
		}
	}
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.SourceLabel = strings.TrimSpace(in.SourceLabel)
	if in.DisplayName == "" || in.SourceLabel == "" {
		return in, "", fmt.Errorf("%w: display_name and source_label are required", ErrInvalidIntake)
	}
	if (in.ParentID == "") != (in.ParentRevision == 0) || (in.ParentID == "") != (in.ParentSHA256 == "") || (in.ParentID != "" && (!safeID.MatchString(in.ParentID) || !sha256Pattern.MatchString(in.ParentSHA256))) {
		return in, "", fmt.Errorf("%w: parent lineage must bind an id, revision, and digest", ErrInvalidIntake)
	}
	if in.ParentID == in.CandidateID {
		return in, "", fmt.Errorf("%w: candidate cannot be its own parent", ErrInvalidIntake)
	}
	seen := map[string]bool{}
	if len(in.DeclaredUses) > 4 {
		return in, "", fmt.Errorf("%w: at most four declared uses are allowed", ErrInvalidIntake)
	}
	for i, use := range in.DeclaredUses {
		use = strings.TrimSpace(use)
		if len(use) > 64 || use == "external_ai_upload" {
			return in, "", fmt.Errorf("%w: external AI upload is not an import claim", ErrInvalidIntake)
		}
		if use == "" || seen[use] {
			return in, "", fmt.Errorf("%w: declared uses must be unique non-empty claims", ErrInvalidIntake)
		}
		seen[use] = true
		in.DeclaredUses[i] = use
	}
	return in, filename, nil
}

func normalizeIntakeInspection(in IntakeInspection) IntakeInspection {
	if in.Profile == "" || len(in.Profile) > 96 || in.ProfileVersion == 0 {
		in.Profile = "device-studio-physical-glb-v1"
		in.ProfileVersion = 1
	}
	if in.Supported {
		in.Finding = ""
	} else if in.Finding == "" || len(in.Finding) > 96 {
		in.Finding = "unsupported_by_current_geometry_inspector"
	}
	return in
}

func intakeCandidateDigest(c IntakeCandidate) (string, error) {
	c.RevisionSHA256 = ""
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
func encodeIntakeRecord(r intakeRecord) ([]byte, string, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(b)
	return b, hex.EncodeToString(sum[:]), nil
}
func decodeIntakeRecord(data []byte) (intakeRecord, string, error) {
	var r intakeRecord
	if len(data) == 0 || len(data) > 1<<20 {
		return r, "", errors.New("stored intake record exceeds bounds")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		return r, "", err
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		return r, "", errors.New("stored intake record has trailing data")
	}
	if !safeID.MatchString(r.Candidate.ID) || r.Candidate.Revision != 1 || r.BlobKey == "" || !strings.HasPrefix(r.BlobKey, "intake/candidates/"+r.Candidate.ID+"/staged/") || !sha256Pattern.MatchString(r.Candidate.OriginalSHA256) || r.Candidate.SizeBytes <= 0 || r.Candidate.SizeBytes > MaxGeometryBytes {
		return r, "", errors.New("stored intake identity is invalid")
	}
	if r.Candidate.RevisionSHA256 != mustIntakeDigest(r.Candidate) || r.Candidate.ReviewStatus != "unreviewed" || r.Candidate.RenderStatus != "not_reviewed" {
		return r, "", errors.New("stored intake status or digest is invalid")
	}
	b, sha, err := encodeIntakeRecord(r)
	if err != nil {
		return r, "", err
	}
	if !bytes.Equal(b, data) {
		return r, "", errors.New("stored intake record is not canonical")
	}
	return r, sha, nil
}
func mustIntakeDigest(c IntakeCandidate) string { v, _ := intakeCandidateDigest(c); return v }
func intakeCandidatesEqual(a, b IntakeCandidate) bool {
	a.RevisionSHA256 = ""
	b.RevisionSHA256 = ""
	a.ImportedAt = time.Time{}
	b.ImportedAt = time.Time{}
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}

func readVerifiedIntakeBlob(ctx context.Context, store blobstore.BlobStore, key string, size int64, want string) ([]byte, error) {
	r, _, err := store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if size <= 0 || size > MaxGeometryBytes {
		return nil, fmt.Errorf("intake size is outside the geometry limit")
	}
	b, err := io.ReadAll(io.LimitReader(r, MaxGeometryBytes+1))
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(b)
	if int64(len(b)) != size || hex.EncodeToString(sum[:]) != want {
		return nil, errors.New("intake staged bytes failed size or digest verification")
	}
	return b, nil
}

func validGLBContainer(data []byte) bool {
	if len(data) < 21 || string(data[:4]) != "glTF" || binary.LittleEndian.Uint32(data[4:8]) != 2 || uint64(binary.LittleEndian.Uint32(data[8:12])) != uint64(len(data)) {
		return false
	}
	position, chunk := 12, 0
	for position < len(data) {
		if len(data)-position < 8 {
			return false
		}
		length := uint64(binary.LittleEndian.Uint32(data[position : position+4]))
		typ := binary.LittleEndian.Uint32(data[position+4 : position+8])
		position += 8
		if length == 0 || length%4 != 0 || length > uint64(len(data)-position) {
			return false
		}
		if chunk == 0 {
			if typ != 0x4e4f534a { // JSON
				return false
			}
			payload := bytes.TrimRight(data[position:position+int(length)], " \t\r\n\x00")
			var object map[string]json.RawMessage
			if json.Unmarshal(payload, &object) != nil || object == nil {
				return false
			}
		} else if typ != 0x004e4942 { // BIN
			return false
		}
		position += int(length)
		chunk++
	}
	return position == len(data) && chunk > 0
}
