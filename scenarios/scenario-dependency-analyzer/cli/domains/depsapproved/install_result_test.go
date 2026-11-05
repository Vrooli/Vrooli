package depsapproved

import (
	"testing"

	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

func TestInstallResultReportsFailureWhenApprovedApplicationDidNotInstall(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response *governancev1.InstallDependencyResponse
		ok       bool
	}{
		{"approved preview", &governancev1.InstallDependencyResponse{DryRun: true}, true},
		{"installed", &governancev1.InstallDependencyResponse{Installed: true}, true},
		{"source refreshed without runtime install", &governancev1.InstallDependencyResponse{SourceUpdated: true}, true},
		{"blocked source outcome", &governancev1.InstallDependencyResponse{SourceUpdated: true, Blocked: true}, false},
		{"failed application", &governancev1.InstallDependencyResponse{Message: "Install failed (approved)"}, false},
		{"blocked preview", &governancev1.InstallDependencyResponse{DryRun: true, Blocked: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := installOutcome(tc.response) == nil; got != tc.ok {
				t.Fatalf("result success=%t want %t", got, tc.ok)
			}
		})
	}
}
