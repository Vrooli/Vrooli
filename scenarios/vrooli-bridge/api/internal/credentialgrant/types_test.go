package credentialgrant

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeNodes struct{ kind string }

func (f fakeNodes) NodeKind(context.Context, string) (string, error) { return f.kind, nil }

type fakeRepo struct{ created []Grant }

func (f *fakeRepo) Create(_ context.Context, grant Grant) (Grant, error) {
	f.created = append(f.created, grant)
	return grant, nil
}
func (f *fakeRepo) List(context.Context, string) ([]Grant, error) { return f.created, nil }
func (f *fakeRepo) Revoke(context.Context, string) error          { return nil }
func (f *fakeRepo) Ack(context.Context, string, int64) error      { return nil }

func TestCreateRejectsUnsafeGrantCombinations(t *testing.T) {
	tests := []struct {
		name string
		in   CreateInput
		want string
	}{
		{"durable infrastructure", CreateInput{Class: ClassInfrastructure, Retention: RetentionDurable}, "cannot receive durable"},
		{"per install", CreateInput{Class: ClassPerInstallGenerated, Retention: RetentionEphemeral}, "generated locally"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			svc := NewService(&fakeRepo{}, fakeNodes{kind: "agent"}, time.Now)
			_, err := svc.Create(context.Background(), CreateInput{NodeID: "n1", LogicalID: "vrooli/test", Field: "value", Class: test.in.Class, Retention: test.in.Retention})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestCreateRejectsControlPlaneAndAllowsEphemeralInfrastructure(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeNodes{kind: "control_plane"}, time.Now)
	_, err := svc.Create(context.Background(), CreateInput{NodeID: "cp", LogicalID: "vrooli/test", Field: "value", Class: ClassUserPrompt, Retention: RetentionDurable})
	if err == nil || !strings.Contains(err.Error(), "control-plane") {
		t.Fatalf("control-plane grant error = %v", err)
	}

	repo := &fakeRepo{}
	svc = NewService(repo, fakeNodes{kind: "agent"}, func() time.Time { return time.Unix(10, 0) })
	grant, err := svc.Create(context.Background(), CreateInput{NodeID: "n1", LogicalID: "infra/repo", Field: "passphrase", Class: ClassInfrastructure, Retention: RetentionEphemeral})
	if err != nil {
		t.Fatal(err)
	}
	if grant.Generation != 1 || grant.Retention != RetentionEphemeral || len(repo.created) != 1 {
		t.Fatalf("grant = %+v, repo = %+v", grant, repo.created)
	}
}
