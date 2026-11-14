package hub

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"
)

// maxContextURLLength bounds a context link a producer supplies.
const maxContextURLLength = 2048

// ValidateContextURL checks a context_url a direct producer supplies on Send or
// Relay. A context link is what the operator taps on a notice, so it is refused
// unless it can only take them somewhere ordinary:
//
//   - a Hub-relative path ("/asks/<id>", not "//host/...", which a browser reads
//     as another site), or
//   - an absolute http or https URL with a host and no embedded credentials
//     ("https://trusted.example@evil.example" shows one host and opens another).
//
// Every other form is refused with an error that names the problem: javascript:,
// data:, file: and any other scheme, a scheme-less host, a backslash, whitespace
// or control characters, and an over-long link. An empty link is valid: there is
// none.
//
// The event path (a *.notification.v1 or decision event's context_url) is not
// checked here: its links are rendered after the UI's own same-site check, and
// rejecting an event would lose a notification.
func ValidateContextURL(raw string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	refuse := func(format string, args ...any) error {
		return fmt.Errorf("%w: context_url %s", ErrInvalidArgument, fmt.Sprintf(format, args...))
	}
	if len(raw) > maxContextURLLength {
		return refuse("is longer than %d characters", maxContextURLLength)
	}
	for _, r := range raw {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return refuse("contains whitespace or a control character")
		}
		if r == '\\' {
			return refuse("contains a backslash, which browsers read as a slash")
		}
	}
	if strings.HasPrefix(raw, "/") {
		if strings.HasPrefix(raw, "//") {
			return refuse("starts with //, which opens another site; use a path like /asks/<id> or a full https URL")
		}
		return nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return refuse("is not a valid URL: %v", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	case "":
		return refuse("must be a path starting with / or an http(s) URL")
	default:
		return refuse("scheme %q is not allowed; use https, http, or a path starting with /", parsed.Scheme)
	}
	if parsed.Hostname() == "" {
		return refuse("has no host")
	}
	if parsed.User != nil {
		return refuse("embeds credentials before the host, which disguises where it goes")
	}
	return nil
}

// scriptSchemes are the URL schemes that run code when a browser opens them.
var scriptSchemes = map[string]bool{"javascript": true, "data": true, "vbscript": true}

// ValidateAskContextURL checks the context_url of a direct Ask. It is narrower than
// ValidateContextURL on purpose: the operator's own agents legitimately link a
// local file:// path or a relative path in an ask (the packet check advises on
// those as local_context_url), and refusing them would reject live traffic. It
// refuses only the schemes that run code, named in the error.
//
// A browser ignores whitespace and control characters inside a scheme
// ("java\tscript:"), so they are removed before the scheme is read.
func ValidateAskContextURL(raw string) error {
	compact := strings.Map(func(r rune) rune {
		if r <= ' ' || r == 0x7f {
			return -1
		}
		return r
	}, raw)
	scheme, _, found := strings.Cut(strings.ToLower(compact), ":")
	if !found || strings.ContainsAny(scheme, "/?#") || !scriptSchemes[scheme] {
		return nil
	}
	return fmt.Errorf("%w: context_url scheme %q can run code and is not allowed; link a path, a file:// path or an http(s) URL", ErrInvalidArgument, scheme)
}
