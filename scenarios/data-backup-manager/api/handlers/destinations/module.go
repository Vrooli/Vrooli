package destinations

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"data-backup-manager/internal/destinationreadiness"
	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/modulecontract"
	"data-backup-manager/internal/repohealth"
	"data-backup-manager/internal/sysmounts"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/lifecycle"
	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/api-core/schedule"
	"github.com/vrooli/vrooli/packages/credential-authority-go/hubidentity"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	destinationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations/destinations_v1connect"

	internaldestinations "data-backup-manager/internal/destinations"
)

// Module returns the destinations domain's contribution to the API: the
// generated DestinationsService Connect-RPC handler. Production callers use
// this entry point; it constructs the repository → service → handler chain
// internally so per-domain dependencies never appear on server.Deps.
func Module(db *database.RoutedDB, clk schedule.Clock, eng engine.KopiaEngine, protectedRoot string, recoveryStateRoot string, logger *log.Logger) modulecontract.Module {
	return ModuleWithRecoveryDecisions(db, clk, eng, protectedRoot, recoveryStateRoot, logger, nil, "")
}

// ModuleWithRecoveryDecisions wires the optional shared Notification Hub
// decision path. Publisher remains useful for ordinary drive-failure alerts
// even when a signing secret is unavailable; destructive decisions stay
// disabled unless the signed callback can be mounted.
func ModuleWithRecoveryDecisions(db *database.RoutedDB, clk schedule.Clock, eng engine.KopiaEngine, protectedRoot string, recoveryStateRoot string, logger *log.Logger, publisher operatordecision.Publisher, webhookSecret string) modulecontract.Module {
	m, _ := ModuleWithRecoveryWorker(db, clk, eng, protectedRoot, recoveryStateRoot, logger, publisher, webhookSecret)
	return m
}

// ModuleWithRecoveryWorker returns the destinations API module and its
// lifecycle-managed safe recovery monitor. Production starts the worker under
// api-core's maintenance hooks; tests that only need routes can use the wrapper
// above without starting background work.
func ModuleWithRecoveryWorker(db *database.RoutedDB, clk schedule.Clock, eng engine.KopiaEngine, protectedRoot string, recoveryStateRoot string, logger *log.Logger, publisher operatordecision.Publisher, webhookSecret string) (modulecontract.Module, *VolumeRecoveryWorker) {
	m, workers := ModuleWithBackupRecovery(db, clk, eng, protectedRoot, recoveryStateRoot, logger, publisher, webhookSecret, BackupFailureConfig{})
	return m, workers.Volume
}

// BackupFailureConfig enables the backup-failure recovery path: diagnosing a
// destination that stops backups, asking the operator one question, and warning
// when backups stall. A zero value leaves it off.
type BackupFailureConfig struct {
	History        BackupHistory
	Usage          *repohealth.UsageEngine
	Exceptions     repohealth.ExceptionStore
	RepoRepair     engine.RepoDamageRepair
	DecisionEffect operatordecision.DomainEffect
	Migrations     FreshMigrationCoordinator
	// AskReader confirms an operator's answer with the Hub before the resolution
	// webhook applies it. Nil reads as data-backup-manager's own ask-reader.
	AskReader operatordecision.AskReader
}

// Workers are the lifecycle-managed monitors of the destinations domain.
type Workers struct {
	Volume        *VolumeRecoveryWorker
	BackupFailure *BackupFailureWorker
}

// Start starts every configured monitor.
func (w Workers) Start() error {
	if err := w.Volume.Start(); err != nil {
		return err
	}
	return w.BackupFailure.Start()
}

// Stop stops every configured monitor.
func (w Workers) Stop(ctx context.Context) error {
	return errors.Join(w.BackupFailure.Stop(ctx), w.Volume.Stop(ctx))
}

// LifecycleHooks quiesces and resumes every monitor around maintenance.
func (w Workers) LifecycleHooks() lifecycle.Hooks {
	return lifecycle.Hooks{Quiesce: w.Stop, Resume: func(context.Context) error { return w.Start() }}
}

// ModuleWithBackupRecovery returns the destinations API module and its monitors:
// the volume recovery monitor and, when configured, the backup-failure monitor
// that shares its decision, journal and callback machinery.
func ModuleWithBackupRecovery(db *database.RoutedDB, clk schedule.Clock, eng engine.KopiaEngine, protectedRoot string, recoveryStateRoot string, logger *log.Logger, publisher operatordecision.Publisher, webhookSecret string, failureCfg BackupFailureConfig) (modulecontract.Module, Workers) {
	return ModuleWithBackupRecoveryLifecycle(nil, db, clk, eng, protectedRoot, recoveryStateRoot, logger, publisher, webhookSecret, failureCfg)
}

// ModuleWithBackupRecoveryLifecycle binds the optional Events subscription
// reconciler to the owning server lifecycle. Use this entry point in servers;
// the non-lifecycle wrapper is retained for callers that only mount routes.
func ModuleWithBackupRecoveryLifecycle(lifecycle context.Context, db *database.RoutedDB, clk schedule.Clock, eng engine.KopiaEngine, protectedRoot string, recoveryStateRoot string, logger *log.Logger, publisher operatordecision.Publisher, webhookSecret string, failureCfg BackupFailureConfig) (modulecontract.Module, Workers) {
	repo := internaldestinations.NewSQLiteRepository(db, clk)
	recoveryStore := &destinationreadiness.FileRecoveryStore{Root: filepath.Join(recoveryStateRoot, "volume-recovery")}
	readinessSvc := destinationreadiness.NewService(
		destinationreadiness.NewReadOnlyInspector(sysmounts.New()),
		destinationreadiness.NewLocalPreparer(),
	).WithRemediator(destinationreadiness.NewControlPlaneRemediator()).
		WithMountedCheckEvidence(recoveryStore).
		WithWriteProber(destinationreadiness.NewFileWriteProber())
	// The same readiness service both reports to the operator and gates
	// creation, so the advice shown and the rule enforced can never disagree.
	svc := internaldestinations.NewService(
		repo, eng, &internaldestinations.FSBundleWriter{}, protectedRoot,
		internaldestinations.WithReadinessGate(readinessSvc),
		internaldestinations.WithVolumeOperationLease(internaldestinations.NewHostVolumeOperationLease("vrooli")),
	)
	var decisions *RecoveryDecisionFlow
	if publisher != nil {
		decisions = newRecoveryDecisionFlow(
			readinessSvc, recoveryStore,
			&destinationreadiness.FileDecisionStore{Root: filepath.Join(recoveryStateRoot, "volume-recovery-decisions")},
			publisher, strings.TrimSpace(webhookSecret) != "", logger,
		)
	}
	recoveryWorker := newVolumeRecoveryWorker(svc, readinessSvc, recoveryStore, decisions, logger)
	workers := Workers{Volume: recoveryWorker}
	var backupFlow *BackupFailureFlow
	if failureCfg.History != nil && failureCfg.Usage != nil && failureCfg.Exceptions != nil {
		incidents := &repohealth.FileIncidentStore{Root: filepath.Join(recoveryStateRoot, "backup-failure")}
		backupFlow = newBackupFailureFlow(BackupFailureFlowDeps{
			Readiness: readinessSvc, Volume: decisions, Recoveries: recoveryStore,
			Asks:      &destinationreadiness.FileDecisionStore{Root: filepath.Join(recoveryStateRoot, "backup-failure-decisions")},
			Publisher: publisher, Incidents: incidents, Exceptions: failureCfg.Exceptions, RepoRepair: failureCfg.RepoRepair,
			Scanner: failureCfg.Usage.Scanner, Usage: failureCfg.Usage, History: failureCfg.History, Dests: svc, Migrations: failureCfg.Migrations,
			Enabled: publisher != nil && strings.TrimSpace(webhookSecret) != "", Logger: logger,
		})
		workers.BackupFailure = newBackupFailureWorker(BackupFailureWorkerDeps{
			History: failureCfg.History, Dests: svc, Readiness: readinessSvc, Flow: backupFlow, Volume: recoveryWorker,
			Usage: failureCfg.Usage, Exceptions: failureCfg.Exceptions, Incidents: incidents,
			Notices:   &repohealth.NoticeStateStore{Path: filepath.Join(recoveryStateRoot, "backup-notices", "state.json")},
			Publisher: publisher, Recoveries: recoveryStore, Logger: logger,
		})
	}
	connectPath, connectHandler := destinationsconnect.NewDestinationsServiceHandler(NewConnectHandler(Deps{
		Service:       svc,
		Readiness:     readinessSvc,
		RecoveryStore: recoveryStore,
		DecisionFlow:  decisions,
		Logger:        logger,
	}))
	return modulecontract.Module{
		Name: "destinations",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			if decisions != nil && strings.TrimSpace(webhookSecret) != "" {
				effects := []operatordecision.DomainEffect{decisions}
				if backupFlow != nil {
					effects = append(effects, backupFlow)
				}
				if failureCfg.DecisionEffect != nil {
					effects = append(effects, failureCfg.DecisionEffect)
				}
				r.Handle(recoveryDecisionWebhookPath, operatordecision.NewWebhook(hubidentity.WebhookConfig(operatordecision.WebhookConfig{Secret: operatordecision.StaticSecret(webhookSecret), Source: recoveryDecisionSource, Reader: failureCfg.AskReader, Effects: effects}))).Methods("POST")
				if port := strings.TrimSpace(os.Getenv("API_PORT")); port != "" {
					target := "http://127.0.0.1:" + port + recoveryDecisionWebhookPath
					go eventbus.RunSubscriptionReconciler(lifecycle, eventbus.SubscriptionReconcilerConfig{
						Interval: 30 * time.Second, AttemptTimeout: 15 * time.Second, Attempts: 1,
					}, func(ctx context.Context) error { return reconcileRecoveryDecisionSubscription(ctx, target) }, func(err error) {
						if logger != nil {
							logger.Printf("volume recovery decision answer subscription reconciliation failed; retry scheduled: %v", err)
						}
					})
				}
			} else if decisions != nil && logger != nil {
				logger.Printf("volume recovery decision callback disabled: events webhook signing secret unavailable")
			}
		},
		Endpoints: Endpoints,
	}, workers
}

// Schema re-exports the destinations domain SQL so the modules registry
// collects endpoints and schema from one symbol per handler package.
func Schema() string { return internaldestinations.Schema() }
