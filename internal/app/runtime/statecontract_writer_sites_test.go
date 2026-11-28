package runtimeapp

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/statecontract"
)

// writerSites identifies functions containing a durable-write operation. The
// inventory is intentionally per function: ownership of one stored type does
// not prove that another writer in the same package writes that type.
func writerSites(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var sites []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			return nil, err
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			function := fn.Name.Name
			if fn.Recv != nil && len(fn.Recv.List) == 1 {
				receiver := fn.Recv.List[0].Type
				start, end := fset.Position(receiver.Pos()).Offset, fset.Position(receiver.End()).Offset
				function = "(" + string(src[start:end]) + ")." + function
			}
			counts := map[string]int{}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				start, end := fset.Position(call.Pos()).Offset, fset.Position(call.End()).Offset
				funEnd := fset.Position(call.Fun.End()).Offset
				name := string(src[start:funEnd])
				if !durableWriter.MatchString(name+"(") && !(name == "os.OpenFile" && durableWriter.Match(src[start:end])) {
					return true
				}
				counts[name]++
				sites = append(sites, fmt.Sprintf("%s:%s:%s#%d", entry.Name(), function, name, counts[name]))
				return true
			})
		}
	}
	sort.Strings(sites)
	for i := 1; i < len(sites); i++ {
		if sites[i] == sites[i-1] {
			return nil, fmt.Errorf("duplicate writer site %s", sites[i])
		}
	}
	return sites, nil
}

func TestWriterSitesFindNewStoreInOwnedPackage(t *testing.T) {
	dir := t.TempDir()
	src := `package fixture
import "os"
type First struct{}
type Second struct{}
func (First) save() { _ = os.WriteFile("first", nil, 0600) }
func (*Second) save() { _ = os.WriteFile("second", nil, 0600) }
func saveOwned() { _ = os.WriteFile("owned", nil, 0600) }
func saveOther() { _ = os.WriteFile("other", nil, 0600); _ = os.WriteFile("third", nil, 0600) }
`
	if err := os.WriteFile(filepath.Join(dir, "state.go"), []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	sites, err := writerSites(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(sites, ",") != "state.go:(*Second).save:os.WriteFile#1,state.go:(First).save:os.WriteFile#1,state.go:saveOther:os.WriteFile#1,state.go:saveOther:os.WriteFile#2,state.go:saveOwned:os.WriteFile#1" {
		t.Fatalf("writer sites: %v", sites)
	}
}

// A malformed or unexpectedly ambiguous source must never silently reuse an
// existing classification. The parser accepts duplicate methods even though
// a later compile rejects them, which exercises the inventory's fail-closed path.
func TestWriterSitesRejectDuplicateIdentity(t *testing.T) {
	dir := t.TempDir()
	src := `package fixture
import "os"
type Owner struct{}
func (Owner) save() { _ = os.WriteFile("one", nil, 0600) }
func (Owner) save() { _ = os.WriteFile("two", nil, 0600) }
`
	if err := os.WriteFile(filepath.Join(dir, "state.go"), []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := writerSites(dir); err == nil || !strings.Contains(err.Error(), "duplicate writer site") {
		t.Fatalf("duplicate site error = %v", err)
	}
}

// auditedWriterSites is an explicit review inventory. Missing entries fail
// rather than assuming that a package's other persistence owners cover them.
var auditedWriterSites = map[string]map[string]string{
	"internal/rootdecision": {"pending.go:Save:config.WriteOwnedFileDurable#1": "pendingRecord root and pending filename fact cover local locator"},
	"internal/capacity": {
		"disk.go:(*SQLiteStore).HeartbeatDisk:s.db.ExecContext#1":                              "updates reservation heartbeat timestamps; capacity schemaSQL, constants and timestamps are contract facts",
		"disk.go:(*SQLiteStore).ReleaseDisk:s.db.ExecContext#1":                                "marks reservation released; capacity schemaSQL, constants and timestamps are contract facts",
		"disk.go:(*SQLiteStore).ReserveDisk:tx.ExecContext#1":                                  "inserts reservation; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).CompleteDiskJob:tx.ExecContext#1":                          "marks job completed and cleanup verified; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).ExtendDiskJob:tx.ExecContext#1":                            "updates path row; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).ExtendDiskJob:tx.ExecContext#2":                            "inserts path row; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).ExtendDiskJob:tx.ExecContext#3":                            "inserts filesystem row; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).ExtendDiskJob:tx.ExecContext#4":                            "updates filesystem row; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).ExtendDiskJob:tx.ExecContext#5":                            "updates job heartbeat row; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).HeartbeatDiskJob:tx.ExecContext#1":                         "updates job heartbeat timestamps; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).ReattachDiskJob:tx.ExecContext#1":                          "updates process identity and capability hash; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).ReconcileDiskJob:tx.ExecContext#1":                         "marks job reconciled and cleanup verified; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).ReserveDiskJob:tx.ExecContext#1":                           "inserts job row; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).ReserveDiskJob:tx.ExecContext#2":                           "inserts filesystem row; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).ReserveDiskJob:tx.ExecContext#3":                           "inserts path row; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).SampleDiskJobPath:tx.ExecContext#1":                        "updates path peak row; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).SampleDiskJobPath:tx.ExecContext#2":                        "updates filesystem peak row; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).SampleDiskJobPath:tx.ExecContext#3":                        "inserts path sample row; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:(*SQLiteStore).SampleDiskJobPath:tx.ExecContext#4":                        "updates job heartbeat row; capacity schemaSQL, constants and timestamps are contract facts",
		"disk_job.go:recordDiskJobGrowth:tx.ExecContext#1":                                     "updates filesystem actual delta; capacity schemaSQL, constants and timestamps are contract facts",
		"placement_repository.go:(*SQLiteStore).RecordPlacementQualification:tx.ExecContext#1": "inserts PlacementQualification JSON and indexed fields; PlacementQualification is a contract root; capacity schemaSQL, constants and timestamps are contract facts",
		"residency_repository.go:(*SQLiteStore).BeginResidencyTransition:tx.ExecContext#1":     "no-op claim update obtains transaction write authority; capacity schemaSQL is a contract fact",
		"residency_repository.go:(*SQLiteStore).BeginResidencyTransition:tx.ExecContext#2":     "locks claim and inserts ResidencyTransitionRecord JSON; capacity schemaSQL, constants and timestamps are contract facts",
		"residency_repository.go:(*SQLiteStore).CompleteResidencyTransition:tx.ExecContext#1":  "updates completed ResidencyTransitionRecord JSON; capacity schemaSQL, constants and timestamps are contract facts",
		"residency_repository.go:(*SQLiteStore).ReconcileResidencyTransition:tx.ExecContext#1": "updates reconciled ResidencyTransitionRecord JSON; capacity schemaSQL, constants and timestamps are contract facts",
		"residency_repository.go:(*SQLiteStore).RecoverResidencyTransition:tx.ExecContext#1":   "updates recovered ResidencyTransitionRecord JSON; capacity schemaSQL, constants and timestamps are contract facts",
		"residency_repository.go:(*SQLiteStore).RecoverResidencyTransition:tx.ExecContext#2":   "updates recovered ResidencyTransitionRecord JSON; capacity schemaSQL, constants and timestamps are contract facts",
		"residency_repository.go:(*SQLiteStore).ReleaseClaimAfterResidency:tx.ExecContext#1":   "updates capacity claim status and generation; capacity schemaSQL, constants and timestamps are contract facts",
		"residency_repository.go:appendResidencyEvent:tx.ExecContext#1":                        "inserts event with ResidencyTransitionRecord JSON; capacity schemaSQL, constants and timestamps are contract facts",
		"schema.go:ensureDiskJobPathSampleColumns:db.ExecContext#1":                            "adds path sample columns present in schemaSQL; capacity schemaSQL, constants and timestamps are contract facts",
		"schema.go:ensureDiskJobPathSampleColumns:db.ExecContext#2":                            "adds path sample columns present in schemaSQL; capacity schemaSQL, constants and timestamps are contract facts",
		"schema.go:ensurePlacementQualificationColumns:db.ExecContext#1":                       "adds qualification claim generation column present in schemaSQL; capacity schemaSQL, constants and timestamps are contract facts",
		"schema.go:ensurePlacementQualificationColumns:db.ExecContext#2":                       "adds qualification claim generation column present in schemaSQL; capacity schemaSQL, constants and timestamps are contract facts",
		"schema.go:(*SQLiteStore).ensureSchema:s.db.ExecContext#1":                             "applies schemaSQL and SQLite user_version; capacity schemaSQL, constants and timestamps are contract facts",
		"schema.go:(*SQLiteStore).ensureSchema:s.db.ExecContext#2":                             "applies schemaSQL and SQLite user_version; capacity schemaSQL, constants and timestamps are contract facts",
		"schema.go:(*SQLiteStore).ensureSchema:s.db.ExecContext#3":                             "applies schemaSQL and SQLite user_version; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).CreateClaim:tx.ExecContext#1":                                "inserts capacity claim including DegradeProfile JSON; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).DegradeClaim:tx.ExecContext#1":                               "updates claim degradation state; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).ExpireStaleClaims:tx.ExecContext#1":                          "updates stale claim state; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).GCTerminalClaims:tx.ExecContext#1":                           "removes terminal claims; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).HeartbeatClaim:tx.ExecContext#1":                             "updates claim heartbeat; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).PreemptClaim:tx.ExecContext#1":                               "updates claim state; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).RecordFootprint:tx.ExecContext#1":                            "upserts footprint high-water mark; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).RecordObserved:tx.ExecContext#1":                             "updates claim observation; capacity schemaSQL, constants and timestamps are contract facts",
		"policy_storage.go:(*SQLiteStore).RecordSweepAt:tx.ExecContext#1":                      "upserts sweep cursor timestamp in policy table; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).ReportActivity:tx.ExecContext#1":                             "updates claim activity and heartbeat; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).ReportActivity:tx.ExecContext#2":                             "updates claim activity and heartbeat; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).ResetFootprints:s.db.ExecContext#1":                          "deletes footprint rows; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).ResizeClaim:tx.ExecContext#1":                                "updates claim amount; capacity schemaSQL, constants and timestamps are contract facts",
		"policy_storage.go:(*SQLiteStore).SetPolicyKey:tx.ExecContext#1":                       "upserts policy key/value; policy_storage.go is whole-source covered; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).UpshiftClaim:tx.ExecContext#1":                               "updates claim rung and amount; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:acquireClaimWriteAuthority:tx.ExecContext#1":                                "no-op update obtains SQLite transaction write authority; capacity schemaSQL, constants and timestamps are contract facts",
		"sqlite.go:(*SQLiteStore).terminate:tx.ExecContext#1":                                  "updates claim terminal state; capacity schemaSQL, constants and timestamps are contract facts",
	},
	"internal/securestore": {
		"backend_selection.go:SelectBackend:config.WriteOwnedFile#1":   "writes BackendSelection contract root",
		"keyringfile.go:writeFileAtomic:config.WriteOwnedFileAtomic#1": "writes repaired keyring; keyringfile.go and values.go are whole-source covered",
		"keyringfile.go:writeKeyringBackup:os.OpenFile#1":              "exclusive copy of original keyring bytes retained for operator recovery",
		"schedule_darwin.go:installNativeCopySchedule:os.WriteFile#1":  "writes systemd or launchd host scheduler definition, not stored control-plane state",
		"schedule_linux.go:installNativeCopySchedule:os.WriteFile#1":   "writes systemd or launchd host scheduler definition, not stored control-plane state",
		"schedule_linux.go:installNativeCopySchedule:os.WriteFile#2":   "writes systemd or launchd host scheduler definition, not stored control-plane state",
		"sealedfile.go:writeSealedFile:config.WriteOwnedFileAtomic#1":  "writes sealedFile root; sealing source is whole-source covered",
		"store_escrow.go:atomicCopy:config.WriteOwnedFileAtomic#1":     "copies sealedFile bytes or CopyStatus receipt root",
		"unlock.go:(fileUnlockCache).Save:os.WriteFile#1":              "writes session unlock cache framing covered by whole-source unlock.go",
	},
	"internal/resources": {
		"blueprint_archive.go:(*Controller).writeBlueprintArchivedResources:os.WriteFile#1":                     "BlueprintArchivedResourceList is a resources root",
		"broker_storage.go:(FileBrokerStore).Save:config.WriteOwnedFileAtomic#1":                                "BrokerState is a resources root",
		"companions.go:appendCompanionLog:os.OpenFile#1":                                                        "append-only companion log; no state reader",
		"companions.go:boundCompanionLog:os.WriteFile#1":                                                        "bounds companion log; no state reader",
		"companions.go:recordCompanionCrashAttempt:os.WriteFile#1":                                              "companion-state whole-source owner covers crash cap marker",
		"companions.go:startCompanionWithNetwork:os.OpenFile#1":                                                 "opens companion log",
		"companions.go:startCompanionWithNetwork:os.WriteFile#1":                                                "companion-state covers PID record",
		"companions.go:startCompanionWithNetwork:os.WriteFile#2":                                                "companion-state covers state record",
		"companions.go:startCompanionWithNetwork:os.WriteFile#3":                                                "companion-state covers PID record",
		"companions.go:startCompanionWithNetwork:os.WriteFile#4":                                                "companion-state covers state record",
		"companions.go:writeCompanionAttempts:os.WriteFile#1":                                                   "companion-state covers persisted timestamp encoding",
		"compose_candidate.go:composeCandidate:os.WriteFile#1":                                                  "copies source resource manifest into candidate snapshot",
		"compose_candidate.go:composeCandidate:os.WriteFile#2":                                                  "ComposeCandidateReceipt is a resources root",
		"compose_candidate.go:snapshotComposeInput:os.OpenFile#1":                                               "copies candidate input bytes, not a control-plane envelope",
		"deprecated.go:copyDir:os.WriteFile#1":                                                                  "copies archived resource files unchanged",
		"deprecated.go:(*Controller).removeResourceConfigEntry:os.WriteFile#1":                                  "edits source-owned service configuration",
		"deprecated.go:writeArchive:os.WriteFile#1":                                                             "copies archive bytes; archive listing is a resources root",
		"deprecated.go:(*Controller).writeDeprecatedResources:os.WriteFile#1":                                   "DeprecatedResourceList is a resources root",
		"deprecated.go:writeJSONMetadata:os.WriteFile#1":                                                        "writes DeprecatedResource metadata or operator-only skipped-path report",
		"drivers_managed_service.go:atomicWriteManagedServiceConfig:os.CreateTemp#1":                            "stages resource-specific config bytes supplied by the caller",
		"drivers_managed_service.go:materializeManagedServiceCredentialFiles:config.WriteOwnedFileAtomic#1":     "materializes credential values for service consumption; credential authority owns their storage",
		"drivers_managed_service.go:runManagedServiceBootstrap:os.WriteFile#1":                                  "temporary bootstrap password consumed by managed service",
		"managed_service_acquisition.go:copyDataArtifact:os.CreateTemp#1":                                       "stages artifact bytes verified against declared checksum",
		"managed_service_acquisition.go:copyManagedComposeSource:os.OpenFile#1":                                 "copies declared compose input bytes",
		"managed_service_acquisition.go:writeDataArtifactChecksum:os.WriteFile#1":                               "checksum sidecar of verified data artifact",
		"managed_service_acquisition.go:writeManagedComposeManifest:os.WriteFile#1":                             "managedComposeManifest is a resources root",
		"managed_service_install_facts.go:writeInstallFacts:os.WriteFile#1":                                     "InstallFacts is a resources root",
		"managed_service_supervisor.go:(*ManagedServiceSupervisor).StartContextWithLauncher:os.OpenFile#1":      "append-only service log",
		"managed_service_supervisor.go:(*ManagedServiceSupervisor).writeState:runtimestorage.WriteFileAtomic#1": "ManagedServiceState is a resources root",
		"model_cache.go:(*modelCacheStore).save:runtimestorage.WriteFileAtomic#1":                               "ModelCacheUsage is a resources root",
		"openrgb_appimage_prepare_windows.go:syncOpenRGBStageDirectory:os.OpenFile#1":                           "opens an existing staged path for sync, no encoded state",
		"openrgb_private_storage.go:writeResourceOwnedFileDurable:config.WriteOwnedFileDurable#1":               "generic writer; callers own serialized state",
		"openrgb_private_storage.go:writeResourceOwnedFileDurableStream:config.WriteOwnedFileDurableStream#1":   "generic stream writer; callers own bytes",
		"openrgb_stage_io_other.go:openRGBWriteStageFile:os.OpenFile#1":                                         "writes caller-supplied artifact bytes into stage",
		"openrgb_stage_io_other.go:prepareOpenRGBStageLock:os.OpenFile#1":                                       "empty synchronization lock",
		"openrgb_tree_io.go:openRGBPrivateTreeIO:os.OpenFile#1":                                                 "generic copied artifact tree bytes",
		"process_limits_linux.go:applyManagedServiceProcessLimits:os.WriteFile#1":                               "writes live process kernel oom_score_adj, not persistent state",
		"qualification_observation.go:recordQualificationObservation:os.CreateTemp#1":                           "resourceQualificationObservation is a resources root",
		"runtime_probe.go:writeRuntimeProbeJSON:os.OpenFile#1":                                                  "caller-requested intent/result forensic outputs, no control-plane reader",
		"scaffold.go:(*Controller).Scaffold:os.WriteFile#1":                                                     "creates source-owned resource manifest",
		"scaffold.go:(*Controller).Scaffold:os.WriteFile#2":                                                     "creates source README",
		"scaffold.go:(*Controller).Scaffold:os.WriteFile#3":                                                     "creates source documentation",
		"scaffold.go:(*Controller).Scaffold:os.WriteFile#4":                                                     "creates source Makefile",
		"scaffold.go:(*Controller).Scaffold:os.WriteFile#5":                                                     "creates source go.mod",
		"scaffold.go:(*Controller).Scaffold:os.WriteFile#6":                                                     "creates source main.go",
		"scaffold.go:(*Controller).Scaffold:os.WriteFile#7":                                                     "updates repository contract",
		"schema_artifacts.go:SyncSchemaArtifacts:os.WriteFile#1":                                                "updates generated repository schema artifact",
		"upstream_liveness.go:CheckUpstream:os.WriteFile#1":                                                     "livenessState is a resources root",
	},

	"internal/peerrecord": {
		"record.go:Write:config.WriteOwnedFileAtomic#1": "Record root and identity.go source cover serialized record and target path",
		"record.go:acquireRecordLock:os.OpenFile#1":     "empty cross-process lock file; no serialized state",
	},
}

func TestPreciseOwnerWriterSitesAreAudited(t *testing.T) {
	root := withRepositoryCodecStamp(t)
	dirs := map[string]bool{}
	for _, inventory := range []map[string][]string{statecontract.PreciseSources, statecontract.ConstantSources} {
		for _, entries := range inventory {
			for _, dir := range entries {
				dirs[dir] = true
			}
		}
	}
	for dir := range dirs {
		sites, err := writerSites(filepath.Join(root, filepath.FromSlash(dir)))
		if err != nil {
			t.Fatal(err)
		}
		for _, site := range sites {
			if auditedWriterSites[dir][site] == "" {
				t.Errorf("unreviewed persistence writer %s:%s", dir, site)
			}
		}
	}
}
