// Package scenarioruntime owns the runtime registry and scenario process-state projection used by control-plane commands. It does not start or repair processes directly; lifecycle owns mutations and this package records or reads their state.
package scenarioruntime
