/**
 * Scenario Details Page
 *
 * Displays detailed information about a single scenario including:
 * - Scenario metadata (name, description, status, priority, tags)
 * - Editable metadata toggles (greenfield, recommendations enabled)
 * - Deletion with confirmation dialog and archive option
 * - Navigation back to the scenarios list
 *
 * Experience Architecture (Phase 29):
 * - Enhanced breadcrumb navigation shows current location context
 * - Reduces cognitive load for returning users navigating hierarchies
 *
 * [REQ:REQ-P0-007] Scenario Metadata Management UI
 * [REQ:REQ-P0-008] Scenario Deletion Workflow with Safeguards
 */

import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useParams, Link, useNavigate } from "react-router-dom";
import { ChevronRight, Package, Settings2, Circle, CheckCircle2, XCircle, Loader2, Trash2, Terminal, Play, Square, RefreshCw, Files } from "lucide-react";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import { ErrorState } from "../components/ui/error-state";
import { TagList } from "../components/ui/tag-list";
import { FileSelectionDialog, type FileSelectionResult } from "../components/scenarios/file-selection-dialog";
import { capitalize, defaultQueryOptions } from "../lib";
import { scenariosService } from "../services";
import { selectors } from "../consts/selectors";
import { SCENARIO_STATUS_COLORS, SCENARIO_STATUS_ICONS } from "../types";
import type { ScenarioFile, PreserveFilesRequest } from "../types";
import { useScenariosStore } from "../stores";

export function ScenarioDetailsPage() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const upsertScenario = useScenariosStore((state) => state.upsertScenario);
  const removeScenario = useScenariosStore((state) => state.removeScenario);

  // Local state for optimistic UI updates
  const [localGreenfield, setLocalGreenfield] = useState<boolean | null>(null);
  const [localRecommendations, setLocalRecommendations] = useState<boolean | null>(null);

  // Delete dialog state
  // [REQ:REQ-P0-008] Delete confirmation dialog state
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [archiveOnDelete, setArchiveOnDelete] = useState(true);

  // File selection dialog state
  const [showFileSelectionDialog, setShowFileSelectionDialog] = useState(false);
  const [fileSelection, setFileSelection] = useState<PreserveFilesRequest | null>(null);
  const [scenarioFiles, setScenarioFiles] = useState<ScenarioFile[]>([]);
  const [filesLoading, setFilesLoading] = useState(false);

  // Fetch scenario details
  // Note: queryFn is only called when enabled is true (i.e., name exists)
  const {
    data: scenario,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: ["scenarios", name],
    queryFn: () => {
      if (!name) throw new Error("Name is required");
      return scenariosService.get(name);
    },
    enabled: !!name,
    ...defaultQueryOptions,
  });

  // Sync local state when scenario data loads
  useEffect(() => {
    if (scenario) {
      setLocalGreenfield(scenario.isGreenfield);
      setLocalRecommendations(scenario.recommendationsEnabled);
    }
  }, [scenario]);

  // Update metadata mutation
  // [REQ:REQ-P0-007] PATCH endpoint for toggling metadata
  const updateMutation = useMutation({
    mutationFn: (updates: { isGreenfield?: boolean; recommendationsEnabled?: boolean }) => {
      if (!name) throw new Error("Name is required");
      return scenariosService.updateMetadata(name, updates);
    },
    onSuccess: (updatedScenario) => {
      // Update cache with new data
      queryClient.setQueryData(["scenarios", name], updatedScenario);
      upsertScenario(updatedScenario);
    },
    onError: () => {
      // Revert local state on error
      if (scenario) {
        setLocalGreenfield(scenario.isGreenfield);
        setLocalRecommendations(scenario.recommendationsEnabled);
      }
    },
  });

  const actionMutation = useMutation({
    mutationFn: async (action: "start" | "stop" | "restart") => {
      if (!name) throw new Error("Name is required");
      if (action === "start") {
        return scenariosService.start(name);
      }
      if (action === "stop") {
        return scenariosService.stop(name);
      }
      return scenariosService.restart(name);
    },
    onSuccess: (updatedScenario) => {
      queryClient.setQueryData(["scenarios", name], updatedScenario);
      upsertScenario(updatedScenario);
    },
  });

  // Toggle handlers with optimistic updates
  const handleGreenfieldToggle = () => {
    const newValue = !localGreenfield;
    setLocalGreenfield(newValue);
    updateMutation.mutate({ isGreenfield: newValue });
  };

  const handleRecommendationsToggle = () => {
    const newValue = !localRecommendations;
    setLocalRecommendations(newValue);
    updateMutation.mutate({ recommendationsEnabled: newValue });
  };

  // Delete mutation
  // [REQ:REQ-P0-008] DELETE endpoint for scenario deletion with archive option
  const deleteMutation = useMutation({
    mutationFn: () => {
      if (!name) throw new Error("Name is required");
      return scenariosService.delete(name, {
        archive: archiveOnDelete,
        preserveFiles: archiveOnDelete && fileSelection ? fileSelection : undefined,
      });
    },
    onSuccess: () => {
      if (name) {
        removeScenario(name);
      }
      // Navigate back to scenarios list
      navigate("/scenarios");
    },
  });

  // Delete handlers
  const handleDeleteClick = () => {
    setShowDeleteDialog(true);
  };

  const handleDeleteConfirm = () => {
    deleteMutation.mutate();
  };

  const handleDeleteCancel = () => {
    setShowDeleteDialog(false);
    setArchiveOnDelete(true); // Reset to default
    setFileSelection(null); // Reset file selection
  };

  // Load scenario files for file selection dialog
  const loadScenarioFiles = async () => {
    if (!name) return;
    setFilesLoading(true);
    try {
      const files = await scenariosService.getFiles(name);
      setScenarioFiles(files);
    } catch (error) {
      console.error("Failed to load scenario files:", error);
      setScenarioFiles([]);
    } finally {
      setFilesLoading(false);
    }
  };

  const handleCustomizeFilesClick = () => {
    loadScenarioFiles();
    setShowFileSelectionDialog(true);
  };

  const handleFileSelectionConfirm = (selection: FileSelectionResult) => {
    setFileSelection({
      preset: selection.preset,
      paths: selection.paths,
    });
    setShowFileSelectionDialog(false);
  };

  const handleFileSelectionCancel = () => {
    setShowFileSelectionDialog(false);
  };

  if (!name) {
    return (
      <div className="space-y-6" data-testid={selectors.scenarioDetails.page}>
        <ErrorState
          error={new Error("Scenario name is required")}
          title="Invalid URL"
        />
      </div>
    );
  }

  // Get status icon
  const StatusIcon = scenario ? (SCENARIO_STATUS_ICONS[scenario.status] || Circle) : Circle;
  const isRunning = scenario?.status === "running";
  const isStopped = scenario?.status === "stopped";
  const actionInFlight = actionMutation.isPending ? actionMutation.variables : null;
  const actionError = actionMutation.isError
    ? `Failed to ${actionMutation.variables ?? "run action"}. Please try again.`
    : null;

  return (
    <div className="space-y-6" data-testid={selectors.scenarioDetails.page}>
      {/* Breadcrumb navigation - shows context and allows quick navigation (Phase 29) */}
      <nav className="flex items-center gap-2 text-sm" data-testid={selectors.scenarioDetails.breadcrumb}>
        <Link
          to="/scenarios"
          className="flex items-center gap-1.5 text-slate-400 hover:text-slate-200 transition-colors"
          data-testid={selectors.scenarioDetails.backButton}
        >
          <Package className="h-4 w-4" />
          <span>Scenarios</span>
        </Link>
        <ChevronRight className="h-4 w-4 text-slate-600" />
        <span className="text-slate-200 truncate max-w-[200px]" title={scenario?.displayName || name}>
          {scenario?.displayName || name}
        </span>
      </nav>

      {/* Loading state */}
      {isLoading && (
        <Card padding="lg" centered>
          <p className="text-slate-400">Loading scenario details...</p>
        </Card>
      )}

      {/* Error state */}
      {error && (
        <ErrorState
          error={error}
          title="Unable to load scenario"
          onRetry={() => refetch()}
        />
      )}

      {/* Scenario details */}
      {scenario && !error && (
        <>
          {/* Scenario header */}
          <Card data-testid={selectors.scenarioDetails.header}>
            <div className="flex items-start justify-between">
              <div className="space-y-2">
                <div className="flex items-center gap-3">
                  <StatusIcon
                    className={`h-4 w-4 ${SCENARIO_STATUS_COLORS[scenario.status]}`}
                    data-testid={selectors.scenarioDetails.status}
                  />
                  <span className="text-sm uppercase tracking-wider text-slate-500">
                    {capitalize(scenario.status)}
                  </span>
                  <span
                    className="rounded-full bg-slate-700 px-3 py-1 text-sm text-slate-300"
                    data-testid={selectors.scenarioDetails.priority}
                  >
                    P{scenario.priority}
                  </span>
                  {localGreenfield && (
                    <span className="rounded-full bg-cyan-500/20 px-2 py-0.5 text-xs text-cyan-400">
                      Greenfield
                    </span>
                  )}
                </div>
                <h1
                  className="text-2xl font-bold text-slate-100"
                  data-testid={selectors.scenarioDetails.title}
                >
                  {scenario.displayName}
                </h1>
                <p
                  className="text-slate-400"
                  data-testid={selectors.scenarioDetails.description}
                >
                  {scenario.description || "No description provided"}
                </p>
                <TagList
                  tags={scenario.tags}
                  maxTags={10}
                  className="mt-4"
                  data-testid={selectors.scenarioDetails.tags}
                />
              </div>

              <div className="flex flex-col items-end gap-2">
                <Package className="h-12 w-12 text-slate-600" />
                {scenario.completenessScore !== undefined && (
                  <div className="flex items-center gap-2">
                    <div className="h-2 w-24 overflow-hidden rounded-full bg-slate-700">
                      <div
                        className="h-full bg-gradient-to-r from-cyan-500 to-purple-500"
                        style={{ width: `${scenario.completenessScore}%` }}
                      />
                    </div>
                    <span className="text-sm text-slate-400">{scenario.completenessScore}%</span>
                  </div>
                )}
                <div className="flex flex-wrap items-center justify-end gap-2" data-testid={selectors.scenarioDetails.actionsSection}>
                  <Button
                    variant={isRunning ? "outline" : "default"}
                    size="sm"
                    onClick={() => actionMutation.mutate("start")}
                    disabled={actionMutation.isPending || isRunning}
                    data-testid={selectors.scenarioDetails.startButton}
                  >
                    {actionInFlight === "start" ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <Play className="mr-2 h-4 w-4" />
                    )}
                    Start
                  </Button>
                  <Button
                    variant={isStopped ? "outline" : "default"}
                    size="sm"
                    onClick={() => actionMutation.mutate("stop")}
                    disabled={actionMutation.isPending || isStopped}
                    data-testid={selectors.scenarioDetails.stopButton}
                  >
                    {actionInFlight === "stop" ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <Square className="mr-2 h-4 w-4" />
                    )}
                    Stop
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => actionMutation.mutate("restart")}
                    disabled={actionMutation.isPending}
                    data-testid={selectors.scenarioDetails.restartButton}
                  >
                    {actionInFlight === "restart" ? (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    ) : (
                      <RefreshCw className="mr-2 h-4 w-4" />
                    )}
                    Restart
                  </Button>
                </div>
                {actionError && (
                  <div
                    className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-1 text-xs text-red-400"
                    data-testid={selectors.scenarioDetails.actionError}
                  >
                    {actionError}
                  </div>
                )}
              </div>
            </div>
          </Card>

          {/* Metadata management section */}
          <Card data-testid={selectors.scenarioDetails.metadataSection}>
            <div className="flex items-center gap-2 mb-4">
              <Settings2 className="h-5 w-5 text-slate-400" />
              <h2 className="text-lg font-medium text-slate-200">Scenario Settings</h2>
              {updateMutation.isPending && (
                <Loader2 className="h-4 w-4 animate-spin text-cyan-400 ml-2" />
              )}
            </div>

            <div className="space-y-4">
              {/* Greenfield toggle */}
              <div className="flex items-center justify-between p-4 rounded-lg bg-slate-700/30">
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-slate-200">Greenfield Mode</span>
                    {localGreenfield ? (
                      <CheckCircle2 className="h-4 w-4 text-cyan-400" />
                    ) : (
                      <XCircle className="h-4 w-4 text-slate-500" />
                    )}
                  </div>
                  <p className="text-sm text-slate-400">
                    Treat this scenario as a new project without existing code base
                  </p>
                </div>
                <Button
                  variant={localGreenfield ? "default" : "outline"}
                  size="sm"
                  onClick={handleGreenfieldToggle}
                  disabled={updateMutation.isPending}
                  data-testid={selectors.scenarioDetails.greenfieldToggle}
                >
                  {localGreenfield ? "Enabled" : "Disabled"}
                </Button>
              </div>

              {/* Recommendations toggle */}
              <div className="flex items-center justify-between p-4 rounded-lg bg-slate-700/30">
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-slate-200">Recommendations</span>
                    {localRecommendations ? (
                      <CheckCircle2 className="h-4 w-4 text-cyan-400" />
                    ) : (
                      <XCircle className="h-4 w-4 text-slate-500" />
                    )}
                  </div>
                  <p className="text-sm text-slate-400">
                    Allow the recommendation engine to suggest improvements for this scenario
                  </p>
                </div>
                <Button
                  variant={localRecommendations ? "default" : "outline"}
                  size="sm"
                  onClick={handleRecommendationsToggle}
                  disabled={updateMutation.isPending}
                  data-testid={selectors.scenarioDetails.recommendationsToggle}
                >
                  {localRecommendations ? "Enabled" : "Disabled"}
                </Button>
              </div>
            </div>

            {/* Error feedback */}
            {updateMutation.isError && (
              <div className="mt-4 p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
                Failed to update settings. Please try again.
              </div>
            )}
          </Card>

          {/* CLI Quick Actions - Helps ops users access common operations (Phase 29 Iteration 5) */}
          <Card data-testid={selectors.scenarioDetails.cliHint}>
            <div className="flex items-center gap-2 mb-3">
              <Terminal className="h-5 w-5 text-slate-400" />
              <h2 className="text-lg font-medium text-slate-200">Quick Actions (CLI)</h2>
            </div>
            <p className="text-sm text-slate-400 mb-4">
              Common operations for this scenario are also available via the command line.
            </p>
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="rounded-lg bg-slate-700/30 p-3">
                <span className="text-xs font-medium text-slate-300">View Logs</span>
                <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
                  vrooli scenario logs {name}
                </code>
              </div>
              <div className="rounded-lg bg-slate-700/30 p-3">
                <span className="text-xs font-medium text-slate-300">Check Status</span>
                <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
                  vrooli scenario status {name}
                </code>
              </div>
              <div className="rounded-lg bg-slate-700/30 p-3">
                <span className="text-xs font-medium text-slate-300">Run Tests</span>
                <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
                  vrooli scenario test {name}
                </code>
              </div>
              <div className="rounded-lg bg-slate-700/30 p-3">
                <span className="text-xs font-medium text-slate-300">Restart Scenario</span>
                <code className="mt-1 block rounded bg-slate-800 px-2 py-1.5 font-mono text-xs text-cyan-400">
                  vrooli scenario restart {name}
                </code>
              </div>
            </div>
          </Card>

          {/* Danger zone - Delete scenario */}
          {/* [REQ:REQ-P0-008] Scenario deletion with safeguards */}
          <div className="rounded-xl border border-red-500/20 bg-slate-800/30 p-6">
            <div className="flex items-center gap-2 mb-4">
              <Trash2 className="h-5 w-5 text-red-400" />
              <h2 className="text-lg font-medium text-red-400">Danger Zone</h2>
            </div>

            <div className="flex items-center justify-between p-4 rounded-lg bg-slate-700/30">
              <div className="space-y-1">
                <span className="font-medium text-slate-200">Delete Scenario</span>
                <p className="text-sm text-slate-400">
                  Permanently remove this scenario from the catalog. This action cannot be undone.
                </p>
              </div>
              <Button
                variant="destructive"
                size="sm"
                onClick={handleDeleteClick}
                disabled={deleteMutation.isPending}
                data-testid={selectors.scenarioDetails.deleteButton}
              >
                {deleteMutation.isPending ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Deleting...
                  </>
                ) : (
                  <>
                    <Trash2 className="mr-2 h-4 w-4" />
                    Delete
                  </>
                )}
              </Button>
            </div>

            {/* Delete error feedback */}
            {deleteMutation.isError && (
              <div className="mt-4 p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
                Failed to delete scenario. Please try again.
              </div>
            )}
          </div>
        </>
      )}

      {/* Delete confirmation dialog */}
      {/* [REQ:REQ-P0-008] Strong confirmation dialog with archive option */}
      <ConfirmDialog
        isOpen={showDeleteDialog}
        onClose={handleDeleteCancel}
        onConfirm={handleDeleteConfirm}
        title="Delete Scenario"
        description={`Are you sure you want to delete "${scenario?.displayName || name}"? This will remove the scenario from the catalog.`}
        confirmationText={scenario?.name}
        confirmLabel="Delete Scenario"
        isLoading={deleteMutation.isPending}
        checkboxContent={{
          label: "Archive to backlog (idea) before deleting (recommended)",
          checked: archiveOnDelete,
          onChange: (checked) => {
            setArchiveOnDelete(checked);
            if (!checked) setFileSelection(null);
          },
          testId: selectors.scenarioDetails.archiveCheckbox,
        }}
        testIds={{
          dialog: selectors.scenarioDetails.deleteDialog,
          confirmButton: selectors.scenarioDetails.deleteConfirmButton,
          cancelButton: selectors.scenarioDetails.deleteCancelButton,
        }}
      />

      {/* "Customize files" link shown below delete dialog when archive is checked */}
      {showDeleteDialog && archiveOnDelete && (
        <div className="fixed inset-0 z-[60] flex items-center justify-center pointer-events-none">
          <div className="pointer-events-auto relative top-[180px] w-full max-w-md">
            <div className="rounded-lg border border-white/10 bg-slate-800/95 p-3 shadow-xl">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Files className="h-4 w-4 text-cyan-400" />
                  <span className="text-sm text-slate-300">
                    {fileSelection
                      ? `${fileSelection.paths?.length || 0} files selected to preserve`
                      : "Preserve files with the archive?"}
                  </span>
                </div>
                <button
                  type="button"
                  onClick={handleCustomizeFilesClick}
                  className="text-sm text-cyan-400 hover:text-cyan-300 underline underline-offset-2"
                  data-testid="customize-files-link"
                >
                  {fileSelection ? "Edit selection" : "Customize files..."}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* File selection dialog */}
      <FileSelectionDialog
        isOpen={showFileSelectionDialog}
        onClose={handleFileSelectionCancel}
        onConfirm={handleFileSelectionConfirm}
        scenarioName={scenario?.displayName || name || ""}
        files={scenarioFiles}
        isLoading={filesLoading}
      />
    </div>
  );
}
