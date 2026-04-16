package domains

import (
	"landing-page-business-suite/cli/domains/account"
	"landing-page-business-suite/cli/domains/admin"
	"landing-page-business-suite/cli/domains/ai"
	"landing-page-business-suite/cli/domains/auth"
	"landing-page-business-suite/cli/domains/billing"
	"landing-page-business-suite/cli/domains/bundles"
	"landing-page-business-suite/cli/domains/content"
	"landing-page-business-suite/cli/domains/coupons"
	"landing-page-business-suite/cli/domains/credits"
	"landing-page-business-suite/cli/domains/docs"
	"landing-page-business-suite/cli/domains/downloads"
	"landing-page-business-suite/cli/domains/feedback"
	"landing-page-business-suite/cli/domains/health"
	"landing-page-business-suite/cli/domains/landing"
	"landing-page-business-suite/cli/domains/metrics"
	"landing-page-business-suite/cli/domains/remoteprofiles"
	"landing-page-business-suite/cli/domains/stripeimport"
	"landing-page-business-suite/cli/domains/users"
	"landing-page-business-suite/cli/domains/variants"
	"landing-page-business-suite/cli/domains/waitlist"
	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(deps support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		health.Register(deps),
		landing.Register(deps),
		auth.Register(deps),
		billing.Register(deps),
		account.Register(deps),
		variants.Register(deps),
		content.Register(deps),
		metrics.Register(deps),
		feedback.Register(deps),
		waitlist.Register(deps),
		credits.Register(deps),
		ai.Register(deps),
		admin.Register(deps),
		remoteprofiles.Register(deps),
		downloads.Register(deps),
		bundles.Register(deps),
		coupons.Register(deps),
		stripeimport.Register(deps),
		users.Register(deps),
		docs.Register(deps),
	}
}
