package planscli

import (
	"io"
	"time"

	"google.golang.org/protobuf/proto"

	planapp "github.com/vrooli/vrooli/internal/app/plans"
	"github.com/vrooli/vrooli/internal/cliout"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// plansRecord maps the internal plan record onto the vrooli.cli.v1 wire type.
// A proto field rename breaks this mapping at compile time — that is the drift guard.
func plansRecord(r planapp.PlanRecord) *cliv1.PlansRecord {
	return &cliv1.PlansRecord{
		Id:          r.ID,
		Title:       r.Title,
		Slug:        r.Slug,
		Path:        r.Path,
		CreatedAt:   formatPlansTime(r.CreatedAt),
		UpdatedAt:   formatPlansTime(r.UpdatedAt),
		Archived:    r.Archived,
		ArchivedAt:  formatPlansTime(r.ArchivedAt),
		SourcePath:  r.SourcePath,
		ContentHash: r.ContentHash,
	}
}

// formatPlansTime renders a timestamp as RFC3339Nano; zero values map to "".
func formatPlansTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

// PlansAddResponse maps `plans add` output onto the wire contract.
func PlansAddResponse(resp planapp.AddOutput) *cliv1.PlansAddOutput {
	return &cliv1.PlansAddOutput{Success: resp.Success, Plan: plansRecord(resp.Plan)}
}

// PlansListResponse maps `plans list` output onto the wire contract.
func PlansListResponse(resp planapp.ListOutput) *cliv1.PlansListOutput {
	out := &cliv1.PlansListOutput{Success: resp.Success}
	for _, p := range resp.Plans {
		out.Plans = append(out.Plans, plansRecord(p))
	}
	return out
}

// PlansShowResponse maps `plans show` output onto the wire contract.
func PlansShowResponse(resp planapp.ShowOutput) *cliv1.PlansShowOutput {
	return &cliv1.PlansShowOutput{Success: resp.Success, Plan: plansRecord(resp.Plan), Content: resp.Content}
}

// PlansPathResponse maps `plans path` output onto the wire contract.
func PlansPathResponse(resp planapp.PathOutput) *cliv1.PlansPathOutput {
	return &cliv1.PlansPathOutput{Success: resp.Success, Id: resp.ID, Path: resp.Path}
}

// PlansArchiveResponse maps `plans archive` output onto the wire contract.
func PlansArchiveResponse(resp planapp.ArchiveOutput) *cliv1.PlansArchiveOutput {
	return &cliv1.PlansArchiveOutput{Success: resp.Success, Plan: plansRecord(resp.Plan)}
}

// PlansImportResponse maps `plans import` output onto the wire contract.
func PlansImportResponse(resp planapp.ImportOutput) *cliv1.PlansImportOutput {
	return &cliv1.PlansImportOutput{Success: resp.Success, Plan: plansRecord(resp.Plan), DeletedSource: resp.Deleted}
}

// PlansExportResponse maps `plans export` output onto the wire contract.
func PlansExportResponse(resp planapp.ExportOutput) *cliv1.PlansExportOutput {
	return &cliv1.PlansExportOutput{Success: resp.Success, Id: resp.ID, Path: resp.Path}
}

// writePlansJSON marshals a plans wire-contract message and writes it to w.
func writePlansJSON(w io.Writer, msg proto.Message) error {
	return cliout.WriteProtoJSON(w, msg)
}
