package collectors

import (
	"context"
	"testing"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/procsampler"
)

type pagingSampler struct{ samples []procsampler.ProcessSample }

func (s pagingSampler) Sample(context.Context) ([]procsampler.ProcessSample, error) {
	return append([]procsampler.ProcessSample(nil), s.samples...), nil
}

func TestGetTopProcessesByPagingRanksFaultsBeforeRSS(t *testing.T) {
	previous := topProcessSampler
	defer func() { topProcessSampler = previous }()
	topProcessSampler = pagingSampler{samples: []procsampler.ProcessSample{
		{PID: 1, Comm: "quiet-large", RSSKB: 900000, MetricsStatus: "measured"},
		{PID: 2, Comm: "faulting-small", RSSKB: 100, MajorFaultsPerSecond: 12, MetricsStatus: "measured"},
	}}
	got, err := GetTopProcessesByPaging(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0]["name"] != "faulting-small" {
		t.Fatalf("paging ranking = %#v", got)
	}
}
