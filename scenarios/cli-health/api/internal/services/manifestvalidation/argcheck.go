package manifestvalidation

import (
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/reflect/protoreflect"
)

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
			for _, positional := range schema.Positionals {
				if positional.LocalOnly {
					continue
				}
				findings = append(findings, checkArgument(request, schema, positional.Name, fmt.Sprintf("%s#/groups/%s/commands/%s/positionals/%s", manifestPath, group.Name, command.Name, positional.Name))...)
			}
			for _, flag := range schema.Flags {
				if flag.LocalOnly {
					continue
				}
				findings = append(findings, checkArgument(request, schema, flag.Name, fmt.Sprintf("%s#/groups/%s/commands/%s/flags/%s", manifestPath, group.Name, command.Name, flag.Name))...)
			}
		}
	}
	return findings
}

func checkArgument(request protoreflect.MessageDescriptor, schema cliapp.ArgSchema, name, location string) []Finding {
	if _, err := cliapp.ResolveArgField(request, name, schema); err != nil {
		return []Finding{{
			Severity:   SeverityError,
			Code:       CodeBindingArgUnmapped,
			Location:   location,
			Message:    fmt.Sprintf("argument %q does not resolve on request type %s: %v", name, request.FullName(), err),
			Suggestion: nearestFieldSuggestion(name),
		}}
	}
	return nil
}

// nearestFieldSuggestion is deliberately conservative. It tells an author
// which declaration to add without inventing a dotted path.
func nearestFieldSuggestion(name string) string {
	return fmt.Sprintf("rename the argument to its proto field, add an alias, or declare bind.field for %q", strings.ReplaceAll(name, "-", "_"))
}
