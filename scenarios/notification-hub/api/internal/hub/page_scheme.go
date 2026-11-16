package hub

import (
	_ "embed"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/vrooli/api-core/pagescheme"
)

// The Hub shows a page in its own light or dark theme, or locks a page whose
// literal colours only work in one scheme to that scheme, and refuses an
// agent's page with text the operator cannot read where it is shown. Both
// readings live in packages/api-core/pagescheme, shared with the page tools
// (decisionpage), so a preview and this check agree. The UI makes the same
// scheme call (ui/src/features/packet/explainer/theme.ts) from the same
// fixtures (testdata/page-schemes).

// Page colour schemes, as PageColorScheme returns them.
const (
	SchemeBoth  = pagescheme.SchemeBoth
	SchemeLight = pagescheme.SchemeLight
	SchemeDark  = pagescheme.SchemeDark
)

// WarnPageUnreadable reports text the operator cannot read in one of the
// Hub's themes.
const WarnPageUnreadable = "page_unreadable"

// kitCSS is the starter kit stylesheet the sealed frame injects into every
// page. It is a copy of ui/src/features/packet/explainer/kit/kit.css (the UI
// owns it); TestExplainerKitMirrorsTheUIKit keeps the two equal. The packet
// check reads the kit's tokens and rules from it, so it judges a page with the
// same colours the operator sees.
//
//go:embed explainer_kit.css
var kitCSS string

var pageKit = pagescheme.NewKit(kitCSS)

// element is one element of a page, as the content rules read it.
type element = pagescheme.Element

func parseElements(page string) *element { return pagescheme.Parse(page) }

// PageColorScheme says which schemes the Hub shows a page in: SchemeBoth (it
// follows the Hub's theme), SchemeLight or SchemeDark.
func PageColorScheme(page string) string { return pageKit.Scheme(page) }

// checkReadable is the page_unreadable finding for an explainer page.
func checkReadable(explainer PacketExplainer) []PacketWarning {
	if explainer.HTML == nil || explainer.IsFolder {
		return nil
	}
	scheme, found := pageKit.Unreadable(explainer.HTML)
	if len(found) == 0 {
		return nil
	}
	var examples []string
	seen := map[string]bool{}
	for _, item := range found {
		key := item.Scheme + item.Where
		if seen[key] {
			continue
		}
		seen[key] = true
		if len(examples) == 3 {
			break
		}
		examples = append(examples, fmt.Sprintf("in %s mode %q (%s) is %s on %s, contrast %.1f:1",
			item.Scheme, shorten(item.Text, 40), item.Where, item.Foreground, item.Background, item.Ratio))
	}
	shown := "in the Hub's light and dark themes"
	if scheme != SchemeBoth {
		shown = "in " + scheme + " mode only, because its own colours only work there"
	}
	return []PacketWarning{advice(WarnPageUnreadable, explainer.Field, fmt.Sprintf(
		"the operator cannot read %d text(s) on the page (shown %s): %s. Fix: colour with the kit's tokens, "+
			"background: var(--hub-surface) with color: var(--hub-fg) (muted text: var(--hub-muted)), and whenever you set a background "+
			"set its text colour in the same rule; never a literal background like white or #fff alone. Then run --dry-run again.",
		len(found), shown, strings.Join(examples, "; ")))}
}

func shorten(text string, max int) string {
	if utf8.RuneCountInString(text) <= max {
		return text
	}
	runes := []rune(text)
	return strings.TrimSpace(string(runes[:max])) + "…"
}
