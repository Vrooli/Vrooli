package runtime

import "encoding/json"

// CloneInstruction performs a deep clone of an instruction.
func CloneInstruction(instr Instruction) (Instruction, error) {
	var clone Instruction
	raw, err := json.Marshal(instr)
	if err != nil {
		return Instruction{}, err
	}
	if err := json.Unmarshal(raw, &clone); err != nil {
		return Instruction{}, err
	}
	return clone, nil
}

// InterpolateInstruction applies context-based interpolation to all string params.
func InterpolateInstruction(instr Instruction, ctx *ExecutionContext) (Instruction, []string, error) {
	clone, err := CloneInstruction(instr)
	if err != nil {
		return Instruction{}, nil, err
	}
	// Convert params to map so interpolation can run recursively
	rawParams, err := json.Marshal(clone.Params)
	if err != nil {
		return Instruction{}, nil, err
	}
	paramsMap := map[string]any{}
	if err := json.Unmarshal(rawParams, &paramsMap); err != nil {
		return Instruction{}, nil, err
	}
	paramsMap, missing := interpolateMap(paramsMap, ctx)
	// Write interpolated params back into struct form
	updatedParams := InstructionParam{}
	updatedRaw, err := json.Marshal(paramsMap)
	if err != nil {
		return Instruction{}, nil, err
	}
	if err := json.Unmarshal(updatedRaw, &updatedParams); err != nil {
		return Instruction{}, nil, err
	}
	clone.Params = updatedParams
	if ctx != nil {
		clone.Context = ctx.Snapshot()
	}
	return clone, missing, nil
}
