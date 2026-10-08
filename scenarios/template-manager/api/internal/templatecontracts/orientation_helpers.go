package templatecontracts

import (
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

type InfoOutput struct {
	Scenario InfoScenarioData `json:"scenario"`
}

type InfoScenarioData struct {
	Generation      *scenariomodel.GenerationMetadata `json:"generation,omitempty"`
	TemplateDrifted bool                              `json:"template_drifted,omitempty"`
}
