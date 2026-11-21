package pagescheme

import (
	"testing"
)

// A small kit with the token shape of the Hub's: text and surface follow the
// theme.
const testKit = `:root{--hub-fg:#0f172a;--hub-surface:#ffffff;color:var(--hub-fg)}
@media (prefers-color-scheme: dark){:root{--hub-fg:#f8fafc;--hub-surface:#0f172a}}
:root[data-theme="dark"]{--hub-fg:#f8fafc;--hub-surface:#0f172a}`

func page(head, body string) string {
	return "<!doctype html><html><head>" + head + "</head><body>" + body + "</body></html>"
}

func TestAPageIsShownInTheSchemeItsOwnColoursWorkIn(t *testing.T) {
	kit := NewKit(testKit)
	for _, tc := range []struct{ name, page, want string }{
		{"tokens only follow the Hub", page("", `<div style="background:var(--hub-surface)">x</div>`), SchemeBoth},
		{"a declared scheme decides", page(`<meta name="color-scheme" content="dark">`, `<div style="background:#fff">x</div>`), SchemeDark},
		{"dark styles of its own follow the Hub", page(`<style>body{background:#fff}@media (prefers-color-scheme: dark){body{background:#111;color:#eee}}</style>`, "x"), SchemeBoth},
		{"a light body is light", page(`<style>body{background:#fafafa}</style>`, "x"), SchemeLight},
		{"a dark body is dark", page(`<style>html{background:#111827}</style>`, "x"), SchemeDark},
		{"an unpaired white card votes light", page(`<style>.card{background:white}</style>`, `<div class="card">x</div>`), SchemeLight},
		{"a paired card does not vote", page(`<style>.card{background:#fff;color:#111}</style>`, `<div class="card">x</div>`), SchemeBoth},
	} {
		if got := kit.Scheme(tc.page); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestUnreadableTextIsNamedWithItsColoursInEachSchemeItIsShownIn(t *testing.T) {
	kit := NewKit(testKit)
	// Paired colours keep the page following the Hub; the card's own text
	// is the kit's, which is light in dark mode.
	html := page(`<meta name="color-scheme" content="light dark"><style>.card{background:#ffffff}.faint{color:#eeeeee;background:#ffffff}</style>`,
		`<div class="card"><h2>Option A</h2></div><p class="faint">Hint</p>`)
	scheme, found := kit.Unreadable([]byte(html))
	if scheme != SchemeBoth {
		t.Fatalf("scheme %q", scheme)
	}
	if len(found) < 2 {
		t.Fatalf("found %+v", found)
	}
	for i := 1; i < len(found); i++ {
		if found[i].Ratio < found[i-1].Ratio {
			t.Fatalf("not lowest contrast first: %+v", found)
		}
	}
	var card *Unreadable
	for i := range found {
		if found[i].Text == "Option A" {
			card = &found[i]
		}
	}
	if card == nil || card.Scheme != SchemeDark || card.Foreground != "#f8fafc" || card.Background != "#ffffff" || card.Where != "h2 in div.card" {
		t.Fatalf("the card heading in dark mode: %+v", card)
	}
	if _, found := kit.Unreadable([]byte(page("", `<div style="background:var(--hub-surface);color:var(--hub-fg)">fine</div>`))); len(found) != 0 {
		t.Fatalf("a page coloured with the tokens is readable in both schemes: %+v", found)
	}
}

func TestParseReadsTheTreeABrowserWouldBuild(t *testing.T) {
	root := Parse(`<p>One<div class="hub-command a" data-x=1>run <code>make it</code></div><script>var s="<b>no</b>"</script>`)
	if root.Tag != "html" || len(root.Children) != 1 || root.Children[0].Tag != "body" {
		t.Fatalf("html and body are added: %+v", root)
	}
	body := root.Children[0]
	if len(body.Children) != 3 || body.Children[0].Tag != "p" || body.Children[1].Tag != "div" {
		t.Fatalf("a div closes the open paragraph: %+v", body.Children)
	}
	div := body.Children[1]
	if !div.Classes["hub-command"] || div.Attrs["data-x"] != "1" || div.Text != "run" || div.Children[0].Text != "make it" || div.Parent != body {
		t.Fatalf("div: %+v", div)
	}
	if script := body.Children[2]; script.Tag != "script" || script.Text != "" || len(script.Children) != 0 {
		t.Fatalf("script source is not text or markup: %+v", script)
	}
}

func TestColourParsing(t *testing.T) {
	for value, want := range map[string]string{
		"#fff":                       "#ffffff",
		"white":                      "#ffffff",
		"rgb(17 24 39)":              "#111827",
		"rgba(17, 24, 39, .9)":       "#111827",
		"hsl(0, 0%, 100%)":           "#ffffff",
		"#fff url(x.png) repeat":     "#ffffff",
		"linear-gradient(#000,#fff)": "#000000",
		"color-mix(in srgb, #000000 50%, #ffffff)": "#808080",
	} {
		color, ok := firstColor(value)
		if !ok || color.c.hex() != want {
			t.Errorf("%s: got %v %q, want %q", value, ok, color.c.hex(), want)
		}
	}
	tinted, ok := firstColor("color-mix(in srgb, #2563eb 10%, transparent)")
	if !ok || tinted.alpha < 0.099 || tinted.alpha > 0.101 {
		t.Errorf("mixing with transparent gives a translucent tint, not the full colour: %v %v", ok, tinted.alpha)
	}
	if _, ok := firstColor("url(data:image/png;base64,AAAA)"); ok {
		t.Error("an image is not a colour")
	}
}

func TestStylesNameTheDeclarationThatWinsForAnElement(t *testing.T) {
	kit := NewKit(`img{height:auto}`)
	html := page(`<style>.shot img{height:220px;object-fit:cover}</style>`, `<figure class="shot"><img src="a.png"></figure><img src="b.png" style="max-height:10rem">`)
	root := Parse(html)
	styles := kit.Styles(html, root, SchemeLight)
	body := root.Children[len(root.Children)-1]
	cropped, inline := body.Children[0].Children[0], body.Children[1]
	if styles.Value(cropped, "height") != "220px" || styles.Value(cropped, "object-fit") != "cover" {
		t.Fatalf("the page's more specific rule wins over the kit: %q %q", styles.Value(cropped, "height"), styles.Value(cropped, "object-fit"))
	}
	if styles.Value(inline, "max-height") != "10rem" || styles.Value(inline, "height") != "auto" || styles.Value(inline, "aspect-ratio") != "" {
		t.Fatalf("style attribute and kit: %q %q", styles.Value(inline, "max-height"), styles.Value(inline, "height"))
	}
}
