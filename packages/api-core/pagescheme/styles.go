package pagescheme

// Styles is a page's cascade in one scheme, the way the frame applies it: the
// kit's rules under the page's own, then each element's style attribute.
type Styles struct {
	cascade cascade
	inline  map[*Element]map[string]string
}

// Styles reads the page's styles over root, the page's tree from Parse, in
// scheme (SchemeLight or SchemeDark).
func (k *Kit) Styles(page string, root *Element, scheme string) Styles {
	return Styles{cascade: newCascade(k.readPageStyles(page), scheme), inline: inlineStyles(root)}
}

// Value is the declaration that wins for e, for the first of properties that
// any rule declares ("" when none). It is the value as written: custom
// properties are not substituted, and inherited values are not followed.
func (s Styles) Value(e *Element, properties ...string) string {
	return s.cascade.value(e, s.inline, properties...)
}
