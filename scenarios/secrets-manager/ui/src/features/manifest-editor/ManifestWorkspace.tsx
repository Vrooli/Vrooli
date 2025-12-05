import { FileEdit, RefreshCw, Loader2, AlertCircle, HelpCircle, ChevronRight, ChevronDown } from "lucide-react";
import { Button } from "../../components/ui/button";
import { ConfirmDialog } from "../../components/ui/ConfirmDialog";
import { JsonCodePreview } from "../../components/ui/JsonCodePreview";
import { useManifestWorkspace } from "./useManifestWorkspace";
import { WorkspaceToolbar } from "./WorkspaceToolbar";
import { ResourceTree } from "./ResourceTree";
import { ManifestSecretDetailPanel } from "./ManifestSecretDetailPanel";
import { ManifestSummaryBar } from "./ManifestSummaryBar";
import { secretIdToString } from "./types";
import type { DeploymentManifestResponse } from "../../lib/api";

interface ManifestWorkspaceProps {
  initialScenario?: string;
  initialTier?: string;
  availableScenarios?: Array<{ name: string; display_name?: string }>;
  onScenarioChange?: (scenario: string) => void;
  onOpenResource?: (resourceName: string, secretKey?: string, tier?: string) => void;
  isCollapsed?: boolean;
  onToggleCollapse?: () => void;
}

export function ManifestWorkspace({
  initialScenario,
  initialTier,
  availableScenarios = [],
  onScenarioChange,
  onOpenResource,
  isCollapsed = false,
  onToggleCollapse
}: ManifestWorkspaceProps) {
  const workspace = useManifestWorkspace({
    initialScenario,
    initialTier,
    availableScenarios,
    onScenarioChange
  });

  const handleSaveOverride = () => {
    if (workspace.selectedSecretId) {
      workspace.saveOverride(workspace.selectedSecretId.resource, workspace.selectedSecretId.key);
    }
  };

  const handleResetOverride = () => {
    if (workspace.selectedSecretId) {
      workspace.resetOverride(workspace.selectedSecretId.resource, workspace.selectedSecretId.key);
    }
  };

  const handleToggleSelectedExclude = () => {
    if (workspace.selectedSecretId) {
      workspace.toggleSecretExclusion(workspace.selectedSecretId.resource, workspace.selectedSecretId.key);
    }
  };

  const handleUpdatePendingChange = (changes: Parameters<typeof workspace.updatePendingChange>[2]) => {
    if (workspace.selectedSecretId) {
      workspace.updatePendingChange(workspace.selectedSecretId.resource, workspace.selectedSecretId.key, changes);
    }
  };

  const selectedSecretKey = workspace.selectedSecretId
    ? secretIdToString(workspace.selectedSecretId)
    : undefined;
  const pendingEdit = selectedSecretKey ? workspace.pendingOverrides.get(selectedSecretKey) : undefined;

  const exportPreview = workspace.getExportPreview();

  return (
    <section id="anchor-deployment" className="rounded-3xl border border-white/10 bg-white/5 overflow-hidden">
      {/* Collapsible Header */}
      <div
        className={`flex items-center gap-3 bg-black/30 px-5 py-4 ${onToggleCollapse ? "cursor-pointer hover:bg-white/5" : ""} ${isCollapsed ? "" : "border-b border-white/10"}`}
        onClick={onToggleCollapse}
      >
        {onToggleCollapse && (
          <div className="flex items-center justify-center">
            {isCollapsed ? (
              <ChevronRight className="h-5 w-5 text-white/40" />
            ) : (
              <ChevronDown className="h-5 w-5 text-white/40" />
            )}
          </div>
        )}
        <div className="rounded-full border border-emerald-400/40 bg-emerald-500/10 p-2">
          <FileEdit className="h-5 w-5 text-emerald-200" />
        </div>
        <div className="flex-1">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold text-white">Manifest Workspace</h2>
            <Tooltip content="The Manifest Workspace lets you preview, customize, and export deployment manifests. Select a scenario and tier to load the manifest, then adjust strategies or exclude secrets before exporting." />
          </div>
          {isCollapsed && workspace.scenario ? (
            <p className="text-sm text-emerald-300/80">
              {workspace.scenario} ({workspace.tier})
              {workspace.hasPendingChanges && <span className="ml-2 text-amber-300">• unsaved changes</span>}
            </p>
          ) : (
            <p className="text-sm text-white/60">
              Preview and customize deployment manifests before export
            </p>
          )}
        </div>
        {/* Summary badges visible when collapsed */}
        {isCollapsed && workspace.summary && (
          <div className="flex gap-2" onClick={(e) => e.stopPropagation()}>
            <div className="rounded-xl border border-white/10 bg-black/30 px-2 py-1 text-xs text-white/70">
              {workspace.summary.totalSecrets} secrets
            </div>
            {workspace.summary.blockingSecrets > 0 && (
              <div className="rounded-xl border border-amber-400/30 bg-amber-500/10 px-2 py-1 text-xs text-amber-100">
                {workspace.summary.blockingSecrets} blocking
              </div>
            )}
          </div>
        )}
        {!isCollapsed && (
          <Button
            variant="ghost"
            size="sm"
            onClick={(e) => { e.stopPropagation(); workspace.refreshManifest(); }}
            disabled={workspace.manifestIsLoading}
            className="gap-1.5 text-white/60 hover:text-white"
          >
            <RefreshCw className={`h-4 w-4 ${workspace.manifestIsLoading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        )}
      </div>

      {/* Collapsible content */}
      {!isCollapsed && (
        <>
          {/* Toolbar with scenario/tier selectors */}
          <WorkspaceToolbar
        scenario={workspace.scenario}
        tier={workspace.tier}
        availableTiers={workspace.availableTiers}
        availableScenarios={availableScenarios}
        searchQuery={workspace.searchQuery}
        filter={workspace.filter}
        hasPendingChanges={workspace.hasPendingChanges}
        isSaving={workspace.isSaving}
        isCopying={workspace.isCopying}
        copyError={workspace.copyError}
        onScenarioChange={workspace.setScenario}
        onTierChange={workspace.setTier}
        onSearchChange={workspace.setSearchQuery}
        onFilterChange={workspace.setFilter}
        onSaveAll={workspace.saveAllPending}
        onExport={workspace.exportManifest}
        onCopyFromTier={workspace.copyFromTier}
      />

          {/* Main content */}
          <div className="flex max-h-[700px] min-h-[400px] overflow-hidden">
        {workspace.manifestIsLoading && !workspace.manifest ? (
          <div className="flex flex-1 items-center justify-center">
            <div className="flex flex-col items-center gap-3">
              <Loader2 className="h-8 w-8 animate-spin text-emerald-400" />
              <p className="text-sm text-white/60">Loading manifest...</p>
            </div>
          </div>
        ) : workspace.manifestIsError ? (
          <div className="flex flex-1 items-center justify-center p-8">
            <div className="flex flex-col items-center gap-3 text-center max-w-md">
              <div className="rounded-full bg-red-500/10 p-3">
                <AlertCircle className="h-8 w-8 text-red-400" />
              </div>
              <h3 className="text-lg font-medium text-white">Failed to load manifest</h3>
              <p className="text-sm text-white/60">
                {workspace.manifestError?.message || "An error occurred while generating the manifest"}
              </p>
              <Button variant="outline" size="sm" onClick={workspace.refreshManifest}>
                Try again
              </Button>
            </div>
          </div>
        ) : !workspace.scenario ? (
          <div className="flex flex-1 items-center justify-center p-8">
            <div className="flex flex-col items-center gap-3 text-center max-w-md">
              <div className="rounded-full bg-white/5 p-3">
                <FileEdit className="h-8 w-8 text-white/40" />
              </div>
              <h3 className="text-lg font-medium text-white">Select a scenario</h3>
              <p className="text-sm text-white/60">
                Choose a scenario from the dropdown above to load and preview its deployment manifest.
              </p>
            </div>
          </div>
        ) : (
          <>
            {/* Resource tree (left panel) */}
            <div className="w-[35%] min-w-[260px] max-w-[380px] overflow-y-auto border-r border-white/10">
              <ResourceTree
                groups={workspace.resources}
                expandedResources={workspace.expandedResources}
                selectedSecretId={workspace.selectedSecretId}
                excludedResources={workspace.excludedResources}
                excludedSecrets={workspace.excludedSecrets}
                overriddenSecrets={workspace.overriddenSecrets}
                onToggleResource={workspace.toggleResource}
                onToggleResourceExclusion={workspace.toggleResourceExclusion}
                onSelectSecret={workspace.selectSecret}
                onToggleSecretExclusion={workspace.toggleSecretExclusion}
              />
            </div>

            {/* Right panel - shows either Detail or JSON (not both) */}
            <div className="flex flex-1 flex-col min-h-0">
              {/* View toggle header */}
              <div className="flex items-center justify-between border-b border-white/10 bg-black/20 px-3 py-2">
                <div className="flex items-center gap-1">
                  <button
                    onClick={() => workspace.setJsonPanelOpen(false)}
                    className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                      !workspace.jsonPanelOpen
                        ? "bg-white/10 text-white"
                        : "text-white/50 hover:text-white/70"
                    }`}
                  >
                    Details
                  </button>
                  <button
                    onClick={() => workspace.setJsonPanelOpen(true)}
                    className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
                      workspace.jsonPanelOpen
                        ? "bg-white/10 text-white"
                        : "text-white/50 hover:text-white/70"
                    }`}
                  >
                    JSON Preview
                  </button>
                </div>
                {workspace.jsonPanelOpen && (
                  <Tooltip content="This shows what will be exported, with any exclusions already applied. Use the Export button to download this JSON." />
                )}
              </div>

              {/* Content area */}
              <div className="flex-1 overflow-y-auto">
                {workspace.jsonPanelOpen ? (
                  /* JSON preview panel */
                  exportPreview ? (
                    <JsonCodePreview data={exportPreview} className="rounded-b-lg" />
                  ) : (
                    <div className="flex h-full items-center justify-center text-white/50">
                      <p className="text-sm">No preview available</p>
                    </div>
                  )
                ) : (
                  /* Secret detail panel */
                  <ManifestSecretDetailPanel
                    secret={workspace.selectedSecret}
                    isOverridden={
                      workspace.selectedSecretId
                        ? workspace.isSecretOverridden(workspace.selectedSecretId.resource, workspace.selectedSecretId.key)
                        : false
                    }
                    isExcluded={
                      workspace.selectedSecretId
                        ? workspace.isExcluded(workspace.selectedSecretId.resource, workspace.selectedSecretId.key)
                        : false
                    }
                    pendingEdit={pendingEdit}
                    isSaving={workspace.isSaving}
                    isDeleting={workspace.isDeleting}
                    onUpdatePendingChange={handleUpdatePendingChange}
                    onSave={handleSaveOverride}
                    onReset={handleResetOverride}
                    onToggleExclude={handleToggleSelectedExclude}
                    onOpenInResourcePanel={onOpenResource ? () => {
                      if (workspace.selectedSecretId) {
                        onOpenResource(workspace.selectedSecretId.resource, workspace.selectedSecretId.key, workspace.tier);
                      }
                    } : undefined}
                  />
                )}
              </div>
            </div>
            </>
          )}
          </div>

          {/* Summary bar */}
          {workspace.manifest && exportPreview && (
            <ManifestSummaryBar summary={workspace.summary} exportPreview={exportPreview} />
          )}
        </>
      )}

      {/* Confirm dialog for dirty state */}
      {workspace.confirmDialog && (
        <ConfirmDialog
          open={workspace.confirmDialog.open}
          title={workspace.confirmDialog.title}
          message={workspace.confirmDialog.message}
          confirmLabel="Discard changes"
          cancelLabel="Keep editing"
          variant="warning"
          onConfirm={workspace.confirmDialog.onConfirm}
          onCancel={workspace.closeConfirmDialog}
        />
      )}
    </section>
  );
}

// Simple tooltip component for contextual help
function Tooltip({ content }: { content: string }) {
  return (
    <div className="group relative inline-flex">
      <HelpCircle className="h-3.5 w-3.5 text-white/30 cursor-help" />
      <div className="pointer-events-none absolute left-full top-1/2 z-50 ml-2 w-64 -translate-y-1/2 rounded-lg border border-white/10 bg-slate-900 px-3 py-2 text-xs text-white/70 opacity-0 shadow-lg transition-opacity group-hover:opacity-100">
        {content}
        <div className="absolute left-0 top-1/2 -translate-x-1 -translate-y-1/2 border-4 border-transparent border-r-slate-900" />
      </div>
    </div>
  );
}
