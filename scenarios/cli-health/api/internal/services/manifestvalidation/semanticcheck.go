package manifestvalidation

import (
	"fmt"
	"strings"

	validate "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

var semanticControlNames = map[string]struct{}{
	"json": {}, "yes": {}, "dry_run": {}, "format": {}, "out": {}, "output": {},
	"verbose": {}, "quiet": {}, "force": {}, "wait": {}, "no_color": {}, "limit": {},
	"page_size": {}, "offset": {}, "follow": {}, "watch": {}, "all": {},
}

// semanticFindings checks whether an explicit bind carries meaning that the
// manifest author must justify. It consumes commandArgumentMappings so the
// semantic gate and the ordinary unmapped-argument gate cannot disagree about
// what an argument targets.
func semanticFindings(m *cliapp.Manifest, surface ProtoSurface, manifestPath string) []Finding {
	if m == nil || (len(surface.Requests) == 0 && len(surface.RequestCandidates) == 0) {
		return nil
	}
	var findings []Finding
	for _, group := range m.Groups {
		for _, command := range group.Commands {
			if command.Binding.Kind != "connect-rpc" || !command.Governance.RunEligible {
				continue
			}
			request, err := surface.ResolveRequest(command.Binding.Service, command.Binding.Method)
			if err != nil || request == nil {
				continue
			}
			schema, err := cliapp.ManifestArgs(command)
			if err != nil {
				continue
			}
			mappings, unmapped := commandArgumentMappings(request, command, schema, manifestPath, group.Name)
			if len(mappings) == 0 && len(unmapped) == 0 {
				continue
			}
			findings = append(findings, collisionFindings(mappings)...)
			findings = append(findings, controlFlagFindings(mappings)...)
			findings = append(findings, redundantBindFindings(request, mappings)...)
			findings = append(findings, requiredFieldFindings(request, mappings, manifestPath, group.Name, command.Name)...)
			findings = append(findings, scalarBoundToMessageFindings(mappings)...)
		}
	}
	return findings
}

// scalarBoundToMessageFindings catches an argument that resolves cleanly onto a
// structured field it cannot populate. Callability gates pass because the field
// exists; protojson still cannot build a message from one CLI string, so the
// call fails or drops the value at runtime. A structured decoder and a stated
// bind_waiver are the two exits.
func scalarBoundToMessageFindings(mappings []argumentMapping) []Finding {
	var findings []Finding
	for _, mapping := range mappings {
		if len(mapping.Path) == 0 || strings.TrimSpace(mapping.BindWaiver) != "" {
			continue
		}
		if mapping.Bind != nil && cliapp.StructuredDecodeKind(mapping.Bind.Kind) {
			continue
		}
		target := mapping.Path[len(mapping.Path)-1]
		if cliapp.ScalarDecodableField(target) {
			continue
		}
		findings = append(findings, Finding{
			Severity: SeverityError, Code: CodeBindingScalarBoundToMessage, Location: mapping.Location,
			Message: fmt.Sprintf("argument %q targets structured proto field %q (%s) with no structured decoder",
				mapping.Name, protoPath(mapping.Path), describeFieldShape(target)),
			Suggestion: "add bind.kind json_file or json_inline, retarget the argument at a scalar field inside the message, or state a bind_waiver",
		})
	}
	return findings
}

// describeFieldShape names why a field is structured, so the finding states
// what the author must supply rather than only that something is wrong.
func describeFieldShape(field protoreflect.FieldDescriptor) string {
	switch {
	case field.IsMap():
		return "map"
	case field.IsList() && field.Kind() == protoreflect.MessageKind:
		return "repeated " + string(field.Message().FullName())
	case field.IsList():
		return "repeated " + field.Kind().String()
	case field.Kind() == protoreflect.MessageKind, field.Kind() == protoreflect.GroupKind:
		return string(field.Message().FullName())
	default:
		return field.Kind().String()
	}
}

func collisionFindings(mappings []argumentMapping) []Finding {
	byField := make(map[string][]argumentMapping)
	for _, mapping := range mappings {
		if len(mapping.Path) == 0 {
			continue
		}
		byField[protoPath(mapping.Path)] = append(byField[protoPath(mapping.Path)], mapping)
	}
	var findings []Finding
	for field, owners := range byField {
		if len(owners) < 2 {
			continue
		}
		names := make([]string, 0, len(owners))
		for _, owner := range owners {
			names = append(names, owner.Name)
		}
		findings = append(findings, Finding{
			Severity: SeverityError, Code: CodeBindingFieldCollision, Location: owners[0].Location,
			Message:    fmt.Sprintf("proto field %q is targeted by competing arguments %s", field, quoteNames(names)),
			Suggestion: fmt.Sprintf("assign one argument to %s and remove or remap the competing binds", field),
		})
	}
	return findings
}

func controlFlagFindings(mappings []argumentMapping) []Finding {
	var findings []Finding
	for _, mapping := range mappings {
		if !mapping.IsFlag || mapping.Bind == nil || strings.TrimSpace(mapping.Bind.Field) == "" || strings.TrimSpace(mapping.BindWaiver) != "" {
			continue
		}
		if _, ok := semanticControlNames[normalizeSemanticName(mapping.Name)]; !ok {
			continue
		}
		findings = append(findings, Finding{
			Severity: SeverityError, Code: CodeBindingControlFlagBound, Location: mapping.Location,
			Message:    fmt.Sprintf("control flag %q is explicitly bound to proto field %q without a waiver", mapping.Name, mapping.Bind.Field),
			Suggestion: "remove bind because control flags are CLI-local, or add a stated bind_waiver when the flag is intentional request data",
		})
	}
	return findings
}

func redundantBindFindings(request protoreflect.MessageDescriptor, mappings []argumentMapping) []Finding {
	var findings []Finding
	for _, mapping := range mappings {
		if mapping.Bind == nil || strings.TrimSpace(mapping.Bind.Field) == "" {
			continue
		}
		if strings.TrimSpace(mapping.BindWaiver) != "" {
			continue
		}
		if fieldBySemanticName(request, mapping.Name) == nil && normalizeSemanticName(mapping.Bind.Field) != normalizeSemanticName(mapping.Name) {
			continue
		}
		field := protoPath(mapping.Path)
		findings = append(findings, Finding{
			Severity: SeverityWarning, Code: CodeBindingBindWhereRenameSuffices, Location: mapping.Location,
			Message:    fmt.Sprintf("argument %q uses bind for proto field %q even though renaming the argument would select it", mapping.Name, field),
			Suggestion: fmt.Sprintf("rename the argument to %s or remove bind", field),
		})
	}
	return findings
}

func requiredFieldFindings(request protoreflect.MessageDescriptor, mappings []argumentMapping, manifestPath, groupName, commandName string) []Finding {
	mapped := make(map[protoreflect.Name]struct{})
	for _, mapping := range mappings {
		if len(mapping.Path) > 0 {
			mapped[mapping.Path[0].Name()] = struct{}{}
		}
	}
	var findings []Finding
	fields := request.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if !requiredPayloadField(field) {
			continue
		}
		if _, ok := mapped[field.Name()]; ok {
			continue
		}
		findings = append(findings, Finding{
			Severity: SeverityError, Code: CodeBindingRequiredFieldUnpopulated,
			Location:   fmt.Sprintf("%s#/groups/%s/commands/%s", manifestPath, groupName, commandName),
			Message:    fmt.Sprintf("required payload field %q has no CLI argument targeting it", field.JSONName()),
			Suggestion: fmt.Sprintf("declare an argument that populates %s", field.JSONName()),
		})
	}
	return findings
}

func requiredPayloadField(field protoreflect.FieldDescriptor) bool {
	if field.Cardinality() == protoreflect.Required {
		return true
	}
	opts, ok := field.Options().(*descriptorpb.FieldOptions)
	if !ok || opts == nil || !proto.HasExtension(opts, validate.E_Field) {
		return false
	}
	rules, ok := proto.GetExtension(opts, validate.E_Field).(*validate.FieldRules)
	if !ok || rules == nil {
		return false
	}
	if rules.GetRequired() {
		return true
	}
	bytesRules, ok := rules.GetType().(*validate.FieldRules_Bytes)
	return ok && bytesRules.Bytes != nil && bytesRules.Bytes.GetMinLen() > 0
}

func fieldBySemanticName(request protoreflect.MessageDescriptor, name string) protoreflect.FieldDescriptor {
	want := normalizeSemanticName(name)
	fields := request.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if normalizeSemanticName(string(field.Name())) == want || normalizeSemanticName(field.JSONName()) == want {
			return field
		}
	}
	return nil
}

func normalizeSemanticName(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "-", "_"))
}

func protoPath(path []protoreflect.FieldDescriptor) string {
	parts := make([]string, 0, len(path))
	for _, field := range path {
		parts = append(parts, field.JSONName())
	}
	return strings.Join(parts, ".")
}

func quoteNames(names []string) string {
	quoted := make([]string, 0, len(names))
	for _, name := range names {
		quoted = append(quoted, fmt.Sprintf("%q", name))
	}
	return strings.Join(quoted, ", ")
}
