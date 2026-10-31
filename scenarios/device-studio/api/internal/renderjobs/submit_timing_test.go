package renderjobs

import (
	"context"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSubmitTimingSurvivesDetachedContextAndEmitsOnlyFixedSpans(t *testing.T) {
	ctx, timing := NewSubmitTiming(context.Background())
	detached, cancel := boundedDetached(ctx, time.Second)
	defer cancel()
	RecordSubmitTiming(detached, "route_pin", 1260*time.Microsecond)
	RecordSubmitTiming(detached, "blender_profile", 1900*time.Millisecond)
	RecordSubmitTiming(detached, "caller-controlled-path", time.Hour)

	header := httptest.NewRecorder().Header()
	SetSubmitTimingHeader(header, timing)
	value := header.Get("Server-Timing")
	if value == "" || len(value) > 1024 {
		t.Fatalf("Server-Timing header length=%d, want a bounded non-empty value", len(value))
	}
	for _, metric := range strings.Split(value, ", ") {
		fields := strings.Split(metric, ";dur=")
		if len(fields) != 2 || fields[0] == "" {
			t.Fatalf("invalid Server-Timing metric %q", metric)
		}
		duration, err := strconv.ParseFloat(fields[1], 64)
		if err != nil || duration < 0 {
			t.Fatalf("invalid Server-Timing duration %q", metric)
		}
	}
	for _, want := range []string{"route_pin;dur=1.3", "blender_profile;dur=1900.0", "handler_total;dur=", "caller-controlled-path"} {
		if strings.Contains(want, "caller-controlled-path") {
			if strings.Contains(value, want) {
				t.Fatalf("unbounded span name leaked into header: %q", value)
			}
			continue
		}
		if !strings.Contains(value, want) {
			t.Fatalf("Server-Timing %q missing %q", value, want)
		}
	}
}

func TestSubmitTimingHeaderDoesNothingWithoutSuccessfulRecorder(t *testing.T) {
	header := httptest.NewRecorder().Header()
	SetSubmitTimingHeader(header, nil)
	if got := header.Get("Server-Timing"); got != "" {
		t.Fatalf("unexpected Server-Timing without recorder: %q", got)
	}
}
