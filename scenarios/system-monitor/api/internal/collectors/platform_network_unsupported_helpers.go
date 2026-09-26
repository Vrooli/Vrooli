//go:build !linux && !darwin && !windows

package collectors

func networkUnsupported(reason string) platformNetworkReading {
	return platformNetworkReading{values: map[string]interface{}{}, status: "unsupported", reason: reason, provenance: "platform backend"}
}
