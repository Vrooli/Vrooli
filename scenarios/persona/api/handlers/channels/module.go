package channels

import (
	"os"
	"strings"

	"persona/internal/channels"
	"persona/internal/journal"
	"persona/internal/module"
	"persona/internal/personas"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	channelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/channels/channels_v1connect"
)

func Module(db *database.RoutedDB, clock schedule.Clock) module.Module {
	personaService := personas.NewService(personas.NewSQLiteRepository(db, clock))
	actionJournal := journal.NewService(journal.NewSQLiteRepository(db, clock))
	adapters := channels.Registry{
		"email":  channels.EmailAdapter{Source: sourceFromEnv("PERSONA_EMAIL_ADAPTER_URL", "email")},
		"sms":    channels.SMSAdapter{Source: sourceFromEnv("PERSONA_SMS_ADAPTER_URL", "sms")},
		"device": channels.DeviceAdapter{Source: sourceFromEnv("PERSONA_DEVICE_ADAPTER_URL", "device")},
	}
	return ModuleWithService(channels.NewService(channels.NewSQLiteRepository(db, clock), personaService, adapters, actionJournal, clock))
}

func sourceFromEnv(key, name string) channels.CodeSource {
	if baseURL := strings.TrimSpace(os.Getenv(key)); baseURL != "" {
		return channels.HTTPSource{BaseURL: baseURL}
	}
	return channels.NewUnavailableSource(name)
}

func ModuleWithService(service channels.Service) module.Module {
	path, handler := channelsconnect.NewChannelsServiceHandler(NewConnectHandler(service))
	return module.Module{Name: "channels", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return channels.Schema() }
