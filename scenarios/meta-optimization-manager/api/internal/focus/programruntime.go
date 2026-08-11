package focus

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
)

const programRuntimeReadDeadline = 3 * time.Second

type ProgramFailureObservation struct {
	Shape, FirstSeen, LastSeen, SampleProgramID string
	Count                                       int
}

type ProgramRefusalObservation struct {
	BindingID, Reason, LastSeen string
	Count                       int
}

type ProgramUnresolvedObservation struct {
	AttemptedName, LastSeen string
	Count                   int
}

type ProgramFrictionReport struct {
	Failures   []ProgramFailureObservation
	Refusals   []ProgramRefusalObservation
	Unresolved []ProgramUnresolvedObservation
}

// ProgramFrictionReader is deliberately declared by the consumer. The focus
// domain reads a bounded, typed projection and never imports program-runtime's
// persistence or governance implementation.
type ProgramFrictionReader interface {
	ReadFriction(context.Context) (ProgramFrictionReport, error)
}

type programRuntimeGapSource struct{ reader ProgramFrictionReader }

func NewProgramRuntimeGapSource(reader ProgramFrictionReader) GapSource {
	return &programRuntimeGapSource{reader: reader}
}

var _ GapSource = (*programRuntimeGapSource)(nil)

func (*programRuntimeGapSource) Axis() Axis { return AxisEmpirical }

func (s *programRuntimeGapSource) DerivedGaps(ctx context.Context) ([]Gap, error) {
	if s.reader == nil {
		return nil, fmt.Errorf("program-runtime friction reader is not configured")
	}
	report, err := s.reader.ReadFriction(ctx)
	if err != nil {
		return nil, fmt.Errorf("read program-runtime friction: %w", err)
	}
	out := make([]Gap, 0, len(report.Failures)+len(report.Refusals)+len(report.Unresolved))
	for _, failure := range report.Failures {
		if strings.TrimSpace(failure.Shape) == "" || failure.Count <= 0 {
			continue
		}
		out = append(out, Gap{
			ID: "empirical/program-runtime/failure/" + failure.Shape, Axis: AxisEmpirical,
			Title: "program-runtime failure shape recurs: " + failure.Shape, Global: true,
			EvidenceSource: "program-runtime", EvidenceLocator: "program-runtime://programs/" + failure.SampleProgramID,
			Recurrence: failure.Count,
			Notes:      []string{fmt.Sprintf("first_seen=%s; last_seen=%s", failure.FirstSeen, failure.LastSeen)},
		})
	}
	for _, refusal := range report.Refusals {
		if refusal.BindingID == "" || refusal.Count <= 0 {
			continue
		}
		out = append(out, Gap{
			ID: "empirical/program-runtime/refusal/" + refusal.BindingID + "/" + refusal.Reason, Axis: AxisEmpirical,
			Title: "program-runtime binding refusal recurs: " + refusal.BindingID, Global: true,
			EvidenceSource: "program-runtime", EvidenceLocator: "program-runtime://refusals/" + refusal.BindingID,
			Recurrence: refusal.Count, Notes: []string{"reason=" + refusal.Reason, "last_seen=" + refusal.LastSeen},
		})
	}
	for _, unresolved := range report.Unresolved {
		if unresolved.AttemptedName == "" || unresolved.Count <= 0 {
			continue
		}
		out = append(out, Gap{
			ID: "empirical/program-runtime/unresolved/" + unresolved.AttemptedName, Axis: AxisEmpirical,
			Title: "program-runtime attempted an ungoverned binding: " + unresolved.AttemptedName, Global: true,
			EvidenceSource: "program-runtime", EvidenceLocator: "program-runtime://unresolved/" + unresolved.AttemptedName,
			Recurrence: unresolved.Count, Notes: []string{"last_seen=" + unresolved.LastSeen},
		})
	}
	return out, nil
}

type programRuntimeFrictionReader struct {
	resolver scenarioURLResolver
	http     connect.HTTPClient
	deadline time.Duration
}

func NewProgramRuntimeFrictionReader() ProgramFrictionReader {
	return &programRuntimeFrictionReader{
		resolver: discovery.NewResolver(discovery.ResolverConfig{}),
		http:     &http.Client{Timeout: programRuntimeReadDeadline},
		deadline: programRuntimeReadDeadline,
	}
}

func (r *programRuntimeFrictionReader) ReadFriction(ctx context.Context) (ProgramFrictionReport, error) {
	ctx, cancel := context.WithTimeout(ctx, r.deadline)
	defer cancel()
	base, err := r.resolver.ResolveScenarioURLDefault(ctx, "program-runtime")
	if err != nil {
		return ProgramFrictionReport{}, fmt.Errorf("resolve program-runtime: %w", err)
	}
	client := programsconnect.NewProgramServiceClient(r.http, base)
	failures, err := client.MineFailures(ctx, connect.NewRequest(&programsv1.MineFailuresRequest{IncludeOperator: false}))
	if err != nil {
		return ProgramFrictionReport{}, fmt.Errorf("mine program failures: %w", err)
	}
	refusals, err := client.MineRefusals(ctx, connect.NewRequest(&programsv1.MineRefusalsRequest{IncludeOperator: false}))
	if err != nil {
		return ProgramFrictionReport{}, fmt.Errorf("mine binding refusals: %w", err)
	}
	unresolved, err := client.MineUnresolvedBindings(ctx, connect.NewRequest(&programsv1.MineUnresolvedBindingsRequest{}))
	if err != nil {
		return ProgramFrictionReport{}, fmt.Errorf("mine unresolved bindings: %w", err)
	}
	report := ProgramFrictionReport{}
	for _, shape := range failures.Msg.GetShapes() {
		report.Failures = append(report.Failures, ProgramFailureObservation{Shape: shape.GetShape(), Count: int(shape.GetCount()), FirstSeen: shape.GetFirstSeen(), LastSeen: shape.GetLastSeen(), SampleProgramID: shape.GetSampleProgramId()})
	}
	for _, shape := range refusals.Msg.GetShapes() {
		report.Refusals = append(report.Refusals, ProgramRefusalObservation{BindingID: shape.GetBindingId(), Reason: shape.GetReason(), Count: int(shape.GetCount()), LastSeen: shape.GetLastSeen()})
	}
	for _, shape := range unresolved.Msg.GetShapes() {
		report.Unresolved = append(report.Unresolved, ProgramUnresolvedObservation{AttemptedName: shape.GetAttemptedName(), Count: int(shape.GetCount()), LastSeen: shape.GetLastSeen()})
	}
	return report, nil
}
