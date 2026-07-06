package studio

import (
	"fmt"
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	apidb "github.com/vrooli/api-core/database"

	"experience-manager/internal/authoring"
	"experience-manager/internal/module"
	"experience-manager/internal/reconcile"

	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
	contractconnect "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract/contract_v1connect"
)

var ProtoFile = contractv1.File_experience_manager_v1_contract_contract_proto

type moduleConfig struct {
	repository authoring.Repository
	evidence   reconcile.EvidenceRepository
}

type Option func(*moduleConfig)

func WithDatabase(db *apidb.RoutedDB) Option {
	return func(cfg *moduleConfig) {
		if db != nil {
			cfg.repository = authoring.NewSQLiteRepository(db)
			cfg.evidence = reconcile.NewSQLiteRepository(db)
		}
	}
}

func Module(logger *log.Logger, repoRoot string, opts ...Option) module.Module {
	var cfg moduleConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	h := &handler{
		logger: logger,
		service: authoring.Service{
			Repo:     cfg.repository,
			Evidence: cfg.evidence,
			RepoRoot: repoRoot,
		},
	}
	path, svc := contractconnect.NewStudioSessionServiceHandler(h)
	return module.Module{
		Name: "studio",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: svc})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return authoring.Schema() }

func requireRepository(repo authoring.Repository) error {
	if repo == nil {
		return fmt.Errorf("authoring repository is not configured")
	}
	return nil
}

func requireEvidenceRepository(repo reconcile.EvidenceRepository) error {
	if repo == nil {
		return fmt.Errorf("reconciliation evidence repository is not configured")
	}
	return nil
}
