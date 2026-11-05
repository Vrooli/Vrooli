package manifestvalidation

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type argumentMapping struct {
	Name       string
	Location   string
	Path       []protoreflect.FieldDescriptor
	IsFlag     bool
	Bind       *cliapp.ManifestFlagBind
	BindWaiver string
}

type unmappedArgument struct {
	Name     string
	Location string
	Err      error
}

// argumentFindings validates the union of manifest arguments and the bound
// request descriptor. The same resolver used by program-runtime and CLI
// dispatch is the only source of truth for names, aliases, envelopes, and
// bind declarations.
func argumentFindings(m *cliapp.Manifest, surface ProtoSurface, manifestPath string) []Finding {
	if m == nil || (len(surface.Requests) == 0 && len(surface.RequestCandidates) == 0) {
		return nil
	}
	var findings []Finding
	for _, group := range m.Groups {
		for _, command := range group.Commands {
			if command.Binding.Kind != "connect-rpc" || !command.Governance.RunEligible {
				continue
			}
			request, resolveErr := surface.ResolveRequest(command.Binding.Service, command.Binding.Method)
			if resolveErr != nil {
				findings = append(findings, Finding{
					Severity:   SeverityError,
					Code:       CodeBindingAmbiguousSvc,
					Location:   fmt.Sprintf("%s#/groups/%s/commands/%s/binding", manifestPath, group.Name, command.Name),
					Message:    resolveErr.Error(),
					Suggestion: "rename the service or declare one owning package in the shared contract list",
				})
				continue
			}
			if request == nil {
				continue
			}
			schema, err := cliapp.ManifestArgs(command)
			if err != nil {
				continue
			}
			_, unmapped := commandArgumentMappings(request, command, schema, manifestPath, group.Name)
			for _, argument := range unmapped {
				findings = append(findings, Finding{
					Severity:   SeverityError,
					Code:       CodeBindingArgUnmapped,
					Location:   argument.Location,
					Message:    fmt.Sprintf("argument %q does not resolve on request type %s: %v", argument.Name, request.FullName(), argument.Err),
					Suggestion: nearestFieldSuggestion(argument.Name),
				})
			}
		}
	}
	return findings
}

// commandArgumentMappings is the one manifest-argument-to-proto mapping seam
// used by both the legacy unmapped-argument check and the semantic checks. A
// semantic rule must never invent a second resolution ladder.
func commandArgumentMappings(request protoreflect.MessageDescriptor, command cliapp.ManifestCommand, schema cliapp.ArgSchema, manifestPath, groupName string) ([]argumentMapping, []unmappedArgument) {
	var mappings []argumentMapping
	var unmapped []unmappedArgument
	for _, positional := range command.Positionals {
		if positional.LocalOnly {
			continue
		}
		location := fmt.Sprintf("%s#/groups/%s/commands/%s/positionals/%s", manifestPath, groupName, command.Name, positional.Name)
		resolved, err := cliapp.ResolveArgField(request, positional.Name, schema)
		if err != nil {
			unmapped = append(unmapped, unmappedArgument{Name: positional.Name, Location: location, Err: err})
			continue
		}
		mappings = append(mappings, argumentMapping{Name: positional.Name, Location: location, Path: resolved.Path, Bind: positional.Bind, BindWaiver: positional.BindWaiver})
	}
	for _, flag := range command.Flags {
		if flag.LocalOnly {
			continue
		}
		location := fmt.Sprintf("%s#/groups/%s/commands/%s/flags/%s", manifestPath, groupName, command.Name, flag.Name)
		resolved, err := cliapp.ResolveArgField(request, flag.Name, schema)
		if err != nil {
			unmapped = append(unmapped, unmappedArgument{Name: flag.Name, Location: location, Err: err})
			continue
		}
		mappings = append(mappings, argumentMapping{Name: flag.Name, Location: location, Path: resolved.Path, IsFlag: true, Bind: flag.Bind, BindWaiver: flag.BindWaiver})
	}
	return mappings, unmapped
}

// nearestFieldSuggestion is deliberately conservative. It tells an author
// which declaration to add without inventing a dotted path.
func nearestFieldSuggestion(name string) string {
	return fmt.Sprintf("rename the argument to its proto field, add an alias, or declare bind.field for %q", strings.ReplaceAll(name, "-", "_"))
}
