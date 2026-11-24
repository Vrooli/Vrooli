package contracts

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"unicode"
)

type ReplayDisposition string

const (
	ReplayUnsafe       ReplayDisposition = "unsafe"
	ReplayReadOnly     ReplayDisposition = "read_only"
	ReplayIdempotent   ReplayDisposition = "idempotent"
	ReplayWithReceipts ReplayDisposition = "owner_receipt"
)

// EffectReceipt is an owner-read observation, not authority inferred from a
// missing runtime row. Only an explicitly verified applied/not_applied result
// can support continuation.
type EffectReceipt struct {
	BindingID      string
	ReceiptBinding string
	Owner          string
	EffectKey      string
	ReceiptID      string
	Outcome        string
	// Set only by the trusted owner-receipt reader after it validates the
	// response against the declared receipt schema. Never accept this from a
	// program input or caller-supplied field.
	Verified bool
}

type ReplayAssessment struct {
	Disposition    ReplayDisposition
	Reason         string
	EffectIDs      []string
	EffectOutcomes map[string]string
}

type replaySafetyDeclaration struct {
	Version int                  `json:"version"`
	Effects []replayEffectPolicy `json:"effects"`
}

type replayEffectPolicy struct {
	BindingID            string `json:"binding_id"`
	Owner                string `json:"owner"`
	Mode                 string `json:"mode"`
	KeyInput             string `json:"key_input"`
	IdempotencyGuarantee string `json:"idempotency_guarantee"`
	ReceiptBinding       string `json:"receipt_binding"`
	ReceiptIDField       string `json:"receipt_id_field"`
	OutcomeField         string `json:"outcome_field"`
}

// ContractMayHaveEffects is conservative: indirect bindings, unknown verbs,
// and undeclared binding effects count as potentially effectful.
func ContractMayHaveEffects(contract Contract) bool {
	if contract.ValidationError != "" || contract.SourceMissing || len(contract.Bindings) == 0 {
		return true
	}
	for _, binding := range contract.Bindings {
		if binding.Effect != "read" || (binding.Via != "" && binding.Via != "binding") {
			return true
		}
	}
	for _, capability := range contract.Learning.Capabilities {
		for _, effect := range capability.Effects {
			if effect != "read" {
				return true
			}
		}
	}
	for _, verb := range contract.Verbs {
		if !replayReadOnlyVerb(verb) {
			return true
		}
	}
	return false
}

// AssessReplaySafety classifies a pinned declaration without replaying it.
// Missing, unknown, partial, duplicated, or unverifiable effect evidence always
// returns unsafe; callers must retain runtime_interrupted and reconcile with
// the effect owner. This assessment is deliberately separate from execution.
func AssessReplaySafety(contract Contract, inputs map[string]any, receipts []EffectReceipt) ReplayAssessment {
	unsafe := func(reason string) ReplayAssessment {
		return ReplayAssessment{Disposition: ReplayUnsafe, Reason: reason}
	}
	if contract.ValidationError != "" || contract.SourceMissing || len(contract.Declaration) == 0 {
		return unsafe("contract_missing_or_invalid")
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(contract.Declaration, &document); err != nil || document == nil {
		return unsafe("contract_declaration_invalid")
	}
	var policy *replaySafetyDeclaration
	if raw, ok := document["replay_safety"]; ok {
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		policy = new(replaySafetyDeclaration)
		if err := decoder.Decode(policy); err != nil {
			return unsafe("replay_safety_policy_invalid")
		}
	}
	effectful := make(map[string]BindingRef)
	for _, binding := range contract.Bindings {
		if binding.ID == "" || binding.Effect == "" {
			return unsafe("binding_effect_unknown")
		}
		if binding.Via != "" && binding.Via != "binding" {
			return unsafe("indirect_binding_effect_unmodeled")
		}
		switch binding.Effect {
		case "read":
		case "write", "destructive":
			if _, duplicate := effectful[binding.ID]; duplicate {
				return unsafe("effect_binding_duplicate")
			}
			effectful[binding.ID] = binding
		default:
			return unsafe("binding_effect_unknown")
		}
	}
	for _, capability := range contract.Learning.Capabilities {
		for _, effect := range capability.Effects {
			if effect != "read" {
				return unsafe("learning_effect_unmodeled")
			}
		}
	}
	for _, verb := range contract.Verbs {
		if !replayReadOnlyVerb(verb) {
			return unsafe("runtime_verb_effect_unmodeled")
		}
	}
	if len(contract.Bindings) == 0 {
		return unsafe("effect_boundary_missing")
	}
	if len(effectful) == 0 {
		if policy != nil && (policy.Version != 1 || len(policy.Effects) != 0) {
			return unsafe("policy_names_unbound_effect")
		}
		return ReplayAssessment{Disposition: ReplayReadOnly, Reason: "all_declared_bindings_are_read"}
	}
	if policy == nil || policy.Version != 1 {
		return unsafe("replay_safety_policy_missing_or_unknown")
	}
	if len(policy.Effects) != len(effectful) {
		return unsafe("effect_policy_coverage_incomplete")
	}

	policies := make(map[string]replayEffectPolicy, len(effectful))
	ids := make([]string, 0, len(effectful))
	for _, effectPolicy := range policy.Effects {
		binding, ok := effectful[effectPolicy.BindingID]
		if !ok || effectPolicy.Owner != ownerOfBinding(binding.ID) {
			return unsafe("effect_policy_binding_or_owner_mismatch")
		}
		if _, duplicate := policies[effectPolicy.BindingID]; duplicate {
			return unsafe("effect_policy_duplicate")
		}
		input, ok := contract.Inputs[effectPolicy.KeyInput]
		if effectPolicy.KeyInput == "" || !ok || !input.Required || input.Type != "string" {
			return unsafe("effect_key_input_not_required_string")
		}
		key, ok := inputs[effectPolicy.KeyInput].(string)
		if !ok || strings.TrimSpace(key) == "" || len(key) > 128 || strings.IndexFunc(key, unicode.IsControl) >= 0 {
			return unsafe("effect_key_input_missing_or_invalid")
		}
		switch effectPolicy.Mode {
		case "idempotent":
			if effectPolicy.IdempotencyGuarantee != "owner_deduplicates_same_key" || effectPolicy.ReceiptBinding != "" || effectPolicy.ReceiptIDField != "" || effectPolicy.OutcomeField != "" {
				return unsafe("idempotency_owner_guarantee_incomplete")
			}
		case "owner_receipt":
			receiptBinding, found := contractBinding(contract.Bindings, effectPolicy.ReceiptBinding)
			if effectPolicy.ReceiptBinding == "" || !found || receiptBinding.Effect != "read" || ownerOfBinding(receiptBinding.ID) != effectPolicy.Owner || !validReceiptField(effectPolicy.ReceiptIDField) || !validReceiptField(effectPolicy.OutcomeField) {
				return unsafe("owner_receipt_route_incomplete")
			}
		case "unsafe":
			return unsafe("owner_declared_effect_unsafe")
		default:
			return unsafe("effect_replay_mode_unknown")
		}
		policies[effectPolicy.BindingID] = effectPolicy
		ids = append(ids, effectPolicy.BindingID)
	}
	for id := range effectful {
		if _, ok := policies[id]; !ok {
			return unsafe("effect_policy_coverage_incomplete")
		}
	}

	needsReceipts := false
	for _, policy := range policies {
		needsReceipts = needsReceipts || policy.Mode == "owner_receipt"
	}
	if needsReceipts {
		if len(receipts) == 0 {
			return unsafe("owner_effect_receipt_missing")
		}
		seen := make(map[string]bool, len(receipts))
		outcomes := make(map[string]string, len(receipts))
		for _, receipt := range receipts {
			policy, ok := policies[receipt.BindingID]
			if !ok || policy.Mode != "owner_receipt" || seen[receipt.BindingID] {
				return unsafe("owner_effect_receipt_incomplete_or_mismatched")
			}
			key, _ := inputs[policy.KeyInput].(string)
			if !receipt.Verified || receipt.Owner != policy.Owner || receipt.ReceiptBinding != policy.ReceiptBinding || strings.TrimSpace(receipt.ReceiptID) == "" || receipt.EffectKey != key {
				return unsafe("owner_effect_receipt_incomplete_or_mismatched")
			}
			if receipt.Outcome != "applied" && receipt.Outcome != "not_applied" {
				return unsafe("owner_effect_receipt_outcome_unknown")
			}
			seen[receipt.BindingID] = true
			outcomes[receipt.BindingID] = receipt.Outcome
		}
		for id, policy := range policies {
			if policy.Mode == "owner_receipt" && !seen[id] {
				return unsafe("owner_effect_receipt_missing")
			}
		}
		sort.Strings(ids)
		return ReplayAssessment{Disposition: ReplayWithReceipts, Reason: "all_effects_have_complete_owner_replay_proof", EffectIDs: ids, EffectOutcomes: outcomes}
	} else if len(receipts) != 0 {
		return unsafe("unexpected_owner_effect_receipt")
	}
	sort.Strings(ids)
	return ReplayAssessment{Disposition: ReplayIdempotent, Reason: "all_effects_have_complete_owner_replay_proof", EffectIDs: ids}
}

func replayReadOnlyVerb(verb string) bool {
	switch verb {
	case "discover", "recall", "validate", "describe", "agent.collect":
		return true
	default:
		return false
	}
}

func ownerOfBinding(id string) string {
	owner, _, _ := strings.Cut(id, "/")
	return owner
}

func validReceiptField(path string) bool {
	if path == "" || len(path) > 128 {
		return false
	}
	for _, segment := range strings.Split(path, ".") {
		if segment == "" {
			return false
		}
		for _, char := range segment {
			if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '_' || char == '-') {
				return false
			}
		}
	}
	return true
}

func contractBinding(bindings []BindingRef, id string) (BindingRef, bool) {
	for _, binding := range bindings {
		if binding.ID == id {
			return binding, true
		}
	}
	return BindingRef{}, false
}
