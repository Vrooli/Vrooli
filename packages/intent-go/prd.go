package intent

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var operationalTargetPattern = regexp.MustCompile(`\bOT-[A-Za-z0-9]+-[A-Za-z0-9]+\b`)

// PRDExtractor extracts outcome claims from PRD.md.
type PRDExtractor interface {
	ExtractPRDClaims(scenarioRoot string) ([]CapabilityClaim, error)
}

type FilePRDExtractor struct{}

func (FilePRDExtractor) ExtractPRDClaims(scenarioRoot string) ([]CapabilityClaim, error) {
	prdPath := filepath.Join(scenarioRoot, "PRD.md")
	file, err := os.Open(prdPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	seen := make(map[string]CapabilityClaim)
	scanner := bufio.NewScanner(file)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		for _, token := range operationalTargetPattern.FindAllString(line, -1) {
			id := strings.ToUpper(strings.TrimSpace(token))
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = CapabilityClaim{
				ID:         id,
				Altitude:   Outcome,
				Text:       cleanTargetText(line),
				Anchor:     "PRD.md:" + strconv.Itoa(lineNo),
				Provenance: "prd",
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	claims := make([]CapabilityClaim, 0, len(seen))
	for _, claim := range seen {
		claims = append(claims, claim)
	}
	sort.Slice(claims, func(i, j int) bool { return claims[i].ID < claims[j].ID })
	return claims, nil
}

func cleanTargetText(line string) string {
	text := strings.TrimSpace(line)
	text = strings.TrimPrefix(text, "- [ ]")
	text = strings.TrimPrefix(text, "- [x]")
	text = strings.TrimPrefix(text, "- [X]")
	return strings.TrimSpace(text)
}
