/**
 * Scenario Details Page
 *
 * Displays detailed information about a single scenario including:
 * - Scenario metadata (name, description, status, priority, tags)
 * - Editable metadata toggle (greenfield)
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

import { useState, useEffect, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useParams, Link, useNavigate } from "react-router-dom";
import {
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  Circle,
  CheckCircle2,
  Loader2,
  MoreHorizontal,
  Package,
  Play,
  RefreshCw,
  Settings2,
  Square,
  Terminal,
  Trash2,
  XCircle,
} from "lucide-react";
import { BottomSheet } from "../components/ui/bottom-sheet";
import { Button } from "../components/ui/button";
import { Card } from "../components/ui/card";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import { ErrorState } from "../components/ui/error-state";
import { InlineLoadingIndicator, PageLoadingState } from "../components/ui/loading-states";
import { Select } from "../components/ui/select";
import { TagList } from "../components/ui/tag-list";
import { FileSelectionDialog, type FileSelectionResult } from "../components/scenarios/file-selection-dialog";
import { capitalize, defaultQueryOptions } from "../lib";
import { scenariosService } from "../services";
import { selectors } from "../consts/selectors";
import { SCENARIO_STATUS_COLORS, SCENARIO_STATUS_ICONS } from "../types";
import type { ScenarioFile, PreserveFilesPreset } from "../types";
import { useScenariosStore } from "../stores";

const ARCHIVE_PRESET_OPTIONS: { value: PreserveFilesPreset; label: string; description: string }[] = [
  { value: "documentation", label: "Documentation", description: "PRD.md, README.md, docs/**, *.md" },
  { value: "requirements", label: "Requirements", description: "PRD.md, requirements/**, specs/**, REQUIREMENTS.md" },
  { value: "planning", label: "Planning", description: "PRD.md, .vrooli/**, planning/**, design/**" },
  {
    value: "all-planning",
    label: "All Planning Files",
    description: "All docs, requirements, specs, planning, design, and markdown files",
  },
];

const ARCHIVE_PREFERENCES_STORAGE_KEY = "swarm-manager.archive.preferences.v1";

interface ArchivePreferences {
  mode: "preset" | "custom";
  preset: PreserveFilesPreset;
  customPaths: string[];
}

const ARCHIVE_PRESET_PATTERNS: Record<PreserveFilesPreset, string[]> = {
  documentation: ["PRD.md", "README.md", "docs/**", "*.md"],
  requirements: ["PRD.md", "requirements/**", "specs/**", "REQUIREMENTS.md"],
  planning: ["PRD.md", ".vrooli/**", "planning/**", "design/**"],
  "all-planning": [
    "PRD.md",
    "README.md",
    "docs/**",
    "requirements/**",
    "specs/**",
    "planning/**",
    "design/**",
    ".vrooli/**",
    "*.md",
  ],
};

const ARCHIVE_IGNORED_DIRS = new Set([
  "node_modules",
  ".git",
  "dist",
  "build",
  "coverage",
  ".next",
  ".turbo",
  "target",
  "vendor",
]);

function isIgnoredArchivePath(path: string): boolean {
  return path.split("/").some((segment) => ARCHIVE_IGNORED_DIRS.has(segment));
}

function matchesPattern(path: string, pattern: string): boolean {
  if (!pattern.includes("*")) {
    return path === pattern || path.endsWith("/" + pattern);
  }
  if (pattern.includes("**")) {
    const prefix = pattern.split("**")[0];
    return !prefix || path.startsWith(prefix);
  }
  if (pattern.startsWith("*.")) {
    return path.endsWith(pattern.slice(1));
  }
  if (pattern.endsWith("/*")) {
    const prefix = pattern.slice(0, -2);
    return path.startsWith(prefix + "/") && !path.slice(prefix.length + 1).includes("/");
  }
  return false;
}

function collectPaths(file: ScenarioFile): string[] {
  if (file.type === "file") return [file.path];
  if (!file.children) return [];
  return file.children.flatMap(collectPaths);
}

function getDefaultArchivePreferences(): ArchivePreferences {
  return {
    mode: "preset",
    preset: "planning",
    customPaths: [],
  };
}

function isValidPreset(value: string): value is PreserveFilesPreset {
  return value in ARCHIVE_PRESET_PATTERNS;
}

function loadArchivePreferences(): ArchivePreferences {
  if (typeof window === "undefined") {
    return getDefaultArchivePreferences();
  }
  try {
    const raw = window.localStorage.getItem(ARCHIVE_PREFERENCES_STORAGE_KEY);
    if (!raw) return getDefaultArchivePreferences();
    const parsed = JSON.parse(raw) as Partial<ArchivePreferences>;
    const preset = parsed.preset && isValidPreset(parsed.preset) ? parsed.preset : "planning";
    const mode = parsed.mode === "custom" ? "custom" : "preset";
    const customPaths = Array.isArray(parsed.customPaths)
      ? parsed.customPaths.filter((path): path is string => typeof path === "string" && path.length > 0)
      : [];
    return { mode, preset, customPaths };
  } catch {
    return getDefaultArchivePreferences();
  }
}

function persistArchivePreferences(preferences: ArchivePreferences): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(ARCHIVE_PREFERENCES_STORAGE_KEY, JSON.stringify(preferences));
}

export function ScenarioDetailsPage() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const upsertScenario = useScenariosStore((state) => state.upsertScenario);
  const removeScenario = useScenariosStore((state) => state.removeScenario);

  // Local state for optimistic UI updates
  const [localGreenfield, setLocalGreenfield] = useState<boolean | null>(null);

  // Delete dialog state
  // [REQ:REQ-P0-008] Delete confirmation dialog state
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [archiveOnDelete, setArchiveOnDelete] = useState(true);
  const [archivePreset, setArchivePreset] = useState<PreserveFilesPreset>(() => loadArchivePreferences().preset);
  const [archiveMode, setArchiveMode] = useState<"preset" | "custom">(() => loadArchivePreferences().mode);
  const [customPaths, setCustomPaths] = useState<string[]>(() => loadArchivePreferences().customPaths);
  const [showActionsSheet, setShowActionsSheet] = useState(false);
  const [mobileDangerExpanded, setMobileDangerExpanded] = useState(false);

  // File selection dialog state
  const [showFileSelectionDialog, setShowFileSelectionDialog] = useState(false);
  const [scenarioFiles, setScenarioFiles] = useState<ScenarioFile[]>([]);
  const [filesLoading, setFilesLoading] = useState(false);
  const [fileTreeLoaded, setFileTreeLoaded] = useState(false);

  useEffect(() => {
    persistArchivePreferences({
      mode: archiveMode,
      preset: archivePreset,
      customPaths,
    });
  }, [archiveMode, archivePreset, customPaths]);

  useEffect(() => {
    setShowActionsSheet(false);
    setMobileDangerExpanded(false);
  }, [name]);

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
    }
  }, [scenario]);

  // Update metadata mutation
  // [REQ:REQ-P0-007] PATCH endpoint for toggling metadata
  const updateMutation = useMutation({
    mutationFn: (updates: { isGreenfield?: boolean }) => {
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

  // Delete mutation
  // [REQ:REQ-P0-008] DELETE endpoint for scenario deletion with archive option
  const deleteMutation = useMutation({
    mutationFn: () => {
      if (!name) throw new Error("Name is required");
      const preserveFiles = archiveOnDelete
        ? effectiveArchiveMode === "custom"
          ? { paths: customPaths }
          : { preset: archivePreset }
        : undefined;
      return scenariosService.delete(name, {
        archive: archiveOnDelete,
        preserveFiles,
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
    setFileTreeLoaded(false);
    void loadScenarioFiles();
  };

  const handleDeleteConfirm = () => {
    deleteMutation.mutate();
  };

  const handleDeleteCancel = () => {
    setShowDeleteDialog(false);
    setArchiveOnDelete(true); // Reset to default
  };

  // Load scenario files for file selection dialog
  const loadScenarioFiles = async () => {
    if (!name) return;
    setFileTreeLoaded(false);
    setFilesLoading(true);
    try {
      const files = await scenariosService.getFiles(name);
      setScenarioFiles(Array.isArray(files) ? files : []);
    } catch (error) {
      console.error("Failed to load scenario files:", error);
      setScenarioFiles([]);
    } finally {
      setFileTreeLoaded(true);
      setFilesLoading(false);
    }
  };

  const handleCustomizeFilesClick = () => {
    loadScenarioFiles();
    setShowFileSelectionDialog(true);
  };

  const handleFileSelectionConfirm = (selection: FileSelectionResult) => {
    setArchiveMode(selection.preset ? "preset" : "custom");
    if (selection.preset) {
      setArchivePreset(selection.preset);
      setCustomPaths([]);
    } else {
      setCustomPaths(selection.paths);
    }
    setShowFileSelectionDialog(false);
  };

  const handleFileSelectionCancel = () => {
    setShowFileSelectionDialog(false);
  };

  // Get status icon
  const StatusIcon = scenario ? (SCENARIO_STATUS_ICONS[scenario.status] || Circle) : Circle;
  const isPageLoading = isLoading && !scenario;
  const isRunning = scenario?.status === "running";
  const isStopped = scenario?.status === "stopped";
  const actionInFlight = actionMutation.isPending ? actionMutation.variables : null;
  const actionError = actionMutation.isError
    ? `Failed to ${actionMutation.variables ?? "run action"}. Please try again.`
    : null;
  const allScenarioFilePaths = useMemo(
    () => scenarioFiles.flatMap(collectPaths).sort((a, b) => a.localeCompare(b)),
    [scenarioFiles]
  );
  const scenarioPathSet = useMemo(() => new Set(allScenarioFilePaths), [allScenarioFilePaths]);
  const canUseCustomSelection = useMemo(() => {
    if (archiveMode !== "custom" || customPaths.length === 0) return false;
    if (!fileTreeLoaded) return true;
    return customPaths.every((path) => scenarioPathSet.has(path));
  }, [archiveMode, customPaths, fileTreeLoaded, scenarioPathSet]);
  const effectiveArchiveMode = archiveMode === "custom" && canUseCustomSelection ? "custom" : "preset";
  const previewPaths = useMemo(() => {
    if (!archiveOnDelete) return [];
    if (effectiveArchiveMode === "custom") return [...customPaths].sort((a, b) => a.localeCompare(b));
    const patterns = ARCHIVE_PRESET_PATTERNS[archivePreset];
    return allScenarioFilePaths.filter(
      (path) => !isIgnoredArchivePath(path) && patterns.some((pattern) => matchesPattern(path, pattern))
    );
  }, [archiveOnDelete, effectiveArchiveMode, customPaths, archivePreset, allScenarioFilePaths]);
  const previewList = previewPaths.slice(0, 10);

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

  const renderActionButtons = (closeOnAction = false) => {
    const runAction = (action: () => void) => {
      if (closeOnAction) {
        setShowActionsSheet(false);
      }
      action();
    };
    const rowButtonClass =
      "h-10 w-full justify-start rounded-lg border-slate-700/80 bg-slate-900/40 px-3 text-sm text-slate-100 hover:bg-slate-800/70";

    return (
      <div className="space-y-2">
        <Button
          variant={isRunning ? "outline" : "default"}
          size="sm"
          className={rowButtonClass}
          onClick={() => runAction(() => actionMutation.mutate("start"))}
          disabled={actionMutation.isPending || isRunning}
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
          className={rowButtonClass}
          onClick={() => runAction(() => actionMutation.mutate("stop"))}
          disabled={actionMutation.isPending || isStopped}
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
          className={rowButtonClass}
          onClick={() => runAction(() => actionMutation.mutate("restart"))}
          disabled={actionMutation.isPending}
        >
          {actionInFlight === "restart" ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="mr-2 h-4 w-4" />
          )}
          Restart
        </Button>
      </div>
    );
  };

  return (
    <div className="space-y-0 lg:space-y-6" data-testid={selectors.scenarioDetails.page}>
      {/* Breadcrumb navigation - shows context and allows quick navigation (Phase 29) */}
      <nav className="hidden items-center gap-2 text-sm lg:flex" data-testid={selectors.scenarioDetails.breadcrumb}>
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
      {isPageLoading && (
        <PageLoadingState
          label="Loading scenario details..."
          variant="detail"
          testId="scenario-details-loading-state"
        />
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
          <div className="flex min-h-[100dvh] flex-col lg:hidden">
            <div className="sticky top-0 z-30 flex items-center gap-2 border-b border-slate-800 bg-slate-950/95 px-3 py-2 backdrop-blur">
              <Button
                asChild
                variant="outline"
                size="sm"
                className="h-9 w-9 rounded-md border-transparent bg-transparent p-0 hover:bg-slate-800/70"
              >
                <Link to="/scenarios" aria-label="Back to scenarios">
                  <ChevronLeft className="h-4 w-4" />
                </Link>
              </Button>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-semibold text-slate-100">{scenario.displayName}</p>
                <p className="truncate text-xs text-slate-400">
                  {capitalize(scenario.status)} · P{scenario.priority}
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                className="h-9 rounded-lg border-slate-700/80 bg-slate-900/45 px-3 text-xs font-medium text-slate-100 hover:bg-slate-800/70"
                onClick={() => setShowActionsSheet(true)}
                aria-label="Open scenario actions"
              >
                Actions
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="h-9 w-9 rounded-md border-transparent bg-transparent p-0 hover:bg-slate-800/70"
                onClick={() => setMobileDangerExpanded((prev) => !prev)}
                aria-label={mobileDangerExpanded ? "Hide danger section" : "Show danger section"}
              >
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </div>

            <div className="flex-1 space-y-3 overflow-y-auto px-3 py-3 pb-6">
              {actionError && (
                <Card padding="sm" className="rounded-lg border-red-500/30 bg-red-500/10">
                  <p className="text-sm text-red-300">{actionError}</p>
                </Card>
              )}

              <Card className="rounded-lg border-slate-700/60 bg-slate-900/45">
                <div className="space-y-3">
                  <div className="flex items-center gap-2">
                    <StatusIcon className={`h-4 w-4 ${SCENARIO_STATUS_COLORS[scenario.status]}`} />
                    <span className="text-xs uppercase tracking-wider text-slate-500">
                      {capitalize(scenario.status)}
                    </span>
                    <span className="rounded-full bg-slate-700 px-2.5 py-0.5 text-xs text-slate-300">P{scenario.priority}</span>
                    {localGreenfield && (
                      <span className="rounded-full bg-cyan-500/20 px-2 py-0.5 text-[11px] text-cyan-300">
                        Greenfield
                      </span>
                    )}
                  </div>
                  <p className="text-sm leading-relaxed text-slate-300">
                    {scenario.description || "No description provided"}
                  </p>
                  {scenario.completenessScore !== undefined && (
                    <div className="flex items-center gap-2">
                      <div className="h-2 flex-1 overflow-hidden rounded-full bg-slate-700">
                        <div
                          className="h-full bg-gradient-to-r from-cyan-500 to-purple-500"
                          style={{ width: `${scenario.completenessScore}%` }}
                        />
                      </div>
                      <span className="text-xs text-slate-400">{scenario.completenessScore}%</span>
                    </div>
                  )}
                  {scenario.tags.length > 0 && <TagList tags={scenario.tags} maxTags={10} />}
                </div>
              </Card>

              <Card className="rounded-lg border-slate-700/60 bg-slate-900/45">
                <div className="space-y-3">
                  <div className="flex items-center gap-2">
                    <Settings2 className="h-4 w-4 text-slate-400" />
                    <h2 className="text-base font-semibold text-slate-100">Scenario Settings</h2>
                    {updateMutation.isPending && (
                      <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
                    )}
                  </div>

                  <div className="rounded-lg bg-slate-700/30 p-3">
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-medium text-slate-200">Greenfield Mode</span>
                        {localGreenfield ? (
                          <CheckCircle2 className="h-4 w-4 text-cyan-400" />
                        ) : (
                          <XCircle className="h-4 w-4 text-slate-500" />
                        )}
                      </div>
                      <p className="text-xs text-slate-400">
                        Treat this scenario as a new project without existing code base
                      </p>
                    </div>
                    <Button
                      variant={localGreenfield ? "default" : "outline"}
                      size="sm"
                      className="mt-3 w-full"
                      onClick={handleGreenfieldToggle}
                      disabled={updateMutation.isPending}
                    >
                      {localGreenfield ? "Enabled" : "Disabled"}
                    </Button>
                  </div>

                  {updateMutation.isError && (
                    <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                      Failed to update settings. Please try again.
                    </div>
                  )}
                </div>
              </Card>

              <Card className="rounded-lg border-slate-700/60 bg-slate-900/45" data-testid={selectors.scenarioDetails.cliHint}>
                <div className="space-y-3">
                  <div className="flex items-center gap-2">
                    <Terminal className="h-4 w-4 text-slate-400" />
                    <h2 className="text-base font-semibold text-slate-100">Quick Actions (CLI)</h2>
                  </div>
                  <p className="text-sm text-slate-400">
                    Common operations for this scenario are also available via the command line.
                  </p>
                  <div className="space-y-2">
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
                </div>
              </Card>

              <Card className="rounded-lg border border-red-500/20 bg-slate-800/35">
                <button
                  type="button"
                  className="flex w-full items-center justify-between px-3 py-2 text-left"
                  onClick={() => setMobileDangerExpanded((prev) => !prev)}
                >
                  <span className="flex items-center gap-2 text-sm font-medium text-red-300">
                    <Trash2 className="h-4 w-4" />
                    Danger Zone
                  </span>
                  {mobileDangerExpanded ? (
                    <ChevronUp className="h-4 w-4 text-red-300" />
                  ) : (
                    <ChevronDown className="h-4 w-4 text-red-300" />
                  )}
                </button>

                {mobileDangerExpanded && (
                  <div className="space-y-3 border-t border-red-500/20 px-3 py-3">
                    <p className="text-sm text-slate-400">
                      Permanently remove this scenario from the catalog. This action cannot be undone.
                    </p>
                    <Button
                      variant="destructive"
                      size="sm"
                      className="w-full"
                      onClick={handleDeleteClick}
                      disabled={deleteMutation.isPending}
                    >
                      {deleteMutation.isPending ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Deleting...
                        </>
                      ) : (
                        <>
                          <Trash2 className="mr-2 h-4 w-4" />
                          Delete Scenario
                        </>
                      )}
                    </Button>
                    {deleteMutation.isError && (
                      <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
                        Failed to delete scenario. Please try again.
                      </div>
                    )}
                  </div>
                )}
              </Card>
            </div>
          </div>

          <div className="hidden space-y-6 lg:block">
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
              <div className="mb-4 flex items-center gap-2">
                <Settings2 className="h-5 w-5 text-slate-400" />
                <h2 className="text-lg font-medium text-slate-200">Scenario Settings</h2>
                {updateMutation.isPending && (
                  <Loader2 className="ml-2 h-4 w-4 animate-spin text-cyan-400" />
                )}
              </div>

              <div className="space-y-4">
                {/* Greenfield toggle */}
                <div className="flex items-center justify-between rounded-lg bg-slate-700/30 p-4">
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
              </div>

              {/* Error feedback */}
              {updateMutation.isError && (
                <div className="mt-4 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-400">
                  Failed to update settings. Please try again.
                </div>
              )}
            </Card>

            {/* CLI Quick Actions - Helps ops users access common operations (Phase 29 Iteration 5) */}
            <Card data-testid={selectors.scenarioDetails.cliHint}>
              <div className="mb-3 flex items-center gap-2">
                <Terminal className="h-5 w-5 text-slate-400" />
                <h2 className="text-lg font-medium text-slate-200">Quick Actions (CLI)</h2>
              </div>
              <p className="mb-4 text-sm text-slate-400">
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
              <div className="mb-4 flex items-center gap-2">
                <Trash2 className="h-5 w-5 text-red-400" />
                <h2 className="text-lg font-medium text-red-400">Danger Zone</h2>
              </div>

              <div className="flex items-center justify-between rounded-lg bg-slate-700/30 p-4">
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
                <div className="mt-4 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-400">
                  Failed to delete scenario. Please try again.
                </div>
              )}
            </div>
          </div>

          <BottomSheet
            isOpen={showActionsSheet}
            onClose={() => setShowActionsSheet(false)}
            title="Scenario Actions"
            description="Manage scenario lifecycle"
            data-testid="scenario-mobile-actions-sheet"
          >
            {renderActionButtons(true)}
          </BottomSheet>
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
            if (checked) {
              void loadScenarioFiles();
            }
          },
          testId: selectors.scenarioDetails.archiveCheckbox,
        }}
        sidePanel={archiveOnDelete ? (
          <div className="space-y-4" data-testid="archive-preview-panel">
            <div className="space-y-1">
              <h3 className="text-sm font-semibold uppercase tracking-wide text-cyan-300">Archive Preview</h3>
              <p className="text-xs text-slate-400">
                Choose what to keep with the archived idea before deletion.
              </p>
            </div>
            <div className="space-y-2">
              <label className="block text-xs font-medium text-slate-300" htmlFor="archive-preset-select">
                Preset
              </label>
              <Select
                id="archive-preset-select"
                value={archivePreset}
                onChange={(e) => {
                  setArchivePreset(e.target.value as PreserveFilesPreset);
                  setArchiveMode("preset");
                }}
                withChevron
                data-testid="archive-preset-select"
              >
                {ARCHIVE_PRESET_OPTIONS.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </Select>
              <p className="text-xs text-slate-500">
                {ARCHIVE_PRESET_OPTIONS.find((preset) => preset.value === archivePreset)?.description}
              </p>
            </div>
            <div className="rounded-lg border border-white/10 bg-slate-800/70 p-3">
              <div className="flex items-center justify-between">
                <span className="text-xs font-medium text-slate-200">
                  {effectiveArchiveMode === "custom" ? "Custom Selection" : "Included Files"}
                </span>
                <span className="text-xs text-cyan-300" data-testid="archive-preview-count">
                  {previewPaths.length} files
                </span>
              </div>
              <div className="mt-2 max-h-40 space-y-1 overflow-y-auto text-xs text-slate-300">
                {filesLoading ? (
                  <InlineLoadingIndicator
                    label="Loading file tree..."
                    className="border-transparent bg-transparent px-0 text-slate-400"
                    testId="scenario-archive-file-tree-loading"
                  />
                ) : previewList.length > 0 ? (
                  previewList.map((path) => (
                    <p key={path} className="truncate font-mono" title={path}>
                      {path}
                    </p>
                  ))
                ) : (
                  <p className="text-slate-500">No files selected for archive.</p>
                )}
                {!filesLoading && previewPaths.length > previewList.length && (
                  <p className="pt-1 text-slate-500">+{previewPaths.length - previewList.length} more files</p>
                )}
              </div>
            </div>
            <button
              type="button"
              onClick={handleCustomizeFilesClick}
              className="text-sm text-cyan-400 hover:text-cyan-300 underline underline-offset-2"
              data-testid="customize-files-link"
            >
              {effectiveArchiveMode === "custom" ? "Edit custom file selection..." : "Fine-tune file selection..."}
            </button>
          </div>
        ) : null}
        testIds={{
          dialog: selectors.scenarioDetails.deleteDialog,
          confirmButton: selectors.scenarioDetails.deleteConfirmButton,
          cancelButton: selectors.scenarioDetails.deleteCancelButton,
        }}
      />

      {/* File selection dialog */}
      <FileSelectionDialog
        isOpen={showFileSelectionDialog}
        onClose={handleFileSelectionCancel}
        onConfirm={handleFileSelectionConfirm}
        scenarioName={scenario?.displayName || name || ""}
        files={scenarioFiles}
        isLoading={filesLoading}
        initialSelection={effectiveArchiveMode === "custom"
          ? { paths: customPaths }
          : { preset: archivePreset, paths: [] }}
      />
    </div>
  );
}
