package metrics

import (
	"testing"

	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/metrics"
)

func TestMetricRows(t *testing.T) {
	rows := metricRows(&metricsv1.GetResponse{})
	if len(rows) != 8 {
		t.Fatalf("metric rows = %#v", rows)
	}
}
