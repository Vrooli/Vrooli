import { ErrorBoundary } from "../ErrorBoundary";
import { LabelManager } from "../labels/LabelManager";
import { Settings, type ViewMode } from "../settings/Settings";
import { KeyboardShortcuts } from "../settings/KeyboardShortcuts";
import { UsageStats } from "../settings/UsageStats";
import { TemplateEditorModal } from "../chat/TemplateEditorModal";
import type { AppModalsState } from "../../hooks/useAppModals";
import type { TemplateWithSource } from "../../lib/types/templates";
import type { Label } from "../../lib/api-types";
import type { Model } from "../../lib/api-models";

interface AppDialogsProps {
  modals: AppModalsState;
  labels: Label[];
  models: Model[];
  createLabel: (data: { name: string; color: string }) => void;
  deleteLabel: (labelId: string) => void;
  deleteAllChats: () => Promise<unknown>;
  isDeletingAllChats: boolean;
  handleClearArchived: () => Promise<void>;
  isClearingArchived: boolean;
  handleMarkAllAsRead: () => Promise<void>;
  isMarkingAllAsRead: boolean;
  viewMode: ViewMode;
  handleViewModeChange: (mode: ViewMode) => void;
}

export function AppDialogs({
  modals,
  labels,
  models,
  createLabel,
  deleteLabel,
  deleteAllChats,
  isDeletingAllChats,
  handleClearArchived,
  isClearingArchived,
  handleMarkAllAsRead,
  isMarkingAllAsRead,
  viewMode,
  handleViewModeChange,
}: AppDialogsProps) {
  return (
    <>
      {/* Label Manager Dialog */}
      <ErrorBoundary name="LabelManager">
        <LabelManager
          open={modals.showLabelManager}
          onClose={() => modals.setShowLabelManager(false)}
          labels={labels}
          onCreateLabel={createLabel}
          onDeleteLabel={deleteLabel}
        />
      </ErrorBoundary>

      {/* Settings Dialog */}
      <ErrorBoundary name="Settings">
        <Settings
          open={modals.showSettings}
          initialTab={modals.settingsInitialTab}
          onClose={() => modals.setShowSettings(false)}
          onDeleteAllChats={deleteAllChats}
          isDeletingAll={isDeletingAllChats}
          onClearArchived={handleClearArchived}
          isClearingArchived={isClearingArchived}
          onMarkAllAsRead={handleMarkAllAsRead}
          isMarkingAllAsRead={isMarkingAllAsRead}
          onShowKeyboardShortcuts={modals.handleShowKeyboardShortcuts}
          onShowUsageStats={modals.handleShowUsageStats}
          models={models}
          viewMode={viewMode}
          onViewModeChange={handleViewModeChange}
          onEditTemplate={modals.handleEditTemplateFromSettings}
        />
      </ErrorBoundary>

      {/* Template Editor from Settings */}
      {!!modals.settingsEditingTemplate && (
        <ErrorBoundary name="TemplateEditor">
          <TemplateEditorModal
            open={!!modals.settingsEditingTemplate}
            onClose={() => {
              modals.setSettingsEditingTemplate(null);
              modals.setSettingsAllTemplates([]);
            }}
            onSave={modals.handleSaveTemplateFromSettings}
            template={modals.settingsEditingTemplate || undefined}
            templateSource={modals.settingsEditingTemplate?.source}
            allTemplates={modals.settingsAllTemplates}
            onSelectTemplate={(template: TemplateWithSource) => {
              modals.setSettingsEditingTemplate(template);
            }}
            onSaveAll={async (updates: Array<{ id: string; data: Record<string, unknown> }>) => {
              const { updateTemplates, getAllTemplates } = await import("../../data/templates");
              await updateTemplates(updates);
              const updated = await getAllTemplates();
              modals.setSettingsAllTemplates(updated);
            }}
          />
        </ErrorBoundary>
      )}

      {/* Keyboard Shortcuts Dialog */}
      <KeyboardShortcuts
        open={modals.showKeyboardShortcuts}
        onClose={() => modals.setShowKeyboardShortcuts(false)}
      />

      {/* Usage Statistics Dialog */}
      <UsageStats
        isOpen={modals.showUsageStats}
        onClose={() => modals.setShowUsageStats(false)}
      />
    </>
  );
}
