package wireproto

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestProtocolDocumentationPinsFieldsAndReasonCodes(t *testing.T) {
	doc, err := os.ReadFile("../../../docs/internal/TERMINAL-INPUT-PROTOCOL.md")
	if err != nil {
		t.Fatal(err)
	}

	fieldNames := map[string]bool{}
	fieldRow := regexp.MustCompile(`^\| ` + "`" + `([^` + "`" + `]+)` + "`" + ` \|`)
	for _, line := range strings.Split(string(doc), "\n") {
		match := fieldRow.FindStringSubmatch(line)
		if len(match) == 2 && (strings.Contains(line, "| yes |") || strings.Contains(line, "| when ")) {
			fieldNames[match[1]] = true
		}
	}

	jsonFields := map[string]bool{}
	typeOfMessage := reflect.TypeOf(TerminalMessage{})
	for i := 0; i < typeOfMessage.NumField(); i++ {
		tag := strings.Split(typeOfMessage.Field(i).Tag.Get("json"), ",")[0]
		if tag != "" && tag != "-" {
			jsonFields[tag] = true
		}
	}
	for field := range fieldNames {
		if !jsonFields[field] {
			t.Errorf("protocol documentation names JSON field %q, but TerminalMessage does not expose it", field)
		}
	}

	for _, reason := range []string{
		StdinAckReasonTmuxFailed,
		StdinAckReasonPTYClosed,
		StdinAckReasonOffsetGap,
		StdinAckReasonUnreconcilable,
	} {
		if !strings.Contains(string(doc), "`"+reason+"`") {
			t.Errorf("emitted stdin reason %q is missing from the protocol documentation", reason)
		}
	}
}
