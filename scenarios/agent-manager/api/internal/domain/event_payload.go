package domain

import (
	"encoding/json"
	"fmt"
)

// DecodeEventPayload decodes persisted event JSON using its durable event type.
// It also normalizes field names emitted by the retired event-data union,
// so existing SQLite rows migrate forward on their first read without a lossy
// table rebuild.
func DecodeEventPayload(eventType RunEventType, raw []byte) (EventPayload, error) {
	if eventType.IsTypedOperationalEvent() {
		return &TypedEventData{Type: eventType, Body: append(json.RawMessage(nil), raw...)}, nil
	}
	data, err := NormalizeEventPayloadJSON(eventType, raw)
	if err != nil {
		return nil, err
	}
	switch eventType {
	case EventTypeLog:
		return decodePayload[LogEventData](data)
	case EventTypeMessage:
		return decodePayload[MessageEventData](data)
	case EventTypeMessageDeleted:
		return decodePayload[MessageDeletedEventData](data)
	case EventTypeToolCall:
		return decodePayload[ToolCallEventData](data)
	case EventTypeToolResult:
		return decodePayload[ToolResultEventData](data)
	case EventTypeStatus:
		return decodePayload[StatusEventData](data)
	case EventTypeMetric:
		var probe struct {
			PayloadKind string `json:"payloadKind"`
		}
		if err := json.Unmarshal(data, &probe); err != nil {
			return nil, err
		}
		switch probe.PayloadKind {
		case PayloadKindUsage:
			return decodePayload[UsageEventData](data)
		case PayloadKindCharge:
			return decodePayload[ChargeEventData](data)
		default:
			// Historical metric rows without a discriminator are normalized
			// to UsageEventData when they contain token fields. Rows that
			// only have the generic metric shape remain MetricEventData.
			var shape struct {
				InputTokens         int `json:"inputTokens"`
				OutputTokens        int `json:"outputTokens"`
				CacheReadTokens     int `json:"cacheReadTokens"`
				CacheCreationTokens int `json:"cacheCreationTokens"`
			}
			if err := json.Unmarshal(data, &shape); err != nil {
				return nil, err
			}
			if shape.InputTokens != 0 || shape.OutputTokens != 0 || shape.CacheReadTokens != 0 || shape.CacheCreationTokens != 0 {
				return decodePayload[UsageEventData](data)
			}
			return decodePayload[MetricEventData](data)
		}
	case EventTypeArtifact:
		return decodePayload[ArtifactEventData](data)
	case EventTypeError:
		return decodePayload[ErrorEventData](data)
	case EventTypeLifecycle:
		return decodePayload[LifecycleEventData](data)
	default:
		return nil, fmt.Errorf("unsupported run event type %q", eventType)
	}
}

func decodePayload[T any](raw []byte) (EventPayload, error) {
	value := new(T)
	if err := json.Unmarshal(raw, value); err != nil {
		return nil, err
	}
	payload, ok := any(value).(EventPayload)
	if !ok {
		return nil, fmt.Errorf("decoded payload %T does not implement EventPayload", value)
	}
	return payload, nil
}

// NormalizeEventPayloadJSON translates the retired event-union field names to
// the durable typed-payload schema. It returns the original bytes unchanged
// when no forward migration is needed, so callers can safely use it during
// SQLite startup migrations without rewriting already-current rows.
func NormalizeEventPayloadJSON(eventType RunEventType, raw []byte) ([]byte, error) {
	if eventType.IsTypedOperationalEvent() {
		return append([]byte(nil), raw...), nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	changed := false
	rename := func(from, to string) {
		if _, exists := fields[to]; !exists {
			if value, legacy := fields[from]; legacy {
				fields[to] = value
				delete(fields, from)
				changed = true
			}
		}
	}
	switch eventType {
	case EventTypeToolCall:
		rename("toolInput", "input")
	case EventTypeToolResult:
		_, legacyFailure := fields["toolError"]
		rename("toolOutput", "output")
		rename("toolError", "error")
		if _, exists := fields["success"]; !exists {
			fields["success"] = json.RawMessage(fmt.Sprintf("%t", !legacyFailure))
			changed = true
		}
	case EventTypeMetric:
		rename("metricName", "name")
		rename("metricValue", "value")
		if _, hasKind := fields["payloadKind"]; !hasKind {
			var input, output, cacheRead, cacheCreate int
			_ = json.Unmarshal(fields["inputTokens"], &input)
			_ = json.Unmarshal(fields["outputTokens"], &output)
			_ = json.Unmarshal(fields["cacheReadTokens"], &cacheRead)
			_ = json.Unmarshal(fields["cacheCreationTokens"], &cacheCreate)
			if input != 0 || output != 0 || cacheRead != 0 || cacheCreate != 0 {
				fields["payloadKind"] = json.RawMessage(`"usage"`)
				changed = true
				if source, ok := fields["costSource"]; ok {
					var sourceValue string
					_ = json.Unmarshal(source, &sourceValue)
					basis := LegacyChargeBasis(sourceValue)
					charge := map[string]any{
						"payloadKind":    PayloadKindCharge,
						"basis":          basis,
						"amountMicroUsd": nil,
						"currency":       "USD",
					}
					var total float64
					_ = json.Unmarshal(fields["totalCostUsd"], &total)
					if total != 0 {
						amount := int64(total*1_000_000 + 0.5)
						charge["amountMicroUsd"] = amount
					}
					encoded, _ := json.Marshal(charge)
					fields["charge"] = json.RawMessage(encoded)
				}
			}
		}
	case EventTypeArtifact:
		rename("artifactType", "type")
		rename("artifactPath", "path")
	case EventTypeError:
		rename("errorCode", "code")
		rename("errorMessage", "message")
	}
	if !changed {
		return append([]byte(nil), raw...), nil
	}
	return json.Marshal(fields)
}
