// Package artifacts preserves test-genie's artifact-path vocabulary while the
// importable artifactpaths package owns all location knowledge.
package artifacts

import authority "github.com/vrooli/vrooli/packages/artifactpaths"

const (
	CoverageRoot            = authority.CoverageRoot
	LogsDir                 = authority.LogsDir
	LatestDir               = authority.LatestDir
	RunsDir                 = authority.RunsDir
	RunsIndexFile           = authority.RunsIndexFile
	PhaseResultsSubdir      = authority.PhaseResultsSubdir
	UISmokeSubdir           = authority.UISmokeSubdir
	UISmokePagesSubdir      = authority.UISmokePagesSubdir
	AutomationSubdir        = authority.AutomationSubdir
	UnitSubdir              = authority.UnitSubdir
	SyncDir                 = authority.SyncDir
	ManualValidationsDir    = authority.ManualValidationsDir
	RuntimeDir              = authority.RuntimeDir
	LatestManifestFile      = authority.LatestManifestFile
	FindingsArtifactFile    = authority.FindingsArtifactFile
	RunSnapshotFile         = authority.RunSnapshotFile
	DescriptorSnapshotFile  = authority.DescriptorSnapshotFile
	ArtifactCatalogFile     = authority.ArtifactCatalogFile
	PhaseResultsSmoke       = authority.PhaseResultsSmoke
	PhaseResultsUnit        = authority.PhaseResultsUnit
	PhaseResultsPlaybooks   = authority.PhaseResultsPlaybooks
	PhaseResultsPerformance = authority.PhaseResultsPerformance
	SyncMetadataFile        = authority.SyncMetadataFile
	ManualValidationsLog    = authority.ManualValidationsLog
	VitestRequirementsFile  = authority.VitestRequirementsFile
	SeedStateFile           = authority.SeedStateFile
)

var (
	RunDir                          = authority.RunDir
	TargetRunDir                    = authority.TargetRunDir
	RunPhaseResultsDir              = authority.RunPhaseResultsDir
	RunUISmokeDir                   = authority.RunUISmokeDir
	RunUISmokePagesDir              = authority.RunUISmokePagesDir
	RunAutomationDir                = authority.RunAutomationDir
	RunUnitDir                      = authority.RunUnitDir
	RunFindingsArtifactPath         = authority.RunFindingsArtifactPath
	RunSnapshotPath                 = authority.RunSnapshotPath
	RunDescriptorSnapshotPath       = authority.RunDescriptorSnapshotPath
	RunArtifactCatalogPath          = authority.RunArtifactCatalogPath
	RelativeRunFindingsArtifactPath = authority.RelativeRunFindingsArtifactPath
	RunsIndexPath                   = authority.RunsIndexPath
	PhaseResultsPath                = authority.PhaseResultsPath
	AutomationArtifactPath          = authority.AutomationArtifactPath
	UnitArtifactPath                = authority.UnitArtifactPath
	SyncMetadataPath                = authority.SyncMetadataPath
	ManualValidationsPath           = authority.ManualValidationsPath
	SeedStatePath                   = authority.SeedStatePath
	VitestRequirementsPaths         = authority.VitestRequirementsPaths
	AllCoverageSubdirs              = authority.AllCoverageSubdirs
	RelativePhaseResultsPath        = authority.RelativePhaseResultsPath
	RelativeAutomationArtifactPath  = authority.RelativeAutomationArtifactPath
	RunLogsDir                      = authority.RunLogsDir
	LatestDirPath                   = authority.LatestDirPath
	LatestManifestPath              = authority.LatestManifestPath
)
