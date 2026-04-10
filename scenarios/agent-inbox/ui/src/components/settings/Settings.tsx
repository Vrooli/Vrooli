import { useState, useEffect, useMemo } from "react";
import {
  Lightbulb,
  BookOpen,
  Bot,
  ChevronDown,
  Settings2,
  Cpu,
  Wrench,
  Database,
} from "lucide-react";
import { Dialog, DialogHeader, DialogBody } from "../ui/dialog";
import { Dropdown, DropdownItem } from "../ui/dropdown";
import { ModelSelector } from "./ModelSelector";
import { TemplatesSettingsTab } from "./TemplatesSettingsTab";
import { SkillsSettingsTab } from "./SkillsSettingsTab";
import { SkillEditorModal } from "./SkillEditorModal";
import { AgentModeSettings } from "./AgentModeSettings";
import { ManualToolDialog } from "../tools/ManualToolDialog";
import { SettingsSection } from "./SettingsControls";
import { GeneralSettingsTab } from "./GeneralSettingsTab";
import { SuggestionsSettingsTab } from "./SuggestionsSettingsTab";
import { DataSettingsTab } from "./DataSettingsTab";
import { ToolsSettingsTab } from "./ToolsSettingsTab";
import { useSettingsState } from "./useSettingsState";
import { getAllSkills } from "../../data/skills";
import type { Model } from "../../lib/api";
import type { TemplateWithSource } from "../../lib/types/templates";

// Re-export types and utilities so existing imports keep working
export type { Theme, ViewMode, SettingsTab } from "./settingsTypes";
export {
  DEFAULT_MODEL,
  DEFAULT_VIEW_MODE,
  getDefaultModel,
  setDefaultModel,
  getViewMode,
  setViewMode,
} from "./settingsTypes";

import type { ViewMode, SettingsTab, SettingsTabDef } from "./settingsTypes";

const SETTINGS_TABS: SettingsTabDef[] = [
  { value: "general", label: "General", icon: Settings2 },
  { value: "ai", label: "AI", icon: Cpu },
  { value: "agent", label: "Agent", icon: Bot },
  { value: "tools", label: "Tools", icon: Wrench },
  { value: "templates", label: "Templates", icon: Lightbulb },
  { value: "suggestions", label: "Suggestions", icon: Lightbulb },
  { value: "skills", label: "Skills", icon: BookOpen },
  { value: "data", label: "Data", icon: Database },
];
const FALLBACK_TAB: SettingsTabDef = { value: "general", label: "General", icon: Settings2 };

interface SettingsProps {
  open: boolean;
  onClose: () => void;
  onDeleteAllChats: () => Promise<unknown>;
  isDeletingAll: boolean;
  onClearArchived?: () => Promise<unknown>;
  isClearingArchived?: boolean;
  onMarkAllAsRead?: () => Promise<unknown>;
  isMarkingAllAsRead?: boolean;
  onShowKeyboardShortcuts: () => void;
  onShowUsageStats: () => void;
  models: Model[];
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  onEditTemplate?: (template: TemplateWithSource, allTemplates: TemplateWithSource[]) => void;
  initialTab?: SettingsTab;
}

export function Settings({
  open,
  onClose,
  onDeleteAllChats,
  isDeletingAll,
  onClearArchived,
  isClearingArchived = false,
  onMarkAllAsRead,
  isMarkingAllAsRead = false,
  onShowKeyboardShortcuts,
  onShowUsageStats,
  models,
  viewMode,
  onViewModeChange,
  onEditTemplate,
  initialTab,
}: SettingsProps) {
  const [activeTab, setActiveTab] = useState<SettingsTab>(initialTab ?? "general");

  useEffect(() => {
    if (open) setActiveTab(initialTab ?? "general");
  }, [open, initialTab]);

  const s = useSettingsState(open, activeTab, onClose, onEditTemplate);

  const activeTabDef = useMemo(
    () => SETTINGS_TABS.find((tab) => tab.value === activeTab) ?? FALLBACK_TAB,
    [activeTab]
  );

  const renderTabContent = () => {
    switch (activeTab) {
      case "general":
        return <GeneralSettingsTab theme={s.theme} onThemeChange={s.handleThemeChange} viewMode={viewMode} onViewModeChange={onViewModeChange} onShowKeyboardShortcuts={onShowKeyboardShortcuts} onClose={onClose} />;
      case "ai":
        return (
          <SettingsSection title="Default Model" description="New chats will use this model by default">
            <ModelSelector models={models} selectedModel={s.defaultModel} onSelectModel={s.handleDefaultModelChange} />
          </SettingsSection>
        );
      case "agent":
        return <AgentModeSettings settings={s.agentSettings} onSettingsChange={s.setAgentSettings} onReset={s.resetAgentSettings} />;
      case "tools":
        return (
          <ToolsSettingsTab
            yoloMode={s.yoloMode} isLoadingYoloMode={s.isLoadingYoloMode} isUpdatingYoloMode={s.isUpdatingYoloMode}
            onYoloModeToggle={s.handleYoloModeToggle}
            toolsByScenario={s.toolsByScenario} categories={s.toolSet?.categories ?? []} scenarioStatuses={s.scenarios}
            isLoadingTools={s.isLoadingTools} isSyncingTools={s.isSyncingTools} isUpdatingTools={s.isUpdatingTools}
            toolsError={s.toolsError?.message}
            onToggleTool={s.toggleTool} onSetApproval={s.handleSetApproval} onSyncTools={s.syncDiscoveredTools}
            onRunTool={s.handleRunTool} enabledCount={s.enabledTools.length} totalCount={s.toolSet?.tools.length ?? 0}
          />
        );
      case "templates":
        return <TemplatesSettingsTab templates={s.templates} onEditTemplate={s.handleEditTemplate} onDeleteTemplate={s.handleDeleteTemplate} onResetTemplate={s.handleResetTemplate} modeHistory={s.modeHistory} onClearHistory={s.clearModeHistory} isLoading={s.isLoadingTemplates} />;
      case "suggestions":
        return (
          <SuggestionsSettingsTab
            suggestionsVisible={s.suggestionsVisible} onSuggestionsVisibleChange={s.setSuggestionsVisible}
            mergeModel={s.mergeModel} onMergeModelChange={s.setMergeModel} models={models}
            autoSuggestError={s.autoSuggestError} suggestionsSaveError={s.suggestionsSaveError}
            autoSuggestLoading={s.autoSuggestLoading} suggestionsDraft={s.suggestionsDraft}
            onSuggestionsDraftChange={s.setSuggestionsDraft} isSavingSuggestions={s.isSavingSuggestions}
            onSaveSuggestions={s.handleSaveSuggestions}
          />
        );
      case "skills":
        return <SkillsSettingsTab skills={s.skills} onEditSkill={s.handleEditSkill} onDeleteSkill={s.handleDeleteSkill} isLoading={s.isLoadingSkills} onSyncSkills={s.handleSyncSkills} isSyncing={s.isSyncingSkills} />;
      case "data":
        return <DataSettingsTab onDeleteAllChats={onDeleteAllChats} isDeletingAll={isDeletingAll} onClearArchived={onClearArchived} isClearingArchived={isClearingArchived} onMarkAllAsRead={onMarkAllAsRead} isMarkingAllAsRead={isMarkingAllAsRead} onShowUsageStats={onShowUsageStats} onClose={onClose} />;
      default:
        queueMicrotask(() => setActiveTab("general"));
        return null;
    }
  };

  return (
    <>
      <Dialog open={open} onClose={onClose} className="max-w-5xl" disableEscape={s.isCreatingSkill || s.editingSkill !== null}>
        <DialogHeader onClose={onClose}>Settings</DialogHeader>
        <DialogBody className="space-y-4">
          <div className="lg:hidden">
            <Dropdown
              className="w-full"
              trigger={(
                <button type="button" className="w-full flex items-center justify-between gap-2 px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-slate-200" data-testid="settings-mobile-tab-selector">
                  <span className="flex items-center gap-2">
                    <activeTabDef.icon className="h-4 w-4" />
                    {activeTabDef.label}
                  </span>
                  <ChevronDown className="h-4 w-4 text-slate-400" />
                </button>
              )}
            >
              {SETTINGS_TABS.map((tab) => (
                <DropdownItem key={tab.value} onClick={() => setActiveTab(tab.value)} className={tab.value === activeTab ? "bg-white/10 text-white" : ""}>
                  <tab.icon className="h-4 w-4" />
                  {tab.label}
                </DropdownItem>
              ))}
            </Dropdown>
          </div>

          <div className="flex gap-4 min-h-[540px]">
            <nav className="hidden lg:flex w-52 shrink-0 flex-col gap-1 rounded-lg border border-white/10 bg-white/5 p-2">
              {SETTINGS_TABS.map((tab) => {
                const Icon = tab.icon;
                const isActive = tab.value === activeTab;
                return (
                  <button key={tab.value} onClick={() => setActiveTab(tab.value)} className={`w-full flex items-center gap-2 px-3 py-2 rounded-md text-sm text-left transition-colors ${isActive ? "bg-indigo-500/20 text-white" : "text-slate-400 hover:text-white hover:bg-white/5"}`} data-testid={`settings-nav-${tab.value}`}>
                    <Icon className="h-4 w-4" />
                    {tab.label}
                  </button>
                );
              })}
            </nav>
            <div className="flex-1 overflow-y-auto rounded-lg border border-white/10 bg-slate-900/40 p-4">
              {renderTabContent()}
            </div>
          </div>

          {s.selectedToolForRun && (
            <ManualToolDialog open={!!s.selectedToolForRun} onClose={() => s.setSelectedToolForRun(null)} tool={s.selectedToolForRun} />
          )}
        </DialogBody>
      </Dialog>

      <SkillEditorModal
        open={s.isCreatingSkill || s.editingSkill !== null}
        onClose={() => { s.setEditingSkill(null); s.setIsCreatingSkill(false); }}
        skill={s.editingSkill ?? undefined}
        onSave={s.handleSaveSkill}
        allSkills={s.skills}
        onSelectSkill={(skill) => { s.setEditingSkill(skill); s.setIsCreatingSkill(false); }}
        onSaveAll={async (updates) => {
          const { updateSkills } = await import("../../data/skills");
          await updateSkills(updates);
          s.setSkills(await getAllSkills());
        }}
      />
    </>
  );
}
