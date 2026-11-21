package domains

import (
	"development-toolchain-validator/cli/domains/golden"
	"development-toolchain-validator/cli/domains/manifest"
	"development-toolchain-validator/cli/domains/report"
	skillcatalog "development-toolchain-validator/cli/domains/skill_catalog"
	"development-toolchain-validator/cli/domains/staleness"
	validationrecord "development-toolchain-validator/cli/domains/validation_record"
	validationrun "development-toolchain-validator/cli/domains/validation_run"

	"github.com/vrooli/cli-core/cliapp"
)

// CommandGroups aggregates flat command groups from domain packages.
func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	_ = core
	return nil
}

// SubcommandGroups aggregates hierarchical command groups from domain packages.
//
// Each domain package owns a Register(core, manifest) function returning a
// SubcommandGroup built from the scenario's cli/manifest.json. The aggregator
// passes the embedded manifest bytes through unchanged; per-domain Register
// implementations call cliapp.LoadFromManifest with the relevant group name.
func SubcommandGroups(core *cliapp.ScenarioApp, manifestBytes []byte) ([]cliapp.SubcommandGroup, error) {
	registrars := []struct {
		name string
		fn   func(*cliapp.ScenarioApp, []byte) (cliapp.SubcommandGroup, error)
	}{
		{"goldens", golden.Register},
		{"manifest", manifest.Register},
		{"report", report.Register},
		{"skill-catalog", skillcatalog.Register},
		{"staleness", staleness.Register},
		{"record", validationrecord.Register},
		{"validation", validationrun.Register},
	}
	groups := make([]cliapp.SubcommandGroup, 0, len(registrars))
	for _, r := range registrars {
		g, err := r.fn(core, manifestBytes)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}
