package harness

import (
	"context"
	"log"
	"os"
	"path/filepath"

	internalharness "vrooli-memory/internal/harness"
	"vrooli-memory/internal/ledgerclient"
	"vrooli-memory/internal/maintenance"
	"vrooli-memory/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/schedule"
	harnessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/harness"
	harnessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/harness/harness_v1connect"
)

func Module(db *database.RoutedDB, roots *filerouting.RoutedRoots, client *ledgerclient.Client, logger *log.Logger, compactor maintenance.Compactor, clocks ...schedule.Clock) module.Module {
	home, _ := os.UserHomeDir()
	if _, err := internalharness.EnsureCuratedTopology("", home); err != nil {
		panic(err)
	}
	root := os.Getenv("VROOLI_MEMORY_CLAUDE_ROOT")
	if root == "" {
		root = filepath.Join(home, ".claude", "projects", "-home-matthalloran8-Vrooli", "memory")
	}
	projector := internalharness.NewProjector(db.Primary(), client.Recall, roots)
	importer := internalharness.NewImporter(client.Journal, root, projector.TargetPaths(), db.Primary())
	interval, err := maintenance.IntervalFromOS()
	if err != nil {
		panic(err)
	}
	clk := schedule.Clock(schedule.System())
	if len(clocks) > 0 && clocks[0] != nil {
		clk = clocks[0]
	}
	compactLimit, err := maintenance.CompactLimitFromOS()
	if err != nil {
		panic(err)
	}
	maintenanceService := maintenance.NewService(maintenance.NewSQLiteStore(db.Primary()), importer, projector, clk, interval).
		WithCompaction(compactor, compactLimit)
	path, h := harnessconnect.NewHarnessServiceHandler(NewConnectHandler(importer, projector, logger, maintenanceService))
	maintenanceService.Start(context.Background())
	return module.Module{Name: "harness", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}

var ProtoFile = harnessv1.File_vrooli_memory_v1_harness_harness_proto
