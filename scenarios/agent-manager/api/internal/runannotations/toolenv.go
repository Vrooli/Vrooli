package runannotations

import (
	"github.com/google/cel-go/cel"

	"agent-manager/internal/toolfacts"
)

// toolRules are the embedded rules the toolfacts package computes tool facts
// under. The feature source reads runs with them and the rule environment is
// declared from them, so what a rule can name and what is computed never drift:
// a variable added to rules.json is available to packs with no change here.
var toolRules, toolRulesErr = toolfacts.DefaultRules()

// toolActivation is a run's tool facts as rule variables. Without loadable
// rules it is nil and every tool variable is absent, which featureEnv reports.
func toolActivation(facts toolfacts.Facts) map[string]any {
	if toolRulesErr != nil {
		return nil
	}
	return facts.Activation(toolRules)
}

// toolVariables declares the tool facts in the rule environment.
func toolVariables() []cel.EnvOption {
	var options []cel.EnvOption
	for _, variable := range toolfacts.Declarations(toolRules) {
		kind := cel.IntType
		if variable.Type == "string" {
			kind = cel.StringType
		}
		options = append(options, cel.Variable(variable.Name, kind))
	}
	return options
}
