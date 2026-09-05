package rules

import (
	"log"

	"vrooli-memory/internal/ledgerclient"
	"vrooli-memory/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	rulesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/rules"
	rulesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/rules/rules_v1connect"
)

func Module(client *ledgerclient.Client, logger *log.Logger) module.Module {
	path, handler := rulesconnect.NewClassificationRulesServiceHandler(NewConnectHandler(client.Rules, logger))
	return module.Module{Name: "rules", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}

var ProtoFile = rulesv1.File_vrooli_memory_v1_rules_rules_proto
