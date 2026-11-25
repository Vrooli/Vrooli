package wireops

import (
	"fmt"
	"go/format"
	"reflect"
	"sort"
	"strings"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	"google.golang.org/protobuf/reflect/protoreflect"

	"image-tools/internal/ops"
)

// fieldOverrides names the parameters whose compute struct field is not the
// upper-camel form of the wire field. They exist because the flat struct serves
// several operations with one field each (one "Color" for a text watermark, one
// "ScrimColor" for a scrim) and because some names are initialisms.
//
// Every override is a decision, not a convention: an entry that no longer names
// a live field fails the drift test.
var fieldOverrides = map[string]map[string]string{
	"curve":          {"exponent": "Curve"},
	"halftone":       {"lpi": "LPI"},
	"icon_container": {"format": "ContainerFormat"},
	"metadata":       {"strip_gps": "StripGPS"},
	"print_prep":     {"dpi": "DPI", "threshold": "InkThreshold"},
	"scrim":          {"color": "ScrimColor"},
}

// layeredOperations are bound by hand (plate stacks and camera messages).
var layeredOperations = map[string]bool{"composite": true, "parallax": true}

// RenderBindings renders the generated parameter bindings for the deterministic
// wire contract. It is the governed owner of bindings.generated.go.
func RenderBindings() (string, error) {
	oneof := (&opsv1.OpParams{}).ProtoReflect().Descriptor().Oneofs().ByName("op")
	if oneof == nil {
		return "", fmt.Errorf("wireops: OpParams has no \"op\" oneof")
	}
	paramsType := reflect.TypeOf(ops.Params{})

	var b strings.Builder
	b.WriteString("// AUTO-GENERATED — do not edit by hand.\n")
	b.WriteString("//\n")
	b.WriteString("// Source : packages/proto/schemas/image-tools/v1/ops/ops.proto (OpParams)\n")
	b.WriteString("// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)\n")
	b.WriteString("// Drift  : api/internal/wireops/render_test.go fails the build if this file is stale.\n")
	b.WriteString("//\n")
	b.WriteString("// One case per operation whose parameters are flat scalars. The layered views\n")
	b.WriteString("// (composite, parallax) are bound by hand in translate.go.\n\n")
	b.WriteString("package wireops\n\n")
	b.WriteString("import (\n")
	b.WriteString("\topsv1 \"github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops\"\n")
	b.WriteString("\n\t\"image-tools/internal/ops\"\n")
	b.WriteString(")\n\n")
	b.WriteString("// bindParams sets the scalar parameters of one operation. It reports whether\n")
	b.WriteString("// the operation is a generated one.\n")
	b.WriteString("func bindParams(operation string, pb *opsv1.OpParams, p *ops.Params) bool {\n")
	b.WriteString("\tswitch operation {\n")

	names := make([]string, 0, oneof.Fields().Len())
	for i := 0; i < oneof.Fields().Len(); i++ {
		names = append(names, string(oneof.Fields().Get(i).Name()))
	}
	sort.Strings(names)
	for _, name := range names {
		if layeredOperations[name] {
			continue
		}
		member := oneof.Fields().ByName(protoreflect.Name(name))
		fmt.Fprintf(&b, "\tcase %q:\n\t\tif v := pb.Get%s(); v != nil {\n", name, upperCamel(member.Name()))
		sub := member.Message()
		for i := 0; i < sub.Fields().Len(); i++ {
			field := sub.Fields().Get(i)
			targetName := upperCamel(field.Name())
			if override, ok := fieldOverrides[name][string(field.Name())]; ok {
				targetName = override
			}
			target, ok := paramsType.FieldByName(targetName)
			if !ok {
				return "", fmt.Errorf("wireops: %s.%s has no ops.Params field %q", name, field.Name(), targetName)
			}
			assignments, err := assignments("p."+targetName, "v.Get"+upperCamel(field.Name())+"()", field, target.Type)
			if err != nil {
				return "", fmt.Errorf("wireops: %s.%s: %w", name, field.Name(), err)
			}
			for _, line := range assignments {
				fmt.Fprintf(&b, "\t\t\t%s\n", line)
			}
		}
		b.WriteString("\t\t}\n\t\treturn true\n")
	}
	b.WriteString("\t}\n\treturn false\n}\n")

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return "", fmt.Errorf("wireops: format generated bindings: %w", err)
	}
	return string(src), nil
}

// assignments renders the copy of one wire field onto one compute field. It
// chooses the conversion from the two declarations rather than from a table: the
// proto kind says what the getter returns, the Go type says what the struct holds.
func assignments(target, getter string, field protoreflect.FieldDescriptor, targetType reflect.Type) ([]string, error) {
	if targetType.Kind() == reflect.Slice {
		if !field.IsList() {
			return nil, fmt.Errorf("scalar field onto list %s", targetType)
		}
		switch elem := targetType.Elem(); elem.Kind() {
		case reflect.String:
			return []string{fmt.Sprintf("%s = append(%s, %s...)", target, target, getter)}, nil
		case reflect.Int:
			return []string{
				fmt.Sprintf("for _, item := range %s {", getter),
				fmt.Sprintf("\t%s = append(%s, int(item))", target, target),
				"}",
			}, nil
		default:
			return nil, fmt.Errorf("unsupported list element type %s", elem)
		}
	}
	if field.IsList() {
		return nil, fmt.Errorf("repeated field onto scalar %s", targetType)
	}
	switch targetType.Kind() {
	case reflect.String, reflect.Bool, reflect.Float64:
		return []string{fmt.Sprintf("%s = %s", target, getter)}, nil
	case reflect.Int, reflect.Int64:
		switch field.Kind() {
		case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
			protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
			return []string{fmt.Sprintf("%s = int(%s)", target, getter)}, nil
		default:
			return []string{fmt.Sprintf("%s = %s", target, getter)}, nil
		}
	default:
		return nil, fmt.Errorf("unsupported target type %s", targetType)
	}
}

// upperCamel converts a snake_case proto name to the Go accessor suffix
// ("target_bytes" → "TargetBytes"), matching protoc-gen-go's naming.
func upperCamel(name protoreflect.Name) string {
	out := make([]byte, 0, len(name))
	up := true
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c == '_' {
			up = true
			continue
		}
		if up && c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		up = false
		out = append(out, c)
	}
	return string(out)
}
