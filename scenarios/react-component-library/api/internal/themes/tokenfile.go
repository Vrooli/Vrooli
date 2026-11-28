package themes

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type TokenTier string

const (
	TokenTierExpression TokenTier = "Expression"
	TokenTierRhythm     TokenTier = "Rhythm"
	TokenTierContract   TokenTier = "Contract"
)

type DesignToken struct {
	Name  string
	Value string
	Tier  TokenTier
}

var (
	tierAnnotationRE  = regexp.MustCompile(`^\s*/\*\s*@tier\s+(Expression|Rhythm|Contract)\s*\*/\s*$`)
	declarationLineRE = regexp.MustCompile(`^\s*(--[A-Za-z0-9_-]+)\s*:\s*(.+);\s*$`)
)

// ParseTokenCSS reads the authored declaration stream. A tier annotation
// applies to the declaration immediately following it; later declarations of
// the same property replace the value but retain the canonical tier.
func ParseTokenCSS(css string) ([]DesignToken, error) {
	byName := map[string]DesignToken{}
	order := make([]string, 0)
	var pendingTier TokenTier
	scanner := bufio.NewScanner(strings.NewReader(css))
	for scanner.Scan() {
		line := scanner.Text()
		if match := tierAnnotationRE.FindStringSubmatch(line); len(match) == 2 {
			pendingTier = TokenTier(match[1])
			continue
		}
		match := declarationLineRE.FindStringSubmatch(line)
		if len(match) != 3 {
			if strings.TrimSpace(line) != "" && !strings.HasPrefix(strings.TrimSpace(line), "/*") {
				pendingTier = ""
			}
			continue
		}
		name := match[1]
		token, exists := byName[name]
		if !exists {
			order = append(order, name)
		}
		token.Name = name
		token.Value = strings.TrimSpace(match[2])
		if pendingTier != "" {
			token.Tier = pendingTier
		}
		byName[name] = token
		pendingTier = ""
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	result := make([]DesignToken, 0, len(order))
	for _, name := range order {
		result = append(result, byName[name])
	}
	return result, nil
}

func ReadTokenFile(path string) ([]DesignToken, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseTokenCSS(string(raw))
}

// ResolveKitTokens composes the shared base vocabulary followed by one kit's
// deliberate overrides, matching generator and preview cascade order.
func ResolveKitTokens(root, kitID string) ([]DesignToken, error) {
	if strings.TrimSpace(kitID) == "" || filepath.Base(kitID) != kitID || strings.HasPrefix(kitID, ".") {
		return nil, fmt.Errorf("invalid design kit id %q", kitID)
	}
	basePath := filepath.Join(root, "templates", "design", "_base", "tokens.css")
	kitPath := filepath.Join(root, "templates", "design", kitID, "adapters", "react-vite-tailwind", "tokens.css")
	base, err := ReadTokenFile(basePath)
	if err != nil {
		return nil, fmt.Errorf("read shared token base: %w", err)
	}
	overrides, err := ReadTokenFile(kitPath)
	if err != nil {
		return nil, fmt.Errorf("read kit %s tokens: %w", kitID, err)
	}
	resolved := map[string]DesignToken{}
	for _, token := range base {
		resolved[token.Name] = token
	}
	for _, token := range overrides {
		current, exists := resolved[token.Name]
		if exists && token.Tier == "" {
			token.Tier = current.Tier
		}
		resolved[token.Name] = token
	}
	names := make([]string, 0, len(resolved))
	for name := range resolved {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]DesignToken, 0, len(names))
	for _, name := range names {
		result = append(result, resolved[name])
	}
	return result, nil
}

func ComposeKitCSS(root, kitID string) (string, error) {
	basePath := filepath.Join(root, "templates", "design", "_base", "tokens.css")
	kitPath := filepath.Join(root, "templates", "design", kitID, "adapters", "react-vite-tailwind", "tokens.css")
	base, err := os.ReadFile(basePath)
	if err != nil {
		return "", err
	}
	overrides, err := os.ReadFile(kitPath)
	if err != nil {
		return "", err
	}
	return string(base) + "\n/* kit overrides: " + kitID + " */\n" + string(overrides), nil
}
