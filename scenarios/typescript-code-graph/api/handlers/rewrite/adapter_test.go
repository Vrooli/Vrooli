package rewrite

import (
	"testing"

	rewritedom "typescript-code-graph/internal/rewrite"

	rewritepb "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/rewrite"
)

func TestProtoToDomainOperation_FileMove(t *testing.T) {
	in := &rewritepb.Operation{Op: &rewritepb.Operation_FileMove{
		FileMove: &rewritepb.FileMove{FromPath: "a", ToPath: "b"},
	}}
	out := protoToDomainOperation(in)
	if out.FileMove == nil || out.FileMove.FromPath != "a" || out.FileMove.ToPath != "b" {
		t.Errorf("file_move not translated: %+v", out)
	}
	if out.ImportRewrite != nil {
		t.Errorf("import_rewrite arm leaked")
	}
}

func TestProtoToDomainOperation_ImportRewrite(t *testing.T) {
	in := &rewritepb.Operation{Op: &rewritepb.Operation_ImportRewrite{
		ImportRewrite: &rewritepb.ImportRewrite{OldPath: "x", NewPath: "y"},
	}}
	out := protoToDomainOperation(in)
	if out.ImportRewrite == nil || out.ImportRewrite.OldPath != "x" || out.ImportRewrite.NewPath != "y" {
		t.Errorf("import_rewrite not translated: %+v", out)
	}
}

func TestDomainToProtoOperation_RoundTrip(t *testing.T) {
	cases := []rewritedom.Operation{
		{FileMove: &rewritedom.FileMove{FromPath: "src/a.ts", ToPath: "src/b.ts"}},
		{ImportRewrite: &rewritedom.ImportRewrite{OldPath: "./old", NewPath: "./new"}},
	}
	protoOps := domainToProtoOperations(cases)
	back := protoToDomainOperations(protoOps)
	if len(back) != len(cases) {
		t.Fatalf("len mismatch: got %d, want %d", len(back), len(cases))
	}
	for i := range cases {
		if cases[i].OperationTag() != back[i].OperationTag() {
			t.Errorf("op %d tag drift: %q vs %q", i, cases[i].OperationTag(), back[i].OperationTag())
		}
	}
}

func TestStatusToProto(t *testing.T) {
	if statusToProto(rewritedom.StatusOK) != rewritepb.OperationStatus_OPERATION_STATUS_OK {
		t.Errorf("StatusOK -> wrong proto")
	}
	if statusToProto(rewritedom.StatusFailed) != rewritepb.OperationStatus_OPERATION_STATUS_FAILED {
		t.Errorf("StatusFailed -> wrong proto")
	}
	if statusToProto("") != rewritepb.OperationStatus_OPERATION_STATUS_UNSPECIFIED {
		t.Errorf("empty -> wrong proto")
	}
}

func TestDomainResultsToProto(t *testing.T) {
	in := []rewritedom.ApplyResult{
		{Operation: rewritedom.Operation{FileMove: &rewritedom.FileMove{FromPath: "a", ToPath: "b"}}, Status: rewritedom.StatusOK},
		{Operation: rewritedom.Operation{ImportRewrite: &rewritedom.ImportRewrite{OldPath: "x", NewPath: "y"}}, Status: rewritedom.StatusFailed, Message: "boom"},
	}
	got := domainResultsToProto(in)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].GetStatus() != rewritepb.OperationStatus_OPERATION_STATUS_OK {
		t.Errorf("result[0] status wrong")
	}
	if got[1].GetStatus() != rewritepb.OperationStatus_OPERATION_STATUS_FAILED || got[1].GetMessage() != "boom" {
		t.Errorf("result[1] status/message wrong: %+v", got[1])
	}
}
