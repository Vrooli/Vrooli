// Package journal exposes the immutable memory journal over Connect-RPC.
package journal

import (
	"log"

	"vrooli-memory/internal/ledgerclient"
	"vrooli-memory/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal/journal_v1connect"
)

// Module owns the production journal composition. Inference remains behind
// the scenario-owned Client seam, so no provider SDK can leak into the domain.
func Module(client *ledgerclient.Client, logger *log.Logger) module.Module {
	path, handler := journalconnect.NewJournalServiceHandler(NewConnectHandler(client.Journal, logger))
	return module.Module{Name: "journal", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
	}, Endpoints: Endpoints}
}

// ProtoFile is registered for generated-proto/Connect endpoint parity checks.
var ProtoFile = journalv1.File_vrooli_memory_v1_journal_journal_proto
