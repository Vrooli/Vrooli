package services

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

func TestForkRateFromValuesMeasured(t *testing.T) {
	info := forkRateFromValues(map[string]interface{}{
		"fork_rate_status":  "measured",
		"fork_rate_source":  "linux /proc/stat",
		"forks_total":       uint64(58_623_471),
		"forks_per_second":  float64(2481),
		"fork_rate_pending": false,
	})
	if info == nil {
		t.Fatal("measured fork rate did not surface")
	}
	if !info.Supported || info.ForksPerSecond != 2481 || info.ForksTotal != 58_623_471 {
		t.Fatalf("info = %+v", info)
	}
	if info.Pending {
		t.Fatal("measured rate marked pending")
	}
	if info.Source != "linux /proc/stat" {
		t.Fatalf("source = %q", info.Source)
	}
}

// The first cycle must arrive as pending, not as a measured zero: "no rate yet"
// and "zero forks per second" are opposite diagnoses during an incident.
func TestForkRateFromValuesPendingIsNotZeroRate(t *testing.T) {
	info := forkRateFromValues(map[string]interface{}{
		"fork_rate_status":  "measured",
		"forks_total":       uint64(10),
		"forks_per_second":  float64(0),
		"fork_rate_pending": true,
	})
	if info == nil || !info.Pending {
		t.Fatalf("first cycle not marked pending: %+v", info)
	}
}

func TestForkRateFromValuesUnsupportedKeepsReason(t *testing.T) {
	info := forkRateFromValues(map[string]interface{}{
		"fork_rate_status": "unsupported",
		"fork_rate_reason": "no counter on this operating system",
	})
	if info == nil {
		t.Fatal("unsupported platform returned nil; the absence must stay visible")
	}
	if info.Supported {
		t.Fatal("unsupported reading marked supported")
	}
	if info.Reason == "" {
		t.Fatal("unsupported reading lost its reason")
	}
}

func TestForkRateFromValuesAbsent(t *testing.T) {
	if info := forkRateFromValues(map[string]interface{}{}); info != nil {
		t.Fatalf("expected nil when the collector emitted no fork-rate keys, got %+v", info)
	}
}

func TestPopulateNetworkDetailsSurfacesSocketOwners(t *testing.T) {
	detailed := &models.DetailedMetrics{}
	populateNetworkDetails(detailed, &collectors.MetricData{
		Values: map[string]interface{}{
			"socket_owners": collectors.SocketAttribution{
				Supported:  true,
				Total:      57_706,
				Attributed: 55_100,
				Owners: []collectors.SocketOwner{
					{PID: 3241745, Comm: "agent-manager-a", Count: 53_474},
					{PID: 1234, Comm: "git-control-tow", Count: 158},
				},
			},
		},
	})

	owners := detailed.NetworkDetails.SocketOwners
	if owners == nil {
		t.Fatal("socket owners did not reach the model")
	}
	if owners.Total != 57_706 || owners.Attributed != 55_100 {
		t.Fatalf("coverage lost: attributed=%d total=%d", owners.Attributed, owners.Total)
	}
	if len(owners.Owners) != 2 || owners.Owners[0].Name != "agent-manager-a" || owners.Owners[0].Connections != 53_474 {
		t.Fatalf("owners = %+v", owners.Owners)
	}
}

// Attribution runs only above a threshold, so its absence is the normal case and
// must not fabricate an empty-but-supported report.
func TestPopulateNetworkDetailsOmitsAbsentAttribution(t *testing.T) {
	detailed := &models.DetailedMetrics{}
	populateNetworkDetails(detailed, &collectors.MetricData{Values: map[string]interface{}{}})
	if detailed.NetworkDetails.SocketOwners != nil {
		t.Fatal("absent attribution surfaced as a populated report")
	}
}
