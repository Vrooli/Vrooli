package sources

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/workloadowner"
	checksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestHostPressureCellPreservesAuthoredUnits(t *testing.T) {
	observedAt := time.Now().UTC()
	reader := AutohealReader{}
	latest := map[string]*checksv1.CheckResult{
		"system-host-pressure": {
			CheckId:     "system-host-pressure",
			Status:      checksv1.CheckStatus_CHECK_STATUS_OK,
			ObservedAt:  timestamppb.New(observedAt),
			DetailsJson: `{"cpu_pressure_percent":12.5,"stranded_memory_mb":42,"fork_rate_per_second":18}`,
		},
	}
	for _, test := range []struct {
		cell, unit string
		value      float64
	}{
		{cell: "substrate/SB14", unit: "percent", value: 12.5},
		{cell: "substrate/SB15", unit: "megabytes of stranded memory", value: 42},
		{cell: "substrate/SB16", unit: "forks per second", value: 18},
	} {
		observation, handled := reader.readHostPressureCell(context.Background(), test.cell, latest, checkQualifiers{}, observedAt)
		if !handled || observation.Unit != test.unit || observation.Value != test.value {
			t.Fatalf("%s observation=%+v handled=%v", test.cell, observation, handled)
		}
	}
}

func TestHostPressureWorkloadCellUsesOwnershipReport(t *testing.T) {
	reader := AutohealReader{Workloads: func(context.Context) (workloadowner.Report, error) {
		return workloadowner.Report{Findings: []workloadowner.Finding{{Class: workloadowner.Abandoned, Finding: true}}}, nil
	}}
	observation, handled := reader.readHostPressureCell(context.Background(), "substrate/SB17", nil, checkQualifiers{}, time.Now().UTC())
	if !handled || observation.Value != 1 || observation.Unit != "unmanaged workloads" {
		t.Fatalf("ownership observation=%+v handled=%v", observation, handled)
	}
}

func TestHostPressureCellReportsUnreadDetails(t *testing.T) {
	reader := AutohealReader{}
	observation, handled := reader.readHostPressureCell(context.Background(), "substrate/SB14", map[string]*checksv1.CheckResult{
		"system-host-pressure": {DetailsJson: `{}`},
	}, checkQualifiers{}, time.Now().UTC())
	if !handled || !observation.TrustHints.Untrusted || observation.Unit != "percent" {
		t.Fatalf("unread observation=%+v handled=%v", observation, handled)
	}
}
