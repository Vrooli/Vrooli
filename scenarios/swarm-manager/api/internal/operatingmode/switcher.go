package operatingmode

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (e *ActiveItemExecutionsConflict) Error() string {
	return fmt.Sprintf("initiative %q has active member-item executions; confirm cancellation before switching to %q", e.InitiativeName, e.ToMode)
}

func (e *ActiveOperatingModeRoundConflict) Error() string {
	return fmt.Sprintf("initiative %q has active %s round %03d; finish or cancel it before switching to %q", e.InitiativeName, e.FromMode, e.Round.Round, e.ToMode)
}

func (s *Service) SwitchMode(ctx context.Context, req SwitchModeRequest) (SwitchModeResult, error) {
	name := strings.TrimSpace(req.InitiativeName)
	if name == "" {
		return SwitchModeResult{}, fmt.Errorf("initiative name is required")
	}
	targetMode := NormalizeMode(req.Mode)
	// Switching to the member-item workflow strategy writes the sentinel wire
	// value as a plain initiative field — there is no mode definition to load
	// or target policy to check. Genuine modes must exist and target
	// initiatives.
	if targetMode != ModeItemLevel {
		def, err := DefinitionFor(targetMode)
		if err != nil {
			return SwitchModeResult{}, err
		}
		// Mode switching is an initiative lifecycle action: an initiative can
		// only run modes whose declared unit of work is an initiative.
		// Plan-target modes (e.g. the generic phased-plan-drain) run against a
		// plan directly, not through an initiative's mode field.
		if def.Target.Kind != TargetInitiative {
			return SwitchModeResult{}, fmt.Errorf("mode %q targets %s, not an initiative; it runs directly against its target and cannot be set as an initiative's mode", targetMode, def.Target.Kind)
		}
	}
	if s.modeUpdater == nil {
		return SwitchModeResult{}, errors.New("operatingmode: InitiativeModeUpdater is not configured")
	}
	init, err := s.initiatives.LoadInitiative(name)
	if err != nil {
		return SwitchModeResult{}, err
	}
	fromMode := NormalizeMode(init.Mode)
	result := SwitchModeResult{
		InitiativeName:           init.Name,
		FromMode:                 string(fromMode),
		ToMode:                   string(targetMode),
		OperatingModeWorkspaceID: string(targetMode),
	}
	if fromMode == targetMode {
		return result, nil
	}
	if fromMode != ModeItemLevel {
		active, err := s.activeOperatingModeRound(init.Name, fromMode)
		if err != nil {
			return SwitchModeResult{}, err
		}
		if active != nil {
			return SwitchModeResult{}, &ActiveOperatingModeRoundConflict{
				InitiativeName: init.Name,
				FromMode:       string(fromMode),
				ToMode:         string(targetMode),
				Round:          *active,
			}
		}
	}
	if fromMode == ModeItemLevel && targetMode != ModeItemLevel && s.itemExecs != nil {
		active, err := s.itemExecs.ActiveExecutionsForInitiative(ctx, init)
		if err != nil {
			return SwitchModeResult{}, err
		}
		if len(active) > 0 && !req.CancelActiveItemExecutions {
			return SwitchModeResult{}, &ActiveItemExecutionsConflict{
				InitiativeName: init.Name,
				FromMode:       string(fromMode),
				ToMode:         string(targetMode),
				Executions:     active,
			}
		}
		if len(active) > 0 {
			canceled, err := s.itemExecs.CancelActiveExecutionsForInitiative(ctx, init)
			if err != nil {
				return SwitchModeResult{}, err
			}
			result.CanceledItemExecutions = canceled
		}
	}
	updated, err := s.modeUpdater.UpdateInitiativeMode(init.Name, string(targetMode))
	if err != nil {
		return SwitchModeResult{}, err
	}
	result.ToMode = string(NormalizeMode(updated.Mode))
	return result, nil
}

func (s *Service) activeOperatingModeRound(initiativeName string, mode Mode) (*RoundEnvelope, error) {
	rounds, err := s.store.ListRounds(initiativeName, mode)
	if err != nil {
		return nil, err
	}
	for i := range rounds {
		if isRoundActive(rounds[i]) {
			return &rounds[i], nil
		}
	}
	return nil, nil
}
