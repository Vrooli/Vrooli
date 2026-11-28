package pagescheme

import (
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
)

// The readability rule of the packet check: every piece of text on the page
// must stand out from what is behind it, in each scheme the Hub shows the page
// in (Kit.Scheme). It builds the page's element tree, runs a small
// cascade of the kit's rules and the page's own over it (selectors by
// specificity and order, the page above the kit, text colour inherited,
// backgrounds painted from the nearest element that has one), and measures
// the contrast of each text against its background. Text below
// MinReadableContrast is reported (Kit.Unreadable); the Hub refuses an agent's
// ask with such a page, because the operator would get a page they cannot read.
//
// It reads the markup as written: text a script adds later is not seen, and a
// selector it does not understand (:has(), state such as :hover) is left out
// rather than guessed.

// MinReadableContrast is the contrast below which text counts as unreadable:
// WCAG's floor for large text. Body text should reach 4.5:1.
const MinReadableContrast = 3.0

// Element is one element of the page, as Parse reads it.
type Element struct {
	// Tag is the lower-case tag name.
	Tag string
	// Attrs are the attributes, names lowered; Classes are the class names.
	Attrs   map[string]string
	Classes map[string]bool
	Parent  *Element
	// Children are the child elements in order.
	Children []*Element
	// Text is the element's own text (not its children's), whitespace folded.
	Text string
}

var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "param": true, "source": true, "track": true, "wbr": true,
}

// impliedEnd lists, for an opening tag, the open elements it closes when one
// of them is the current element (the HTML parser's implied end tags, for the
// cases pages actually rely on).
var impliedEnd = map[string][]string{
	"li": {"li"}, "dt": {"dt", "dd"}, "dd": {"dt", "dd"}, "tr": {"tr", "td", "th"}, "td": {"td", "th"}, "th": {"td", "th"},
	"option": {"option"}, "p": {"p"},
}

var paragraphClosers = map[string]bool{
	"address": true, "article": true, "aside": true, "blockquote": true, "details": true, "div": true, "dl": true,
	"fieldset": true, "figure": true, "footer": true, "form": true, "h1": true, "h2": true, "h3": true, "h4": true,
	"h5": true, "h6": true, "header": true, "hr": true, "main": true, "nav": true, "ol": true, "pre": true,
	"section": true, "table": true, "ul": true,
}

var htmlTokenPattern = regexp.MustCompile(`(?s)<(/?)([A-Za-z][A-Za-z0-9-]*)((?:[^>"']|"[^"]*"|'[^']*')*)>|<![^>]*>`)

// Parse builds the page's element tree under an html root with a body, the
// way a browser would for the pages agents write. Scripts and styles are
// skipped, so their source is not text.
func Parse(page string) *Element {
	page = commentPattern.ReplaceAllString(page, "")
	page = scriptBodyPattern.ReplaceAllString(styleBodyPattern.ReplaceAllString(page, "$1$2"), "$1$2")
	root := &Element{Tag: "#document", Attrs: map[string]string{}, Classes: map[string]bool{}}
	stack := []*Element{root}
	top := func() *Element { return stack[len(stack)-1] }
	addText := func(text string) {
		text = strings.Join(strings.Fields(html.UnescapeString(text)), " ")
		if text == "" {
			return
		}
		parent := top()
		if parent.Text != "" {
			parent.Text += " "
		}
		parent.Text += text
	}
	last := 0
	for _, loc := range htmlTokenPattern.FindAllStringSubmatchIndex(page, -1) {
		addText(page[last:loc[0]])
		last = loc[1]
		if loc[4] < 0 { // doctype or other <!...>
			continue
		}
		closing := page[loc[2]:loc[3]] == "/"
		tag := strings.ToLower(page[loc[4]:loc[5]])
		if closing {
			for i := len(stack) - 1; i > 0; i-- {
				if stack[i].Tag == tag {
					stack = stack[:i]
					break
				}
			}
			continue
		}
		if closes, ok := impliedEnd[tag]; ok && containsString(closes, top().Tag) {
			stack = stack[:len(stack)-1]
		} else if paragraphClosers[tag] && top().Tag == "p" {
			stack = stack[:len(stack)-1]
		}
		attrs := attributes(page[loc[6]:loc[7]])
		node := &Element{Tag: tag, Attrs: attrs, Classes: map[string]bool{}, Parent: top()}
		for _, class := range strings.Fields(attrs["class"]) {
			node.Classes[class] = true
		}
		top().Children = append(top().Children, node)
		selfClosing := strings.HasSuffix(strings.TrimSpace(page[loc[6]:loc[7]]), "/")
		if !voidElements[tag] && !selfClosing {
			stack = append(stack, node)
		}
	}
	addText(page[last:])
	return documentRoot(root)
}

// documentRoot returns the html Element, adding html and body where the page
// left them out.
func documentRoot(document *Element) *Element {
	var htmlNode *Element
	for _, child := range document.Children {
		if child.Tag == "html" {
			htmlNode = child
		}
	}
	if htmlNode == nil {
		htmlNode = &Element{Tag: "html", Attrs: map[string]string{}, Classes: map[string]bool{}, Children: document.Children}
		for _, child := range htmlNode.Children {
			child.Parent = htmlNode
		}
	}
	htmlNode.Parent = nil
	var body *Element
	var loose []*Element
	kept := htmlNode.Children[:0:0]
	for _, child := range htmlNode.Children {
		switch child.Tag {
		case "body":
			body = child
			kept = append(kept, child)
		case "head":
			kept = append(kept, child)
		default:
			loose = append(loose, child)
		}
	}
	if body == nil {
		body = &Element{Tag: "body", Attrs: map[string]string{}, Classes: map[string]bool{}, Parent: htmlNode, Text: htmlNode.Text}
		htmlNode.Text = ""
		kept = append(kept, body)
	}
	for _, child := range loose {
		child.Parent = body
		body.Children = append(body.Children, child)
	}
	htmlNode.Children = kept
	return htmlNode
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

// --- selectors ----------------------------------------------------------------

// compound is one compound selector: a tag, an id, classes, attribute tests
// and pseudo-classes, joined to the compound before it by combinator.
type compound struct {
	combinator byte // ' ', '>', '+', '~' (to the compound on its left); 0 for the first
	tag, id    string
	classes    []string
	attrs      []attrTest
	pseudos    []pseudo
}

type attrTest struct{ name, op, value string }

type pseudo struct {
	name string
	args []selectorChain // for :not, :is, :where
}

type selectorChain []compound

// specificity is (ids, classes, tags).
type specificity [3]int

func (a specificity) less(b specificity) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

var (
	errUnsupportedSelector = fmt.Errorf("unsupported selector")
	selectorTokenPattern   = regexp.MustCompile(`^(?:\*|[A-Za-z][A-Za-z0-9-]*)`)
	identPattern           = regexp.MustCompile(`^-?[A-Za-z_][A-Za-z0-9_-]*`)
	attrPattern            = regexp.MustCompile(`^\[\s*([A-Za-z_:][A-Za-z0-9_:.-]*)\s*(?:([~|^$*]?=)\s*("[^"]*"|'[^']*'|[^\]\s]+)\s*(?:[iIsS]\s*)?)?\]`)
)

// matchingPseudos are the pseudo-classes the check can decide from the markup.
var matchingPseudos = map[string]bool{
	"root": true, "first-child": true, "last-child": true, "only-child": true, "link": true, "any-link": true,
	"not": true, "is": true, "where": true, "matches": true,
	// Counted as matching: position and emptiness rules rarely change colour.
	"nth-child": true, "nth-last-child": true, "nth-of-type": true, "nth-last-of-type": true,
	"first-of-type": true, "last-of-type": true, "only-of-type": true, "lang": true, "dir": true,
}

// parseSelector reads one complex selector. A pseudo-Element, a state
// pseudo-class or :has() makes it unsupported: the rule is not applied.
func parseSelector(text string) (selectorChain, error) {
	var chain selectorChain
	text = strings.TrimSpace(text)
	current := compound{}
	started := false
	var combinator byte
	for i := 0; i < len(text); {
		c := text[i]
		switch {
		case c == ' ' || c == '>' || c == '+' || c == '~':
			j := i
			found := byte(' ')
			for j < len(text) && strings.IndexByte(" >+~\t\n", text[j]) >= 0 {
				if text[j] != ' ' && text[j] != '\t' && text[j] != '\n' {
					found = text[j]
				}
				j++
			}
			if started {
				chain = append(chain, current)
				current, started = compound{}, false
				combinator = found
			}
			i = j
			continue
		case c == '*' || isLetter(c):
			token := selectorTokenPattern.FindString(text[i:])
			if token != "*" {
				current.tag = strings.ToLower(token)
			}
			i += len(token)
		case c == '#' || c == '.':
			ident := identPattern.FindString(text[i+1:])
			if ident == "" {
				return nil, errUnsupportedSelector
			}
			if c == '#' {
				current.id = ident
			} else {
				current.classes = append(current.classes, ident)
			}
			i += 1 + len(ident)
		case c == '[':
			match := attrPattern.FindStringSubmatch(text[i:])
			if match == nil {
				return nil, errUnsupportedSelector
			}
			current.attrs = append(current.attrs, attrTest{name: strings.ToLower(match[1]), op: match[2], value: strings.Trim(match[3], `"'`)})
			i += len(match[0])
		case c == ':':
			if strings.HasPrefix(text[i:], "::") {
				return nil, errUnsupportedSelector
			}
			name := strings.ToLower(identPattern.FindString(text[i+1:]))
			if name == "" || !matchingPseudos[name] {
				return nil, errUnsupportedSelector
			}
			i += 1 + len(name)
			p := pseudo{name: name}
			if i < len(text) && text[i] == '(' {
				body, end := parenBody(text, i)
				i = end
				if name == "not" || name == "is" || name == "where" || name == "matches" {
					for _, part := range splitTop(body, ',') {
						inner, err := parseSelector(part)
						if err != nil {
							return nil, err
						}
						p.args = append(p.args, inner)
					}
				}
			}
			current.pseudos = append(current.pseudos, p)
		default:
			return nil, errUnsupportedSelector
		}
		if !started {
			current.combinator = combinator
			started = true
		}
	}
	if !started {
		return nil, errUnsupportedSelector
	}
	return append(chain, current), nil
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func parenBody(text string, open int) (string, int) {
	depth := 0
	for j := open; j < len(text); j++ {
		switch text[j] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return text[open+1 : j], j + 1
			}
		}
	}
	return text[open+1:], len(text)
}

func (chain selectorChain) specificity() specificity {
	var s specificity
	for _, c := range chain {
		if c.id != "" {
			s[0]++
		}
		s[1] += len(c.classes) + len(c.attrs)
		if c.tag != "" {
			s[2]++
		}
		for _, p := range c.pseudos {
			switch p.name {
			case "where":
			case "not", "is", "matches":
				var best specificity
				for _, arg := range p.args {
					if spec := arg.specificity(); best.less(spec) {
						best = spec
					}
				}
				for i := range s {
					s[i] += best[i]
				}
			default:
				s[1]++
			}
		}
	}
	return s
}

func (chain selectorChain) matches(e *Element) bool {
	return matchFrom(chain, len(chain)-1, e)
}

func matchFrom(chain selectorChain, index int, e *Element) bool {
	if e == nil || !chain[index].matches(e) {
		return false
	}
	if index == 0 {
		return true
	}
	switch chain[index].combinator {
	case '>':
		return matchFrom(chain, index-1, e.Parent)
	case '+':
		return matchFrom(chain, index-1, previousSibling(e))
	case '~':
		for sibling := previousSibling(e); sibling != nil; sibling = previousSibling(sibling) {
			if matchFrom(chain, index-1, sibling) {
				return true
			}
		}
		return false
	default:
		for ancestor := e.Parent; ancestor != nil; ancestor = ancestor.Parent {
			if matchFrom(chain, index-1, ancestor) {
				return true
			}
		}
		return false
	}
}

func previousSibling(e *Element) *Element {
	if e.Parent == nil {
		return nil
	}
	var previous *Element
	for _, child := range e.Parent.Children {
		if child == e {
			return previous
		}
		previous = child
	}
	return nil
}

func (c compound) matches(e *Element) bool {
	if c.tag != "" && c.tag != e.Tag {
		return false
	}
	if c.id != "" && e.Attrs["id"] != c.id {
		return false
	}
	for _, class := range c.classes {
		if !e.Classes[class] {
			return false
		}
	}
	for _, test := range c.attrs {
		value, ok := e.Attrs[test.name]
		if !ok || !attrMatches(test, value) {
			return false
		}
	}
	for _, p := range c.pseudos {
		if !p.matches(e) {
			return false
		}
	}
	return true
}

func attrMatches(test attrTest, value string) bool {
	switch test.op {
	case "":
		return true
	case "=":
		return value == test.value
	case "~=":
		return containsString(strings.Fields(value), test.value)
	case "|=":
		return value == test.value || strings.HasPrefix(value, test.value+"-")
	case "^=":
		return test.value != "" && strings.HasPrefix(value, test.value)
	case "$=":
		return test.value != "" && strings.HasSuffix(value, test.value)
	case "*=":
		return test.value != "" && strings.Contains(value, test.value)
	}
	return false
}

func (p pseudo) matches(e *Element) bool {
	switch p.name {
	case "root":
		return e.Tag == "html"
	case "first-child":
		return e.Parent == nil || len(e.Parent.Children) > 0 && e.Parent.Children[0] == e
	case "last-child":
		return e.Parent == nil || len(e.Parent.Children) > 0 && e.Parent.Children[len(e.Parent.Children)-1] == e
	case "only-child":
		return e.Parent == nil || len(e.Parent.Children) == 1
	case "link", "any-link":
		_, ok := e.Attrs["href"]
		return e.Tag == "a" && ok
	case "not":
		for _, arg := range p.args {
			if arg.matches(e) {
				return false
			}
		}
		return true
	case "is", "where", "matches":
		for _, arg := range p.args {
			if arg.matches(e) {
				return true
			}
		}
		return false
	}
	return true
}

// --- the cascade ----------------------------------------------------------------

// styleRule is a parsed rule ready for the cascade.
type styleRule struct {
	chain       selectorChain
	specificity specificity
	decls       map[string]string
	scheme      string
	// layer orders origins: the kit (0) loses to the page (1); a style
	// attribute (2) wins over both.
	layer, order int
}

// cascade holds the rules of one page in one scheme.
type cascade struct {
	rules []styleRule
	vars  map[string]string
}

func newCascade(styles pageStyles, scheme string) cascade {
	c := cascade{vars: themeVars(styles, scheme)}
	add := func(rules []cssRule, layer int) {
		for _, rule := range rules {
			if rule.scheme != "" && rule.scheme != scheme {
				continue
			}
			if rule.inline {
				continue
			}
			chain, err := parseSelector(rule.selector)
			if err != nil {
				continue
			}
			c.rules = append(c.rules, styleRule{chain: chain, specificity: chain.specificity(), decls: rule.decls, scheme: rule.scheme, layer: layer, order: len(c.rules)})
		}
	}
	add(styles.kit, 0)
	add(styles.rules, 1)
	return c
}

// value is the winning declaration of any of properties for e ("" when
// none): a style attribute, else the rule that wins the cascade.
func (c cascade) value(e *Element, inline map[*Element]map[string]string, properties ...string) string {
	if decls, ok := inline[e]; ok {
		for _, property := range properties {
			if value, found := decls[property]; found {
				return value
			}
		}
	}
	var best *styleRule
	bestValue := ""
	for i := range c.rules {
		rule := &c.rules[i]
		value, declared := "", false
		for _, property := range properties {
			if v, ok := rule.decls[property]; ok {
				value, declared = v, true
				break
			}
		}
		if !declared || !rule.chain.matches(e) {
			continue
		}
		if best == nil || best.layer < rule.layer || best.layer == rule.layer && (best.specificity.less(rule.specificity) || best.specificity == rule.specificity && best.order < rule.order) {
			best, bestValue = rule, value
		}
	}
	return bestValue
}

// unreadableText is one text the operator cannot read.
type unreadableText struct {
	scheme, words, where string
	fg, bg               rgb
	ratio                float64
}

// readPage measures every text of the page in scheme.
func readPage(root *Element, styles pageStyles, inline map[*Element]map[string]string, scheme string) []unreadableText {
	c := newCascade(styles, scheme)
	// The frame marks the root with the theme it shows the page in.
	root.Attrs["data-theme"] = scheme
	fallbackFG, fallbackBG := rgb{0, 0, 0}, rgb{255, 255, 255}
	if scheme == SchemeDark {
		fallbackFG, fallbackBG = rgb{255, 255, 255}, rgb{18, 18, 18}
	}
	var found []unreadableText
	// overlaid is true inside an absolutely positioned element: its text sits
	// over whatever is drawn beneath it, which the markup does not say, so it is
	// not measured until an element sets its own background.
	var walk func(e *Element, fg, bg rgb, overlaid bool)
	walk = func(e *Element, fg, bg rgb, overlaid bool) {
		switch e.Tag {
		case "head", "script", "style", "template", "noscript", "svg", "math", "title":
			return
		}
		if _, hidden := e.Attrs["hidden"]; hidden {
			return
		}
		if strings.TrimSpace(c.value(e, inline, "display")) == "none" {
			return
		}
		if value := c.value(e, inline, "color"); value != "" {
			if resolved, ok := resolve(value, c.vars); ok {
				if color, found := firstColor(resolved); found && color.alpha > 0 {
					fg = over(color.c, color.alpha, bg)
				}
			}
		}
		switch strings.TrimSpace(c.value(e, inline, "position")) {
		case "absolute", "fixed":
			overlaid = true
		}
		if value := c.value(e, inline, "background-color", "background"); value != "" {
			if resolved, ok := resolve(value, c.vars); ok {
				if color, found := firstColor(resolved); found && color.alpha > 0 {
					bg = over(color.c, color.alpha, bg)
					if color.alpha >= 0.9 {
						overlaid = false
					}
				}
			}
		}
		if e.Text != "" && e.Tag != "html" && !overlaid {
			if ratio := contrast(fg, bg); ratio < MinReadableContrast {
				found = append(found, unreadableText{scheme: scheme, words: e.Text, where: describe(e), fg: fg, bg: bg, ratio: ratio})
			}
		}
		for _, child := range e.Children {
			walk(child, fg, bg, overlaid)
		}
	}
	walk(root, fallbackFG, fallbackBG, false)
	return found
}

// describe names an element the way its author wrote it: h1.title in section.ask.
func describe(e *Element) string {
	name := func(e *Element) string {
		out := e.Tag
		if id := e.Attrs["id"]; id != "" {
			out += "#" + id
		}
		classes := strings.Fields(e.Attrs["class"])
		if len(classes) > 2 {
			classes = classes[:2]
		}
		for _, class := range classes {
			out += "." + class
		}
		return out
	}
	out := name(e)
	for ancestor := e.Parent; ancestor != nil && ancestor.Tag != "body" && ancestor.Tag != "html"; ancestor = ancestor.Parent {
		if ancestor.Attrs["class"] != "" || ancestor.Attrs["id"] != "" {
			return out + " in " + name(ancestor)
		}
	}
	return out
}

// inlineStyles reads the style attributes of the tree.
func inlineStyles(root *Element) map[*Element]map[string]string {
	out := map[*Element]map[string]string{}
	var walk func(e *Element)
	walk = func(e *Element) {
		if style := strings.TrimSpace(e.Attrs["style"]); style != "" {
			out[e] = parseDeclarations(style)
		}
		for _, child := range e.Children {
			walk(child)
		}
	}
	walk(root)
	return out
}

// pageSchemes are the schemes the Hub shows a page in.
func pageSchemes(scheme string) []string {
	if scheme == SchemeBoth {
		return []string{SchemeLight, SchemeDark}
	}
	return []string{scheme}
}

// Unreadable is one piece of text the operator cannot read in one scheme.
type Unreadable struct {
	// Scheme is the scheme it was read in: SchemeLight or SchemeDark.
	Scheme string
	// Text is the text, and Where names its element ("h2 in div.card").
	Text, Where string
	// Foreground and Background are the colours as #rrggbb.
	Foreground, Background string
	// Ratio is their contrast.
	Ratio float64
}

// Unreadable reads the page in each scheme the Hub shows it in (Scheme) and
// returns that scheme and the text below MinReadableContrast, lowest first.
func (k *Kit) Unreadable(page []byte) (string, []Unreadable) {
	text := string(page)
	styles := k.readPageStyles(text)
	scheme := styles.scheme()
	root := Parse(text)
	inline := inlineStyles(root)
	var found []Unreadable
	for _, each := range pageSchemes(scheme) {
		for _, item := range readPage(root, styles, inline, each) {
			found = append(found, Unreadable{Scheme: item.scheme, Text: item.words, Where: item.where, Foreground: item.fg.hex(), Background: item.bg.hex(), Ratio: item.ratio})
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].Ratio < found[j].Ratio })
	return scheme, found
}
