package contractcli

import (
	"io"

	"google.golang.org/protobuf/proto"

	repocontract "github.com/vrooli/repo-contract-go"
	contractapp "github.com/vrooli/vrooli/internal/app/contract"
	"github.com/vrooli/vrooli/internal/cliout"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// ContractValidationOutputMessage maps the internal contract validation output
// onto the vrooli.cli.v1 wire contract. A proto field rename breaks this
// mapping at compile time — that is the drift guard. Exported so the hygiene
// producer (which embeds this output) can reuse the mapping.
func ContractValidationOutputMessage(output contractapp.ValidationOutput) *cliv1.ContractValidationOutput {
	report := &cliv1.ContractCheckReport{
		Root:         output.Report.Root,
		ContractPath: output.Report.ContractPath,
		Success:      output.Report.Success,
	}
	for _, check := range output.Report.Checks {
		report.Checks = append(report.Checks, &cliv1.ContractCheckResult{
			Name:    check.Name,
			Passed:  check.Passed,
			Message: check.Message,
		})
	}
	return &cliv1.ContractValidationOutput{
		Success: output.Success,
		Root:    output.Root,
		Schema: &cliv1.ContractValidationCheck{
			Passed:  output.Schema.Passed,
			Message: output.Schema.Message,
		},
		Report: report,
	}
}

func contractShowOutputMessage(output contractapp.ShowOutput) *cliv1.ContractShowOutput {
	msg := &cliv1.ContractShowOutput{
		Success:      output.Success,
		Root:         output.Root,
		ContractPath: output.ContractPath,
		Schema:       output.Schema,
		Version:      output.Version,
		Platform: &cliv1.ContractPlatform{
			Mode:                       output.Platform.Mode,
			LegacyProjectBashSupported: output.Platform.LegacyProjectBashSupported,
		},
		Markers: &cliv1.ContractRootMarkers{
			RequiredDirs:  output.Markers.RequiredDirs,
			RequiredFiles: output.Markers.RequiredFiles,
		},
		Layout: &cliv1.ContractLayout{
			ProjectConfigDir: output.Layout.ProjectConfigDir,
			ScenarioDir:      output.Layout.ScenarioDir,
			ResourceDir:      output.Layout.ResourceDir,
			TemplateDir:      output.Layout.TemplateDir,
			PackageDir:       output.Layout.PackageDir,
			CommandDir:       output.Layout.CommandDir,
			InternalDir:      output.Layout.InternalDir,
			DocsDir:          output.Layout.DocsDir,
		},
		Scenario: &cliv1.ContractScenarioSpec{
			RequiredFiles:  output.Scenario.RequiredFiles,
			WellKnownPaths: output.Scenario.WellKnownPaths,
		},
		Resource: &cliv1.ContractResourceSpec{
			Manifest:       output.Resource.Manifest,
			WellKnownPaths: output.Resource.WellKnownPaths,
		},
		Globs: &cliv1.ContractGlobSpec{
			Syntax:        output.Globs.Syntax,
			RootRelative:  output.Globs.RootRelative,
			CaseSensitive: output.Globs.CaseSensitive,
			AllowAbsolute: output.Globs.AllowAbsolute,
			PathFormat:    output.Globs.PathFormat,
		},
		Environment: output.Environment,
		Sandbox: &cliv1.ContractShowSandbox{
			FullRepoScopes:      output.Sandbox.FullRepoScopes,
			ScenarioScopePrefix: output.Sandbox.ScenarioScopePrefix,
		},
	}
	if len(output.Profiles) > 0 {
		msg.Profiles = make(map[string]*cliv1.ContractProfile, len(output.Profiles))
		for name, profile := range output.Profiles {
			msg.Profiles[name] = contractProfileMessage(profile)
		}
	}
	return msg
}

func contractProfileMessage(profile repocontract.Profile) *cliv1.ContractProfile {
	return &cliv1.ContractProfile{
		Description:     profile.Description,
		Parameters:      profile.Parameters,
		Include:         profile.Include,
		OptionalInclude: profile.OptionalInclude,
		Exclude:         profile.Exclude,
	}
}

func contractResolveScenarioOutputMessage(output contractapp.ResolveScenarioOutput) *cliv1.ContractResolveScenarioOutput {
	return &cliv1.ContractResolveScenarioOutput{
		Success:  output.Success,
		Root:     output.Root,
		Scenario: output.Scenario,
		File:     output.File,
		Path:     output.Path,
	}
}

func contractMatchGlobOutputMessage(output contractapp.MatchGlobOutput) *cliv1.ContractMatchGlobOutput {
	return &cliv1.ContractMatchGlobOutput{
		Success: output.Success,
		Pattern: output.Pattern,
		Path:    output.Path,
		Matched: output.Matched,
	}
}

func writeContractValidationJSON(w io.Writer, output contractapp.ValidationOutput) error {
	return writeContractMessage(w, ContractValidationOutputMessage(output))
}

func writeContractShowJSON(w io.Writer, output contractapp.ShowOutput) error {
	return writeContractMessage(w, contractShowOutputMessage(output))
}

func writeContractResolveScenarioJSON(w io.Writer, output contractapp.ResolveScenarioOutput) error {
	return writeContractMessage(w, contractResolveScenarioOutputMessage(output))
}

func writeContractMatchGlobJSON(w io.Writer, output contractapp.MatchGlobOutput) error {
	return writeContractMessage(w, contractMatchGlobOutputMessage(output))
}

func writeContractMessage(w io.Writer, msg proto.Message) error {
	return cliout.WriteProtoJSON(w, msg)
}
