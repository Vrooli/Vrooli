package message

import (
	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	contextcapturehandler "portal/handlers/contextcapture"
	"portal/internal/chatauth"

	messageconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/message/message_v1connect"

	"portal/internal/agentchat"
	internalbrief "portal/internal/brief"
	internalchat "portal/internal/chat"
	"portal/internal/completion"
	"portal/internal/module"
	internalsearch "portal/internal/search"

	"github.com/vrooli/api-core/schedule"
)

func Module(db *database.RoutedDB, clk schedule.Clock, searchService *internalsearch.Service, briefService *internalbrief.Service, contextServices ...contextcapturehandler.Service) module.Module {
	repo := internalchat.NewSQLiteRepository(db, clk)
	chatService := internalchat.NewService(repo)
	openRouter, _ := completion.NewOpenRouterStreamerFromEnv()
	var contextService contextcapturehandler.Service
	if len(contextServices) > 0 {
		contextService = contextServices[0]
	}
	imageResolver := newContextImageResolver(chatService, contextService)
	completionService := completion.NewService(completion.Config{
		Chat:          chatService,
		OpenRouter:    openRouter,
		SkillResolver: completion.NewPromptManagerSkillResolver(),
		Briefs:        briefService,
		BriefStore:    briefService,
		Images:        imageResolver,
	})
	agentManager, _ := agentchat.NewAgentManagerFromEnv()
	agentService := agentchat.NewService(agentchat.Config{
		Chat:         chatService,
		AgentManager: agentManager,
		Runs:         agentchat.NewSQLiteRepository(db),
		Images:       imageResolver,
		Briefs:       briefService,
		BriefStore:   briefService,
	})
	connectPath, connectHandler := messageconnect.NewMessageServiceHandler(NewHandler(chatService, completionService, agentService, searchService, contextService), connect.WithInterceptors(chatauth.FromEnvironment(clk)))
	return module.Module{
		Name: "message",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return agentchat.Schema() }
