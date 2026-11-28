package lpbsidentity

import "github.com/vrooli/api-core/principalrole"

const AccountCreditLifecycleAuthorizationHeader = "X-Vrooli-Account-Credit-Authorization"

// AccountCreditLifecycle authorizes an app to manage its own authenticated
// user's account-credit reservation lifecycle. It is separate from the
// inference-call role because callers can retain, settle, or release holds.
var AccountCreditLifecycle = principalrole.NewRole(
	Audience,
	"lpbs:account-credit.lifecycle",
	Resource,
	"-lpbs-account-credit-lifecycle",
	"LPBS account-credit lifecycle",
)
