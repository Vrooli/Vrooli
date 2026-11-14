package inference

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// inventoryReservationPolicy is the deterministic contract used by the
// 2026-10-25 closed-set screen-short-policy workload. The prompt is parsed
// into operands and a threshold; the action is evaluated in code, without a
// model call. This role deliberately supports one declared rule shape rather
// than pretending to interpret arbitrary natural language.
var inventoryReservationPolicy = regexp.MustCompile(`(?is)^\s*Return only JSON with keys action and net\.\s*Rule: if inventory after reservations is below ([0-9]+), action is ([a-z][a-z0-9_-]*); otherwise ([a-z][a-z0-9_-]*)\.\s*Current inventory ([0-9]+), reservations ([0-9]+)\. net is inventory minus reservations\. Do not explain\.\s*$`)

var ErrUnsupportedClosedSetTask = errors.New("source does not match the inventory-reservation policy contract")
var ErrCodeRoleCallerSchema = errors.New("code role output violates caller schema")

// evaluateDeclaredCodeRole executes the handler named in a validated catalog
// contract and validates its result against both the catalog's stable output
// contract and the caller's narrower request schema. The code role stays
// bounded and provider-free even when a caller asks for a permissive schema.
func evaluateDeclaredCodeRole(contract CodeRoleContract, requestSchema map[string]any, source string) (string, error) {
	if contract.MaxSourceBytes < 1 || contract.MaxOutputBytes < 1 {
		return "", errors.New("code role contract must declare positive source and output byte limits")
	}
	if len(source) > contract.MaxSourceBytes {
		return "", fmt.Errorf("code role source exceeds the declared %d-byte limit", contract.MaxSourceBytes)
	}
	var value string
	switch contract.HandlerID {
	case "inventory-reservation-policy/v1":
		var err error
		value, err = evaluateInventoryReservationPolicy(source)
		if err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("code role handler %q is not registered", contract.HandlerID)
	}
	if len(value) > contract.MaxOutputBytes {
		return "", fmt.Errorf("code role output exceeds the declared %d-byte limit", contract.MaxOutputBytes)
	}
	contractJSON, err := json.Marshal(contract.SchemaContract)
	if err != nil {
		return "", fmt.Errorf("encode code role schema contract: %w", err)
	}
	contractSchema, err := (SchemaGate{}).Parse(string(contractJSON))
	if err != nil {
		return "", fmt.Errorf("parse code role schema contract: %w", err)
	}
	if err := ValidateJSON(contractSchema, []byte(value)); err != nil {
		return "", fmt.Errorf("code role output violates its declared schema contract: %w", err)
	}
	if requestSchema != nil {
		if err := ValidateJSON(requestSchema, []byte(value)); err != nil {
			return "", fmt.Errorf("%w: %v", ErrCodeRoleCallerSchema, err)
		}
	}
	return value, nil
}

func evaluateInventoryReservationPolicy(source string) (string, error) {
	parts := inventoryReservationPolicy.FindStringSubmatch(source)
	if parts == nil {
		return "", ErrUnsupportedClosedSetTask
	}
	values := make([]int64, 3)
	for i, raw := range []string{parts[1], parts[4], parts[5]} {
		value, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || value < 0 {
			return "", fmt.Errorf("inventory policy operand %q is outside the supported non-negative integer range", raw)
		}
		values[i] = value
	}
	threshold, inventory, reservations := values[0], values[1], values[2]
	net := inventory - reservations
	action := parts[3]
	if net < threshold {
		action = parts[2]
	}
	encoded, err := json.Marshal(struct {
		Action string `json:"action"`
		Net    int64  `json:"net"`
	}{Action: action, Net: net})
	return string(encoded), err
}
