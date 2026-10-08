package metrics

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"connectrpc.com/connect"
	devicegraphpb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/devicegraph"

	"github.com/vrooli/cli-core/cliapp"
)

// devices renders the graded hardware device graph.
//
// It is the command-line half of the same read the Substrate Board consumes, so
// every figure on that board can be checked from a terminal. The rendering
// keeps three distinctions the graph is built to preserve, because collapsing
// any of them here would undo the collector's work:
//
//   - An UNAVAILABLE graph is a failure to OBSERVE, never a host with no
//     hardware. It prints the reason and no device count.
//   - A rung grade other than MEASURED always prints its reason, and its
//     remediation where the collector names one.
//   - Virtual network interfaces are named rather than silently dropped, so
//     their exclusion from hardware grading is visible.
func (h *handlers) devices(ctx cliapp.RunContext) error {
	resp, err := h.deviceGraph.GetDeviceGraph(context.Background(), connect.NewRequest(&devicegraphpb.GetDeviceGraphRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get device graph", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetGraph() == nil {
		return fmt.Errorf("server returned no device graph")
	}
	graph := resp.Msg.GetGraph()

	// A walk that produced nothing is reported as a failure to observe. Printing
	// "0 devices" here would be indistinguishable from a machine with nothing
	// attached, which is the one thing this surface must never imply.
	if !graph.GetAvailable() {
		reason := strings.TrimSpace(graph.GetUnavailableReason())
		if reason == "" {
			reason = "the device walk reported no devices and no graded subsystem, and gave no reason"
		}
		return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
			Summary: []string{
				fmt.Sprintf("Platform: %s", graph.GetPlatform()),
				"Device graph: UNAVAILABLE",
				fmt.Sprintf("Reason: %s", reason),
			},
			ResultsHeading: "Devices",
			Results:        []string{"No device list is shown, because the walk did not observe one."},
		})
	}

	devices := graph.GetDevices()
	sort.SliceStable(devices, func(i, j int) bool {
		if devices[i].GetClass() != devices[j].GetClass() {
			return devices[i].GetClass() < devices[j].GetClass()
		}
		return devices[i].GetId() < devices[j].GetId()
	})

	results := make([]string, 0, len(devices)+len(graph.GetSubsystems()))
	for _, device := range devices {
		label := strings.TrimSpace(device.GetModel())
		if label == "" {
			label = strings.TrimSpace(device.GetVendor())
		}
		header := fmt.Sprintf("%s  %s", device.GetId(), device.GetClass())
		if label != "" {
			header += "  " + label
		}
		if driver := strings.TrimSpace(device.GetDriver()); driver != "" {
			header += fmt.Sprintf("  [driver %s]", driver)
		}
		results = append(results, header)
		results = append(results, renderRungs(device.GetRungs())...)
	}
	for _, subsystem := range graph.GetSubsystems() {
		results = append(results, fmt.Sprintf("%s  (subsystem)", subsystem.GetName()))
		results = append(results, renderRungs(subsystem.GetRungs())...)
	}

	summary := []string{
		fmt.Sprintf("Platform: %s", graph.GetPlatform()),
		fmt.Sprintf("Devices: %d", len(devices)),
		fmt.Sprintf("Graded subsystems: %d", len(graph.GetSubsystems())),
	}
	// Named, not dropped: an interface excluded from hardware grading is a
	// decision the reader should be able to see and disagree with.
	if virtual := graph.GetVirtualNetworkInterfaces(); len(virtual) > 0 {
		summary = append(summary, fmt.Sprintf("Virtual interfaces excluded from grading: %d (%s)",
			len(virtual), strings.Join(virtual, ", ")))
	}
	for _, warning := range graph.GetWarnings() {
		summary = append(summary, "Warning: "+warning)
	}

	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Devices And Graded Subsystems",
		Results:        results,
		RetrievalHints: []string{
			"infrastructure-manager ladder status",
			"infrastructure-manager coverage show substrate",
		},
	})
}

// renderRungs prints one line per ladder rung. Every grade other than MEASURED
// carries its reason, and its remediation when the collector names one — an
// explained gap is actionable and an unexplained one is just a blank.
func renderRungs(rungs []*devicegraphpb.RungState) []string {
	out := make([]string, 0, len(rungs))
	for _, rung := range rungs {
		line := fmt.Sprintf("    %-13s %s", shortRung(rung.GetRung()), shortGrade(rung.GetGrade()))
		if reason := strings.TrimSpace(rung.GetReason()); reason != "" {
			line += " — " + reason
		}
		if remediation := strings.TrimSpace(rung.GetRemediation()); remediation != "" {
			line += "  [to close: " + remediation + "]"
		}
		out = append(out, line)
	}
	return out
}

func shortRung(rung devicegraphpb.Rung) string {
	return strings.ToLower(strings.TrimPrefix(rung.String(), "RUNG_"))
}

func shortGrade(grade devicegraphpb.RungGrade) string {
	return strings.ToLower(strings.TrimPrefix(grade.String(), "RUNG_GRADE_"))
}
