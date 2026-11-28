package depsapproved

import (
	"strings"
	"testing"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

func TestUpdateOutcomeSucceedsForEveryConsistentResult(t *testing.T) {
	for name, msg := range map[string]*governancev1.UpdateLockedDependencyResponse{
		"preview":       {DryRun: true, Verdict: "updatable"},
		"applied":       {Updated: true, Verdict: "updatable"},
		"lock only":     {SourceUpdated: true, Verdict: "updatable"},
		"nothing newer": {Verdict: "current"},
		"not exposed":   {Verdict: "not_affected"},
	} {
		if err := updateOutcome(msg); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, msg := range map[string]*governancev1.UpdateLockedDependencyResponse{
		"blocked preview": {DryRun: true, Blocked: true, Verdict: "update_blocked"},
		"failed apply":    {Verdict: "update_failed"},
	} {
		if err := updateOutcome(msg); err == nil || !strings.Contains(err.Error(), "did not succeed") {
			t.Errorf("%s: err = %v, want a failure", name, err)
		}
	}
}

func TestUpdateTakesNamedPackagesLikeRemoveDoes(t *testing.T) {
	ecosystem, names, err := parseRemoveSpecs([]string{"npm/body-parser", "npm/proxy-addr"})
	if err != nil || ecosystem != "npm" || strings.Join(names, ",") != "body-parser,proxy-addr" {
		t.Fatalf("ecosystem=%q names=%v err=%v", ecosystem, names, err)
	}
	schema := updateArgumentSchema()
	if len(schema.Positionals) != 1 || !schema.Positionals[0].Repeated || schema.Positionals[0].Required {
		t.Fatalf("update takes repeated dependencies that --advisory can replace, so none is schema-required: %+v", schema.Positionals)
	}
}

func TestUpdateSelectsPackagesOrOneAdvisoryNeverBothOrNeither(t *testing.T) {
	cases := []struct {
		name     string
		advisory string
		specs    []string
		wantErr  bool
		wantEco  string
		wantN    int
	}{
		{"packages", "", []string{"npm/body-parser", "npm/proxy-addr"}, false, "npm", 2},
		{"an advisory", "GHSA-27r3-jxmh-p828", nil, false, "", 0},
		{"both", "GHSA-27r3-jxmh-p828", []string{"npm/body-parser"}, true, "", 0},
		{"neither", "", nil, true, "", 0},
		{"a spec with no ecosystem", "", []string{"body-parser"}, true, "", 0},
	}
	for _, tc := range cases {
		eco, names, err := updateSelection(tc.advisory, tc.specs)
		if (err != nil) != tc.wantErr || eco != tc.wantEco || len(names) != tc.wantN {
			t.Errorf("%s: ecosystem=%q names=%v err=%v", tc.name, eco, names, err)
		}
	}
}
