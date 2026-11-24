package provision

import "github.com/vrooli/vrooli/packages/provider-effect-gate"

func fixtureLiveEffects() *effectgate.Gate {
	gate, _ := effectgate.New(effectgate.Config{Default: effectgate.Live})
	return gate
}

func fixtureStubEffects() *effectgate.Gate {
	gate, _ := effectgate.New(effectgate.Config{Default: effectgate.Stub})
	return gate
}
