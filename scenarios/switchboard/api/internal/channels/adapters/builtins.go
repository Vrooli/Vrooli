package adapters

import (
	"switchboard/internal/channels"
	"switchboard/internal/channels/adapters/imessage"
	"switchboard/internal/channels/adapters/inapp"
	"switchboard/internal/channels/adapters/slack"
	"switchboard/internal/channels/adapters/telegram"
)

func init() {
	Register(func() channels.Adapter { return inapp.New() })
	Register(func() channels.Adapter { return telegram.New() })
	Register(func() channels.Adapter { return slack.New() })
	Register(func() channels.Adapter { return imessage.New() })
}
