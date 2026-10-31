//go:build !linux && !darwin && !windows

package collectors

func platformMemoryUnsupported(reason string) platformMemoryReading {
	return platformMemoryReading{status: "unsupported", reason: reason, provenance: "platform backend"}
}
