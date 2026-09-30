package predicate

import "testing"

func lookupMap(m map[string]string) Lookup {
	return func(name string) (string, bool) {
		v, ok := m[name]
		return v, ok
	}
}

func TestEvalCases(t *testing.T) {
	ctx := map[string]string{
		"auth":         "logged_in",
		"role":         "admin",
		"viewport":     "desktop",
		"feature_beta": "false",
	}
	cases := []struct {
		src  string
		want bool
	}{
		{"auth=logged_in", true},
		{"auth=logged_out", false},
		{"auth=logged_in AND role=admin", true},
		{"auth=logged_in AND role!=admin", false},
		{"auth=logged_out OR role=admin", true},
		{"NOT auth=logged_out", true},
		{"viewport IN [mobile, tablet]", false},
		{"viewport IN [desktop, mobile]", true},
		{"(auth=logged_in AND role=viewer) OR feature_beta=false", true},
		{"feature_beta=false AND viewport=desktop", true},
	}
	for _, c := range cases {
		p, err := Parse(c.src)
		if err != nil {
			t.Fatalf("Parse(%q): %v", c.src, err)
		}
		got, err := p.Eval(lookupMap(ctx))
		if err != nil {
			t.Fatalf("Eval(%q): %v", c.src, err)
		}
		if got != c.want {
			t.Errorf("Eval(%q) = %v, want %v", c.src, got, c.want)
		}
	}
}

func TestContainsForDeepLink(t *testing.T) {
	// deep_link_policy use case: identifier resolves to a string
	// containing the equality.
	p, err := Parse("requires CONTAINS auth=logged_in")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, err := p.Eval(lookupMap(map[string]string{"requires": "auth=logged_in AND role=admin"}))
	if err != nil || !got {
		t.Errorf("CONTAINS auth=logged_in: got=%v err=%v", got, err)
	}
	got, err = p.Eval(lookupMap(map[string]string{"requires": "auth=logged_out"}))
	if err != nil || got {
		t.Errorf("negative CONTAINS: got=%v err=%v", got, err)
	}
}

func TestEmptyIsTrue(t *testing.T) {
	p, err := Parse("")
	if err != nil {
		t.Fatalf("parse empty: %v", err)
	}
	got, err := p.Eval(lookupMap(nil))
	if err != nil || !got {
		t.Errorf("empty predicate should be true: got=%v err=%v", got, err)
	}
}

func TestParseErrors(t *testing.T) {
	bad := []string{
		"auth=",              // missing rhs
		"auth=logged_in AND", // trailing AND
		"= logged_in",        // missing ident
		"auth ! logged_in",   // bare !
		"viewport IN mobile", // missing brackets
		"(auth=logged_in",    // unmatched paren
	}
	for _, s := range bad {
		if _, err := Parse(s); err == nil {
			t.Errorf("Parse(%q) should error", s)
		}
	}
}

func TestUnknownIdent(t *testing.T) {
	p, _ := Parse("missing=foo")
	if _, err := p.Eval(lookupMap(map[string]string{})); err == nil {
		t.Errorf("unknown ident should error")
	}
}
