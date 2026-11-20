package designassist

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// DecodeProposal re-checks the JSON the gateway claimed to validate, then
// decodes it strictly. Two independent checks are cheap; a lenient decode is
// how an unknown field silently becomes an operation nobody validated.
//
// Both transports (the legacy design assist route and the workspace assist
// RPC) use this, so a model answer cannot be accepted by one and refused by
// the other.
func DecodeProposal(schemaJSON, valueJSON string) (Proposal, error) {
	if strings.TrimSpace(valueJSON) == "" {
		return Proposal{}, errors.New("the model returned no value")
	}
	var value any
	if err := json.Unmarshal([]byte(valueJSON), &value); err != nil {
		return Proposal{}, fmt.Errorf("value is not JSON: %w", err)
	}
	schema, err := jsonschema.CompileString("design-assist.json", schemaJSON)
	if err != nil {
		return Proposal{}, fmt.Errorf("patch schema is invalid: %w", err)
	}
	if err := schema.Validate(value); err != nil {
		return Proposal{}, err
	}
	var proposal Proposal
	decoder := json.NewDecoder(strings.NewReader(valueJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&proposal); err != nil {
		return Proposal{}, err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return Proposal{}, errors.New("value has trailing JSON")
	}
	return proposal, nil
}
