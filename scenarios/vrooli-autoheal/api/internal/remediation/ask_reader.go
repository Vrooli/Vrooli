package remediation

import (
	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/vrooli/packages/credential-authority-go/hubidentity"
)

// askReaderSource is autoheal's name on the events it raises, and so the asks
// Notification Hub lets its ask-reader principal read back.
const askReaderSource = "vrooli-autoheal"

// NewAskReaderTokens is autoheal's ask-reader identity: the credential the
// operator issues with serviceprincipal.IssueCommand, exchanged at scenario-authenticator
// by the shared client. There is no fallback. Without the credential the
// verifier refuses, so no remediation is approved on a read the Hub did not
// authenticate.
func NewAskReaderTokens() *askreader.Tokens {
	return hubidentity.NewAskReaderTokens(askReaderSource)
}
