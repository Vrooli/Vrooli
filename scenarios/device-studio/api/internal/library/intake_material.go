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
	"io"
	"math"
	"reflect"
	"strings"
	"unicode"

	"device-studio/internal/revisionstore"
)

const maxMaterialCount = 256
const maxMaterialNameBytes = 256

type IntakeMaterial struct {
	Index           uint32     `json:"index"`
	Name            string     `json:"name"`
	BaseColorFactor [4]float64 `json:"base_color_factor"`
	MetallicFactor  float64    `json:"metallic_factor"`
	RoughnessFactor float64    `json:"roughness_factor"`
	EmissiveFactor  [3]float64 `json:"emissive_factor"`
	Editable        bool       `json:"editable"`
}

type IntakeMaterialInventory struct {
	CandidateID       string                     `json:"candidate_id"`
	Revision          uint32                     `json:"revision"`
	RevisionSHA256    string                     `json:"revision_sha256"`
	GLBSHA256         string                     `json:"glb_sha256"`
	Materials         []IntakeMaterial           `json:"materials"`
	CorrectionLineage *MaterialCorrectionLineage `json:"material_correction_lineage,omitempty"`
}

type MaterialCorrectionRequest struct {
	ParentCandidateID    string      `json:"parent_candidate_id"`
	ParentRevision       uint32      `json:"parent_revision"`
	ParentRevisionSHA256 string      `json:"parent_revision_sha256"`
	ChildCandidateID     string      `json:"child_candidate_id"`
	MaterialIndex        uint32      `json:"material_index"`
	MaterialName         string      `json:"material_name"`
	BaseColorFactor      *[4]float64 `json:"base_color_factor,omitempty"`
	MetallicFactor       *float64    `json:"metallic_factor,omitempty"`
	RoughnessFactor      *float64    `json:"roughness_factor,omitempty"`
	EmissiveFactor       *[3]float64 `json:"emissive_factor,omitempty"`
}

type MaterialCorrectionLineage struct {
	ParentCandidateID     string         `json:"parent_candidate_id"`
	ParentRevision        uint32         `json:"parent_revision"`
	ParentSHA256          string         `json:"parent_sha256"`
	RequestSHA256         string         `json:"request_sha256"`
	MaterialIndex         uint32         `json:"material_index"`
	MaterialName          string         `json:"material_name"`
	Before                map[string]any `json:"before"`
	After                 map[string]any `json:"after"`
	BeforeGLBSHA256       string         `json:"before_glb_sha256"`
	AfterGLBSHA256        string         `json:"after_glb_sha256"`
	BeforeBINChunksSHA256 []string       `json:"before_bin_chunks_sha256"`
	AfterBINChunksSHA256  []string       `json:"after_bin_chunks_sha256"`
	GeometryUnchanged     bool           `json:"geometry_unchanged"`
}

func (s *IntakeService) MaterialInventory(ctx context.Context, id string, revision uint32, revisionSHA string) (IntakeMaterialInventory, error) {
	candidate, data, err := s.readExactIntake(ctx, id, revision, revisionSHA)
	if err != nil {
		return IntakeMaterialInventory{}, err
	}
	doc, _, _, err := decodeMaterialGLB(data)
	if err != nil {
		return IntakeMaterialInventory{}, err
	}
	materials, err := inventoryMaterials(doc)
	if err != nil {
		return IntakeMaterialInventory{}, err
	}
	return IntakeMaterialInventory{CandidateID: candidate.ID, Revision: candidate.Revision, RevisionSHA256: candidate.RevisionSHA256, GLBSHA256: candidate.OriginalSHA256, Materials: materials, CorrectionLineage: candidate.Correction}, nil
}

func (s *IntakeService) CorrectMaterial(ctx context.Context, req MaterialCorrectionRequest) (IntakeCandidate, error) {
	requestSHA, err := digestMaterialCorrection(req)
	if err != nil {
		return IntakeCandidate{}, err
	}
	if !safeID.MatchString(req.ChildCandidateID) || !safeID.MatchString(req.ParentCandidateID) || req.ParentRevision == 0 || !sha256Pattern.MatchString(req.ParentRevisionSHA256) {
		return IntakeCandidate{}, fmt.Errorf("%w: exact parent and child identities are required", ErrInvalidIntake)
	}
	if row, getErr := s.repo.Get(ctx, req.ChildCandidateID, 1); getErr == nil {
		lineage := row.Value.Candidate.Correction
		if lineage != nil && lineage.RequestSHA256 == requestSHA && lineage.ParentCandidateID == req.ParentCandidateID && lineage.ParentRevision == req.ParentRevision && lineage.ParentSHA256 == req.ParentRevisionSHA256 {
			_, reader, openErr := s.Open(ctx, req.ChildCandidateID, 1)
			if openErr != nil {
				return IntakeCandidate{}, openErr
			}
			_, readErr := io.Copy(io.Discard, reader)
			closeErr := reader.Close()
			if readErr != nil || closeErr != nil {
				return IntakeCandidate{}, errors.Join(readErr, closeErr)
			}
			return row.Value.Candidate, nil
		}
		return IntakeCandidate{}, ErrIntakeConflict
	} else if !errors.Is(getErr, revisionstore.ErrNotFound) {
		return IntakeCandidate{}, fmt.Errorf("read existing corrected candidate: %w", getErr)
	}
	parent, original, err := s.readExactIntake(ctx, req.ParentCandidateID, req.ParentRevision, req.ParentRevisionSHA256)
	if err != nil {
		return IntakeCandidate{}, err
	}
	doc, chunks, binChunks, err := decodeMaterialGLB(original)
	if err != nil {
		return IntakeCandidate{}, err
	}
	before, after, err := updateMaterial(doc, req)
	if err != nil {
		return IntakeCandidate{}, err
	}
	corrected, err := encodeMaterialGLB(doc, chunks)
	if err != nil {
		return IntakeCandidate{}, err
	}
	newBinChunks := binChunkDigests(chunks)
	if !reflect.DeepEqual(binChunks, newBinChunks) {
		return IntakeCandidate{}, errors.New("material correction changed embedded geometry bytes")
	}
	beforeSum, afterSum := sha256.Sum256(original), sha256.Sum256(corrected)
	claims := IntakeClaims{CandidateID: req.ChildCandidateID, DisplayName: parent.DisplayName, SourceLabel: parent.SourceLabel, LicenseClaim: parent.LicenseClaim, Attribution: parent.Attribution, DeclaredUses: append([]string(nil), parent.DeclaredUses...), ParentID: parent.ID, ParentRevision: parent.Revision, ParentSHA256: parent.RevisionSHA256}
	staged, err := s.BeginImport(ctx, claims, parent.OriginalName, int64(len(corrected)), bytes.NewReader(corrected))
	if err != nil {
		return IntakeCandidate{}, err
	}
	candidate := staged.Candidate()
	if candidate.Inspection.NodeCount != parent.Inspection.NodeCount || candidate.Inspection.MeshCount != parent.Inspection.MeshCount || candidate.Inspection.TriangleCount != parent.Inspection.TriangleCount {
		return IntakeCandidate{}, errors.Join(errors.New("material-only correction changed inspected geometry counts"), staged.Abort(ctx))
	}
	candidate.ImportMethod = "local_material_correction"
	candidate.Correction = &MaterialCorrectionLineage{
		ParentCandidateID: parent.ID, ParentRevision: parent.Revision, ParentSHA256: parent.RevisionSHA256,
		RequestSHA256: requestSHA, MaterialIndex: req.MaterialIndex, MaterialName: req.MaterialName,
		Before: before, After: after, BeforeGLBSHA256: hex.EncodeToString(beforeSum[:]), AfterGLBSHA256: hex.EncodeToString(afterSum[:]),
		BeforeBINChunksSHA256: binChunks, AfterBINChunksSHA256: newBinChunks, GeometryUnchanged: true,
	}
	candidate.RevisionSHA256, err = intakeCandidateDigest(candidate)
	if err != nil {
		return IntakeCandidate{}, errors.Join(err, staged.Abort(ctx))
	}
	staged.(*stagedIntake).record.Candidate = candidate
	return staged.Commit(ctx)
}

func (s *IntakeService) readExactIntake(ctx context.Context, id string, revision uint32, wantSHA string) (IntakeCandidate, []byte, error) {
	candidate, reader, err := s.Open(ctx, id, revision)
	if err != nil {
		return IntakeCandidate{}, nil, err
	}
	defer reader.Close()
	if wantSHA == "" || candidate.RevisionSHA256 != wantSHA {
		return IntakeCandidate{}, nil, fmt.Errorf("%w: candidate revision digest does not match", ErrInvalidIntake)
	}
	data, err := io.ReadAll(io.LimitReader(reader, MaxGeometryBytes+1))
	if err != nil {
		return IntakeCandidate{}, nil, err
	}
	if len(data) == 0 || int64(len(data)) > MaxGeometryBytes {
		return IntakeCandidate{}, nil, ErrIntakeTooLarge
	}
	return candidate, data, nil
}

func digestMaterialCorrection(req MaterialCorrectionRequest) (string, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

type materialGLBChunks struct {
	kind uint32
	data []byte
}

func decodeMaterialGLB(data []byte) (map[string]json.RawMessage, []materialGLBChunks, []string, error) {
	if !validGLBContainer(data) {
		return nil, nil, nil, fmt.Errorf("%w: GLB container is invalid", ErrInvalidIntake)
	}
	chunks := make([]materialGLBChunks, 0, 2)
	position := 12
	for position < len(data) {
		n := int(binary.LittleEndian.Uint32(data[position : position+4]))
		kind := binary.LittleEndian.Uint32(data[position+4 : position+8])
		position += 8
		chunks = append(chunks, materialGLBChunks{kind: kind, data: append([]byte(nil), data[position:position+n]...)})
		position += n
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimRight(chunks[0].data, " \t\r\n\x00"), &doc); err != nil || doc == nil {
		return nil, nil, nil, fmt.Errorf("%w: GLB JSON is invalid", ErrInvalidIntake)
	}
	bins := binChunkDigests(chunks)
	return doc, chunks, bins, nil
}

func encodeMaterialGLB(doc map[string]json.RawMessage, old []materialGLBChunks) ([]byte, error) {
	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	for len(jsonBytes)%4 != 0 {
		jsonBytes = append(jsonBytes, ' ')
	}
	chunks := append([]materialGLBChunks(nil), old...)
	chunks[0].data = jsonBytes
	total := uint64(12)
	for _, c := range chunks {
		total += 8 + uint64(len(c.data))
	}
	if total > uint64(MaxGeometryBytes) || total > math.MaxUint32 {
		return nil, ErrIntakeTooLarge
	}
	out := make([]byte, int(total))
	copy(out, "glTF")
	binary.LittleEndian.PutUint32(out[4:8], 2)
	binary.LittleEndian.PutUint32(out[8:12], uint32(total))
	position := 12
	for _, c := range chunks {
		binary.LittleEndian.PutUint32(out[position:position+4], uint32(len(c.data)))
		binary.LittleEndian.PutUint32(out[position+4:position+8], c.kind)
		position += 8
		copy(out[position:], c.data)
		position += len(c.data)
	}
	return out, nil
}

func binChunkDigests(chunks []materialGLBChunks) []string {
	var out []string
	for _, c := range chunks {
		if c.kind == 0x004e4942 {
			sum := sha256.Sum256(c.data)
			out = append(out, hex.EncodeToString(sum[:]))
		}
	}
	return out
}

func inventoryMaterials(doc map[string]json.RawMessage) ([]IntakeMaterial, error) {
	raw := doc["materials"]
	if len(raw) == 0 {
		return []IntakeMaterial{}, nil
	}
	var materials []json.RawMessage
	if err := json.Unmarshal(raw, &materials); err != nil || len(materials) > maxMaterialCount {
		return nil, fmt.Errorf("%w: material inventory exceeds bounds", ErrInvalidIntake)
	}
	out := make([]IntakeMaterial, len(materials))
	for i, materialJSON := range materials {
		var material map[string]json.RawMessage
		if err := json.Unmarshal(materialJSON, &material); err != nil || material == nil {
			return nil, fmt.Errorf("%w: material record is invalid", ErrInvalidIntake)
		}
		var name string
		_ = json.Unmarshal(material["name"], &name)
		if !validMaterialName(name) {
			return nil, fmt.Errorf("%w: material name exceeds the inventory limit", ErrInvalidIntake)
		}
		base := [4]float64{1, 1, 1, 1}
		metal, rough := 1.0, 1.0
		emissive := [3]float64{}
		var pbr map[string]json.RawMessage
		if len(material["pbrMetallicRoughness"]) != 0 {
			if err := json.Unmarshal(material["pbrMetallicRoughness"], &pbr); err != nil {
				return nil, fmt.Errorf("%w: PBR material is invalid", ErrInvalidIntake)
			}
		}
		if pbr != nil {
			if err := readFactor(pbr["baseColorFactor"], base[:]); err != nil {
				return nil, err
			}
			if err := readScalar(pbr["metallicFactor"], &metal); err != nil {
				return nil, err
			}
			if err := readScalar(pbr["roughnessFactor"], &rough); err != nil {
				return nil, err
			}
		}
		if err := readFactor(material["emissiveFactor"], emissive[:]); err != nil {
			return nil, err
		}
		out[i] = IntakeMaterial{Index: uint32(i), Name: name, BaseColorFactor: base, MetallicFactor: metal, RoughnessFactor: rough, EmissiveFactor: emissive, Editable: !hasMaterialExtensions(material)}
	}
	return out, nil
}

func updateMaterial(doc map[string]json.RawMessage, req MaterialCorrectionRequest) (map[string]any, map[string]any, error) {
	materials, err := rawMaterials(doc)
	if err != nil {
		return nil, nil, err
	}
	if int(req.MaterialIndex) >= len(materials) {
		return nil, nil, fmt.Errorf("%w: selected material index is unavailable", ErrInvalidIntake)
	}
	var material map[string]json.RawMessage
	if err := json.Unmarshal(materials[req.MaterialIndex], &material); err != nil || material == nil {
		return nil, nil, fmt.Errorf("%w: selected material record is invalid", ErrInvalidIntake)
	}
	var name string
	_ = json.Unmarshal(material["name"], &name)
	if !validMaterialName(name) {
		return nil, nil, fmt.Errorf("%w: selected material name exceeds the inventory limit", ErrInvalidIntake)
	}
	if name != req.MaterialName {
		return nil, nil, fmt.Errorf("%w: selected material name no longer matches the observed inventory", ErrInvalidIntake)
	}
	if hasMaterialExtensions(material) {
		return nil, nil, fmt.Errorf("%w: materials with extensions cannot be edited by this operation", ErrInvalidIntake)
	}
	before, after := map[string]any{}, map[string]any{}
	var pbr map[string]json.RawMessage
	if len(material["pbrMetallicRoughness"]) != 0 {
		if err := json.Unmarshal(material["pbrMetallicRoughness"], &pbr); err != nil {
			return nil, nil, fmt.Errorf("%w: PBR material is invalid", ErrInvalidIntake)
		}
	}
	if pbr == nil {
		pbr = map[string]json.RawMessage{}
	}
	base := [4]float64{1, 1, 1, 1}
	metal, rough := 1.0, 1.0
	emissive := [3]float64{}
	if err := readFactor(pbr["baseColorFactor"], base[:]); err != nil {
		return nil, nil, err
	}
	if err := readScalar(pbr["metallicFactor"], &metal); err != nil {
		return nil, nil, err
	}
	if err := readScalar(pbr["roughnessFactor"], &rough); err != nil {
		return nil, nil, err
	}
	if err := readFactor(material["emissiveFactor"], emissive[:]); err != nil {
		return nil, nil, err
	}
	changed := false
	if v := req.BaseColorFactor; v != nil {
		if err := unitValues(v[:]); err != nil {
			return nil, nil, err
		}
		before["base_color_factor"], after["base_color_factor"] = base, *v
		if base != *v {
			changed = true
		}
		pbr["baseColorFactor"], _ = json.Marshal(v)
	}
	if v := req.MetallicFactor; v != nil {
		if !unitValue(*v) {
			return nil, nil, fmt.Errorf("%w: metallic factor must be finite and between 0 and 1", ErrInvalidIntake)
		}
		before["metallic_factor"], after["metallic_factor"] = metal, *v
		if metal != *v {
			changed = true
		}
		pbr["metallicFactor"], _ = json.Marshal(v)
	}
	if v := req.RoughnessFactor; v != nil {
		if !unitValue(*v) {
			return nil, nil, fmt.Errorf("%w: roughness factor must be finite and between 0 and 1", ErrInvalidIntake)
		}
		before["roughness_factor"], after["roughness_factor"] = rough, *v
		if rough != *v {
			changed = true
		}
		pbr["roughnessFactor"], _ = json.Marshal(v)
	}
	if v := req.EmissiveFactor; v != nil {
		if err := unitValues(v[:]); err != nil {
			return nil, nil, err
		}
		before["emissive_factor"], after["emissive_factor"] = emissive, *v
		if emissive != *v {
			changed = true
		}
		material["emissiveFactor"], _ = json.Marshal(v)
	}
	if !changed {
		return nil, nil, fmt.Errorf("%w: at least one material value must change", ErrInvalidIntake)
	}
	pbrJSON, _ := json.Marshal(pbr)
	material["pbrMetallicRoughness"] = pbrJSON
	materials[req.MaterialIndex], _ = json.Marshal(material)
	doc["materials"], _ = json.Marshal(materials)
	return before, after, nil
}

func hasMaterialExtensions(material map[string]json.RawMessage) bool {
	if hasExtensions := func(raw json.RawMessage) bool { return len(raw) != 0 && strings.TrimSpace(string(raw)) != "{}" }(material["extensions"]); hasExtensions {
		return true
	}
	var pbr map[string]json.RawMessage
	if len(material["pbrMetallicRoughness"]) != 0 {
		if json.Unmarshal(material["pbrMetallicRoughness"], &pbr) != nil {
			return true
		}
		return len(pbr["extensions"]) != 0 && strings.TrimSpace(string(pbr["extensions"])) != "{}"
	}
	return false
}

func rawMaterials(doc map[string]json.RawMessage) ([]json.RawMessage, error) {
	var materials []json.RawMessage
	if err := json.Unmarshal(doc["materials"], &materials); err != nil || len(materials) > maxMaterialCount {
		return nil, fmt.Errorf("%w: material inventory exceeds bounds", ErrInvalidIntake)
	}
	return materials, nil
}
func readFactor(raw json.RawMessage, out []float64) error {
	if len(raw) == 0 {
		return nil
	}
	var values []float64
	if err := json.Unmarshal(raw, &values); err != nil || len(values) != len(out) {
		return fmt.Errorf("%w: material factor has invalid shape", ErrInvalidIntake)
	}
	for i, v := range values {
		if !unitValue(v) {
			return fmt.Errorf("%w: material factor is outside its finite range", ErrInvalidIntake)
		}
		out[i] = v
	}
	return nil
}
func readScalar(raw json.RawMessage, out *float64) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil || !unitValue(*out) {
		return fmt.Errorf("%w: material factor is outside its finite range", ErrInvalidIntake)
	}
	return nil
}
func unitValues(values []float64) error {
	for _, v := range values {
		if !unitValue(v) {
			return fmt.Errorf("%w: material factor must be finite and between 0 and 1", ErrInvalidIntake)
		}
	}
	return nil
}
func unitValue(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1 }

func validMaterialName(name string) bool {
	return len(name) <= maxMaterialNameBytes && strings.IndexFunc(name, unicode.IsControl) < 0
}
