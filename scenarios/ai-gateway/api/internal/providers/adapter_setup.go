package providers

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func NewDefaultAdapters(runner CommandRunner) []Adapter {
	if runner == nil {
		runner = ExecRunner{}
	}
	specs := DefaultProviderSpecs()
	adapters := make([]Adapter, 0, len(specs))
	for _, spec := range specs {
		adapters = append(adapters, Adapter{Provider: spec.ID, CommandName: spec.CommandName, Locality: spec.Locality, Family: spec.Family, Runner: runner})
	}
	return adapters
}

func (a Adapter) family() string {
	if strings.TrimSpace(a.Family) != "" {
		return a.Family
	}
	if spec, ok := SpecFor(a.Provider); ok {
		return spec.Family
	}
	return ""
}

func (a Adapter) command() Command {
	return Command{Name: a.CommandName, Args: []string{"policy", "roles", "--json"}, Timeout: DefaultCommandTimeout}
}

func (a Adapter) runner() CommandRunner {
	if a.Runner == nil {
		return ExecRunner{}
	}
	return a.Runner
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
