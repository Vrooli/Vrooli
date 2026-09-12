package eval

import (
	"strings"
	"testing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

func TestComposeRoutingSuite(t *testing.T) {
	base := func(provider string, cases ...*evalv1.EvalCase) *evalv1.EvalSuite {
		return &evalv1.EvalSuite{SuiteId: provider + ".primary", ProviderId: provider, Cases: cases}
	}
	positive := func(id, query string) *evalv1.EvalCase {
		return &evalv1.EvalCase{CaseId: id, Query: query, ExpectIds: []string{"hit"}}
	}
	tests := []struct {
		name       string
		suites     []*evalv1.EvalSuite
		registered map[string]struct{}
		wantCases  int
		wantText   string
	}{
		{name: "empty registry", registered: map[string]struct{}{}},
		{
			name:       "all candidates",
			registered: map[string]struct{}{"p": {}},
			suites:     []*evalv1.EvalSuite{base("p", &evalv1.EvalCase{CaseId: "candidate", Query: "candidate", Status: "candidate", ExpectIds: []string{"x"}})},
			wantText:   "candidate=1",
		},
		{
			name:       "unregistered owner excluded",
			registered: map[string]struct{}{},
			suites:     []*evalv1.EvalSuite{base("missing", positive("one", "where is one"))},
			wantText:   "unregistered=1",
		},
		{
			name:       "duplicate query remains independently labeled",
			registered: map[string]struct{}{"a": {}, "b": {}},
			suites: []*evalv1.EvalSuite{
				base("a", positive("one", "same question")),
				base("b", positive("two", "same question")),
			},
			wantCases: 2,
		},
		{
			name:       "healthy suite",
			registered: map[string]struct{}{"p": {}},
			suites:     []*evalv1.EvalSuite{base("p", positive("one", "where is one"))},
			wantCases:  1,
		},
		{
			name:       "degenerate positive excluded",
			registered: map[string]struct{}{"p": {}},
			suites: []*evalv1.EvalSuite{base("p", []*evalv1.EvalCase{
				{CaseId: "token", Query: "api", ExpectIds: []string{"pkg/api"}},
				{CaseId: "question", Query: "where is api handled", ExpectIds: []string{"pkg/api"}},
			}...)},
			wantCases: 1,
			wantText:  "degenerate=1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComposeRoutingSuite(tt.suites, tt.registered)
			if len(got.GetCases()) != tt.wantCases {
				t.Fatalf("cases=%d, want %d (%s)", len(got.GetCases()), tt.wantCases, got.GetDescription())
			}
			if !strings.Contains(got.GetDescription(), tt.wantText) {
				t.Fatalf("description %q does not contain %q", got.GetDescription(), tt.wantText)
			}
		})
	}
}

func TestComposeRoutingSuiteCarriesOwnerAndHasReservedIdentity(t *testing.T) {
	suite := ComposeRoutingSuite([]*evalv1.EvalSuite{{ProviderId: "p", Cases: []*evalv1.EvalCase{{CaseId: "c", Query: "question", ExpectIds: []string{"id"}}}}}, map[string]struct{}{"p": {}})
	if suite.GetSuiteId() != RouterSuiteID || suite.GetProviderId() != RouterSuiteID {
		t.Fatalf("unexpected composed identity: %s/%s", suite.GetSuiteId(), suite.GetProviderId())
	}
	if suite.GetCases()[0].GetExpectedProviderId() != "p" {
		t.Fatalf("owner=%q, want p", suite.GetCases()[0].GetExpectedProviderId())
	}
	if err := Validate(&evalv1.EvalSuite{SuiteId: RouterSuiteID, ProviderId: "p", Cases: suite.GetCases()}); err == nil {
		t.Fatal("reserved suite id unexpectedly validated")
	}
}
