package sources

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/discovery"
)

func TestReadTypedReportsADeadlineRatherThanHanging(t *testing.T) {
	started := time.Now()
	result := ReadTyped(context.Background(), "slow", func(ctx context.Context) (int, error) {
		select {
		case <-time.After(time.Second):
			return 1, nil
		case <-ctx.Done():
			return 0, ctx.Err()
		}
	}, 20*time.Millisecond)
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("the read took %s; the deadline did not bound it", elapsed)
	}
	if result.Available {
		t.Fatal("a source that exceeded its deadline was reported available")
	}
	if !strings.Contains(result.Reason, "deadline exceeded") {
		t.Errorf("the reason reads %q; it does not name the deadline", result.Reason)
	}
}

func TestReadTypedCarriesTheUnderlyingErrorVerbatim(t *testing.T) {
	result := ReadTyped(context.Background(), "broken", func(context.Context) (int, error) {
		return 0, errors.New("unexpected EOF")
	}, time.Second)
	if result.Available {
		t.Fatal("a failing source was reported available")
	}
	if result.Reason != "unexpected EOF" {
		t.Errorf("the reason reads %q; the underlying error was rewritten", result.Reason)
	}
}

func TestReadTypedReportsAnUnconfiguredReader(t *testing.T) {
	result := ReadTyped[int](context.Background(), "unwired", nil, time.Second)
	if result.Available {
		t.Fatal("an unconfigured reader was reported available")
	}
	if result.Reason == "" {
		t.Fatal("an unconfigured reader produced no reason")
	}
}

// TestReadTypedDefaultsToTheDocumentedDeadline pins the trust model's
// readDeadline. A deadline tight enough to make healthy sources intermittently
// vanish manufactures exactly the false blindness the trust axis prevents.
func TestReadTypedDefaultsToTheDocumentedDeadline(t *testing.T) {
	if DefaultTimeout != 10*time.Second {
		t.Fatalf("the default per-source deadline is %s; the trust model documents 10s", DefaultTimeout)
	}
}

type stubTransport struct {
	graph DeviceGraph
	err   error
	base  string
}

func (s *stubTransport) DeviceGraph(_ context.Context, baseURL string) (DeviceGraph, error) {
	s.base = baseURL
	return s.graph, s.err
}

// TestDeviceGraphReaderDistinguishesAnUnwiredVerbFromAnOutage is the
// difference between "the owner is down" — an outage, which is not this
// instrument's work — and "the owner is up and publishes no typed device-graph
// verb", which is an unclosed join and IS this instrument's work. Collapsing
// them routes the second to the wrong owner.
func TestDeviceGraphReaderDistinguishesAnUnwiredVerbFromAnOutage(t *testing.T) {
	t.Run("owner reachable, verb unwired", func(t *testing.T) {
		reader := DeviceGraphReader{
			Resolver: discovery.NewResolver(discovery.ResolverConfig{StaticBaseURL: "http://127.0.0.1:65000"}),
		}
		_, err := reader.ReadGraph(context.Background())
		if err == nil {
			t.Fatal("an unwired transport produced no error")
		}
		if !errors.Is(err, ErrDeviceGraphVerbUnpublished) {
			t.Fatalf("an unwired transport produced %v, want ErrDeviceGraphVerbUnpublished", err)
		}
		for _, fragment := range []string{deviceGraphScenario, deviceGraphVerb, "http://127.0.0.1:65000"} {
			if !strings.Contains(err.Error(), fragment) {
				t.Errorf("the error %q does not name %q", err, fragment)
			}
		}
	})

	t.Run("owner unreachable", func(t *testing.T) {
		reader := DeviceGraphReader{
			Resolver:  discovery.NewResolver(discovery.ResolverConfig{VrooliPath: "/nonexistent/vrooli"}),
			Transport: &stubTransport{},
		}
		_, err := reader.ReadGraph(context.Background())
		if err == nil {
			t.Fatal("an unresolvable owner produced no error")
		}
		if errors.Is(err, ErrDeviceGraphVerbUnpublished) {
			t.Fatalf("an unresolvable owner was reported as an unpublished verb: %v", err)
		}
		if !strings.Contains(err.Error(), deviceGraphScenario) {
			t.Errorf("the error %q does not name the owning scenario", err)
		}
	})
}

// TestDeviceGraphReaderPassesTheResolvedBaseToTheTransport pins that the
// transport reads the scenario discovery resolved, not a hard-coded address.
func TestDeviceGraphReaderPassesTheResolvedBaseToTheTransport(t *testing.T) {
	transport := &stubTransport{graph: DeviceGraph{Platform: "linux"}}
	reader := DeviceGraphReader{
		Resolver:  discovery.NewResolver(discovery.ResolverConfig{StaticBaseURL: "http://127.0.0.1:65000"}),
		Transport: transport,
	}
	graph, err := reader.ReadGraph(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if graph.Platform != "linux" {
		t.Errorf("the reader returned platform %q, want the transport's value", graph.Platform)
	}
	if transport.base != "http://127.0.0.1:65000" {
		t.Errorf("the transport was given base %q, want the resolved address", transport.base)
	}
}

// TestDeviceGraphReaderNeverReturnsAnEmptyGraphOnFailure guards the claim an
// empty graph would make: that the host has no hardware.
func TestDeviceGraphReaderNeverReturnsAnEmptyGraphOnFailure(t *testing.T) {
	reader := DeviceGraphReader{
		Resolver:  discovery.NewResolver(discovery.ResolverConfig{StaticBaseURL: "http://127.0.0.1:65000"}),
		Transport: &stubTransport{err: errors.New("unexpected EOF")},
	}
	graph, err := reader.ReadGraph(context.Background())
	if err == nil {
		t.Fatal("a failing transport produced no error")
	}
	if len(graph.Devices) != 0 || len(graph.Subsystems) != 0 {
		t.Fatal("a failing read returned graph content")
	}
	if !strings.Contains(err.Error(), "unexpected EOF") {
		t.Errorf("the error %q loses the underlying failure", err)
	}
}

func TestCheckPlatformsEmptyListMeansEveryPlatform(t *testing.T) {
	universal := CheckPlatforms{CheckID: "everywhere"}
	for _, hostOS := range []string{"linux", "macos", "windows"} {
		if !universal.AppliesTo(hostOS) {
			t.Errorf("a check declaring no platforms was read as not applying on %s; an empty list means all, not unknown", hostOS)
		}
	}
	linuxOnly := CheckPlatforms{CheckID: "linux-only", Platforms: []string{"linux"}}
	if !linuxOnly.AppliesTo("linux") {
		t.Error("a linux-only check was read as not applying on linux")
	}
	if linuxOnly.AppliesTo("windows") {
		t.Error("a linux-only check was read as applying on windows")
	}
}
