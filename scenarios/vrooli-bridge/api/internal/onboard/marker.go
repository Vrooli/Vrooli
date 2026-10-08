package onboard

import "strings"

// markerPrefix is the leading token every bootstrap progress line carries on
// stdout. Pairing authorization is never extracted from marker text; the
// structured node-id marker is only a recovery hint for an already-paired node.
const markerPrefix = "VBOOTSTRAP "

const (
	eventRunStart  = "run-start"
	eventRunOK     = "run-ok"
	eventRunFail   = "run-fail"
	eventStepStart = "step-start"
	eventStepOK    = "step-ok"
	eventStepSkip  = "step-skip"
	eventStepFail  = "step-fail"
	eventNodeID    = "node-id"
)

type Marker struct{ Event, Step, NodeID, Detail string }

func parseMarker(line string) (Marker, bool) {
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, markerPrefix) {
		return Marker{}, false
	}
	rest := line[len(markerPrefix):]
	var marker Marker
	for rest != "" {
		if strings.HasPrefix(rest, "detail=") {
			marker.Detail = unquoteDetail(strings.TrimPrefix(rest, "detail="))
			break
		}
		field := rest
		if i := strings.IndexByte(rest, ' '); i >= 0 {
			field, rest = rest[:i], rest[i+1:]
		} else {
			rest = ""
		}
		key, value, ok := strings.Cut(field, "=")
		if !ok {
			continue
		}
		switch key {
		case "event":
			marker.Event = value
		case "step":
			marker.Step = value
		case "node-id":
			marker.NodeID = value
		}
	}
	return marker, marker.Event != ""
}

func unquoteDetail(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
		return value[1 : len(value)-1]
	}
	return value
}

func stepStatusForEvent(event string) StepStatus {
	switch event {
	case eventStepStart:
		return StepStatusStarted
	case eventStepOK:
		return StepStatusOK
	case eventStepSkip:
		return StepStatusSkipped
	case eventStepFail:
		return StepStatusFailed
	default:
		return StepStatusUnspecified
	}
}
