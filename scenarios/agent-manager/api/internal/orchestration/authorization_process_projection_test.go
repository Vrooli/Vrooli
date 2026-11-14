//go:build linux

package orchestration

import (
	"context"
	"errors"
	"os"
	"testing"

	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/vrooli/cli-core/cliutil"
)

type processProjectionRepository struct {
	repository.RunRepository
	candidates []repository.RunProcessBinding
	err        error
	calls      int
}

func (r *processProjectionRepository) ListManagedRunProcesses(context.Context) ([]repository.RunProcessBinding, error) {
	r.calls++
	return r.candidates, r.err
}

func TestManagedAuthorizationProjectionRetainsLiveProofAndAmbiguityChecks(t *testing.T) {
	binding, err := cliutil.ObserveProcessBinding(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	for _, tc := range []struct {
		name       string
		candidates []repository.RunProcessBinding
		readErr    error
		want       uuid.UUID
		wantErr    bool
	}{
		{name: "empty"},
		{name: "verified same process", candidates: []repository.RunProcessBinding{{ID: id, RunnerPID: os.Getpid()}}, want: id},
		{name: "same identity twice", candidates: []repository.RunProcessBinding{{ID: id, RunnerPID: os.Getpid()}, {ID: id, RunnerPID: os.Getpid()}}, want: id},
		{name: "ambiguous", candidates: []repository.RunProcessBinding{{ID: id, RunnerPID: os.Getpid()}, {ID: uuid.New(), RunnerPID: os.Getpid()}}, wantErr: true},
		{name: "dead locator", candidates: []repository.RunProcessBinding{{ID: id, RunnerPID: 1 << 30}}},
		{name: "invalid PID", candidates: []repository.RunProcessBinding{{ID: id, RunnerPID: 1}}, wantErr: true},
		{name: "invalid identity", candidates: []repository.RunProcessBinding{{RunnerPID: os.Getpid()}}, wantErr: true},
		{name: "inventory error", readErr: errors.New("inventory failed"), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &processProjectionRepository{candidates: tc.candidates, err: tc.readErr}
			o := &Orchestrator{runs: repo}
			got, err := o.managedAuthorizationRun(context.Background(), binding)
			if (err != nil) != tc.wantErr || got != tc.want || repo.calls != 1 {
				t.Fatalf("got %s, err=%v, calls=%d; want %s, error=%v", got, err, repo.calls, tc.want, tc.wantErr)
			}
		})
	}
}
