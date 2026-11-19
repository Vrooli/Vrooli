package capability

import (
	"fmt"
	"strings"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// layeredOps are the deterministic operations whose parameters cannot be
// expressed as scalar flags: they carry multipart plate stacks (and, for
// parallax, a pair of cameras), so the CLI builds them explicitly in
// cli/domains/ops/layered.go. Naming them here keeps the exclusion reviewable:
// TestDeterministicWireCoverage in the conformance suite fails if an operation
// reaches this table without a wire-parameter message, or a wire-parameter
// operation fails to reach the generated table.
var layeredOps = map[string]struct{}{
	"composite": {},
	"parallax":  {},
}

// RenderOpsCLI renders the deterministic CLI run-command table from the
// OpParams wire contract. The table is derived from the protobuf descriptor, so
// a parameter added to ops.proto becomes a flag without a register.go edit, and
// a flag can never name a field the server does not read.
//
// Only shape comes from the descriptor; human descriptions live in the CLI's
// flag-definition table, exactly as the AI group handles its shared flags.
func RenderOpsCLI() (string, error) {
	specs, err := opsRunSpecs()
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("// AUTO-GENERATED — do not edit by hand.\n")
	b.WriteString("//\n")
	b.WriteString("// Source : packages/proto/schemas/image-tools/v1/ops/ops.proto (OpParams)\n")
	b.WriteString("// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)\n")
	b.WriteString("// Drift  : api/internal/capability/render_ops_test.go fails the build if this file is stale.\n")
	b.WriteString("//\n")
	b.WriteString("// One spec per deterministic operation reachable over the REST run edge. The\n")
	b.WriteString("// command construction, flag definitions, and parameter binding live in\n")
	b.WriteString("// register.go and params.go; this file is shape only.\n\n")
	b.WriteString("package ops\n\n")
	b.WriteString("var generatedRunSpecs = []runSpec{\n")
	for _, s := range specs {
		fmt.Fprintf(&b, "\t{Op: %q, Desc: %q, Flags: []flagSpec{\n", s.op, s.desc)
		for _, f := range s.flags {
			fmt.Fprintf(&b, "\t\t{Name: %q, Field: %q, Kind: %s},\n", f.name, f.field, f.kind)
		}
		b.WriteString("\t}},\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

// flagKindName is the generated identifier for a scalar field kind. Message and
// repeated-message fields are rejected rather than skipped: a parameter the
// generator cannot express is a decision to make, not to lose silently.
func flagKindName(fd protoreflect.FieldDescriptor) (string, error) {
	if fd.IsMap() {
		return "", fmt.Errorf("map field %q", fd.Name())
	}
	prefix := ""
	if fd.IsList() {
		prefix = "List"
	}
	switch fd.Kind() {
	case protoreflect.StringKind:
		return "flagString" + prefix, nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "flagInt32" + prefix, nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "flagInt64" + prefix, nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "flagUint32" + prefix, nil
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return "flagDouble" + prefix, nil
	case protoreflect.BoolKind:
		return "flagBool" + prefix, nil
	case protoreflect.MessageKind, protoreflect.EnumKind, protoreflect.BytesKind:
		return "", fmt.Errorf("field %q has kind %s, which the generated CLI table cannot express", fd.Name(), fd.Kind())
	default:
		return "", fmt.Errorf("field %q has unsupported kind %s", fd.Name(), fd.Kind())
	}
}

type opsFlag struct {
	name  string
	field string
	kind  string
}

type opsSpec struct {
	op    string
	desc  string
	flags []opsFlag
}

// opsRunSpecs projects the deterministic manifests onto the OpParams oneof. The
// flag name is the field name in kebab case, so there is exactly one name for a
// parameter across proto, wire, and CLI.
func opsRunSpecs() ([]opsSpec, error) {
	oneof := (&opsv1.OpParams{}).ProtoReflect().Descriptor().Oneofs().ByName("op")
	if oneof == nil {
		return nil, fmt.Errorf("capability: OpParams has no \"op\" oneof")
	}
	specs := make([]opsSpec, 0, oneof.Fields().Len())
	for _, m := range registry.manifests {
		if m.Engine != EngineCompute {
			continue
		}
		if _, skip := layeredOps[m.ID]; skip {
			continue
		}
		fd := oneof.Fields().ByName(protoreflect.Name(m.ID))
		if fd == nil {
			return nil, fmt.Errorf("capability: deterministic operation %q has no OpParams member", m.ID)
		}
		sub := fd.Message()
		flags := make([]opsFlag, 0, sub.Fields().Len())
		for i := 0; i < sub.Fields().Len(); i++ {
			field := sub.Fields().Get(i)
			kind, err := flagKindName(field)
			if err != nil {
				return nil, fmt.Errorf("capability: operation %q: %w", m.ID, err)
			}
			flags = append(flags, opsFlag{
				name:  strings.ReplaceAll(string(field.Name()), "_", "-"),
				field: string(field.Name()),
				kind:  kind,
			})
		}
		specs = append(specs, opsSpec{op: m.ID, desc: m.Summary, flags: flags})
	}
	return specs, nil
}
