package message

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	messageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/message/message_v1connect"

	"portal/internal/agentchat"
	internalchat "portal/internal/chat"
	"portal/internal/completion"
	"portal/internal/module"
	internalsearch "portal/internal/search"

	"github.com/vrooli/api-core/schedule"
)

func Module(db *database.RoutedDB, clk schedule.Clock, searchService *internalsearch.Service) module.Module {
	repo := internalchat.NewSQLiteRepository(db, clk)
	chatService := internalchat.NewService(repo)
	openRouter, _ := completion.NewOpenRouterStreamerFromEnv()
	completionService := completion.NewService(completion.Config{
		Chat:          chatService,
		OpenRouter:    openRouter,
		SkillResolver: completion.NewPromptManagerSkillResolver(),
		SearchContext: searchService,
	})
	agentManager, _ := agentchat.NewAgentManagerFromEnv()
	agentService := agentchat.NewService(agentchat.Config{
		Chat:         chatService,
		AgentManager: agentManager,
	})
	connectPath, connectHandler := messageconnect.NewMessageServiceHandler(NewHandler(chatService, completionService, agentService, searchService))
	return module.Module{
		Name: "message",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }
