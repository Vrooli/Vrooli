package resources

import (
	"context"
	"errors"
	"strings"
	"testing"

	platform "github.com/vrooli/platform-go"
)

func updateOriginalScopeFixture(u ManagedServiceUpdate) *platform.ServiceScopeRef {
	name := resourceServiceScopeName(u.Resource, "service")
	_, ticks, _ := strings.Cut(u.Original.Incarnation, ":")
	return &platform.ServiceScopeRef{Scope: platform.ScopeRef{Name: name, Kind: platform.ScopeKindCgroup, Path: "/user.slice/" + platform.ServicesSlice + "/" + name + ".scope"}, RootPID: u.Original.PID, RootStartTime: ticks, NativeInvocationID: strings.Repeat("a", 32), ScopeDevice: 31, ScopeInode: 99}
}

func TestManagedUpdateOriginalScopeBindsExactLegacyNativeIdentity(t *testing.T) {
	u := updateTestInitial(t)
	u.OriginalScope = updateOriginalScopeFixture(u)
	if err := u.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"root", "incarnation", "resource", "inode", "invocation"} {
		t.Run(field, func(t *testing.T) {
			bad := updateTestClone(t, u)
			switch field {
			case "root":
				bad.OriginalScope.RootPID++
			case "incarnation":
				bad.OriginalScope.RootStartTime += "0"
			case "resource":
				bad.OriginalScope.Scope.Name = "vrooli-service-resource-other-service"
				bad.OriginalScope.Scope.Path = "/user.slice/" + platform.ServicesSlice + "/" + bad.OriginalScope.Scope.Name + ".scope"
			case "inode":
				bad.OriginalScope.ScopeInode = 0
			case "invocation":
				bad.OriginalScope.NativeInvocationID = ""
			}
			if bad.Validate() == nil {
				t.Fatal("mismatched original scope was accepted")
			}
		})
	}
}

func TestManagedUpdateOriginalScopeCannotChangeOnReplayOrTransition(t *testing.T) {
	u := updateTestInitial(t)
	u.OriginalScope = updateOriginalScopeFixture(u)
	s := updateTestStore(t, u)
	if _, _, err := s.Create(context.Background(), u, false); err != nil {
		t.Fatal(err)
	}
	changed := updateTestClone(t, u)
	changed.OriginalScope.ScopeInode++
	if _, _, err := s.Create(context.Background(), changed, false); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatal("lost-reply replay changed the original native owner")
	}
	changed.Candidate.Intent = updateTestIntent("candidate", changed.Target)
	if _, err := s.Commit(context.Background(), changed, UpdateLaunchCandidate); !errors.Is(err, ErrManagedUpdateConflict) {
		t.Fatal("transition replaced the original native owner")
	}
}
