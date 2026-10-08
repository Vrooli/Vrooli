package intent

type FakePRDExtractor struct {
	Claims []CapabilityClaim
	Err    error
}

func (f FakePRDExtractor) ExtractPRDClaims(string) ([]CapabilityClaim, error) {
	return f.Claims, f.Err
}

type FakeRequirementsExtractor struct {
	Claims []CapabilityClaim
	Err    error
}

func (f FakeRequirementsExtractor) ExtractRequirementClaims(string) ([]CapabilityClaim, error) {
	return f.Claims, f.Err
}
