import { CheckCircle2, Loader2, XCircle } from "lucide-react";
import { Button } from "../ui/button";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { InlineLoadingIndicator } from "../ui/loading-states";
import { Select } from "../ui/select";
import { FileSelectionDialog, type FileSelectionResult } from "./file-selection-dialog";
import { ARCHIVE_PRESET_OPTIONS } from "../../hooks/useArchivePreferences";
import { selectors } from "../../consts/selectors";
import type { PreserveFilesPreset, ScenarioFile } from "../../types";
import type { SpecSyncPhase } from "../../hooks/useScenarioDetailData";

interface ScenarioDeleteDialogProps {
  isOpen: boolean;
  scenarioDisplayName: string;
  scenarioName: string | undefined;
  isDeleteLoading: boolean;
  // Archive preferences
  archiveOnDelete: boolean;
  onArchiveOnDeleteChange: (checked: boolean) => void;
  archivePreset: PreserveFilesPreset;
  onArchivePresetChange: (preset: PreserveFilesPreset) => void;
  effectiveArchiveMode: "preset" | "custom";
  customPaths: string[];
  onCustomPathsChange: (paths: string[]) => void;
  onArchiveModeChange: (mode: "preset" | "custom") => void;
  previewPaths: string[];
  previewList: string[];
  filesLoading: boolean;
  // Spec sync
  specSyncEnabled: boolean;
  onSpecSyncEnabledChange: (enabled: boolean) => void;
  specSyncPhase: SpecSyncPhase;
  specSyncError: string | null;
  isSpecSyncInProgress: boolean;
  onSpecSyncRetry: () => void;
  onArchiveWithoutSync: () => void;
  onSpecSyncCancel: () => void;
  // File selection dialog
  showFileSelectionDialog: boolean;
  onCustomizeFilesClick: () => void;
  onFileSelectionConfirm: (selection: FileSelectionResult) => void;
  onFileSelectionCancel: () => void;
  scenarioFiles: ScenarioFile[];
  // Callbacks
  onClose: () => void;
  onConfirm: () => void;
}

export function ScenarioDeleteDialog({
  isOpen,
  scenarioDisplayName,
  scenarioName,
  isDeleteLoading,
  archiveOnDelete,
  onArchiveOnDeleteChange,
  archivePreset,
  onArchivePresetChange,
  effectiveArchiveMode,
  customPaths,
  onCustomPathsChange: _onCustomPathsChange,
  onArchiveModeChange,
  previewPaths,
  previewList,
  filesLoading,
  specSyncEnabled,
  onSpecSyncEnabledChange,
  specSyncPhase,
  specSyncError,
  isSpecSyncInProgress,
  onSpecSyncRetry,
  onArchiveWithoutSync,
  onSpecSyncCancel,
  showFileSelectionDialog,
  onCustomizeFilesClick,
  onFileSelectionConfirm,
  onFileSelectionCancel,
  scenarioFiles,
  onClose,
  onConfirm,
}: ScenarioDeleteDialogProps) {
  return (
    <>
      {/* [REQ:REQ-P0-008] Strong confirmation dialog with archive option */}
      <ConfirmDialog
        isOpen={isOpen}
        onClose={onClose}
        onConfirm={onConfirm}
        title="Delete Scenario"
        description={`Are you sure you want to delete "${scenarioDisplayName}"? This will remove the scenario from the catalog.`}
        confirmationText={scenarioName}
        confirmLabel={specSyncEnabled && archiveOnDelete ? "Sync & Archive" : "Delete Scenario"}
        isLoading={isDeleteLoading || isSpecSyncInProgress}
        checkboxContent={{
          label: "Archive to backlog (idea) before deleting (recommended)",
          checked: archiveOnDelete,
          onChange: onArchiveOnDeleteChange,
          testId: selectors.scenarioDetails.archiveCheckbox,
        }}
        sidePanel={archiveOnDelete ? (
          <div className="space-y-4" data-testid="archive-preview-panel">
            {/* Spec-sync progress overlay */}
            {specSyncPhase !== "idle" && (
              <div className="rounded-lg border border-cyan-500/30 bg-cyan-500/10 p-3 space-y-2">
                {specSyncPhase === "syncing" && (
                  <>
                    <div className="flex items-center gap-2">
                      <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
                      <span className="text-sm font-medium text-cyan-300">Syncing specs with code...</span>
                    </div>
                    <p className="text-xs text-slate-400">
                      An agent is updating docs to match the implementation. This may take several minutes.
                    </p>
                    <Button
                      variant="outline"
                      size="sm"
                      className="mt-1"
                      onClick={onSpecSyncCancel}
                    >
                      Cancel
                    </Button>
                  </>
                )}
                {specSyncPhase === "archiving" && (
                  <div className="flex items-center gap-2">
                    <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
                    <span className="text-sm font-medium text-cyan-300">Specs synced. Archiving...</span>
                  </div>
                )}
                {specSyncPhase === "done" && (
                  <div className="flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4 text-green-400" />
                    <span className="text-sm font-medium text-green-300">Archive complete!</span>
                  </div>
                )}
                {specSyncPhase === "failed" && (
                  <>
                    <div className="flex items-center gap-2">
                      <XCircle className="h-4 w-4 text-red-400" />
                      <span className="text-sm font-medium text-red-300">Spec sync failed</span>
                    </div>
                    <p className="text-xs text-red-300/80">{specSyncError}</p>
                    <div className="flex gap-2 pt-1">
                      <Button variant="outline" size="sm" onClick={onSpecSyncRetry}>
                        Retry
                      </Button>
                      <Button variant="outline" size="sm" onClick={onArchiveWithoutSync}>
                        Archive Without Sync
                      </Button>
                      <Button variant="outline" size="sm" onClick={onClose}>
                        Cancel
                      </Button>
                    </div>
                  </>
                )}
              </div>
            )}

            <div className="space-y-1">
              <h3 className="text-sm font-semibold uppercase tracking-wide text-cyan-300">Archive Preview</h3>
              <p className="text-xs text-slate-400">
                Choose what to keep with the archived idea before deletion.
              </p>
            </div>

            {/* Spec-sync toggle */}
            <label className="flex items-start gap-2 rounded-lg bg-slate-800/50 p-3 cursor-pointer" data-testid="spec-sync-toggle">
              <input
                type="checkbox"
                checked={specSyncEnabled}
                onChange={(e) => onSpecSyncEnabledChange(e.target.checked)}
                disabled={isSpecSyncInProgress}
                className="mt-0.5 rounded border-slate-600 bg-slate-700 text-cyan-500 focus:ring-cyan-500/50"
              />
              <div className="space-y-0.5">
                <span className="text-xs font-medium text-slate-200">Sync specs with code before archiving</span>
                <p className="text-[11px] text-slate-400">
                  An agent will update PRD, requirements, and docs to match the current code. Takes several minutes.
                </p>
              </div>
            </label>

            <div className="space-y-2">
              <label className="block text-xs font-medium text-slate-300" htmlFor="archive-preset-select">
                Preset
              </label>
              <Select
                id="archive-preset-select"
                value={archivePreset}
                onChange={(e) => {
                  onArchivePresetChange(e.target.value as PreserveFilesPreset);
                  onArchiveModeChange("preset");
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
              onClick={onCustomizeFilesClick}
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
        onClose={onFileSelectionCancel}
        onConfirm={onFileSelectionConfirm}
        scenarioName={scenarioDisplayName}
        files={scenarioFiles}
        isLoading={filesLoading}
        initialSelection={effectiveArchiveMode === "custom"
          ? { paths: customPaths }
          : { preset: archivePreset, paths: [] }}
      />
    </>
  );
}
