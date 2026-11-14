package hub

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateContextURLAcceptsTheFormsProducersUse(t *testing.T) {
	for _, raw := range []string{
		"",
		"   ",
		"/asks/ask-1",
		"/asks/ask-1?from=notice#top",
		"https://agent-manager.example.test/api/v1/authorizations/native/approve",
		"http://127.0.0.1:18227/runs/1",
		"https://host.example:8443/path?q=a%20b",
		" https://host.example/trimmed ",
	} {
		if err := ValidateContextURL(raw); err != nil {
			t.Errorf("%q refused: %v", raw, err)
		}
	}
}

func TestValidateContextURLRefusesLinksThatCouldPhishTheOperator(t *testing.T) {
	cases := map[string]string{
		"javascript:alert(1)":                        "scheme",
		"JavaScript:alert(1)":                        "scheme",
		"data:text/html;base64,PHNjcmlwdD4=":         "scheme",
		"file:///etc/passwd":                         "scheme",
		"ftp://host.example/x":                       "scheme",
		"vbscript:msgbox(1)":                         "scheme",
		"//evil.example/asks/1":                      "//",
		"evil.example/asks/1":                        "must be a path",
		"asks/1":                                     "must be a path",
		"https://trusted.example@evil.example/login": "credentials",
		"https://user:pass@host.example/":            "credentials",
		"https:///no-host":                           "no host",
		"/asks/1\\..\\evil":                          "backslash",
		"/asks/1\n/evil":                             "whitespace",
		"https://host.example/a b":                   "whitespace",
		"https://host.example/" + strings.Repeat("a", maxContextURLLength): "longer than",
	}
	for raw, want := range cases {
		err := ValidateContextURL(raw)
		if err == nil {
			t.Errorf("%q accepted", raw)
			continue
		}
		if !errors.Is(err, ErrInvalidArgument) || !strings.Contains(err.Error(), "context_url") || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: err = %v, want a context_url ErrInvalidArgument naming %q", raw, err, want)
		}
	}
}

func TestValidateAskContextURLRefusesOnlyScriptSchemes(t *testing.T) {
	for _, raw := range []string{
		"", "/asks/1", "steward/delegations/x/RETURN.md", "file:///home/op/.vrooli/research/runs/1/RETURN.md",
		"https://host.example/a", "http://127.0.0.1:1/x", "//host.example/x", "ftp://host.example/x",
		"https://host.example/?next=javascript:alert(1)", "/path/data:text", "mailto:me@example.test",
	} {
		if err := ValidateAskContextURL(raw); err != nil {
			t.Errorf("%q refused: %v", raw, err)
		}
	}
	for raw, scheme := range map[string]string{
		"javascript:alert(1)":                "javascript",
		"JavaScript:alert(1)":                "javascript",
		"  javascript:alert(1)":              "javascript",
		"java\tscript:alert(1)":              "javascript",
		"jav\nascript:alert(1)":              "javascript",
		"data:text/html;base64,PHNjcmlwdD4=": "data",
		"vbscript:msgbox(1)":                 "vbscript",
	} {
		err := ValidateAskContextURL(raw)
		if err == nil {
			t.Errorf("%q accepted", raw)
			continue
		}
		if !errors.Is(err, ErrInvalidArgument) || !strings.Contains(err.Error(), "context_url scheme \""+scheme+"\"") {
			t.Errorf("%q: err = %v, want a named %q refusal", raw, err, scheme)
		}
	}
}
