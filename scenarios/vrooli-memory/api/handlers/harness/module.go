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
	clk := schedule.Clock(schedule.System())
	if len(clocks) > 0 && clocks[0] != nil {
		clk = clocks[0]
	}
	projector := internalharness.NewProjectorWithClock(db.Primary(), client.Recall, clk, roots)
	root := os.Getenv("VROOLI_MEMORY_CLAUDE_ROOT")
	if root == "" {
		if target, ok := projector.Target("claude-code"); ok {
			root = filepath.Dir(target)
		}
	}
	importer := internalharness.NewImporterWithClock(client.Journal, root, projector.TargetPaths(), clk, db.Primary())
	interval, err := maintenance.IntervalFromOS()
	if err != nil {
		panic(err)
	}
	compactLimit, err := maintenance.CompactLimitFromOS()
	if err != nil {
		panic(err)
	}
	maintenanceService := maintenance.NewService(maintenance.NewSQLiteStore(db.Primary()), importer, projector, clk, interval).
		WithLogger(logger).
		WithCompaction(compactor, compactLimit).
		WithEmbeddingRetries(client)
	path, h := harnessconnect.NewHarnessServiceHandler(NewConnectHandler(importer, projector, logger, maintenanceService))
	maintenanceService.Start(context.Background())
	return module.Module{Name: "harness", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}

var ProtoFile = harnessv1.File_vrooli_memory_v1_harness_harness_proto
