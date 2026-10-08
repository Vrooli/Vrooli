package cliresolve

import (
	"fmt"
	"strings"
)

// ShellMetacharacters is the defense-in-depth set shared by typed command
// producers and consumers. Commands still execute as argv, never through a
// shell, but malformed fields are rejected before crossing that boundary.
const ShellMetacharacters = "|&;<>()$`\\\"'\n\r\t*?[]{}!#~"

// ValidateArgvToken rejects a token that could carry shell syntax.
func ValidateArgvToken(token string) error {
	if i := strings.IndexAny(token, ShellMetacharacters); i >= 0 {
		return fmt.Errorf("contains shell metacharacter %q", string(token[i]))
	}
	return nil
}
