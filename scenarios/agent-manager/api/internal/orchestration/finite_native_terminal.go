// finite_native_terminal.go couples the finite-only native factory to exact unit settlement at owner-only startup.
package orchestration

import (
	"agent-manager/internal/adapters/runner"
	"fmt"
)

// InstallFiniteNativeIsolation is explicit startup composition, never an HTTP
// operation. Install before exposing the target. Disabled factory preserves
// ordinary human/operator paths; enabled target accepts only finite native work.
// All registered native runners must support the exact factory before install.
func (o *Orchestrator) InstallFiniteNativeIsolation(f *runner.FiniteNativeFactory) error {
	if f == nil || o.runners == nil {
		return fmt.Errorf("finite native factory/registry missing")
	}
	setters := []interface {
		SetFiniteNativeFactory(*runner.FiniteNativeFactory) error
	}{}
	for _, r := range o.runners.List() {
		s, ok := r.(interface {
			SetFiniteNativeFactory(*runner.FiniteNativeFactory) error
		})
		if !ok {
			return fmt.Errorf("registered runner lacks finite native migration")
		}
		setters = append(setters, s)
	}
	if len(setters) == 0 {
		return fmt.Errorf("finite native registry empty")
	}
	o.finiteNativeFactory = f
	o.finiteNativeTerminal = f
	for _, s := range setters {
		if e := s.SetFiniteNativeFactory(f); e != nil {
			return fmt.Errorf("finite-only target remains held after failed install: %w", e)
		}
	}
	o.finiteNativeFactory = f
	o.finiteNativeTerminal = f
	return nil
}
