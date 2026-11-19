package capability

import (
	"encoding/json"
	"fmt"
	"go/format"
	"sort"
	"strconv"
	"strings"
)

// RenderParamTypes renders the generated typed views of every operation that
// declares a params_schema.
//
// Why this exists: the manifest owns each operation's parameter contract as JSON
// Schema, but the execution spine reads parameters from the persisted job payload
// — a map[string]string — and used to parse them by hand at every argument
// builder. That left the last hop unguarded: a reader could misspell a key or
// disagree about a value's type and nothing failed until a render was wrong.
//
// The generated structs make the keys and shapes compile-checked at the reader,
// and Decode<Op>Params fails closed when a persisted value no longer parses
// (which can only mean the schema and the reader diverged). The string map stays
// the persisted representation on purpose: it is the stable cross-restart job
// contract, so this replaces the hand-parsing, not the wire format.
func RenderParamTypes() (string, error) {
	var b strings.Builder
	b.WriteString("// AUTO-GENERATED — do not edit by hand.\n")
	b.WriteString("//\n")
	b.WriteString("// Source : api/internal/capability/manifests.seed.json (params_schema)\n")
	b.WriteString("// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)\n")
	b.WriteString("// Drift  : api/internal/capability/render_paramtypes_test.go fails the build if this file is stale.\n")
	b.WriteString("//\n")
	b.WriteString("// One typed view per operation that declares a params_schema. Persisted job\n")
	b.WriteString("// payloads keep the string map; these decoders are the single place that turns\n")
	b.WriteString("// it back into typed values, so an argument builder cannot misspell a key or\n")
	b.WriteString("// disagree about a type.\n\n")
	b.WriteString("package opparams\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"encoding/json\"\n")
	b.WriteString("\t\"fmt\"\n")
	b.WriteString("\t\"strconv\"\n")
	b.WriteString(")\n\n")
	b.WriteString("// float64Bits is the bit size the decoders parse a JSON number with.\n")
	b.WriteString("const float64Bits = 64\n")

	emitted := 0
	for _, m := range registry.manifests {
		if len(m.ParamsSchema) == 0 {
			continue
		}
		doc, err := parseSchemaObject(m.ParamsSchema)
		if err != nil {
			return "", fmt.Errorf("capability: operation %q params_schema: %w", m.ID, err)
		}
		root := goTypeName(m.ID) + "Params"
		decls, err := collectTypes(root, doc)
		if err != nil {
			return "", fmt.Errorf("capability: operation %q params_schema: %w", m.ID, err)
		}
		for _, d := range decls {
			b.WriteString("\n" + d.declaration + "\n")
		}
		decoder, err := renderDecoder(m.ID, root, doc)
		if err != nil {
			return "", fmt.Errorf("capability: operation %q params_schema: %w", m.ID, err)
		}
		b.WriteString("\n" + decoder + "\n")
		emitted++
	}
	if emitted == 0 {
		return "", fmt.Errorf("capability: no manifest declares a params_schema; nothing to generate")
	}
	return formatGoSource(b.String())
}

// formatGoSource runs the canonical Go formatter so a generated file is clean
// under `gofmt -l` without a second, unmanaged formatting step in the build.
// The drift test compares this exact output, so formatting belongs here rather
// than in the caller.
func formatGoSource(src string) (string, error) {
	out, err := format.Source([]byte(src))
	if err != nil {
		return "", fmt.Errorf("capability: format generated Go: %w", err)
	}
	return string(out), nil
}

// RenderParamTypesTS renders the TypeScript mirror of the same schemas so the UI
// builders that hand-write `opParams` literals are checked against the contract
// instead of against a comment.
func RenderParamTypesTS() (string, error) {
	var b strings.Builder
	b.WriteString("// AUTO-GENERATED — do not edit by hand.\n")
	b.WriteString("//\n")
	b.WriteString("// Source : api/internal/capability/manifests.seed.json (params_schema)\n")
	b.WriteString("// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)\n")
	b.WriteString("// Drift  : api/internal/capability/render_paramtypes_test.go fails the build if this file is stale.\n")
	b.WriteString("//\n")
	b.WriteString("// The TypeScript mirror of the Go param types. An `opParams` literal typed with\n")
	b.WriteString("// one of these interfaces cannot use a key the operation's schema does not\n")
	b.WriteString("// declare.\n\n")

	names := make([]string, 0, len(registry.manifests))
	entries := make(map[string]string, len(registry.manifests))
	for _, m := range registry.manifests {
		if len(m.ParamsSchema) == 0 {
			continue
		}
		doc, err := parseSchemaObject(m.ParamsSchema)
		if err != nil {
			return "", fmt.Errorf("capability: operation %q params_schema: %w", m.ID, err)
		}
		root := goTypeName(m.ID) + "OpParams"
		decls, err := collectTypesTS(root, doc)
		if err != nil {
			return "", fmt.Errorf("capability: operation %q params_schema: %w", m.ID, err)
		}
		for _, d := range decls {
			b.WriteString(d + "\n\n")
		}
		names = append(names, m.ID)
		entries[m.ID] = root
	}
	sort.Strings(names)
	b.WriteString("export interface AiOpParamsByOperation {\n")
	for _, name := range names {
		fmt.Fprintf(&b, "  %s: %s;\n", strconv.Quote(name), entries[name])
	}
	b.WriteString("}\n")
	return b.String(), nil
}

// schemaObject is the node model the generators walk. The executor only supports
// the small JSON-Schema subset `internal/paramschema` enforces; an unsupported
// keyword fails generation rather than producing a wrong type.
type schemaObject struct {
	typ        string
	properties map[string]*schemaObject
	required   map[string]bool
	item       *schemaObject
}

type generatedType struct {
	declaration   string
	tsDeclaration string
	goType        string
	tsType        string
}

func parseSchemaObject(raw []byte) (*schemaObject, error) {
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return readSchemaObject(doc, "schema")
}

func readSchemaObject(doc map[string]any, path string) (*schemaObject, error) {
	typ, _ := doc["type"].(string)
	obj := &schemaObject{typ: typ, required: map[string]bool{}}
	switch typ {
	case "object":
		props, _ := doc["properties"].(map[string]any)
		if len(props) == 0 {
			return nil, fmt.Errorf("%s: object has no properties", path)
		}
		obj.properties = make(map[string]*schemaObject, len(props))
		for _, name := range sortedKeys(props) {
			child, ok := props[name].(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s.%s: property must be an object schema", path, name)
			}
			parsed, err := readSchemaObject(child, path+"."+name)
			if err != nil {
				return nil, err
			}
			obj.properties[name] = parsed
		}
		if req, ok := doc["required"].([]any); ok {
			for _, r := range req {
				s, ok := r.(string)
				if !ok {
					return nil, fmt.Errorf("%s.required: entries must be strings", path)
				}
				if _, declared := obj.properties[s]; !declared {
					return nil, fmt.Errorf("%s.required: %q is not a declared property", path, s)
				}
				obj.required[s] = true
			}
		}
		return obj, nil
	case "array":
		item, ok := doc["items"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: array has no items schema", path)
		}
		parsed, err := readSchemaObject(item, path+"[]")
		if err != nil {
			return nil, err
		}
		obj.item = parsed
		return obj, nil
	case "integer", "number", "boolean", "string":
		return obj, nil
	default:
		return nil, fmt.Errorf("%s: unsupported schema type %q", path, typ)
	}
}

// collectTypes returns every Go declaration for the node, nested types first and
// the node's own declaration last.
func collectTypes(name string, obj *schemaObject) ([]generatedType, error) {
	if obj.typ != "object" {
		return nil, fmt.Errorf("%s: a parameter contract must be an object", name)
	}
	var decls []generatedType
	var fields strings.Builder
	for _, prop := range sortedKeys(obj.properties) {
		child := obj.properties[prop]
		field := fieldName(name, prop)
		tag := prop
		if !obj.required[prop] {
			tag += ",omitempty"
		}
		switch child.typ {
		case "object":
			nested, err := collectTypes(name+field, child)
			if err != nil {
				return nil, err
			}
			decls = append(decls, nested...)
			goType := nested[len(nested)-1].goType
			if !obj.required[prop] {
				goType = "*" + goType
			}
			fmt.Fprintf(&fields, "\t%s %s `json:%q`\n", field, goType, tag)
		case "array":
			itemType, extra, err := collectArrayItem(name+field, child.item)
			if err != nil {
				return nil, err
			}
			decls = append(decls, extra...)
			fmt.Fprintf(&fields, "\t%s []%s `json:%q`\n", field, itemType, tag)
		default:
			goType, err := scalarType(child.typ)
			if err != nil {
				return nil, fmt.Errorf("%s.%s: %w", name, prop, err)
			}
			if !obj.required[prop] {
				goType = "*" + goType
			}
			fmt.Fprintf(&fields, "\t%s %s `json:%q`\n", field, goType, tag)
		}
	}
	decls = append(decls, generatedType{
		declaration: fmt.Sprintf("// %s is the typed view of this operation's parameter object.\ntype %s struct {\n%s}", name, name, fields.String()),
		goType:      name,
	})
	return decls, nil
}

func collectArrayItem(name string, item *schemaObject) (string, []generatedType, error) {
	if item.typ == "object" {
		nested, err := collectTypes(name+"Item", item)
		if err != nil {
			return "", nil, err
		}
		return nested[len(nested)-1].goType, nested, nil
	}
	t, err := scalarType(item.typ)
	if err != nil {
		return "", nil, err
	}
	return t, nil, nil
}

// collectTypesTS returns every TypeScript declaration for the node, nested types
// first and the node's own interface last.
func collectTypesTS(name string, obj *schemaObject) ([]string, error) {
	if obj.typ != "object" {
		return nil, fmt.Errorf("%s: a parameter contract must be an object", name)
	}
	var decls []string
	var fields strings.Builder
	for _, prop := range sortedKeys(obj.properties) {
		child := obj.properties[prop]
		optional := ""
		if !obj.required[prop] {
			optional = "?"
		}
		switch child.typ {
		case "object":
			nested, err := collectTypesTS(name+goFieldName(prop), child)
			if err != nil {
				return nil, err
			}
			decls = append(decls, nested...)
			fmt.Fprintf(&fields, "  %s%s: %s;\n", prop, optional, tsTypeName(nested[len(nested)-1]))
		case "array":
			if child.item.typ == "object" {
				nested, err := collectTypesTS(name+goFieldName(prop)+"Item", child.item)
				if err != nil {
					return nil, err
				}
				decls = append(decls, nested...)
				fmt.Fprintf(&fields, "  %s%s: %s[];\n", prop, optional, tsTypeName(nested[len(nested)-1]))
			} else {
				fmt.Fprintf(&fields, "  %s%s: %s[];\n", prop, optional, tsScalarType(child.item.typ))
			}
		default:
			fmt.Fprintf(&fields, "  %s%s: %s;\n", prop, optional, tsScalarType(child.typ))
		}
	}
	decls = append(decls, fmt.Sprintf("export interface %s {\n%s}", name, fields.String()))
	return decls, nil
}

// tsTypeName extracts the interface name from a rendered TypeScript declaration.
func tsTypeName(decl string) string {
	for _, prefix := range []string{"export interface "} {
		if strings.HasPrefix(decl, prefix) {
			rest := strings.TrimPrefix(decl, prefix)
			if i := strings.IndexAny(rest, " {\n"); i >= 0 {
				return rest[:i]
			}
			return rest
		}
	}
	return decl
}

func scalarType(typ string) (string, error) {
	switch typ {
	case "integer":
		return "int", nil
	case "number":
		return "float64", nil
	case "boolean":
		return "bool", nil
	case "string":
		return "string", nil
	default:
		return "", fmt.Errorf("unsupported value type %q", typ)
	}
}

func tsScalarType(typ string) string {
	switch typ {
	case "integer", "number":
		return "number"
	case "boolean":
		return "boolean"
	default:
		return "string"
	}
}

// renderDecoder emits the string-map decoder for one root object. It only
// touches declared keys; an absent key leaves the field nil so the operation's
// own default applies.
func renderDecoder(op, name string, root *schemaObject) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "// Decode%s turns the persisted string map into a typed view of the %q\n", name, op)
	fmt.Fprintf(&b, "// parameters. It fails closed on a value that no longer parses.\n")
	fmt.Fprintf(&b, "func Decode%s(m map[string]string) (%s, error) {\n", name, name)
	fmt.Fprintf(&b, "\tvar out %s\n", name)
	for _, prop := range sortedKeys(root.properties) {
		child := root.properties[prop]
		field := fieldName(name, prop)
		fmt.Fprintf(&b, "\tif raw, ok := m[%q]; ok && raw != \"\" {\n", prop)
		assign := func(target, goType string) {
			fmt.Fprintf(&b, "\t\tvar v %s\n", goType)
			fmt.Fprintf(&b, "\t\tif err := json.Unmarshal([]byte(raw), &v); err != nil {\n")
			fmt.Fprintf(&b, "\t\t\treturn out, fmt.Errorf(%q, err)\n", op+": "+prop+": %w")
			b.WriteString("\t\t}\n")
			if strings.HasPrefix(target, "*") {
				fmt.Fprintf(&b, "\t\tout.%s = &v\n", field)
			} else {
				fmt.Fprintf(&b, "\t\tout.%s = v\n", field)
			}
		}
		switch child.typ {
		case "object":
			base := name + field
			if root.required[prop] {
				assign("", base)
			} else {
				assign("*", base)
			}
		case "array":
			itemType, _, err := collectArrayItem(name+field, child.item)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "\t\tvar v []%s\n", itemType)
			fmt.Fprintf(&b, "\t\tif err := json.Unmarshal([]byte(raw), &v); err != nil {\n")
			fmt.Fprintf(&b, "\t\t\treturn out, fmt.Errorf(%q, err)\n", op+": "+prop+": %w")
			b.WriteString("\t\t}\n")
			fmt.Fprintf(&b, "\t\tout.%s = v\n", field)
		default:
			assignValue := "&v"
			if root.required[prop] {
				assignValue = "v"
			}
			switch child.typ {
			case "boolean":
				fmt.Fprintf(&b, "\t\tv, err := strconv.ParseBool(raw)\n\t\tif err != nil {\n\t\t\treturn out, fmt.Errorf(%q, err)\n\t\t}\n\t\tout.%s = %s\n", op+": "+prop+": %w", field, assignValue)
			case "integer":
				fmt.Fprintf(&b, "\t\tv, err := strconv.Atoi(raw)\n\t\tif err != nil {\n\t\t\treturn out, fmt.Errorf(%q, err)\n\t\t}\n\t\tout.%s = %s\n", op+": "+prop+": %w", field, assignValue)
			case "number":
				fmt.Fprintf(&b, "\t\tv, err := strconv.ParseFloat(raw, float64Bits)\n\t\tif err != nil {\n\t\t\treturn out, fmt.Errorf(%q, err)\n\t\t}\n\t\tout.%s = %s\n", op+": "+prop+": %w", field, assignValue)
			case "string":
				fmt.Fprintf(&b, "\t\tv := raw\n\t\tout.%s = %s\n", field, assignValue)
			}
		}
		b.WriteString("\t}\n")
	}
	b.WriteString("\treturn out, nil\n}\n")
	return b.String(), nil
}

// goTypeName converts an operation id to the exported Go type prefix.
func goTypeName(op string) string {
	parts := strings.FieldsFunc(op, func(r rune) bool { return r == '_' || r == '-' || r == '.' })
	var b strings.Builder
	for _, p := range parts {
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

// fieldName converts a schema property to a Go field name and drops an
// operation prefix that would otherwise repeat the type name
// (InpaintParams.InpaintCropToMask → CropToMask). The rule is shared by the type
// and decoder generators so they cannot disagree.
func fieldName(typeName, prop string) string {
	field := goFieldName(prop)
	prefix := strings.TrimSuffix(typeName, "Params")
	if prefix != "" && strings.HasPrefix(field, prefix) && len(field) > len(prefix) && isUpper(field[len(prefix)]) {
		return field[len(prefix):]
	}
	return field
}

func isUpper(b byte) bool { return b >= 'A' && b <= 'Z' }

// goFieldName converts a schema property name (snake_case or camelCase) to an
// exported Go field name.
func goFieldName(prop string) string {
	parts := strings.FieldsFunc(prop, func(r rune) bool { return r == '_' || r == '-' })
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
