package rules

import (
	"log"

	internalfacets "source-ledger/internal/facets"
	"source-ledger/internal/inference"
	"source-ledger/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	rulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/rules"
	rulesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/rules/rules_v1connect"
)

func Module(db *database.RoutedDB, logger *log.Logger, classifiers ...inference.Client) module.Module {
	svc := internalfacets.NewService(internalfacets.NewSQLiteRepository(db.Primary()), classifiers...)
	path, handler := rulesconnect.NewClassificationRulesServiceHandler(NewConnectHandler(svc, logger))
	return module.Module{Name: "rules", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

var ProtoFile = rulesv1.File_source_ledger_v1_rules_rules_proto
