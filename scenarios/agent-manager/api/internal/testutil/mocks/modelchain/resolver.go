package modelchain

import "agent-manager/internal/modelregistry"

// FakeResolver resolves every preset lookup to the configured chain.
type FakeResolver struct {
	Chain modelregistry.PresetChain
}

func NewFakeResolver(chain modelregistry.PresetChain) *FakeResolver {
	return &FakeResolver{Chain: append(modelregistry.PresetChain(nil), chain...)}
}

func (r *FakeResolver) ResolvePreset(_, _ string) (modelregistry.PresetChain, bool) {
	if len(r.Chain) == 0 {
		return nil, false
	}
	return append(modelregistry.PresetChain(nil), r.Chain...), true
}
