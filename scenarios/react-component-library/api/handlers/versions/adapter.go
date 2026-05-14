package versions

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"react-component-library/internal/versions"

	versionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/versions"
)

func versionToProto(v versions.Version, _ bool) *versionsv1.Version {
	out := &versionsv1.Version{
		Id:            v.ID,
		ComponentId:   v.ComponentID,
		LibraryId:     v.LibraryID,
		Version:       v.Version,
		ContentSha256: v.ContentSHA256,
		ChangelogMd:   v.ChangelogMD,
		RecordedAt:    timestamppb.New(v.RecordedAt.UTC()),
		Status:        v.Status,
		SourcePath:    v.SourcePath,
	}
	if !v.ReleasedAt.IsZero() {
		out.ReleasedAt = timestamppb.New(v.ReleasedAt.UTC())
	}
	return out
}

func diffToProto(d versions.DiffResult) *versionsv1.DiffVersionsResponse {
	out := &versionsv1.DiffVersionsResponse{
		Rows:      make([]*versionsv1.DiffRow, 0, len(d.Rows)),
		Additions: int32(d.Additions),
		Removals:  int32(d.Removals),
		FromLabel: d.FromLabel,
		ToLabel:   d.ToLabel,
	}
	for _, r := range d.Rows {
		out.Rows = append(out.Rows, &versionsv1.DiffRow{
			Left:  cellToProto(r.Left),
			Right: cellToProto(r.Right),
		})
	}
	return out
}

func cellToProto(c versions.DiffCell) *versionsv1.DiffCell {
	return &versionsv1.DiffCell{
		LineNumber: int32(c.LineNumber),
		Text:       c.Text,
		Op:         opToProto(c.Op),
	}
}

func opToProto(op versions.DiffOp) versionsv1.DiffOp {
	switch op {
	case versions.DiffOpEqual:
		return versionsv1.DiffOp_DIFF_OP_EQUAL
	case versions.DiffOpAdd:
		return versionsv1.DiffOp_DIFF_OP_ADD
	case versions.DiffOpRemove:
		return versionsv1.DiffOp_DIFF_OP_REMOVE
	case versions.DiffOpEmpty:
		return versionsv1.DiffOp_DIFF_OP_EMPTY
	}
	return versionsv1.DiffOp_DIFF_OP_UNSPECIFIED
}
