package rules

import (
	"log"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	rulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/rules"
	rulesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/rules/rules_v1connect"
	internalfacets "vrooli-memory/internal/facets"
	"vrooli-memory/internal/inference"
	"vrooli-memory/internal/module"
)

func Module(db *database.RoutedDB, logger *log.Logger, classifiers ...inference.Client) module.Module {
	svc := internalfacets.NewService(internalfacets.NewSQLiteRepository(db.Primary()), classifiers...)
	path, handler := rulesconnect.NewClassificationRulesServiceHandler(NewConnectHandler(svc, logger))
	return module.Module{Name: "rules", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

var ProtoFile = rulesv1.File_vrooli_memory_v1_rules_rules_proto
