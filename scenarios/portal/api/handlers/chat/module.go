package chat

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	chatconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/chat/chat_v1connect"

	internalchat "portal/internal/chat"
	"portal/internal/clock"
	"portal/internal/module"
)

func Module(db *database.RoutedDB, clk clock.Clock) module.Module {
	repo := internalchat.NewSQLiteRepository(db, clk)
	service := internalchat.NewService(repo)
	connectPath, connectHandler := chatconnect.NewChatServiceHandler(NewHandler(service))
	return module.Module{
		Name: "chat",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return internalchat.Schema() }
