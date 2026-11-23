package proposals

import (
	"context"
	"errors"
	"testing"

	"git-control-tower/internal/trailers"
)

type fixedTrailerResolver struct {
	ownerID string
	status  trailers.ResolutionStatus
	detail  string
	err     error
	keys    []string
}

func (r *fixedTrailerResolver) Resolve(_ context.Context, key, value string) (string, trailers.ResolutionStatus, string, error) {
	r.keys = append(r.keys, key+":"+value)
	return r.ownerID, r.status, r.detail, r.err
}

func TestLocalResolverResolvesProposalWithinItsRepository(t *testing.T) {
	service := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "before"})
	git.Write("a.go", "after")
	view := createProposal(t, service, repoFor(git), "a.go")

	ownerID, status, detail, err := service.Resolver(repoFor(git)).Resolve(context.Background(), "Vrooli-Proposal", view.ID)
	if err != nil || ownerID != view.ID || status != trailers.Resolved || detail != "proposal open" {
		t.Fatalf("resolution = (%q, %q, %q, %v)", ownerID, status, detail, err)
	}
}

func TestLocalResolverDoesNotExposeProposalsFromOtherRepositories(t *testing.T) {
	service := testService(t)
	git := NewMemoryGit(map[string]string{"a.go": "before"})
	git.Write("a.go", "after")
	view := createProposal(t, service, repoFor(git), "a.go")

	ownerID, status, detail, err := service.Resolver(Repo{ID: "2", Git: git}).Resolve(context.Background(), "Vrooli-Proposal", view.ID)
	if err != nil || ownerID != "" || status != trailers.Inaccessible || detail != "" {
		t.Fatalf("cross-repository resolution = (%q, %q, %q, %v)", ownerID, status, detail, err)
	}
}

func TestLocalResolverReportsMissingProposalAndCommitWithoutErrors(t *testing.T) {
	service := testService(t)
	git := NewMemoryGit(nil)
	resolver := service.Resolver(repoFor(git))

	cases := []struct {
		name, key, value, wantDetail string
	}{
		{name: "proposal", key: "Vrooli-Proposal", value: "gctp-aaaaaaaa", wantDetail: "no proposal with this ID in Git Control Tower"},
		{name: "commit", key: "Vrooli-Continues", value: "0123456", wantDetail: "commit not found in this repository"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ownerID, status, detail, err := resolver.Resolve(context.Background(), tc.key, tc.value)
			if err != nil || ownerID != "" || status != trailers.Unresolved || detail != tc.wantDetail {
				t.Fatalf("resolution = (%q, %q, %q, %v)", ownerID, status, detail, err)
			}
		})
	}
}

func TestLocalResolverDelegatesOwnerKindsAndKeepsUnavailableEvidenceUnresolved(t *testing.T) {
	owner := &fixedTrailerResolver{ownerID: "effort-17", status: trailers.Resolved, detail: "current effort"}
	service := testService(t).WithResolver(owner)
	resolver := service.Resolver(repoFor(NewMemoryGit(nil)))

	ownerID, status, detail, err := resolver.Resolve(context.Background(), "Vrooli-Effort", "effort:bas@9")
	if err != nil || ownerID != "effort-17" || status != trailers.Resolved || detail != "current effort" {
		t.Fatalf("delegated resolution = (%q, %q, %q, %v)", ownerID, status, detail, err)
	}
	if len(owner.keys) != 1 || owner.keys[0] != "Vrooli-Effort:effort:bas@9" {
		t.Fatalf("delegated references = %v", owner.keys)
	}

	withoutOwner := testService(t).Resolver(repoFor(NewMemoryGit(nil)))
	ownerID, status, detail, err = withoutOwner.Resolve(context.Background(), "Vrooli-Effort", "effort:bas@9")
	if err != nil || ownerID != "" || status != trailers.Unresolved || detail != "owner evidence unavailable for effort" {
		t.Fatalf("unavailable resolution = (%q, %q, %q, %v)", ownerID, status, detail, err)
	}
}

func TestLocalResolverPropagatesOwnerAndStoreFailures(t *testing.T) {
	ownerErr := errors.New("owner read failed")
	service := testService(t).WithResolver(&fixedTrailerResolver{err: ownerErr})
	_, _, _, err := service.Resolver(repoFor(NewMemoryGit(nil))).Resolve(context.Background(), "Vrooli-Effort", "effort:bas@9")
	if !errors.Is(err, ownerErr) {
		t.Fatalf("owner error = %v", err)
	}

	service = testService(t)
	if err := service.store.db.Close(); err != nil {
		t.Fatal(err)
	}
	_, _, _, err = service.Resolver(repoFor(NewMemoryGit(nil))).Resolve(context.Background(), "Vrooli-Proposal", "gctp-aaaaaaaa")
	if err == nil {
		t.Fatal("closed proposal store resolved without an error")
	}
}
