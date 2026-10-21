package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

func newIntakeHandlers(core *cliapp.ScenarioApp) *intakeHandlers {
	hc, base := cliapp.NewConnectHTTPClient(core)
	return &intakeHandlers{core: core, client: connectv1.NewLibraryCatalogServiceClient(hc, base)}
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
