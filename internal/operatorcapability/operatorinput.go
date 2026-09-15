package operatorcapability

// OperatorInputs translates a descriptor into the durable requests setup may
// queue. The request ID includes the capability ID so two providers can use
// the same field name without colliding. Answers are still resolved by the
// provider through the descriptor's typed validation path.
func (d Descriptor) OperatorInputs() ([]Request, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	requests := make([]Request, 0, len(d.Inputs))
	for _, input := range d.Inputs {
		kind := Kind(input.Kind)
		if input.Kind == KindEnum {
			kind = KindEnum
		}
		if input.Kind == KindConfirmation {
			kind = KindConfirmation
		}
		candidates := make([]Candidate, 0, len(input.Candidates))
		for _, candidate := range input.Candidates {
			candidates = append(candidates, Candidate{
				ID: candidate.ID, Kind: candidate.Kind, Label: candidate.Label, Location: candidate.Location,
				StableIdentity: candidate.StableIdentity, DeviceIdentity: candidate.DeviceIdentity,
				Writable: candidate.Writable, Status: candidate.Status, Risk: candidate.Risk, Remediation: candidate.Remediation,
				Metadata: cloneMetadata(candidate.Metadata),
			})
		}
		requests = append(requests, Request{
			ID: d.ID + ":" + input.ID, Kind: kind, ContractVersion: d.Version, Owner: d.Owner,
			CapabilityID: d.ID, ActionID: "apply", InputID: input.ID, Title: input.Label, Description: input.Description,
			Default: input.Default, Options: append([]string(nil), input.Options...), Candidates: candidates,
			Remediation: d.Remediation, Validation: input.Validation, Required: input.Required,
			Declinable: input.Declinable, Decision: DecisionPending,
			CredentialLogicalID: input.CredentialLogicalID, CredentialField: input.CredentialField, Provider: input.Provider,
			RequirementGroup: input.RequirementGroup, ConsumerRefs: append([]string(nil), input.ConsumerRefs...), CompanionSettings: append([]string(nil), input.CompanionSettings...), CompanionCredentials: append([]string(nil), input.CompanionCredentials...),
			AcquisitionRef: input.AcquisitionRef, VerificationRef: input.VerificationRef, RecoveryRef: input.RecoveryRef, HelpRef: input.HelpRef, EvidencePolicy: input.EvidencePolicy,
		})
	}
	return requests, nil
}

func cloneMetadata(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
