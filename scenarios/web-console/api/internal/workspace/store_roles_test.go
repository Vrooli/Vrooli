package workspace

import (
	"context"
	"testing"
)

// roleStore is the slice of Store the role tests exercise. Running the same
// sequence through both implementations is the point: a behaviour that holds
// only in MemStore would pass every handler test and fail in production.
type roleStore interface {
	CreateGroup(ctx context.Context, name, color string) (Group, error)
	DeleteGroup(ctx context.Context, id string) (bool, error)
	GetLayout(ctx context.Context) (Layout, error)
	ListRoles(ctx context.Context, groupID string) ([]Role, error)
	CreateRole(ctx context.Context, req CreateRoleRequest) (Role, error)
	UpdateRole(ctx context.Context, req UpdateRoleRequest) (Role, error)
	DeleteRole(ctx context.Context, id string) (bool, error)
	ReassignRoleSession(ctx context.Context, oldSessionID, newSessionID string) error
}

func eachRoleStore(t *testing.T, run func(t *testing.T, s roleStore)) {
	t.Helper()
	t.Run("mem", func(t *testing.T) { run(t, NewMemStore()) })
	t.Run("sql", func(t *testing.T) { run(t, newTestSQLStore(t)) })
}

func TestRoleLifecycleMatchesAcrossStores(t *testing.T) {
	eachRoleStore(t, func(t *testing.T, s roleStore) {
		ctx := context.Background()
		group, err := s.CreateGroup(ctx, "Ship it", "#22d3ee")
		if err != nil {
			t.Fatal(err)
		}

		// Three roles, deliberately not two: a pair-shaped model would pass a
		// two-role test and fail here.
		labels := []string{"Planner", "Implementer", "Critic"}
		created := make([]Role, 0, len(labels))
		for _, label := range labels {
			r, err := s.CreateRole(ctx, CreateRoleRequest{GroupID: group.ID, Label: label, Command: "agent " + label})
			if err != nil {
				t.Fatalf("create %s: %v", label, err)
			}
			created = append(created, r)
		}

		roles, err := s.ListRoles(ctx, group.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(roles) != 3 {
			t.Fatalf("ListRoles returned %d roles, want 3", len(roles))
		}
		for i, r := range roles {
			if r.Label != labels[i] {
				t.Fatalf("role %d label = %q, want %q (sort_order not honoured)", i, r.Label, labels[i])
			}
			if !r.IsWaiting() {
				t.Fatalf("role %q should be waiting on create with no session id", r.Label)
			}
		}

		// Starting a role is an ordinary update that sets the session id.
		started, err := s.UpdateRole(ctx, UpdateRoleRequest{ID: created[1].ID, HasSessionID: true, SessionID: "sess-impl"})
		if err != nil {
			t.Fatal(err)
		}
		if started.IsWaiting() || started.SessionID != "sess-impl" {
			t.Fatalf("started role = %#v, want session sess-impl", started)
		}

		// ...and returning it to waiting is the same update with an empty id.
		back, err := s.UpdateRole(ctx, UpdateRoleRequest{ID: created[1].ID, HasSessionID: true, SessionID: ""})
		if err != nil {
			t.Fatal(err)
		}
		if !back.IsWaiting() {
			t.Fatalf("cleared role = %#v, want waiting", back)
		}

		removed, err := s.DeleteRole(ctx, created[2].ID)
		if err != nil || !removed {
			t.Fatalf("DeleteRole = %v, %v", removed, err)
		}
		// Idempotent: deleting it again is the state the caller asked for.
		removed, err = s.DeleteRole(ctx, created[2].ID)
		if err != nil || removed {
			t.Fatalf("second DeleteRole = %v, %v; want false, nil", removed, err)
		}
	})
}

func TestUpdateRoleRejectsUnknownID(t *testing.T) {
	eachRoleStore(t, func(t *testing.T, s roleStore) {
		if _, err := s.UpdateRole(context.Background(), UpdateRoleRequest{ID: "missing"}); err != ErrRoleNotFound {
			t.Fatalf("UpdateRole(missing) = %v, want ErrRoleNotFound", err)
		}
	})
}

func TestCreateRoleRejectsBlankGroup(t *testing.T) {
	eachRoleStore(t, func(t *testing.T, s roleStore) {
		if _, err := s.CreateRole(context.Background(), CreateRoleRequest{Label: "Orphan"}); err != ErrInvalidRole {
			t.Fatalf("CreateRole(no group) = %v, want ErrInvalidRole", err)
		}
	})
}

// TestManyRolesMayWaitButOneSessionBacksOneRole is the partial-unique-index
// contract. Both halves matter: if NULLs collided, a group could hold only
// one waiting role; if running ids did not collide, two roles could claim the
// same terminal and a handoff would be delivered twice.
func TestManyRolesMayWaitButOneSessionBacksOneRole(t *testing.T) {
	eachRoleStore(t, func(t *testing.T, s roleStore) {
		ctx := context.Background()
		group, err := s.CreateGroup(ctx, "Ship it", "#22d3ee")
		if err != nil {
			t.Fatal(err)
		}

		for i := 0; i < 3; i++ {
			if _, err := s.CreateRole(ctx, CreateRoleRequest{GroupID: group.ID, Label: "Waiting"}); err != nil {
				t.Fatalf("waiting role %d rejected: %v", i, err)
			}
		}

		if _, err := s.CreateRole(ctx, CreateRoleRequest{GroupID: group.ID, Label: "Running", SessionID: "sess-1"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.CreateRole(ctx, CreateRoleRequest{GroupID: group.ID, Label: "Duplicate", SessionID: "sess-1"}); err != ErrInvalidRole {
			t.Fatalf("second role for sess-1 = %v, want ErrInvalidRole", err)
		}
	})
}

// TestDeletingGroupDeletesItsRoles pins the deliberate difference from panes:
// a pane survives its group (ON DELETE SET NULL) because it owns a live
// session; a role has no meaning outside its group.
func TestDeletingGroupDeletesItsRoles(t *testing.T) {
	eachRoleStore(t, func(t *testing.T, s roleStore) {
		ctx := context.Background()
		group, err := s.CreateGroup(ctx, "Ship it", "#22d3ee")
		if err != nil {
			t.Fatal(err)
		}
		other, err := s.CreateGroup(ctx, "Keep me", "#f59e0b")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.CreateRole(ctx, CreateRoleRequest{GroupID: group.ID, Label: "Doomed"}); err != nil {
			t.Fatal(err)
		}
		if _, err := s.CreateRole(ctx, CreateRoleRequest{GroupID: other.ID, Label: "Survivor"}); err != nil {
			t.Fatal(err)
		}

		if _, err := s.DeleteGroup(ctx, group.ID); err != nil {
			t.Fatal(err)
		}

		roles, err := s.ListRoles(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		if len(roles) != 1 || roles[0].Label != "Survivor" {
			t.Fatalf("after group delete roles = %#v, want only Survivor", roles)
		}
	})
}

// TestGetLayoutIncludesRoles pins the one-round-trip constraint.
func TestGetLayoutIncludesRoles(t *testing.T) {
	eachRoleStore(t, func(t *testing.T, s roleStore) {
		ctx := context.Background()
		group, err := s.CreateGroup(ctx, "Ship it", "#22d3ee")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.CreateRole(ctx, CreateRoleRequest{GroupID: group.ID, Label: "Implementer"}); err != nil {
			t.Fatal(err)
		}

		layout, err := s.GetLayout(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(layout.Roles) != 1 || layout.Roles[0].Label != "Implementer" {
			t.Fatalf("GetLayout roles = %#v, want one Implementer", layout.Roles)
		}
	})
}

// TestReassignRoleSessionFollowsRecovery is the answer to the session-recovery
// question: a recovered session KEEPS its role rather than dropping back to
// waiting, so the operator's running work is not silently discarded.
func TestReassignRoleSessionFollowsRecovery(t *testing.T) {
	eachRoleStore(t, func(t *testing.T, s roleStore) {
		ctx := context.Background()
		group, err := s.CreateGroup(ctx, "Ship it", "#22d3ee")
		if err != nil {
			t.Fatal(err)
		}
		role, err := s.CreateRole(ctx, CreateRoleRequest{GroupID: group.ID, Label: "Implementer", SessionID: "old-session"})
		if err != nil {
			t.Fatal(err)
		}

		if err := s.ReassignRoleSession(ctx, "old-session", "new-session"); err != nil {
			t.Fatal(err)
		}

		roles, err := s.ListRoles(ctx, group.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(roles) != 1 || roles[0].ID != role.ID || roles[0].SessionID != "new-session" {
			t.Fatalf("after reassignment roles = %#v, want %s pointing at new-session", roles, role.ID)
		}
	})
}

// TestReassignRoleSessionReleasesTheReplacement covers the case that would
// otherwise trip the unique index: recovery created a fresh session that a
// default role already claims before the original is moved onto it.
func TestReassignRoleSessionReleasesTheReplacement(t *testing.T) {
	eachRoleStore(t, func(t *testing.T, s roleStore) {
		ctx := context.Background()
		group, err := s.CreateGroup(ctx, "Ship it", "#22d3ee")
		if err != nil {
			t.Fatal(err)
		}
		original, err := s.CreateRole(ctx, CreateRoleRequest{GroupID: group.ID, Label: "Implementer", SessionID: "old-session"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.CreateRole(ctx, CreateRoleRequest{GroupID: group.ID, Label: "Stand-in", SessionID: "new-session"}); err != nil {
			t.Fatal(err)
		}

		if err := s.ReassignRoleSession(ctx, "old-session", "new-session"); err != nil {
			t.Fatalf("reassignment blocked by the replacement claim: %v", err)
		}

		roles, err := s.ListRoles(ctx, group.ID)
		if err != nil {
			t.Fatal(err)
		}
		var holders int
		for _, r := range roles {
			if r.SessionID == "new-session" {
				holders++
				if r.ID != original.ID {
					t.Fatalf("new-session held by %q, want the original role", r.Label)
				}
			}
		}
		if holders != 1 {
			t.Fatalf("%d roles hold new-session, want exactly 1", holders)
		}
	})
}
