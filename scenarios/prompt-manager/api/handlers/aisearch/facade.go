// Package aisearch exposes the semantic-search transport boundary.
package aisearch

import domain "prompt-manager/internal/aisearch"

type (
	CollectionDescriptor = domain.CollectionDescriptor
	Service              = domain.Service
	VectorStore          = domain.VectorStore
)

var (
	ErrReconcileBusy              = domain.ErrReconcileBusy
	LoadConfigFromEnv             = domain.LoadConfigFromEnv
	NewActionDescriptor           = domain.NewActionDescriptor
	NewAgentDescriptor            = domain.NewAgentDescriptor
	NewBudgetConfigStore          = domain.NewBudgetConfigStore
	NewDiscoverFilterConfigStore  = domain.NewDiscoverFilterConfigStore
	NewDiscoverRankingConfigStore = domain.NewDiscoverRankingConfigStore
	NewEmbedder                   = domain.NewEmbedder
	NewHandlers                   = domain.NewHandlers
	NewReconciler                 = domain.NewReconciler
	NewService                    = domain.NewService
	NewSkillDescriptor            = domain.NewSkillDescriptor
	NewSyncLoop                   = domain.NewSyncLoop
	NewTeamDescriptor             = domain.NewTeamDescriptor
	NewTopicDescriptor            = domain.NewTopicDescriptor
	NewVectorStoreForRole         = domain.NewVectorStoreForRole
)
