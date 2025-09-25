package testkit

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

type DocCase struct {
	ID                 string
	Description        string
	Input              string
	Language           string
	ShouldFail         bool
	ExpectedViolations *int
	ExpectedMessage    string
	Path               string
	Scenario           string
}

var testCaseRegex = regexp.MustCompile(`(?s)<test-case\s+([^>]+)>(.*?)</test-case>`)
var attrRegex = regexp.MustCompile(`([A-Za-z0-9_-]+)="([^"]*)"`)

func LoadDocCases(t *testing.T, filename string) []DocCase {
	t.Helper()

	path := filename
	if !filepath.IsAbs(path) {
		path = filepath.Clean(path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", filename, err)
	}

	matches := testCaseRegex.FindAllSubmatch(data, -1)
	cases := make([]DocCase, 0, len(matches))

	for _, match := range matches {
		attrs := parseAttributes(string(match[1]))
		body := string(match[2])

		tc := DocCase{
			ID:       attrs["id"],
			Path:     attrs["path"],
			Scenario: attrs["scenario"],
		}
		if tc.ID == "" {
			t.Fatalf("test case missing id attribute in %s", filename)
		}
		if v, ok := attrs["should-fail"]; ok {
			tc.ShouldFail = strings.EqualFold(v, "true")
		}

		inputLang, inputContent := extractTag(body, "input")
		tc.Language = inputLang
		tc.Input = strings.TrimSpace(inputContent)

		if desc := extractSimpleTag(body, "description"); desc != "" {
			tc.Description = desc
		}

		if ev := extractSimpleTag(body, "expected-violations"); ev != "" {
			if n, err := strconv.Atoi(strings.TrimSpace(ev)); err == nil {
				tc.ExpectedViolations = &n
			}
		}

		if msg := extractSimpleTag(body, "expected-message"); msg != "" {
			tc.ExpectedMessage = msg
		}

		cases = append(cases, tc)
	}

	return cases
}

func EvaluateDocCase(t *testing.T, tc DocCase, count int, messages []string, runErr error) {
	t.Helper()

	if runErr != nil {
		t.Fatalf("execution error for %s: %v", tc.ID, runErr)
	}

	if tc.ExpectedViolations != nil {
		if count != *tc.ExpectedViolations {
			t.Fatalf("%s: expected %d violations, got %d", tc.ID, *tc.ExpectedViolations, count)
		}
	} else if tc.ShouldFail {
		if count == 0 {
			t.Fatalf("%s: expected failure but rule returned no violations", tc.ID)
		}
	} else {
		if count != 0 {
			t.Fatalf("%s: expected success but rule returned %d violations", tc.ID, count)
		}
	}

	if tc.ExpectedMessage != "" && count > 0 {
		if !containsMessage(messages, tc.ExpectedMessage) {
			t.Fatalf("%s: expected message containing %q in %v", tc.ID, tc.ExpectedMessage, messages)
		}
	}
}

func containsMessage(messages []string, needle string) bool {
	for _, msg := range messages {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func parseAttributes(spec string) map[string]string {
	attrs := make(map[string]string)
	for _, match := range attrRegex.FindAllStringSubmatch(spec, -1) {
		attrs[match[1]] = match[2]
	}
	return attrs
}

func extractTag(body, name string) (string, string) {
	startTag := fmt.Sprintf("<%s", name)
	i := strings.Index(body, startTag)
	if i == -1 {
		return "", ""
	}
	rest := body[i:]
	// parse attribute using xml decoder for robustness
	decoder := xml.NewDecoder(strings.NewReader(rest))
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch elem := tok.(type) {
		case xml.StartElement:
			if elem.Name.Local != name {
				continue
			}
			var lang string
			for _, attr := range elem.Attr {
				if attr.Name.Local == "language" {
					lang = attr.Value
					break
				}
			}
			var content strings.Builder
			for {
				tok, err = decoder.Token()
				if err != nil {
					break
				}
				switch v := tok.(type) {
				case xml.EndElement:
					if v.Name.Local == name {
						return lang, content.String()
					}
				case xml.CharData:
					content.Write([]byte(v))
				}
			}
			return lang, content.String()
		}
	}
	return "", ""
}

func extractSimpleTag(body, name string) string {
	tag := fmt.Sprintf("<%s>", name)
	start := strings.Index(body, tag)
	if start == -1 {
		return ""
	}
	start += len(tag)
	endTag := fmt.Sprintf("</%s>", name)
	end := strings.Index(body[start:], endTag)
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(body[start : start+end])
}

func DefaultPath(tc DocCase, fallback string) string {
	if tc.Path != "" {
		return tc.Path
	}
	return fallback
}

func DefaultScenario(tc DocCase, fallback string) string {
	if strings.TrimSpace(tc.Scenario) != "" {
		return strings.TrimSpace(tc.Scenario)
	}
	return fallback
}
