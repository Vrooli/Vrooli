package system

import (
	"context"
	"io"
	"testing"

	"github.com/vrooli/vrooli/internal/hostpressure"
	"github.com/vrooli/vrooli/internal/resources"
)

type reclaimLifecycleFake struct {
	names     []string
	serving   bool
	run       []string
	statusErr error
}

func (f *reclaimLifecycleFake) EnabledResourceNames() ([]string, error) { return f.names, nil }
func (f *reclaimLifecycleFake) Status(string, bool) (resources.Status, error) {
	if f.statusErr != nil {
		return resources.Status{}, f.statusErr
	}
	return resources.Status{Serving: &f.serving}, nil
}

func (f *reclaimLifecycleFake) Run(name string, args []string, _, _ io.Writer) error {
	f.run = append(f.run, name+":"+args[0])
	return nil
}

func TestProductionReclaimerRecyclesOneDeclaredIdleResource(t *testing.T) {
	fake := &reclaimLifecycleFake{names: []string{"reranker"}}
	reclaim := newHostPressureReclaimer(fake, 50, func(context.Context) hostpressure.PressureSnapshot {
		return hostpressure.PressureSnapshot{
			CPUPressure: hostpressure.NewRead(10, "test"),
			Processes:   []hostpressure.Process{{PID: 77, Name: "reranker_linux_amd64", Resident: 10, Swapped: 2_000_000_000}},
		}
	})
	message, err := reclaim(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.run) != 1 || fake.run[0] != "reranker:restart" {
		t.Fatalf("run=%v, want one governed reranker restart", fake.run)
	}
	if message == "" {
		t.Fatal("expected reclaim evidence")
	}
}

func TestProductionReclaimerBrakesSaturationAndUnreadServingState(t *testing.T) {
	tests := []struct {
		name      string
		cpu       float64
		statusErr error
		wantRun   int
	}{
		{name: "saturated", cpu: 90, wantRun: 0},
		{name: "serving unread", cpu: 10, statusErr: context.DeadlineExceeded, wantRun: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &reclaimLifecycleFake{names: []string{"reranker"}, statusErr: tt.statusErr}
			reclaim := newHostPressureReclaimer(fake, 50, func(context.Context) hostpressure.PressureSnapshot {
				return hostpressure.PressureSnapshot{
					CPUPressure: hostpressure.NewRead(tt.cpu, "test"),
					Processes:   []hostpressure.Process{{PID: 77, Name: "reranker_linux_amd64", Resident: 10, Swapped: 2_000_000_000}},
				}
			})
			if _, err := reclaim(context.Background()); err == nil && tt.statusErr != nil {
				// A serving-state probe error must be surfaced rather than treated
				// as permission to recycle.
				t.Fatal("expected unread serving state to refuse reclaim")
			}
			if len(fake.run) != tt.wantRun {
				t.Fatalf("run=%v, want %d actions", fake.run, tt.wantRun)
			}
		})
	}
}
