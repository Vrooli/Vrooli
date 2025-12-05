import { FileEdit } from "lucide-react";
import type { DeploymentManifestResponse } from "../../lib/api";
import { useManifestEditor } from "./useManifestEditor";
import { ManifestToolbar } from "./ManifestToolbar";
import { ResourceTree } from "./ResourceTree";
import { ManifestSecretDetailPanel } from "./ManifestSecretDetailPanel";
import { ManifestSummaryBar } from "./ManifestSummaryBar";
import { secretIdToString } from "./types";

interface ManifestEditorProps {
  scenario: string;
  tier: string;
  availableTiers?: string[];
  initialManifest: DeploymentManifestResponse;
  onClose: () => void;
  onExport?: () => void;
}

const DEFAULT_TIERS = [
  "tier-1-local",
  "tier-2-desktop",
  "tier-3-mobile",
  "tier-4-saas",
  "tier-5-enterprise"
];

export function ManifestEditor({
  scenario,
  tier,
  availableTiers = DEFAULT_TIERS,
  initialManifest,
  onClose,
  onExport
}: ManifestEditorProps) {
  const editor = useManifestEditor({ scenario, tier, initialManifest });

  const handleExport = () => {
    editor.exportManifest();
    onExport?.();
  };

  const handleSaveOverride = () => {
    if (editor.selectedSecretId) {
      editor.saveOverride(editor.selectedSecretId.resource, editor.selectedSecretId.key);
    }
  };

  const handleResetOverride = () => {
    if (editor.selectedSecretId) {
      editor.resetOverride(editor.selectedSecretId.resource, editor.selectedSecretId.key);
    }
  };

  const handleToggleSelectedExclude = () => {
    if (editor.selectedSecretId) {
      editor.toggleSecretExclusion(editor.selectedSecretId.resource, editor.selectedSecretId.key);
    }
  };

  const handleUpdatePendingChange = (changes: Parameters<typeof editor.updatePendingChange>[2]) => {
    if (editor.selectedSecretId) {
      editor.updatePendingChange(editor.selectedSecretId.resource, editor.selectedSecretId.key, changes);
    }
  };

  const selectedSecretKey = editor.selectedSecretId
    ? secretIdToString(editor.selectedSecretId)
    : undefined;
  const pendingEdit = selectedSecretKey ? editor.pendingOverrides.get(selectedSecretKey) : undefined;

  return (
    <div className="flex h-full flex-col overflow-hidden rounded-2xl border border-white/10 bg-slate-900/95">
      <div className="flex items-center gap-3 border-b border-white/10 bg-black/30 px-4 py-3">
        <FileEdit className="h-5 w-5 text-emerald-400" />
        <div>
          <h2 className="text-sm font-medium text-white">Manifest Editor</h2>
          <p className="text-xs text-white/50">
            {scenario} / {tier}
          </p>
        </div>
      </div>

      <ManifestToolbar
        searchQuery={editor.searchQuery}
        filter={editor.filter}
        hasPendingChanges={editor.hasPendingChanges}
        isSaving={editor.isSaving}
        currentTier={tier}
        availableTiers={availableTiers}
        isCopying={editor.isCopying}
        copyError={editor.copyError}
        onSearchChange={editor.setSearchQuery}
        onFilterChange={editor.setFilter}
        onSaveAll={editor.saveAllPending}
        onExport={handleExport}
        onClose={onClose}
        onCopyFromTier={editor.copyFromTier}
      />

      <div className="flex flex-1 min-h-0">
        <div className="w-[40%] min-w-[280px] max-w-[400px] overflow-y-auto border-r border-white/10">
          <ResourceTree
            groups={editor.resources}
            expandedResources={editor.expandedResources}
            selectedSecretId={editor.selectedSecretId}
            excludedResources={editor.excludedResources}
            excludedSecrets={editor.excludedSecrets}
            overriddenSecrets={editor.overriddenSecrets}
            onToggleResource={editor.toggleResource}
            onToggleResourceExclusion={editor.toggleResourceExclusion}
            onSelectSecret={editor.selectSecret}
            onToggleSecretExclusion={editor.toggleSecretExclusion}
          />
        </div>

        <div className="flex-1 overflow-y-auto">
          <ManifestSecretDetailPanel
            secret={editor.selectedSecret}
            isOverridden={
              editor.selectedSecretId
                ? editor.isSecretOverridden(editor.selectedSecretId.resource, editor.selectedSecretId.key)
                : false
            }
            isExcluded={
              editor.selectedSecretId
                ? editor.isExcluded(editor.selectedSecretId.resource, editor.selectedSecretId.key)
                : false
            }
            pendingEdit={pendingEdit}
            isSaving={editor.isSaving}
            isDeleting={editor.isDeleting}
            onUpdatePendingChange={handleUpdatePendingChange}
            onSave={handleSaveOverride}
            onReset={handleResetOverride}
            onToggleExclude={handleToggleSelectedExclude}
          />
        </div>
      </div>

      <ManifestSummaryBar summary={editor.summary} exportPreview={editor.getExportPreview()} />
    </div>
  );
}
