// Package lpbsidentity names service-principal roles accepted by Landing Page
// Business Suite. Each role has one audience and one capability so a token
// issued for a reporting or deployment client cannot grant customer credits.
package lpbsidentity

import "github.com/vrooli/api-core/principalrole"

const (
	Audience                        = "scenario-authenticator:landing-page-business-suite"
	Resource                        = "landing-page-business-suite"
	CreditCallerAuthorizationHeader = "X-Vrooli-Credit-Caller-Authorization"
)

// TreasuryAcceptedReceiptCredit grants only the ability to submit an already
// accepted Money Ledger receipt for credit-lot attribution.
var TreasuryAcceptedReceiptCredit = principalrole.NewRole(
	Audience,
	"lpbs:credits.grant-accepted-crypto-receipt",
	Resource,
	"-lpbs-credit-grant",
	"LPBS accepted-receipt credit grant",
)

// GatewayCreditInference identifies a calling scenario at LPBS's standard
// metered inference boundary. The end-user bearer still supplies the human
// account context; this separate principal proves only the invoking scenario.
// LPBS binds the verified source to the canonical feature registry and request
// operation before issuing or admitting any credit assertion.
var GatewayCreditInference = principalrole.NewRole(
	Audience,
	"lpbs:ai-inference.call",
	Resource,
	"-lpbs-ai-inference",
	"LPBS Gateway credit inference",
)

// AgentManagerBrowserSessionSettlement permits Agent Manager to ask LPBS to
// close an already-reserved browser session after AM has validated the run
// grant and BAS has supplied a consumed close proof. It does not create holds
// or authorize inference.
var AgentManagerBrowserSessionSettlement = principalrole.NewRole(
	Audience,
	"lpbs:browser-session.settle-and-receipt",
	Resource,
	"-browser-session-settler",
	"LPBS browser session settlement and receipt",
)

// ScenarioToDesktopDeployment manages deploy-readiness and the LPBS remote
// profile handoff used by the desktop pipeline. It carries no billing scope.
var ScenarioToDesktopDeployment = principalrole.NewRole(
	Audience,
	"lpbs:deployment.manage",
	Resource,
	"-lpbs-deployment-manager",
	"LPBS deployment management",
)

// DeploymentManagerRelease manages LPBS release readiness and delivery
// operations. It is a separate principal from the desktop deployment client.
var DeploymentManagerRelease = principalrole.NewRole(
	Audience,
	"lpbs:release.manage",
	Resource,
	"-lpbs-release-manager",
	"LPBS release management",
)

// CommandCenterAnalytics reads LPBS business analytics without administration
// or credit-mutation privileges.
var CommandCenterAnalytics = principalrole.NewRole(
	Audience,
	"lpbs:analytics.read",
	Resource,
	"-lpbs-analytics-reader",
	"LPBS business analytics",
)

// CommandCenterRemoteProfiles permits the production broker to list and proxy
// through one configured remote profile. It does not permit profile changes.
var CommandCenterRemoteProfiles = principalrole.NewRole(
	Audience,
	"lpbs:remote-profiles.proxy",
	Resource,
	"-lpbs-profile-proxy",
	"LPBS remote profile proxy",
)

// ScenarioToCloudRemoteProfiles reconciles a profile for a local deployment
// consumer. It is deliberately separate from profile proxy access, credit
// grants, and general deployment actions.
var ScenarioToCloudRemoteProfiles = principalrole.NewRole(
	Audience,
	"lpbs:remote-profiles.reconcile",
	Resource,
	"-lpbs-profile-reconciler",
	"LPBS remote profile reconciliation",
)

// LocalAdminRecovery is a local LPBS CLI principal restricted to the two
// account recovery operations that do not have an interactive session path.
var LocalAdminRecovery = principalrole.NewRole(
	Audience,
	"lpbs:admin-recovery.reset",
	Resource,
	"-lpbs-admin-recovery",
	"LPBS local administrator recovery",
)
