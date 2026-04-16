package domains

import (
	"notification-hub/cli/domains/analytics"
	"notification-hub/cli/domains/config"
	"notification-hub/cli/domains/contacts"
	"notification-hub/cli/domains/notifications"
	"notification-hub/cli/domains/profiles"
	"notification-hub/cli/domains/templates"
	"notification-hub/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(d support.Dependencies) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		config.Register(d),
	}
}

func SubcommandGroups(d support.Dependencies) []cliapp.SubcommandGroup {
	return []cliapp.SubcommandGroup{
		profiles.Register(d),
		contacts.Register(d),
		templates.Register(d),
		notifications.Register(d),
		analytics.Register(d),
	}
}
