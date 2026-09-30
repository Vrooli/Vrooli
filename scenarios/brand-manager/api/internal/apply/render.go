package apply

import (
	"encoding/json"
	"fmt"
	"strings"
)

// nameValue pairs a CSS custom-property name with its value.
type nameValue struct {
	name  string
	value string
}

// colorPairs returns the canonical color name→value pairs for a brand's color
// system, in a stable order.
func colorPairs(c Colors) []nameValue {
	return []nameValue{
		{"primary", c.Primary},
		{"secondary", c.Secondary},
		{"accent", c.Accent},
		{"background", c.Background},
		{"surface", c.Surface},
		{"text", c.Text},
		{"error", c.Error},
	}
}

// typographyPairs returns the canonical typography name→value pairs.
func typographyPairs(t Typography) []nameValue {
	return []nameValue{
		{"heading-font", t.HeadingFont},
		{"body-font", t.BodyFont},
		{"mono-font", t.MonoFont},
		{"base-font-size", t.BaseFontSize},
	}
}

// generateCSSBlock builds a :root CSS block from name/value pairs with
// brand-manager markers, so a re-apply can recognise and overwrite managed
// declarations. Empty values are omitted.
func generateCSSBlock(section string, pairs []nameValue) string {
	var b strings.Builder
	fmt.Fprintf(&b, "/* brand-manager:%s - Auto-generated brand %s */\n", section, section)
	b.WriteString(":root {\n")
	for _, p := range pairs {
		if p.value != "" {
			fmt.Fprintf(&b, "  --brand-%s: %s; /* brand-manager:%s */\n", p.name, p.value, p.name)
		}
	}
	b.WriteString("}\n")
	return b.String()
}

// generateColorCSS produces CSS custom properties from brand colors.
func generateColorCSS(c Colors) string {
	return generateCSSBlock("colors", colorPairs(c))
}

// generateTypographyCSS produces CSS custom properties from brand typography.
func generateTypographyCSS(t Typography) string {
	return generateCSSBlock("typography", typographyPairs(t))
}

// mergeManifest applies the brand's identity onto an existing manifest.json
// payload (or an empty document when existing is nil/empty), returning the
// re-marshalled bytes. The _brand_* keys track provenance so a later apply or
// audit can tell the values came from this brand.
func mergeManifest(existing []byte, brand BrandView) ([]byte, error) {
	manifest := map[string]any{}
	if len(existing) > 0 {
		// Ignore a corrupt existing manifest and start fresh — apply is the
		// authority for the managed keys, and refusing to write because an
		// unrelated key is malformed would be surprising.
		_ = json.Unmarshal(existing, &manifest)
	}

	if brand.DisplayName != "" {
		manifest["name"] = brand.DisplayName
		manifest["short_name"] = brand.DisplayName
		manifest["_brand_display_name"] = brand.DisplayName
	}
	if brand.Tagline != "" {
		manifest["description"] = brand.Tagline
		manifest["_brand_tagline"] = brand.Tagline
	}
	manifest["_brand_id"] = brand.ID
	manifest["_brand_version"] = brand.Version

	out, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
