package devicegraph

import (
	"context"
	"runtime"
)

// Collect observes the host once and returns the graded device graph.
//
// Collect never returns an error. A probe that could not run is a graded state
// inside the graph — StateUnavailable or StateUnmeasurable with a reason — so
// a caller can always tell the difference between "no devices of this kind"
// and "this host would not tell us". Callers that need structural assurance
// call Graph.Validate.
func Collect(ctx context.Context, env Env) Graph {
	env = env.normalized()
	graph := &Graph{CollectedAt: env.Now().UTC(), Platform: runtime.GOOS}
	b := newBuilder(env, graph, grader{at: graph.CollectedAt})
	collectPlatformDevices(ctx, b)
	b.setParents()
	b.dropDanglingParents()
	return *graph
}

// unsupportedPlatform records that no device-graph backend exists for this
// build target. It reports the absence explicitly rather than returning an
// empty graph, which would be indistinguishable from a host with no hardware.
func unsupportedPlatform(b *builder, reason string) {
	for _, name := range []string{"pci-bus", "usb-bus", "block-storage", "network-interfaces", "thermal", SubsystemMemoryErrors} {
		b.graph.addSubsystem(Subsystem{
			Name: name,
			Rungs: rungSet(
				b.grader.unavailable(RungIdentity, reason, "none"),
				b.grader.unavailable(RungTelemetry, reason, "none"),
				b.grader.unavailable(RungEvidence, "nothing to retain: "+reason, evidenceMechanism),
				b.grader.unavailable(RungControl, reason, "none"),
				b.grader.unavailable(RungAnticipation, "no forward-looking signal: "+reason, "none"),
			),
		})
	}
}
