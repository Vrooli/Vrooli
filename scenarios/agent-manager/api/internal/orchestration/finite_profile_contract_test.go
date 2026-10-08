package orchestration

import (
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/effortauthority"
	"testing"
	"time"
)

// Every unimplemented repository operation panics. Owner-contract reads cannot
// accidentally create/update/ensure a profile while passing this fixture.
type readinessProfileRepository struct {
	repository.ProfileRepository
	reads   int
	profile *domain.AgentProfile
	err     error
}

func (r *readinessProfileRepository) GetByKey(context.Context, string) (*domain.AgentProfile, error) {
	r.reads++
	return r.profile, r.err
}
func TestReadinessNativeProfileContractUsesCompleteOwnerObject(t *testing.T) {
	p := &domain.AgentProfile{ID: uuid.New(), ProfileKey: "bounded", Name: "owner profile", RoleRef: "code.default"}
	r := &readinessProfileRepository{profile: p}
	o := &Orchestrator{profiles: r}
	got, e := o.ReadFiniteProfileContract(context.Background(), "bounded")
	if e != nil || r.reads != 1 || got.ID != p.ID.String() || got.Key != p.ProfileKey || got.Digest != effortauthority.Digest(p) {
		t.Fatalf("complete owner contract mismatch: %#v %v reads=%d", got, e, r.reads)
	}
	p.Name = "changed owner name"
	changed, e := o.ReadFiniteProfileContract(context.Background(), "bounded")
	if e != nil || changed.Digest == got.Digest {
		t.Fatal("complete owner mutation absent from digest", e)
	}
}
func TestReadinessNativeProfileContractRefusesMissingOwnerBeforeEffects(t *testing.T) {
	for _, name := range []string{"nil-orchestrator", "nil-repository", "empty-key", "missing", "error", "foreign-key"} {
		t.Run(name, func(t *testing.T) {
			r := &readinessProfileRepository{profile: &domain.AgentProfile{ID: uuid.New(), ProfileKey: "bounded"}}
			o := &Orchestrator{profiles: r}
			key := "bounded"
			wantReads := 1
			switch name {
			case "nil-orchestrator":
				o = nil
				wantReads = 0
			case "nil-repository":
				o.profiles = nil
				wantReads = 0
			case "empty-key":
				key = ""
				wantReads = 0
			case "missing":
				r.profile = nil
			case "error":
				r.err = errors.New("owner read unavailable")
			case "foreign-key":
				r.profile.ProfileKey = "foreign"
			}
			if _, e := o.ReadFiniteProfileContract(context.Background(), key); e == nil {
				t.Fatal("invalid owner contract accepted")
			}
			if r.reads != wantReads {
				t.Fatalf("reads=%d want=%d", r.reads, wantReads)
			}
		})
	}
}

func TestReadinessNativeProfileDigestIncludesPermissionsAndMetadata(t *testing.T) {
	cases := map[string]func(*domain.AgentProfile){
		"allowed-tools": func(p *domain.AgentProfile) { p.AllowedTools = []string{"Write"} },
		"denied-tools":  func(p *domain.AgentProfile) { p.DeniedTools = []string{"Read"} },
		"scopes":        func(p *domain.AgentProfile) { p.DeclaredScopes = []string{"agent-manager:write"} },
		"paths":         func(p *domain.AgentProfile) { p.AllowedPaths = []string{"/approved"} },
		"permission":    func(p *domain.AgentProfile) { p.SkipPermissionPrompt = true },
		"network":       func(p *domain.AgentProfile) { p.NetworkAccess = domain.NetworkAccessNone },
		"timestamp":     func(p *domain.AgentProfile) { p.UpdatedAt = time.Unix(100, 0) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := &domain.AgentProfile{ID: uuid.New(), ProfileKey: "bounded"}
			repo := &readinessProfileRepository{profile: p}
			o := &Orchestrator{profiles: repo}
			before, e := o.ReadFiniteProfileContract(context.Background(), p.ProfileKey)
			if e != nil {
				t.Fatal(e)
			}
			mutate(p)
			after, e := o.ReadFiniteProfileContract(context.Background(), p.ProfileKey)
			if e != nil || before.Digest == after.Digest || repo.reads != 2 {
				t.Fatal("permission/metadata drift omitted", e)
			}
		})
	}
}
