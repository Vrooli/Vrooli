package ladder

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	ladderv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/ladder"
	internalcondition "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/condition"
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/focus"
	internalladder "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/ladder"
)

type stubSnapshotter struct{ snapshot internalladder.Snapshot }

func (s stubSnapshotter) Snapshot(context.Context) internalladder.Snapshot { return s.snapshot }

func fixture() internalladder.Snapshot {
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
	return internalladder.Snapshot{
		HostOS:            "linux",
		ComputedAt:        now,
		CoverageAvailable: true,
		Cells: []internalladder.Cell{
			{
				Key:         internalladder.CellKey{DeviceClass: "thermal-sensor", Rung: internalladder.RungTelemetry, HostOS: "linux"},
				CellRef:     "substrate/SB11",
				Status:      "IN-REACH",
				Observation: internalladder.ObservationMeasured,
				Trust:       internalcondition.TrustValid,
				Band:        internalcondition.BandInBand,
				Graded:      true,
				BarID:       "substrate-thermal-telemetry",
				ObservedAt:  now,
			},
			{
				Key:         internalladder.CellKey{DeviceClass: "block-device", Rung: internalladder.RungAnticipation, HostOS: "macos"},
				CellRef:     "substrate/SB10",
				Status:      "IN-REACH",
				Observation: internalladder.ObservationUnread,
				Trust:       internalcondition.TrustUnavailable,
				Band:        internalcondition.BandNotEvaluated,
				ObservedAt:  now,
			},
		},
		Sources: []internalladder.SourceState{
			{ID: "system-monitor/device-graph", Available: false, Reason: "connection refused", CheckedAt: now},
		},
		Findings: []focus.RankedFinding{
			{
				Finding:         focus.Finding{ID: "source-unavailable/system-monitor/device-graph", Stage: focus.StageIntegrity, Severity: 10},
				Rank:            1,
				RankExplanation: "sensor-channel integrity outranks plant condition",
			},
			{
				Finding:         focus.Finding{ID: "ungraded/substrate/SB10", Stage: focus.StageMeasurement, Severity: 1},
				Rank:            2,
				RankExplanation: "measurement improvement follows operational findings",
			},
		},
	}
}

func handler() *connectHandler { return &connectHandler{service: stubSnapshotter{snapshot: fixture()}} }

// TestListCellsFiltersAreConjunctive pins that each filter narrows rather than
// widens: a filter that silently matched everything would report a targeted
// question's answer as the whole grid.
func TestListCellsFiltersAreConjunctive(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		request *ladderv1.ListCellsRequest
		want    int
	}{
		{"no filter", &ladderv1.ListCellsRequest{}, 2},
		{"device class", &ladderv1.ListCellsRequest{DeviceClass: "thermal-sensor"}, 1},
		{"host os", &ladderv1.ListCellsRequest{HostOs: "macos"}, 1},
		{"cell ref", &ladderv1.ListCellsRequest{CellRef: "substrate/SB11"}, 1},
		{"rung", &ladderv1.ListCellsRequest{Rung: ladderv1.Rung_RUNG_ANTICIPATION}, 1},
		{"contradictory filters", &ladderv1.ListCellsRequest{DeviceClass: "thermal-sensor", HostOs: "macos"}, 0},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			response, err := handler().ListCells(context.Background(), connect.NewRequest(testCase.request))
			if err != nil {
				t.Fatal(err)
			}
			if got := len(response.Msg.GetCells()); got != testCase.want {
				t.Fatalf("got %d cells, want %d", got, testCase.want)
			}
		})
	}
}

// TestRankFindingsStatesTheCascadeItApplied is operating-model rule 7's
// reporting half: a ranking whose order the reader cannot see is
// indistinguishable from a ranking bug.
func TestRankFindingsStatesTheCascadeItApplied(t *testing.T) {
	response, err := handler().RankFindings(context.Background(), connect.NewRequest(&ladderv1.RankFindingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetAppliedCascade() != appliedCascade {
		t.Fatalf("the response states cascade %q, want %q", response.Msg.GetAppliedCascade(), appliedCascade)
	}
	findings := response.Msg.GetFindings()
	if len(findings) != 2 {
		t.Fatalf("got %d findings, want 2", len(findings))
	}
	if findings[0].GetStage() != ladderv1.CascadeStage_CASCADE_STAGE_SENSOR_CHANNEL_INTEGRITY {
		t.Errorf("the first finding is stage %v, want sensor-channel integrity", findings[0].GetStage())
	}
	if findings[1].GetStage() != ladderv1.CascadeStage_CASCADE_STAGE_MEASUREMENT_IMPROVEMENT {
		t.Errorf("the second finding is stage %v, want measurement improvement", findings[1].GetStage())
	}
	for _, finding := range findings {
		if finding.GetStageExplanation() == "" {
			t.Errorf("finding %q states no stage explanation", finding.GetId())
		}
	}
}

func TestRankFindingsStageFilter(t *testing.T) {
	response, err := handler().RankFindings(context.Background(), connect.NewRequest(&ladderv1.RankFindingsRequest{
		Stage: ladderv1.CascadeStage_CASCADE_STAGE_SENSOR_CHANNEL_INTEGRITY,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(response.Msg.GetFindings()); got != 1 {
		t.Fatalf("got %d findings for one stage, want 1", got)
	}
}

// TestProjectionsNeverRenderAsUnspecified guards the failure mode where a
// shipped token resolves to no enum value and renders as UNSPECIFIED, which is
// indistinguishable from a deliberate one.
func TestProjectionsNeverRenderAsUnspecified(t *testing.T) {
	response, err := handler().GetLadder(context.Background(), connect.NewRequest(&ladderv1.GetLadderRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, cell := range response.Msg.GetLadder().GetCells() {
		if cell.GetRung() == ladderv1.Rung_RUNG_UNSPECIFIED {
			t.Errorf("%s renders its rung as UNSPECIFIED", cell.GetKey())
		}
		if cell.GetObservation() == ladderv1.Observation_OBSERVATION_UNSPECIFIED {
			t.Errorf("%s renders its observation as UNSPECIFIED", cell.GetKey())
		}
		if cell.GetStatus() == ladderv1.CellStatus_CELL_STATUS_UNSPECIFIED {
			t.Errorf("%s renders its status as UNSPECIFIED", cell.GetKey())
		}
		if cell.GetTrust() == ladderv1.TrustVerdict_TRUST_VERDICT_UNSPECIFIED {
			t.Errorf("%s renders its trust as UNSPECIFIED", cell.GetKey())
		}
		if cell.GetBand() == ladderv1.BandVerdict_BAND_VERDICT_UNSPECIFIED {
			t.Errorf("%s renders its band as UNSPECIFIED", cell.GetKey())
		}
	}
}

// TestEveryRungAndObservationTokenProjects walks both vocabularies so a token
// added to the Go model without a matching proto value fails here rather than
// rendering as UNSPECIFIED in production.
func TestEveryRungAndObservationTokenProjects(t *testing.T) {
	for _, rung := range internalladder.Rungs {
		if protoRung(rung) == ladderv1.Rung_RUNG_UNSPECIFIED {
			t.Errorf("rung %s has no proto value", rung)
		}
	}
	for _, observation := range []internalladder.Observation{
		internalladder.ObservationMeasured, internalladder.ObservationUnmeasurable,
		internalladder.ObservationUnavailable, internalladder.ObservationNotApplicable,
		internalladder.ObservationBlocked, internalladder.ObservationUnread,
	} {
		if protoObservation(observation) == ladderv1.Observation_OBSERVATION_UNSPECIFIED {
			t.Errorf("observation %s has no proto value", observation)
		}
	}
	for _, trust := range []internalcondition.TrustVerdict{
		internalcondition.TrustValid, internalcondition.TrustGhost, internalcondition.TrustSaturated,
		internalcondition.TrustShelved, internalcondition.TrustUnitMismatch,
		internalcondition.TrustUnavailable, internalcondition.TrustUntrusted,
	} {
		if protoTrust(trust) == ladderv1.TrustVerdict_TRUST_VERDICT_UNSPECIFIED {
			t.Errorf("trust verdict %s has no proto value", trust)
		}
	}
	for _, stage := range []focus.Stage{
		focus.StageIntegrity, focus.StageSubstrate, focus.StageAvailability,
		focus.StageEfficiency, focus.StageMeasurement,
	} {
		if protoStage(stage) == ladderv1.CascadeStage_CASCADE_STAGE_UNSPECIFIED {
			t.Errorf("cascade stage %v has no proto value", stage)
		}
	}
}
