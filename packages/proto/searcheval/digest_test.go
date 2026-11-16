package searcheval

import (
	"testing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	"google.golang.org/protobuf/proto"
)

func TestSuiteCorpusDigestIgnoresStoreMetadataAndTracksLabels(t *testing.T) {
	suite := &evalv1.EvalSuite{
		SuiteId: "docs.primary", ProviderId: "docs", Name: "Original", Description: "old",
		CreatedAt: "2026-11-01T00:00:00Z", UpdatedAt: "2026-11-02T00:00:00Z",
		Cases: []*evalv1.EvalCase{{CaseId: "case-1", Query: "query", ExpectIds: []string{"doc-1"}}},
	}
	baseline := SuiteCorpusDigest(suite)
	metadataChanged := proto.Clone(suite).(*evalv1.EvalSuite)
	metadataChanged.Name = "Renamed"
	metadataChanged.Description = "new"
	metadataChanged.UpdatedAt = "2026-11-03T00:00:00Z"
	if got := SuiteCorpusDigest(metadataChanged); got != baseline {
		t.Fatalf("metadata-only edit changed corpus digest: %q != %q", got, baseline)
	}
	labelChanged := proto.Clone(suite).(*evalv1.EvalSuite)
	labelChanged.Cases[0].ExpectIds[0] = "doc-2"
	if got := SuiteCorpusDigest(labelChanged); got == baseline {
		t.Fatalf("label edit did not change corpus digest: %q", got)
	}
	if SuiteCorpusDigest(nil) != "" {
		t.Fatal("nil suite should have no corpus digest")
	}
}

func TestCaseIDsDigestIsOrderIndependentAndPreservesDuplicates(t *testing.T) {
	left := CaseIDsDigest([]string{"b", "a"})
	right := CaseIDsDigest([]string{"a", "b"})
	if left == "" || left != right {
		t.Fatalf("ID set digest depends on input order: %q != %q", left, right)
	}
	if duplicate := CaseIDsDigest([]string{"a", "b", "a"}); duplicate == left {
		t.Fatal("duplicate ID unexpectedly disappeared from the digest")
	}
}
