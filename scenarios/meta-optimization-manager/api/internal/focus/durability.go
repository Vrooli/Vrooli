package focus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

const durabilityMinimumSample = 3

const (
	LaneVerified = "verified"
	LaneObserved = "observed"
	LaneUnlinked = "unlinked"
)

type DurabilityObservation struct {
	RunID     string
	Verdict   string
	Sample    int
	Lane      string
	Reference string
}

type DurabilityReader interface {
	ReadDurability(context.Context) ([]DurabilityObservation, error)
}

type agentManagerDurabilityReader struct {
	client *http.Client
}

func NewAgentManagerDurabilityReader() DurabilityReader {
	return &agentManagerDurabilityReader{client: &http.Client{Timeout: 3 * time.Second}}
}

func (r *agentManagerDurabilityReader) ReadDurability(ctx context.Context) ([]DurabilityObservation, error) {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return nil, err
	}
	client := r.client
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/api/v1/runs?limit=25", nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return nil, fmt.Errorf("agent-manager runs returned %s", response.Status)
	}
	var runs struct {
		Runs []struct {
			ID string `json:"id"`
		} `json:"runs"`
	}
	if err := json.NewDecoder(response.Body).Decode(&runs); err != nil {
		return nil, err
	}
	out := make([]DurabilityObservation, 0, len(runs.Runs))
	for _, run := range runs.Runs {
		if strings.TrimSpace(run.ID) == "" {
			continue
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(base, "/")+"/api/v1/runs/"+run.ID+"/durability", nil)
		if err != nil {
			return nil, err
		}
		response, err := client.Do(request)
		if err != nil {
			return nil, err
		}
		var projection struct {
			Verdict    string `json:"verdict"`
			SampleSize int    `json:"sampleSize"`
			Lane       string `json:"lane"`
		}
		decodeErr := json.NewDecoder(response.Body).Decode(&projection)
		response.Body.Close()
		if decodeErr != nil || response.StatusCode/100 != 2 || projection.SampleSize < durabilityMinimumSample {
			continue
		}
		// The lane is the work's own attribution, reported by the projection.
		// Deriving it from evidence coverage would call work "unlinked" purely
		// because it happened to have no findings.
		lane := LaneUnlinked
		switch projection.Lane {
		case LaneVerified, LaneObserved:
			lane = projection.Lane
		}
		out = append(out, DurabilityObservation{RunID: run.ID, Verdict: projection.Verdict, Sample: projection.SampleSize, Lane: lane, Reference: "agent-manager://runs/" + run.ID + "/durability"})
	}
	return out, nil
}

type durabilityGapSource struct{ reader DurabilityReader }

func NewDurabilityGapSource(reader DurabilityReader) GapSource {
	return &durabilityGapSource{reader: reader}
}

func (s *durabilityGapSource) Axis() Axis { return AxisEmpirical }

func (s *durabilityGapSource) DerivedGaps(ctx context.Context) ([]Gap, error) {
	if s.reader == nil {
		return nil, fmt.Errorf("durability reader is not configured")
	}
	observations, err := s.reader.ReadDurability(ctx)
	if err != nil {
		return nil, err
	}
	if len(observations) < durabilityMinimumSample {
		return nil, nil
	}
	verified, observed, unlinked := 0, 0, 0
	for _, item := range observations {
		switch item.Lane {
		case LaneVerified:
			verified++
		case LaneObserved:
			observed++
		default:
			unlinked++
		}
	}
	newest := observations[len(observations)-1]
	return []Gap{{
		ID:              "empirical/agent-manager/durability",
		Axis:            AxisEmpirical,
		Title:           "agent-manager durability lane",
		Global:          true,
		EvidenceSource:  "agent-manager",
		EvidenceLocator: newest.Reference,
		Recurrence:      len(observations),
		Notes:           []string{fmt.Sprintf("sample_size=%d; verified=%d; observed=%d; unlinked=%d; verdicts are evidence collection, not threshold conclusions", len(observations), verified, observed, unlinked)},
	}}, nil
}
