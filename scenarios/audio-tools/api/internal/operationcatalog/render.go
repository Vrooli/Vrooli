package operationcatalog

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// RenderTypeScript renders the transform-family operations as the checked-in
// UI spec module. The generated file is drift-gated by catalog_test.go, so the
// seed remains the only place an operation is declared.
func RenderTypeScript(c Catalog) ([]byte, error) {
	transforms := c.Transform()
	if len(transforms) == 0 {
		return nil, fmt.Errorf("operationcatalog: no transform operations to render")
	}
	var b strings.Builder
	b.WriteString(tsHeader)
	b.WriteString("\nexport type AudioOpControl =\n")
	b.WriteString("  | \"number\"\n  | \"text\"\n  | \"select\"\n  | \"file\"\n  | \"files\"\n  | \"checkbox\";\n\n")

	b.WriteString("export type AudioOpId =\n")
	for i, op := range transforms {
		sep := "  |"
		if i == 0 {
			sep = "  "
		}
		fmt.Fprintf(&b, "%s %q\n", sep, op.ID)
	}
	b.WriteString(";\n\n")

	b.WriteString("export type AudioOpMethod =\n")
	for i, op := range transforms {
		sep := "  |"
		if i == 0 {
			sep = "  "
		}
		fmt.Fprintf(&b, "%s %q\n", sep, lowerCamel(op.CLI.Method))
	}
	b.WriteString(";\n\n")

	writeUnion(&b, "AudioOpParamName", orderedParamNames(transforms))
	writeUnion(&b, "AudioOpLabelKey", orderedLabelKeys(transforms, false))
	writeUnion(&b, "AudioOpDescriptionKey", orderedLabelKeys(transforms, true))
	writeUnion(&b, "AudioOpOptionLabelKey", orderedOptionLabelKeys(transforms))

	b.WriteString(tsInterfaces)
	b.WriteString("export const audioOps: readonly AudioOpSpec[] = [\n")
	for _, op := range transforms {
		b.WriteString("  {\n")
		fmt.Fprintf(&b, "    id: %q,\n", op.ID)
		fmt.Fprintf(&b, "    method: %q,\n", lowerCamel(op.CLI.Method))
		fmt.Fprintf(&b, "    summary: %q,\n", op.Summary)
		fmt.Fprintf(&b, "    labelKey: %q,\n", op.Presentation.LabelKey)
		fmt.Fprintf(&b, "    descriptionKey: %q,\n", op.Presentation.DescKey)
		fmt.Fprintf(&b, "    order: %d,\n", op.Presentation.Order)
		b.WriteString("    params: [\n")
		for _, p := range op.Params {
			b.WriteString("      {\n")
			fmt.Fprintf(&b, "        name: %q,\n", p.Name)
			fmt.Fprintf(&b, "        type: %q,\n", p.Type)
			fmt.Fprintf(&b, "        control: %q,\n", p.Control)
			fmt.Fprintf(&b, "        required: %t,\n", p.Required)
			fmt.Fprintf(&b, "        defaultValue: %q,\n", p.Default)
			if p.Min != nil {
				fmt.Fprintf(&b, "        min: %s,\n", formatFloat(*p.Min))
			}
			if p.Max != nil {
				fmt.Fprintf(&b, "        max: %s,\n", formatFloat(*p.Max))
			}
			if p.Step != nil {
				fmt.Fprintf(&b, "        step: %s,\n", formatFloat(*p.Step))
			}
			if p.Unit != "" {
				fmt.Fprintf(&b, "        unit: %q,\n", p.Unit)
			}
			if p.Advanced {
				b.WriteString("        advanced: true,\n")
			}
			if len(p.Options) > 0 {
				b.WriteString("        options: [\n")
				for _, o := range p.Options {
					fmt.Fprintf(&b, "          { value: %q, labelKey: %q },\n", o.Value, o.LabelKey)
				}
				b.WriteString("        ],\n")
			}
			b.WriteString("      },\n")
		}
		b.WriteString("    ],\n")
		b.WriteString("  },\n")
	}
	b.WriteString("];\n\n")
	b.WriteString("export const audioOpsById = (id: AudioOpId): AudioOpSpec | undefined =>\n")
	b.WriteString("  audioOps.find((op) => op.id === id);\n")
	return []byte(b.String()), nil
}

func writeUnion(b *strings.Builder, name string, values []string) {
	fmt.Fprintf(b, "export type %s =\n", name)
	for i, value := range values {
		sep := "  |"
		if i == 0 {
			sep = "  "
		}
		fmt.Fprintf(b, "%s %q\n", sep, value)
	}
	b.WriteString(";\n\n")
}

func orderedParamNames(ops []Operation) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, op := range ops {
		for _, p := range op.Params {
			if _, ok := seen[p.Name]; ok {
				continue
			}
			seen[p.Name] = struct{}{}
			out = append(out, p.Name)
		}
	}
	return out
}

func orderedLabelKeys(ops []Operation, description bool) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, op := range ops {
		key := op.Presentation.LabelKey
		if description {
			key = op.Presentation.DescKey
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func orderedOptionLabelKeys(ops []Operation) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, op := range ops {
		for _, p := range op.Params {
			for _, option := range p.Options {
				if _, ok := seen[option.LabelKey]; ok {
					continue
				}
				seen[option.LabelKey] = struct{}{}
				out = append(out, option.LabelKey)
			}
		}
	}
	return out
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func lowerCamel(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

const tsHeader = `// AUTO-GENERATED — do not edit by hand.
//
// Source : api/internal/operationcatalog/operations.seed.json
// Codegen: cd api && go run ./cmd/gen-operations
//
// Drift gate: api/internal/operationcatalog/catalog_test.go compares this
// file to a fresh render of the seed. Edit the seed, regenerate, and commit.
`

const tsInterfaces = `export interface AudioOpParamOption {
  readonly value: string;
  readonly labelKey: AudioOpOptionLabelKey;
}

export interface AudioOpParam {
  readonly name: AudioOpParamName;
  readonly type: string;
  readonly control: AudioOpControl;
  readonly required: boolean;
  readonly defaultValue: string;
  readonly min?: number;
  readonly max?: number;
  readonly step?: number;
  readonly unit?: string;
  readonly advanced?: boolean;
  readonly options?: readonly AudioOpParamOption[];
}

export interface AudioOpSpec {
  readonly id: AudioOpId;
  readonly method: AudioOpMethod;
  readonly summary: string;
  readonly labelKey: AudioOpLabelKey;
  readonly descriptionKey: AudioOpDescriptionKey;
  readonly order: number;
  readonly params: readonly AudioOpParam[];
}

`
