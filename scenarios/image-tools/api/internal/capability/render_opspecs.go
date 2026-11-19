package capability

import (
	"fmt"
	"strconv"
	"strings"
)

// RenderOpSpecs renders the deterministic operations' parameter forms from the
// manifest declaration, cross-checked against the OpParams descriptor. The UI
// therefore learns which controls exist from data, and a manifest field that the
// wire message does not carry fails generation instead of being silently dropped
// by the client.
//
// Only operations that declare presentation.fields appear here. A deterministic
// operation without a workspace form (an advanced or CLI-first transform) simply
// omits the block.
func RenderOpSpecs() (string, error) {
	specs, err := opsRunSpecs()
	if err != nil {
		return "", err
	}
	wireFields := make(map[string]map[string]struct{}, len(specs))
	for _, spec := range specs {
		set := make(map[string]struct{}, len(spec.flags))
		for _, f := range spec.flags {
			set[f.field] = struct{}{}
		}
		wireFields[spec.op] = set
	}

	var b strings.Builder
	b.WriteString("// AUTO-GENERATED — do not edit by hand.\n")
	b.WriteString("//\n")
	b.WriteString("// Source : api/internal/capability/manifests.seed.json (presentation.fields)\n")
	b.WriteString("//         cross-checked against packages/proto/schemas/image-tools/v1/ops/ops.proto\n")
	b.WriteString("// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)\n")
	b.WriteString("// Drift  : api/internal/capability/render_opspecs_test.go fails the build if this file is stale.\n")
	b.WriteString("//\n")
	b.WriteString("// One parameter form per deterministic operation that declares one. The\n")
	b.WriteString("// Inspector renders the field's control; the field name is the OpParams field\n")
	b.WriteString("// the value is sent as. Adding a knob is a manifest + proto edit, not a UI map.\n\n")
	b.WriteString("import { ENCODE_FORMATS, type OpSpec } from \"./opSpecTypes\";\n")
	b.WriteString("import { strings } from \"../../consts/strings\";\n\n")
	b.WriteString("export const OP_SPECS: Readonly<Record<string, OpSpec>> = {\n")
	for _, m := range registry.manifests {
		if m.Presentation == nil || len(m.Presentation.Fields) == 0 {
			continue
		}
		known, ok := wireFields[m.ID]
		if !ok {
			return "", fmt.Errorf("capability: operation %q declares a UI form but has no OpParams member", m.ID)
		}
		for _, f := range m.Presentation.Fields {
			if _, ok := known[f.Name]; !ok {
				return "", fmt.Errorf("capability: operation %q form field %q is not an OpParams field", m.ID, f.Name)
			}
		}
		fmt.Fprintf(&b, "  %s: {\n", strconv.Quote(m.ID))
		if m.Presentation.AcceptsOverlay {
			b.WriteString("    acceptsOverlay: true,\n")
		}
		b.WriteString("    fields: [\n")
		for _, f := range m.Presentation.Fields {
			b.WriteString("      { ")
			b.WriteString("name: " + strconv.Quote(f.Name))
			b.WriteString(", control: " + strconv.Quote(f.Control))
			b.WriteString(", labelKey: strings." + f.LabelKey)
			b.WriteString(", default: " + string(f.Default))
			switch {
			case f.OptionsFrom == "encode_formats":
				b.WriteString(", options: ENCODE_FORMATS")
			case len(f.Options) > 0:
				quoted := make([]string, 0, len(f.Options))
				for _, opt := range f.Options {
					quoted = append(quoted, strconv.Quote(opt))
				}
				b.WriteString(", options: [" + strings.Join(quoted, ", ") + "]")
			}
			if f.Min != nil {
				b.WriteString(", min: " + formatUINumber(*f.Min))
			}
			if f.Max != nil {
				b.WriteString(", max: " + formatUINumber(*f.Max))
			}
			if f.Step != nil {
				b.WriteString(", step: " + formatUINumber(*f.Step))
			}
			if f.Unit != "" {
				b.WriteString(", unit: " + strconv.Quote(f.Unit))
			}
			if f.Advanced {
				b.WriteString(", advanced: true")
			}
			b.WriteString(" },\n")
		}
		b.WriteString("    ],\n")
		b.WriteString("  },\n")
	}
	b.WriteString("};\n")
	return b.String(), nil
}

// formatUINumber keeps an integer literal an integer (256, not 256.0) so the
// generated file stays readable and the drift diff stays stable.
func formatUINumber(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }
