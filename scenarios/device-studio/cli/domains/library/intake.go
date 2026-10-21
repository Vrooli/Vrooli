package library

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/library/library_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

type intakeHandlers struct {
	core   *cliapp.ScenarioApp
	client connectv1.LibraryCatalogServiceClient
}

type materialInventoryResult struct {
	CandidateID    string `json:"candidate_id"`
	Revision       uint32 `json:"revision"`
	RevisionSHA256 string `json:"revision_sha256"`
	GLBSHA256      string `json:"glb_sha256"`
	Materials      []struct {
		Index     uint32    `json:"index"`
		Name      string    `json:"name"`
		BaseColor []float64 `json:"base_color_factor"`
		Metallic  float64   `json:"metallic_factor"`
		Roughness float64   `json:"roughness_factor"`
		Emissive  []float64 `json:"emissive_factor"`
		Editable  bool      `json:"editable"`
	} `json:"materials"`
}
type materialCorrectionResult struct {
	Candidate materialCorrectionCandidate `json:"candidate"`
}

type materialCorrectionCandidate struct {
	ID                string `json:"id"`
	Revision          uint32 `json:"revision"`
	RevisionSHA256    string `json:"revision_sha256"`
	OriginalSHA256    string `json:"original_sha256"`
	ReviewStatus      string `json:"review_status"`
	RenderStatus      string `json:"render_status"`
	ParentCandidateID string `json:"parent_candidate_id"`
	ParentRevision    uint32 `json:"parent_revision"`
	ParentSHA256      string `json:"parent_sha256"`
	ImportMethod      string `json:"import_method"`
}

func newIntakeHandlers(core *cliapp.ScenarioApp) *intakeHandlers {
	hc, base := cliapp.NewConnectHTTPClient(core)
	return &intakeHandlers{core: core, client: connectv1.NewLibraryCatalogServiceClient(hc, base)}
}

func (h *intakeHandlers) materialsCall(ctx cliapp.OperationContext) (materialInventoryResult, error) {
	revision, err := parseRequiredRevision(ctx.Flag("revision"))
	if err != nil {
		return materialInventoryResult{}, err
	}
	id := ctx.Positional("candidate-id")
	if !validStableID(id) {
		return materialInventoryResult{}, fmt.Errorf("candidate-id must be a stable identifier")
	}
	digest := strings.ToLower(strings.TrimSpace(ctx.Flag("revision-sha256")))
	if len(digest) != 64 {
		return materialInventoryResult{}, fmt.Errorf("revision-sha256 must be a 64-character digest")
	}
	path := "/library/intake-candidates/" + url.PathEscape(id) + fmt.Sprintf("/revisions/%d/materials?revision_sha256=%s", revision, url.QueryEscape(digest))
	data, err := h.core.Request("GET", path, nil, nil)
	if err != nil {
		return materialInventoryResult{}, cliapp.WrapAPIError("read imported model materials", err, nil)
	}
	var result materialInventoryResult
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("decode material inventory: %w", err)
	}
	return result, nil
}

func (h *intakeHandlers) correctMaterialCall(ctx cliapp.OperationContext) (materialCorrectionResult, error) {
	revision, err := parseRequiredRevision(ctx.Flag("revision"))
	if err != nil {
		return materialCorrectionResult{}, err
	}
	id := ctx.Positional("candidate-id")
	if !validStableID(id) {
		return materialCorrectionResult{}, fmt.Errorf("candidate-id must be a stable identifier")
	}
	index, err := strconv.ParseUint(strings.TrimSpace(ctx.Flag("material-index")), 10, 32)
	if err != nil {
		return materialCorrectionResult{}, fmt.Errorf("material-index must be a non-negative integer")
	}
	childID := strings.TrimSpace(ctx.Flag("child-candidate-id"))
	parentSHA := strings.ToLower(strings.TrimSpace(ctx.Flag("revision-sha256")))
	if !validStableID(childID) || !validSHA256(parentSHA) || strings.TrimSpace(ctx.Flag("material-name")) == "" {
		return materialCorrectionResult{}, fmt.Errorf("provide exact parent digest, child ID, and observed material name")
	}
	body := map[string]any{"parent_candidate_id": id, "parent_revision": revision, "parent_revision_sha256": parentSHA, "child_candidate_id": childID, "material_index": uint32(index), "material_name": ctx.Flag("material-name")}
	changes := 0
	for flag, spec := range map[string]struct {
		key   string
		count int
	}{"base-color": {"base_color_factor", 4}, "emissive": {"emissive_factor", 3}} {
		if raw := strings.TrimSpace(ctx.Flag(flag)); raw != "" {
			values, err := parseMaterialValues(raw, spec.count)
			if err != nil {
				return materialCorrectionResult{}, fmt.Errorf("%s: %w", flag, err)
			}
			body[spec.key] = values
			changes++
		}
	}
	for flag, key := range map[string]string{"metallic": "metallic_factor", "roughness": "roughness_factor"} {
		if raw := strings.TrimSpace(ctx.Flag(flag)); raw != "" {
			v, err := strconv.ParseFloat(raw, 64)
			if err != nil || v < 0 || v > 1 || math.IsNaN(v) || math.IsInf(v, 0) {
				return materialCorrectionResult{}, fmt.Errorf("%s must be between 0 and 1", flag)
			}
			body[key] = v
			changes++
		}
	}
	if changes == 0 {
		return materialCorrectionResult{}, fmt.Errorf("provide at least one changed PBR factor")
	}
	data, err := json.Marshal(body)
	if err != nil {
		return materialCorrectionResult{}, err
	}
	path := "/library/intake-candidates/" + url.PathEscape(id) + fmt.Sprintf("/revisions/%d/material-corrections", revision)
	response, err := h.core.Request("POST", path, nil, data)
	if err != nil {
		return materialCorrectionResult{}, cliapp.WrapAPIError("correct imported material", err, nil)
	}
	var result materialCorrectionResult
	if err := json.Unmarshal(response, &result); err != nil {
		return result, fmt.Errorf("decode corrected candidate: %w", err)
	}
	if err := validateMaterialCorrectionCandidate(result.Candidate, childID, id, revision, parentSHA); err != nil {
		return result, err
	}
	return result, nil
}

func validateMaterialCorrectionCandidate(candidate materialCorrectionCandidate, childID, parentID string, parentRevision uint32, parentSHA string) error {
	if candidate.ID != childID || candidate.Revision != 1 || !validSHA256(candidate.RevisionSHA256) || !validSHA256(candidate.OriginalSHA256) ||
		candidate.ParentCandidateID != parentID || candidate.ParentRevision != parentRevision || candidate.ParentSHA256 != parentSHA ||
		candidate.ImportMethod != "local_material_correction" || candidate.ReviewStatus != "unreviewed" || candidate.RenderStatus != "not_reviewed" {
		return fmt.Errorf("server returned a corrected candidate that does not match the exact child and parent request")
	}
	return nil
}

func validSHA256(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func (h *intakeHandlers) materialsReport(_ cliapp.OperationContext, result materialInventoryResult) cliapp.MutationReport {
	rows := make([]string, 0, len(result.Materials))
	for _, m := range result.Materials {
		rows = append(rows, fmt.Sprintf("[%d] %q editable=%t base=%v metallic=%.3f roughness=%.3f emissive=%v", m.Index, m.Name, m.Editable, m.BaseColor, m.Metallic, m.Roughness, m.Emissive))
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Observed %d bounded material record(s) on %s@%d with GLB SHA-256 %s. Inventory is read-only; edits create an unreviewed child.", len(rows), result.CandidateID, result.Revision, result.GLBSHA256)}, Changes: rows}
}

func (h *intakeHandlers) correctionReport(_ cliapp.OperationContext, result materialCorrectionResult) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created or recovered unreviewed material correction %s@%d with GLB SHA-256 %s.", result.Candidate.ID, result.Candidate.Revision, result.Candidate.OriginalSHA256)}, Changes: []string{fmt.Sprintf("review=%s render=%s", result.Candidate.ReviewStatus, result.Candidate.RenderStatus)}, NextCommand: []string{fmt.Sprintf("`library intake-get %s --revision %d` — inspect the derived candidate; its parent remains unchanged", result.Candidate.ID, result.Candidate.Revision)}}
}

func parseMaterialValues(raw string, count int) ([]float64, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != count {
		return nil, fmt.Errorf("expected %d comma-separated values", count)
	}
	values := make([]float64, count)
	for i, part := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil || v < 0 || v > 1 || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("all values must be between 0 and 1")
		}
		values[i] = v
	}
	return values, nil
}

func (h *intakeHandlers) importCall(ctx cliapp.OperationContext) (*v1.GetIntakeCandidateResponse, error) {
	path := ctx.Flag("file")
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open local geometry: %w", err)
	}
	defer file.Close()
	claims := &v1.IntakeCandidateClaims{
		CandidateId: ctx.Flag("candidate-id"), DisplayName: ctx.Flag("display-name"),
		SourceLabel: ctx.Flag("source-label"), LicenseClaim: ctx.Flag("license-claim"),
		Attribution: ctx.Flag("attribution"), DeclaredUses: splitClaims(ctx.Flag("declared-uses")),
		ParentCandidateId: ctx.Flag("parent-id"), ParentSha256: ctx.Flag("parent-sha256"),
	}
	if raw := ctx.Flag("parent-revision"); raw != "" {
		claims.ParentRevision, err = parseRequiredRevision(raw)
		if err != nil {
			return nil, fmt.Errorf("parent-revision: %w", err)
		}
	}
	if (claims.ParentCandidateId == "") != (claims.ParentRevision == 0) || (claims.ParentCandidateId == "") != (claims.ParentSha256 == "") {
		return nil, fmt.Errorf("parent-id, parent-revision, and parent-sha256 must be supplied together")
	}
	metadata, err := protojson.Marshal(claims)
	if err != nil {
		return nil, fmt.Errorf("encode intake claims: %w", err)
	}
	body, err := cliapp.UploadFile(h.core, "/library/intake-candidates", map[string]string{"metadata": string(metadata)}, cliapp.UploadedFile{Name: filepath.Base(path), ContentType: "model/gltf-binary", Reader: file})
	if err != nil {
		return nil, cliapp.WrapAPIError("import local geometry candidate", err, nil)
	}
	response, err := cliapp.DecodeUploadResponse[*v1.GetIntakeCandidateResponse](body)
	if err != nil {
		return nil, err
	}
	if response.Candidate == nil {
		return nil, fmt.Errorf("server returned no intake candidate")
	}
	return response, nil
}

func (h *intakeHandlers) importReport(_ cliapp.OperationContext, response *v1.GetIntakeCandidateResponse) cliapp.MutationReport {
	c := response.Candidate
	return cliapp.MutationReport{
		Result:      []string{fmt.Sprintf("Imported original geometry bytes as candidate %s@%d.", c.Id, c.Revision)},
		Changes:     []string{formatIntake(c)},
		NextCommand: []string{fmt.Sprintf("`library intake-get %s --revision %d` — inspect the retained source facts", c.Id, c.Revision), "The import is unreviewed and cannot be used for rendering until a separate review and qualification flow exists."},
	}
}

func (h *intakeHandlers) listCall(ctx cliapp.OperationContext) (*v1.ListIntakeCandidatesResponse, error) {
	pageSize, err := boundedPageSize(ctx.Flag("page-size"))
	if err != nil {
		return nil, err
	}
	after := ctx.Flag("after-candidate-id")
	if after != "" && !validStableID(after) {
		return nil, fmt.Errorf("after-candidate-id must be a stable identifier of at most 128 characters")
	}
	response, err := h.client.ListIntakeCandidates(context.Background(), connect.NewRequest(&v1.ListIntakeCandidatesRequest{AfterCandidateId: after, PageSize: pageSize}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list local geometry candidates", err, nil)
	}
	return response.Msg, nil
}

func (h *intakeHandlers) listReport(_ cliapp.OperationContext, response *v1.ListIntakeCandidatesResponse) cliapp.ListReport {
	rows := make([]string, 0, len(response.Candidates))
	for _, candidate := range response.Candidates {
		rows = append(rows, formatIntake(candidate))
	}
	var hints []string
	if response.NextAfterCandidateId != "" {
		hints = append(hints, fmt.Sprintf("`library intake-list --after-candidate-id %s` — next page", response.NextAfterCandidateId))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d locally imported geometry candidate(s). Review and render qualification are separate.", len(rows))}, ResultsHeading: "Geometry intake candidates", Results: rows, RetrievalHints: hints}
}

func (h *intakeHandlers) getCall(ctx cliapp.OperationContext) (*v1.GetIntakeCandidateResponse, error) {
	revision, err := parseRequiredRevision(ctx.Flag("revision"))
	if err != nil {
		return nil, err
	}
	response, err := h.client.GetIntakeCandidate(context.Background(), connect.NewRequest(&v1.GetIntakeCandidateRequest{CandidateId: ctx.Positional("candidate-id"), Revision: revision}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get local geometry candidate", err, nil)
	}
	if response.Msg.Candidate == nil {
		return nil, fmt.Errorf("server returned no intake candidate")
	}
	return response.Msg, nil
}

func (h *intakeHandlers) getReport(_ cliapp.OperationContext, response *v1.GetIntakeCandidateResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Loaded immutable local geometry intake metadata. Original bytes remain available through the candidate content endpoint."}, ResultsHeading: "Geometry intake candidate", Results: []string{formatIntake(response.Candidate)}}
}

func formatIntake(c *v1.IntakeCandidateRevision) string {
	if c == nil {
		return "invalid intake candidate response"
	}
	facts := fmt.Sprintf("%s@%d %s (%d bytes, sha256=%s) review=%s render=%s", c.Id, c.Revision, c.OriginalFilename, c.SizeBytes, c.OriginalSha256, c.ReviewStatus, c.RenderStatus)
	if c.Inspection != nil {
		facts += fmt.Sprintf(" inspection=%s/v%d supported=%t", c.Inspection.Profile, c.Inspection.ProfileVersion, c.Inspection.Supported)
		if c.Inspection.Finding != "" {
			facts += " finding=" + c.Inspection.Finding
		}
	}
	if c.ParentCandidateId != "" {
		facts += fmt.Sprintf(" parent=%s@%d sha256=%s", c.ParentCandidateId, c.ParentRevision, c.ParentSha256)
	}
	return facts
}

func splitClaims(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}
