package repocontract

import (
	"fmt"
	"regexp"
	"strings"
)

var placeholderPattern = regexp.MustCompile(`\{[^}]+\}`)

func (c *Contract) ResolveProfile(name string, params ResolveParams) (ResolvedProfile, error) {
	profile, err := c.Profile(name)
	if err != nil {
		return ResolvedProfile{}, err
	}

	allowedScalars := map[string]struct{}{}
	allowedLists := map[string]struct{}{}
	for _, parameter := range profile.Parameters {
		if strings.HasSuffix(parameter, "[*]") {
			allowedLists[strings.TrimSuffix(parameter, "[*]")] = struct{}{}
			continue
		}
		allowedScalars[parameter] = struct{}{}
	}

	for key := range params.Values {
		if _, ok := allowedScalars[key]; !ok {
			return ResolvedProfile{}, &Error{Kind: ErrInvalidInput, Message: "unexpected scalar profile parameter", Details: key}
		}
	}
	for key := range params.Lists {
		if _, ok := allowedLists[key]; !ok {
			return ResolvedProfile{}, &Error{Kind: ErrInvalidInput, Message: "unexpected list profile parameter", Details: key}
		}
	}

	include, err := expandProfileEntries(profile.Include, params)
	if err != nil {
		return ResolvedProfile{}, err
	}
	optionalInclude, err := expandProfileEntries(profile.OptionalInclude, params)
	if err != nil {
		return ResolvedProfile{}, err
	}
	exclude, err := expandProfileEntries(profile.Exclude, params)
	if err != nil {
		return ResolvedProfile{}, err
	}

	return ResolvedProfile{
		Name:            name,
		Description:     profile.Description,
		Include:         include,
		OptionalInclude: optionalInclude,
		Exclude:         exclude,
	}, nil
}

func expandProfileEntries(entries []string, params ResolveParams) ([]string, error) {
	var out []string
	for _, entry := range entries {
		expanded, err := expandProfileEntry(entry, params)
		if err != nil {
			return nil, err
		}
		out = append(out, expanded...)
	}
	return dedupePreserveOrder(out), nil
}

func expandProfileEntry(entry string, params ResolveParams) ([]string, error) {
	tokens := placeholderPattern.FindAllString(entry, -1)
	if len(tokens) == 0 {
		return []string{entry}, nil
	}

	items := []string{entry}
	for _, token := range tokens {
		key := strings.TrimSuffix(strings.TrimPrefix(token, "{"), "}")
		if strings.HasSuffix(key, "[*]") {
			listKey := strings.TrimSuffix(key, "[*]")
			values := params.Lists[listKey]
			if len(values) == 0 {
				return nil, nil
			}
			var next []string
			for _, current := range items {
				for _, value := range values {
					next = append(next, strings.ReplaceAll(current, token, value))
				}
			}
			items = next
			continue
		}

		value, ok := params.Values[key]
		if !ok || strings.TrimSpace(value) == "" {
			return nil, &Error{Kind: ErrInvalidInput, Message: "missing required profile parameter", Details: key}
		}
		for i := range items {
			items[i] = strings.ReplaceAll(items[i], token, value)
		}
	}

	for _, item := range items {
		if placeholderPattern.MatchString(item) {
			return nil, &Error{Kind: ErrInvalidInput, Message: "unresolved profile placeholder", Details: fmt.Sprintf("%q", item)}
		}
	}
	return items, nil
}

func dedupePreserveOrder(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
