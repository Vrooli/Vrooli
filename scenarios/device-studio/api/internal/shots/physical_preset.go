package shots

import (
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

	"device-studio/internal/library"
	"device-studio/internal/physical"
	"device-studio/internal/revisionstore"
	"github.com/google/uuid"
	physicalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/physical"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const maxPhysicalPresetBytes = 256 << 10

var (
	ErrPhysicalPresetNotFound = errors.New("physical shot preset revision not found")
	ErrPhysicalPresetConflict = errors.New("physical shot preset head conflict")
)

type PhysicalPresetRevision struct {
	PresetID  string
	Revision  uint32
	SHA256    string
	Name      string
	Setup     *shotsv1.PhysicalShotSetup
	CreatedAt time.Time
	canonical []byte
}

type PhysicalPresetValidator interface {
	ValidatePhysicalSetup(context.Context, *shotsv1.PhysicalShotSetup) error
}

type PhysicalPresetRepository interface {
	Save(context.Context, PhysicalPresetRevision, uint32) (PhysicalPresetRevision, error)
	Get(context.Context, string, uint32) (PhysicalPresetRevision, error)
	Head(context.Context, string) (PhysicalPresetRevision, error)
	ListHeads(context.Context, string, int) ([]PhysicalPresetRevision, error)
	ListRevisions(context.Context, string, uint32, int) ([]PhysicalPresetRevision, error)
}

type PhysicalPresetService struct {
	repo      PhysicalPresetRepository
	validator PhysicalPresetValidator
	now       func() time.Time
	newID     func() string
}

func NewPhysicalPresetService(repo PhysicalPresetRepository, validator PhysicalPresetValidator, now func() time.Time, newID func() string) (*PhysicalPresetService, error) {
	if repo == nil || validator == nil {
		return nil, errors.New("physical preset service requires its repository and physical setup validator")
	}
	if now == nil {
		now = time.Now
	}
	if newID == nil {
		newID = uuid.NewString
	}
	return &PhysicalPresetService{repo: repo, validator: validator, now: now, newID: newID}, nil
}

func (s *PhysicalPresetService) Save(ctx context.Context, id string, expected uint32, name string, setup *shotsv1.PhysicalShotSetup) (PhysicalPresetRevision, error) {
	id = strings.TrimSpace(id)
	if id == "" && expected == 0 {
		id = s.newID()
	}
	if !library.IsValidStableID(id) || len(id) > 128 {
		return PhysicalPresetRevision{}, ErrInvalidPhysicalBinding{Reason: "preset ID must be a stable identifier"}
	}
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 {
		return PhysicalPresetRevision{}, ErrInvalidPhysicalBinding{Reason: "preset name must contain 1 to 120 characters"}
	}
	canonicalSetup, setupJSON, err := canonicalPhysicalSetup(setup)
	if err != nil {
		return PhysicalPresetRevision{}, ErrInvalidPhysicalBinding{Reason: err.Error()}
	}
	if err := s.validator.ValidatePhysicalSetup(ctx, canonicalSetup); err != nil {
		return PhysicalPresetRevision{}, err
	}
	data, err := json.Marshal(physicalPresetSnapshot{PresetID: id, Name: name, Setup: setupJSON})
	if err != nil {
		return PhysicalPresetRevision{}, fmt.Errorf("encode physical preset: %w", err)
	}
	if len(data) > maxPhysicalPresetBytes {
		return PhysicalPresetRevision{}, ErrInvalidPhysicalBinding{Reason: "physical setup preset exceeds its size limit"}
	}
	sum := sha256.Sum256(data)
	proposed := PhysicalPresetRevision{PresetID: id, SHA256: hex.EncodeToString(sum[:]), Name: name, Setup: canonicalSetup, CreatedAt: s.now().UTC(), canonical: data}
	return s.repo.Save(ctx, proposed, expected)
}

func (s *PhysicalPresetService) Get(ctx context.Context, id string, revision uint32) (PhysicalPresetRevision, error) {
	return s.repo.Get(ctx, strings.TrimSpace(id), revision)
}

func (s *PhysicalPresetService) Head(ctx context.Context, id string) (PhysicalPresetRevision, error) {
	id = strings.TrimSpace(id)
	if !library.IsValidStableID(id) {
		return PhysicalPresetRevision{}, ErrInvalidPhysicalBinding{Reason: "preset ID must be a stable identifier"}
	}
	return s.repo.Head(ctx, id)
}

func (s *PhysicalPresetService) ListHeads(ctx context.Context, after string, limit int) ([]PhysicalPresetRevision, error) {
	if limit < 1 || limit > 100 || (after != "" && (!library.IsValidStableID(after) || len(after) > 128)) {
		return nil, ErrInvalidPhysicalBinding{Reason: "preset page requires a valid cursor and page size from 1 to 100"}
	}
	return s.repo.ListHeads(ctx, after, limit)
}

func (s *PhysicalPresetService) ListRevisions(ctx context.Context, id string, after uint32, limit int) ([]PhysicalPresetRevision, error) {
	id = strings.TrimSpace(id)
	if !library.IsValidStableID(id) || limit < 1 || limit > 100 {
		return nil, ErrInvalidPhysicalBinding{Reason: "preset revision page requires a stable ID and page size from 1 to 100"}
	}
	return s.repo.ListRevisions(ctx, id, after, limit)
}

type physicalPresetSnapshot struct {
	PresetID string          `json:"preset_id"`
	Name     string          `json:"name"`
	Setup    json.RawMessage `json:"setup"`
}

func canonicalPhysicalSetup(input *shotsv1.PhysicalShotSetup) (*shotsv1.PhysicalShotSetup, []byte, error) {
	if input == nil || input.GetDevice() == nil || input.GetScene() == nil || input.GetMotion() == nil {
		return nil, nil, errors.New("device, scene, and motion revision references are required")
	}
	setup := proto.Clone(input).(*shotsv1.PhysicalShotSetup)
	for label, ref := range map[string]*shotsv1.ComponentRevisionRef{"device": setup.GetDevice(), "scene": setup.GetScene(), "motion": setup.GetMotion()} {
		if !library.IsValidStableID(ref.GetId()) || ref.GetRevision() == 0 || !isSHA256(ref.GetSha256()) {
			return nil, nil, fmt.Errorf("%s revision reference must contain a stable ID, positive revision, and lowercase SHA-256", label)
		}
	}
	for label, id := range map[string]string{"display": setup.GetDisplayId(), "device support": setup.GetDeviceSupportId(), "scene support": setup.GetSceneSupportId(), "camera": setup.GetCameraId()} {
		if !library.IsValidStableID(id) {
			return nil, nil, fmt.Errorf("%s selection must be a stable identifier", label)
		}
	}
	if setup.GetSceneSupportWorld() == nil || setup.GetDeviceRootRest() == nil || setup.GetSupportPivotM() == nil {
		return nil, nil, errors.New("explicit scene support, support pivot, and device rest transforms are required")
	}
	for label, transform := range map[string]*physicalv1.TRS{"scene support": setup.GetSceneSupportWorld(), "device rest": setup.GetDeviceRootRest()} {
		canonical, err := fromProtoTRS(transform)
		if err != nil {
			return nil, nil, fmt.Errorf("%s transform: %w", label, err)
		}
		setProtoTRS(transform, canonical)
	}
	pivot := setup.GetSupportPivotM()
	if err := (physical.Vec3{X: pivot.GetX(), Y: pivot.GetY(), Z: pivot.GetZ()}).Validate(); err != nil {
		return nil, nil, errors.New("support pivot must be finite")
	}
	if len(setup.GetStillProfiles())+len(setup.GetMotionProfiles()) == 0 || len(setup.GetStillProfiles()) > 2 || len(setup.GetMotionProfiles()) > 2 {
		return nil, nil, errors.New("one or more supported output profiles are required within the finite profile limits")
	}
	seenStill, seenMotion := map[shotsv1.PhysicalStillOutputProfile]bool{}, map[shotsv1.PhysicalMotionOutputProfile]bool{}
	for _, profile := range setup.GetStillProfiles() {
		if profile != shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1 && profile != shotsv1.PhysicalStillOutputProfile_PHYSICAL_STILL_OUTPUT_PROFILE_3840X2160_V1 {
			return nil, nil, errors.New("unsupported still output profile")
		}
		if seenStill[profile] {
			return nil, nil, errors.New("duplicate still output profile")
		}
		seenStill[profile] = true
	}
	for _, profile := range setup.GetMotionProfiles() {
		if profile != shotsv1.PhysicalMotionOutputProfile_PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V1 && profile != shotsv1.PhysicalMotionOutputProfile_PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V2 {
			return nil, nil, errors.New("unsupported motion output profile")
		}
		if seenMotion[profile] {
			return nil, nil, errors.New("duplicate motion output profile")
		}
		seenMotion[profile] = true
	}
	sort.Slice(setup.StillProfiles, func(i, j int) bool { return setup.StillProfiles[i] < setup.StillProfiles[j] })
	sort.Slice(setup.MotionProfiles, func(i, j int) bool { return setup.MotionProfiles[i] < setup.MotionProfiles[j] })
	data, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(setup)
	if err != nil {
		return nil, nil, errors.New("physical setup cannot be canonically encoded within its size limit")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, data); err != nil {
		return nil, nil, errors.New("physical setup cannot be canonically compacted")
	}
	if compact.Len() > maxPhysicalPresetBytes {
		return nil, nil, errors.New("physical setup cannot be canonically encoded within its size limit")
	}
	return setup, compact.Bytes(), nil
}

func setProtoTRS(target *physicalv1.TRS, value physical.TRS) {
	target.TranslationM.X, target.TranslationM.Y, target.TranslationM.Z = value.Translation.X, value.Translation.Y, value.Translation.Z
	target.RotationXyzw.X, target.RotationXyzw.Y, target.RotationXyzw.Z, target.RotationXyzw.W = value.Rotation.X, value.Rotation.Y, value.Rotation.Z, value.Rotation.W
	target.Scale.X, target.Scale.Y, target.Scale.Z = value.Scale.X, value.Scale.Y, value.Scale.Z
}

func isSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && hex.EncodeToString(decoded) == value
}

type sqlitePhysicalPresetRepository struct {
	store *revisionstore.SQLite[physicalPresetSnapshot]
}

func NewSQLitePhysicalPresetRepository(db revisionstore.Database) (PhysicalPresetRepository, error) {
	store, err := revisionstore.NewSQLite[physicalPresetSnapshot](db, revisionstore.PhysicalShotPresetRevisions, revisionstore.Codec[physicalPresetSnapshot]{
		ID: func(value physicalPresetSnapshot) string { return value.PresetID },
		Decode: func(data []byte) (physicalPresetSnapshot, string, error) {
			return decodePhysicalPresetSnapshot(data)
		},
	})
	if err != nil {
		return nil, err
	}
	return &sqlitePhysicalPresetRepository{store: store}, nil
}

func (r *sqlitePhysicalPresetRepository) Save(ctx context.Context, proposed PhysicalPresetRevision, expected uint32) (PhysicalPresetRevision, error) {
	if expected == ^uint32(0) {
		return PhysicalPresetRevision{}, ErrInvalidPhysicalBinding{Reason: "preset revision number is exhausted"}
	}
	validated, err := decodePhysicalPreset(proposed.PresetID, expected+1, proposed.SHA256, proposed.canonical, proposed.CreatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return PhysicalPresetRevision{}, fmt.Errorf("physical preset repository input cannot be decoded: %w", err)
	}
	if validated.Name != proposed.Name || !proto.Equal(validated.Setup, proposed.Setup) {
		return PhysicalPresetRevision{}, errors.New("physical preset repository input is not canonical")
	}
	_, setupJSON, err := canonicalPhysicalSetup(validated.Setup)
	if err != nil {
		return PhysicalPresetRevision{}, errors.New("physical preset repository setup is not canonical")
	}
	stored, err := r.store.Save(ctx, revisionstore.Record[physicalPresetSnapshot]{
		ID: proposed.PresetID, SHA256: proposed.SHA256,
		Value:     physicalPresetSnapshot{PresetID: proposed.PresetID, Name: proposed.Name, Setup: append(json.RawMessage(nil), setupJSON...)},
		Canonical: append([]byte(nil), proposed.canonical...), CreatedAt: proposed.CreatedAt,
	}, expected)
	if errors.Is(err, revisionstore.ErrConflict) {
		return PhysicalPresetRevision{}, fmt.Errorf("%w: %v", ErrPhysicalPresetConflict, err)
	}
	if err != nil {
		return PhysicalPresetRevision{}, err
	}
	return physicalPresetFromRecord(stored)
}

func (r *sqlitePhysicalPresetRepository) Get(ctx context.Context, id string, revision uint32) (PhysicalPresetRevision, error) {
	if !library.IsValidStableID(id) || revision == 0 {
		return PhysicalPresetRevision{}, ErrPhysicalPresetNotFound
	}
	stored, err := r.store.Get(ctx, id, revision)
	if errors.Is(err, revisionstore.ErrNotFound) {
		return PhysicalPresetRevision{}, ErrPhysicalPresetNotFound
	}
	if err != nil {
		return PhysicalPresetRevision{}, err
	}
	return physicalPresetFromRecord(stored)
}

func (r *sqlitePhysicalPresetRepository) Head(ctx context.Context, id string) (PhysicalPresetRevision, error) {
	stored, err := r.store.Head(ctx, id)
	if errors.Is(err, revisionstore.ErrNotFound) {
		return PhysicalPresetRevision{}, ErrPhysicalPresetNotFound
	}
	if err != nil {
		return PhysicalPresetRevision{}, err
	}
	return physicalPresetFromRecord(stored)
}

func (r *sqlitePhysicalPresetRepository) ListHeads(ctx context.Context, after string, limit int) ([]PhysicalPresetRevision, error) {
	stored, err := r.store.ListHeads(ctx, after, limit)
	if err != nil {
		return nil, err
	}
	result := make([]PhysicalPresetRevision, 0, len(stored))
	for _, item := range stored {
		revision, err := physicalPresetFromRecord(item)
		if err != nil {
			return nil, err
		}
		result = append(result, revision)
	}
	return result, nil
}

func (r *sqlitePhysicalPresetRepository) ListRevisions(ctx context.Context, id string, after uint32, limit int) ([]PhysicalPresetRevision, error) {
	stored, err := r.store.List(ctx, id, after, limit)
	if err != nil {
		return nil, err
	}
	result := make([]PhysicalPresetRevision, 0, len(stored))
	for _, item := range stored {
		revision, err := physicalPresetFromRecord(item)
		if err != nil {
			return nil, err
		}
		result = append(result, revision)
	}
	return result, nil
}

func physicalPresetFromRecord(stored revisionstore.Record[physicalPresetSnapshot]) (PhysicalPresetRevision, error) {
	var setup shotsv1.PhysicalShotSetup
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(stored.Value.Setup, &setup); err != nil {
		return PhysicalPresetRevision{}, fmt.Errorf("decode stored physical preset setup: %w", err)
	}
	canonical, _, err := canonicalPhysicalSetup(&setup)
	if err != nil {
		return PhysicalPresetRevision{}, fmt.Errorf("validate stored physical preset setup: %w", err)
	}
	return PhysicalPresetRevision{PresetID: stored.ID, Revision: stored.Revision, SHA256: stored.SHA256, Name: stored.Value.Name, Setup: canonical, CreatedAt: stored.CreatedAt.UTC(), canonical: append([]byte(nil), stored.Canonical...)}, nil
}

func decodePhysicalPresetSnapshot(data []byte) (physicalPresetSnapshot, string, error) {
	if len(data) == 0 || len(data) > maxPhysicalPresetBytes {
		return physicalPresetSnapshot{}, "", errors.New("stored physical preset size is invalid")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var snapshot physicalPresetSnapshot
	if err := decoder.Decode(&snapshot); err != nil {
		return physicalPresetSnapshot{}, "", err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return physicalPresetSnapshot{}, "", errors.New("physical preset contains trailing JSON")
	}
	if !library.IsValidStableID(snapshot.PresetID) || len(snapshot.PresetID) > 128 || strings.TrimSpace(snapshot.Name) == "" || len(snapshot.Name) > 120 || len(snapshot.Setup) == 0 {
		return physicalPresetSnapshot{}, "", errors.New("stored physical preset identity is invalid")
	}
	var setup shotsv1.PhysicalShotSetup
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(snapshot.Setup, &setup); err != nil {
		return physicalPresetSnapshot{}, "", err
	}
	_, setupJSON, err := canonicalPhysicalSetup(&setup)
	if err != nil {
		return physicalPresetSnapshot{}, "", err
	}
	if !bytes.Equal(setupJSON, snapshot.Setup) {
		return physicalPresetSnapshot{}, "", errors.New("stored physical preset setup is not canonical")
	}
	canonical, err := json.Marshal(physicalPresetSnapshot{PresetID: snapshot.PresetID, Name: snapshot.Name, Setup: setupJSON})
	if err != nil || !bytes.Equal(canonical, data) {
		return physicalPresetSnapshot{}, "", errors.New("stored physical preset is not canonical")
	}
	sum := sha256.Sum256(data)
	snapshot.Setup = append(json.RawMessage(nil), setupJSON...)
	return snapshot, hex.EncodeToString(sum[:]), nil
}

func decodePhysicalPreset(id string, revision uint32, digest string, data []byte, at string) (PhysicalPresetRevision, error) {
	snapshot, actualDigest, err := decodePhysicalPresetSnapshot(data)
	if err != nil {
		return PhysicalPresetRevision{}, fmt.Errorf("decode stored physical preset snapshot: %w", err)
	}
	if actualDigest != digest {
		return PhysicalPresetRevision{}, errors.New("stored physical preset digest mismatch")
	}
	if snapshot.PresetID != id {
		return PhysicalPresetRevision{}, errors.New("stored physical preset identity mismatch")
	}
	var setup shotsv1.PhysicalShotSetup
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(snapshot.Setup, &setup); err != nil {
		return PhysicalPresetRevision{}, err
	}
	canonical, _, err := canonicalPhysicalSetup(&setup)
	if err != nil {
		return PhysicalPresetRevision{}, err
	}
	created, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return PhysicalPresetRevision{}, errors.New("stored physical preset timestamp is invalid")
	}
	return PhysicalPresetRevision{PresetID: id, Revision: revision, SHA256: digest, Name: snapshot.Name, Setup: canonical, CreatedAt: created.UTC(), canonical: append([]byte(nil), data...)}, nil
}
