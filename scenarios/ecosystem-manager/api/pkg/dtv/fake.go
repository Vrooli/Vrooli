package dtv

import "context"

// FakeProvider is an in-memory SkillFitnessProvider for tests. A skill absent
// from Fits resolves to a zero-value (UNKNOWN) Fitness — the fail-open default.
// When Err is set, every call returns it (with an UNKNOWN fitness) to exercise
// the controller's degradation path.
type FakeProvider struct {
	Fits map[string]Fitness
	Err  error
}

var _ SkillFitnessProvider = (*FakeProvider)(nil)

// Fitness implements SkillFitnessProvider.
func (f *FakeProvider) Fitness(_ context.Context, skillID string) (Fitness, error) {
	if f.Err != nil {
		return Fitness{SkillID: skillID}, f.Err
	}
	if fit, ok := f.Fits[skillID]; ok {
		fit.SkillID = skillID
		return fit, nil
	}
	return Fitness{SkillID: skillID}, nil
}
