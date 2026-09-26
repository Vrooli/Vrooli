package bindings

import (
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

type semanticCounts struct {
	fieldCollisions           int
	controlFlagsBound         int
	requiredFieldsUnpopulated int
	bindsWhereRenameSuffices  int
	scalarBoundToMessage      int
}

type semanticArgument struct {
	name       string
	isFlag     bool
	path       []protoreflect.FieldDescriptor
	bind       *cliapp.ManifestFlagBind
	bindWaiver string
}

// addSemanticCounts is the runtime doctor's independent projection of the
// manifest semantic gate. It deliberately uses cliapp.ResolveArgField, the
// same resolver used by Execute and Describe, while retaining the raw manifest
// bind/waiver metadata needed for semantic diagnostics.
func (r *Registry) addSemanticCounts(scenario string, command cliapp.ManifestCommand, request protoreflect.MessageDescriptor, schema cliapp.ArgSchema) {
	args := make([]semanticArgument, 0, len(command.Positionals)+len(command.Flags))
	for _, positional := range command.Positionals {
		if positional.LocalOnly {
			continue
		}
		if resolved, err := cliapp.ResolveArgField(request, positional.Name, schema); err == nil {
			args = append(args, semanticArgument{name: positional.Name, path: resolved.Path, bind: positional.Bind, bindWaiver: positional.BindWaiver})
		}
	}
	for _, flag := range command.Flags {
		if flag.LocalOnly {
			continue
		}
		if resolved, err := cliapp.ResolveArgField(request, flag.Name, schema); err == nil {
			args = append(args, semanticArgument{name: flag.Name, isFlag: true, path: resolved.Path, bind: flag.Bind, bindWaiver: flag.BindWaiver})
		}
	}
	if len(args) == 0 {
		return
	}
	counts := r.semantic[scenario]
	byField := make(map[string]int)
	mappedTopLevel := make(map[protoreflect.Name]struct{})
	for _, argument := range args {
		if len(argument.path) == 0 {
			continue
		}
		field := semanticPath(argument.path)
		byField[field]++
		mappedTopLevel[argument.path[0].Name()] = struct{}{}
		// A structured target with no structured decoder resolves cleanly and
		// still cannot be populated from one CLI string, so callability checks
		// never see it. Count it as its own dimension.
		if strings.TrimSpace(argument.bindWaiver) == "" &&
			!(argument.bind != nil && cliapp.StructuredDecodeKind(argument.bind.Kind)) &&
			!cliapp.ScalarDecodableField(argument.path[len(argument.path)-1]) {
			counts.scalarBoundToMessage++
		}
		if argument.bind != nil && strings.TrimSpace(argument.bind.Field) != "" {
			if argument.isFlag && strings.TrimSpace(argument.bindWaiver) == "" {
				if _, ok := semanticControlNames[normalizeRuntimeName(argument.name)]; ok {
					counts.controlFlagsBound++
				}
			}
			if strings.TrimSpace(argument.bindWaiver) == "" && (fieldByRuntimeName(request, argument.name) != nil || normalizeRuntimeName(argument.bind.Field) == normalizeRuntimeName(argument.name)) {
				counts.bindsWhereRenameSuffices++
			}
		}
	}
	for _, owners := range byField {
		if owners > 1 {
			counts.fieldCollisions++
		}
	}
	fields := request.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if requiredRuntimePayloadField(field) {
			if _, ok := mappedTopLevel[field.Name()]; !ok {
				counts.requiredFieldsUnpopulated++
			}
		}
	}
	r.semantic[scenario] = counts
}

func fieldByRuntimeName(request protoreflect.MessageDescriptor, name string) protoreflect.FieldDescriptor {
	want := normalizeRuntimeName(name)
	fields := request.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if normalizeRuntimeName(string(field.Name())) == want || normalizeRuntimeName(field.JSONName()) == want {
			return field
		}
	}
	return nil
}

func requiredRuntimePayloadField(field protoreflect.FieldDescriptor) bool {
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

func normalizeRuntimeName(name string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "-", "_"))
}

func semanticPath(path []protoreflect.FieldDescriptor) string {
	parts := make([]string, 0, len(path))
	for _, field := range path {
		parts = append(parts, field.JSONName())
	}
	return strings.Join(parts, ".")
}
