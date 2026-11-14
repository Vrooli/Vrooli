package control

import (
	"encoding/json"
	"sort"
)

// compactDevices projects the full inventory payload to one short row per
// reachable device: what it is, whether it can be used, and which one-call
// verbs its transports support. Unreachable and retained-but-absent rows are
// counted, not listed; `device list --all` or `--full` shows them.
func compactDevices(raw []byte, includeUnreachable bool) (map[string]any, error) {
	var payload struct {
		Devices []struct {
			ID           string                          `json:"id"`
			Name         string                          `json:"name"`
			Kind         string                          `json:"kind"`
			Status       string                          `json:"status"`
			Health       string                          `json:"health"`
			HealthReason string                          `json:"health_reason"`
			Capabilities []struct{ Name, Status string } `json:"capabilities"`
			Transports   []struct {
				Name         string `json:"name"`
				StrategyID   string `json:"strategy_id"`
				Health       string `json:"health"`
				Capabilities map[string]struct {
					Status string `json:"status"`
				} `json:"capabilities"`
			} `json:"transports"`
		} `json:"devices"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(payload.Devices))
	hidden := 0
	for _, device := range payload.Devices {
		usable := device.Status == "available" || device.Health == "available"
		if !usable && !includeUnreachable {
			hidden++
			continue
		}
		caps := map[string]bool{}
		for _, capability := range device.Capabilities {
			if capability.Status == "available" {
				caps[capability.Name] = true
			}
		}
		transports := make([]string, 0, len(device.Transports))
		for _, transport := range device.Transports {
			transports = append(transports, transport.Name+"("+transport.Health+")")
			if transport.Health != "available" {
				continue
			}
			for name, capability := range transport.Capabilities {
				if capability.Status == "available" {
					caps[name] = true
				}
			}
		}
		row := map[string]any{"id": device.ID, "name": device.Name, "kind": device.Kind, "status": device.Status, "can": userVerbsFor(caps)}
		if len(transports) > 0 {
			sort.Strings(transports)
			row["transports"] = transports
		}
		if !usable && device.HealthReason != "" {
			row["reason"] = device.HealthReason
		}
		rows = append(rows, row)
	}
	out := map[string]any{"devices": rows}
	if hidden > 0 {
		out["hidden_unreachable"] = hidden
		out["hint"] = "device list --all shows unreachable rows; device list --full shows every capability probe"
	}
	return out, nil
}

// userVerbsFor maps probed capabilities to the one-call verbs they enable.
func userVerbsFor(caps map[string]bool) []string {
	verbs := []string{}
	add := func(verb string, ok bool) {
		if ok {
			verbs = append(verbs, verb)
		}
	}
	add("screenshot", caps["screenshot"])
	add("record", caps["native-recording"] || caps["screen-recording"])
	add("open", caps["app-lifecycle"])
	add("wake", caps["screen-control"])
	add("input", caps["input"])
	add("media", caps["media"])
	add("semantics", caps["semantic-tree"])
	return verbs
}
