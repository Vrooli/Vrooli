package preview

import (
	"fmt"
	"sort"
	"strings"

	"react-component-library/internal/utilityclass"
)

// ConsumerTokenSet is the explicit vocabulary supplied by an adopter preview.
// It is deliberately a set of utility token names, not a Tailwind config AST;
// the preview contract only needs to answer whether emitted semantic classes
// have a consumer definition.
type ConsumerTokenSet struct {
	Name   string
	Tokens map[string]struct{}
}

type UndefinedToken struct {
	Class string
	Token string
}

// ValidateAdoptedContext is the fail-closed gate used by adopted-context
// preview. It scans the same semantic utility classes shipped by the asset and
// reports every missing consumer token in stable order.
func ValidateAdoptedContext(source string, set ConsumerTokenSet) []UndefinedToken {
	seen := map[string]struct{}{}
	var missing []UndefinedToken
	for _, hit := range utilityclass.EmitsAny(source) {
		class, token, ok := semanticToken(hit.Class)
		if !ok {
			continue
		}
		if _, ok := set.Tokens[token]; ok {
			continue
		}
		key := class + "\x00" + token
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		missing = append(missing, UndefinedToken{Class: class, Token: token})
	}
	sort.Slice(missing, func(i, j int) bool {
		if missing[i].Token == missing[j].Token {
			return missing[i].Class < missing[j].Class
		}
		return missing[i].Token < missing[j].Token
	})
	return missing
}

func semanticToken(class string) (string, string, bool) {
	base := class
	if colon := strings.LastIndex(base, ":"); colon >= 0 {
		base = base[colon+1:]
	}
	for _, prefix := range []string{"bg-", "text-", "border-", "ring-", "outline-", "divide-"} {
		if !strings.HasPrefix(base, prefix) {
			continue
		}
		token := strings.SplitN(strings.TrimPrefix(base, prefix), "/", 2)[0]
		return class, token, token != ""
	}
	return "", "", false
}

func AdoptedContextError(set ConsumerTokenSet, missing []UndefinedToken) error {
	if len(missing) == 0 {
		return nil
	}
	parts := make([]string, 0, len(missing))
	for _, item := range missing {
		parts = append(parts, item.Class+" requires "+item.Token)
	}
	return fmt.Errorf("adopted-context preview %q has undefined consumer token(s): %s", set.Name, strings.Join(parts, ", "))
}
