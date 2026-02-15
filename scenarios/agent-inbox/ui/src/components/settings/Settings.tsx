import { useState, useCallback, useEffect, useMemo } from "react";
import {
  Moon,
  Sun,
  Trash2,
  AlertTriangle,
  Keyboard,
  BarChart3,
  Wrench,
  Settings2,
  Cpu,
  Database,
  MessageCircle,
  AlignLeft,
  Zap,
  Lightbulb,
  BookOpen,
  MailCheck,
  Archive,
  Loader2,
  Bot,
  ChevronDown,
  type LucideIcon,
} from "lucide-react";
import { Dialog, DialogHeader, DialogBody } from "../ui/dialog";
import { Button } from "../ui/button";
import { Dropdown, DropdownItem } from "../ui/dropdown";
import { Input } from "../ui/input";
import { ModelSelector } from "./ModelSelector";
import { ToolConfiguration } from "./ToolConfiguration";
import { TemplatesSettingsTab } from "./TemplatesSettingsTab";
import { SkillsSettingsTab } from "./SkillsSettingsTab";
import { SkillEditorModal } from "./SkillEditorModal";
import { AgentModeSettings } from "./AgentModeSettings";
import { ManualToolDialog } from "../tools/ManualToolDialog";
import { SettingsSection, SettingsSwitchRow, SettingsNumberField } from "./SettingsControls";
import { useTools } from "../../hooks/useTools";
import { useYoloMode } from "../../hooks/useSettings";
import { useSuggestionsSettings } from "../../hooks/useSuggestionsSettings";
import { useModeHistory } from "../../hooks/useModeHistory";
import { useAgentSettings } from "../../hooks/useAgentSettings";
import {
  getAllTemplates,
  deleteTemplate as deleteTemplateFromAPI,
  resetTemplate as resetTemplateFromAPI,
} from "../../data/templates";
import {
  getAllSkills,
  deleteSkill as deleteSkillFromAPI,
  createSkill as createSkillFromAPI,
  updateSkill as updateSkillFromAPI,
  syncSkills as syncSkillsFromAPI,
} from "../../data/skills";
import type { Model, ApprovalOverride, EffectiveTool } from "../../lib/api";
import type { TemplateWithSource, SkillWithSource, Skill } from "../../lib/types/templates";

export type Theme = "dark" | "light";
export type ViewMode = "bubble" | "compact";
export type SettingsTab = "general" | "ai" | "agent" | "tools" | "templates" | "suggestions" | "skills" | "data";

// Default model used when none is set
export const DEFAULT_MODEL = "anthropic/claude-3.5-sonnet";

// Default view mode
export const DEFAULT_VIEW_MODE: ViewMode = "bubble";

interface SettingsTabDef {
  value: SettingsTab;
  label: string;
  icon: LucideIcon;
}

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


// Get/set default model from localStorage
export function getDefaultModel(): string {
  if (typeof window !== "undefined") {
    return localStorage.getItem("defaultModel") || DEFAULT_MODEL;
  }
  return DEFAULT_MODEL;
}

export function setDefaultModel(modelId: string): void {
  if (typeof window !== "undefined") {
    localStorage.setItem("defaultModel", modelId);
  }
}

// Get/set view mode from localStorage
export function getViewMode(): ViewMode {
  if (typeof window !== "undefined") {
    const stored = localStorage.getItem("viewMode");
    if (stored === "bubble" || stored === "compact") {
      return stored;
    }
  }
  return DEFAULT_VIEW_MODE;
}

export function setViewMode(mode: ViewMode): void {
  if (typeof window !== "undefined") {
    localStorage.setItem("viewMode", mode);
  }
}

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

interface ChoiceButtonProps {
  label: string;
  icon: React.ReactNode;
  selected: boolean;
  onClick: () => void;
  testId?: string;
}

function ChoiceButton({ label, icon, selected, onClick, testId }: ChoiceButtonProps) {
  return (
    <button
      onClick={onClick}
      className={`flex-1 flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors ${
        selected
          ? "bg-indigo-500/20 border-indigo-500 text-white"
          : "bg-white/5 border-white/10 text-slate-400 hover:text-white hover:border-white/20"
      }`}
      data-testid={testId}
    >
      {icon}
      <span className="text-sm">{label}</span>
    </button>
  );
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
  const [theme, setTheme] = useState<Theme>(() => {
    if (typeof window !== "undefined") {
      return (localStorage.getItem("theme") as Theme) || "dark";
    }
    return "dark";
  });
  const [defaultModel, setDefaultModelState] = useState<string>(getDefaultModel);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteConfirmText, setDeleteConfirmText] = useState("");
  const [showClearArchivedConfirm, setShowClearArchivedConfirm] = useState(false);
  const [selectedToolForRun, setSelectedToolForRun] = useState<EffectiveTool | null>(null);

  // YOLO mode setting
  const {
    yoloMode,
    isLoading: isLoadingYoloMode,
    isUpdating: isUpdatingYoloMode,
    setYoloMode,
  } = useYoloMode(open && activeTab === "tools");

  // Tool configuration (global defaults)
  const {
    toolsByScenario,
    toolSet,
    scenarios,
    enabledTools,
    isLoading: isLoadingTools,
    isSyncing: isSyncingTools,
    isUpdating: isUpdatingTools,
    error: toolsError,
    toggleTool,
    setApproval,
    syncDiscoveredTools,
  } = useTools({ enabled: open && activeTab === "tools" });

  // Suggestions settings
  const {
    visible: suggestionsVisible,
    setVisible: setSuggestionsVisible,
    mergeModel,
    setMergeModel,
    autoSuggest,
    autoSuggestLoading,
    autoSuggestError,
    updateAutoSuggest,
  } = useSuggestionsSettings();

  const { history: modeHistory, clearHistory: clearModeHistory } = useModeHistory();

  // Agent mode settings
  const {
    settings: agentSettings,
    setSettings: setAgentSettings,
    resetSettings: resetAgentSettings,
  } = useAgentSettings();

  // Templates state - refresh when tab changes to templates
  const [templates, setTemplates] = useState<TemplateWithSource[]>([]);
  const [isLoadingTemplates, setIsLoadingTemplates] = useState(false);

  // Skills state - refresh when tab changes to skills
  const [skills, setSkills] = useState<SkillWithSource[]>([]);
  const [isLoadingSkills, setIsLoadingSkills] = useState(false);
  const [isSyncingSkills, setIsSyncingSkills] = useState(false);
  const [editingSkill, setEditingSkill] = useState<SkillWithSource | null>(null);
  const [isCreatingSkill, setIsCreatingSkill] = useState(false);

  const [suggestionsDraft, setSuggestionsDraft] = useState(autoSuggest);
  const [isSavingSuggestions, setIsSavingSuggestions] = useState(false);
  const [suggestionsSaveError, setSuggestionsSaveError] = useState<string | null>(null);

  // Refresh templates when switching to templates tab
  useEffect(() => {
    if (activeTab === "templates") {
      setIsLoadingTemplates(true);
      getAllTemplates()
        .then(setTemplates)
        .finally(() => setIsLoadingTemplates(false));
    }
  }, [activeTab]);

  // Refresh skills when switching to skills tab
  useEffect(() => {
    if (activeTab === "skills") {
      setIsLoadingSkills(true);
      getAllSkills()
        .then(setSkills)
        .finally(() => setIsLoadingSkills(false));
    }
  }, [activeTab]);

  useEffect(() => {
    setSuggestionsDraft(autoSuggest);
  }, [autoSuggest]);

  const handleDeleteTemplate = useCallback(async (templateId: string) => {
    await deleteTemplateFromAPI(templateId);
    const updated = await getAllTemplates();
    setTemplates(updated);
  }, []);

  const handleResetTemplate = useCallback(async (templateId: string) => {
    await resetTemplateFromAPI(templateId);
    const updated = await getAllTemplates();
    setTemplates(updated);
  }, []);

  const handleEditTemplate = useCallback((template: TemplateWithSource) => {
    onEditTemplate?.(template, templates);
    onClose();
  }, [onEditTemplate, onClose, templates]);

  // Skill handlers
  const handleDeleteSkill = useCallback(async (skillId: string) => {
    await deleteSkillFromAPI(skillId);
    const updated = await getAllSkills();
    setSkills(updated);
  }, []);

  const handleSyncSkills = useCallback(async () => {
    setIsSyncingSkills(true);
    try {
      await syncSkillsFromAPI();
      const updated = await getAllSkills();
      setSkills(updated);
    } finally {
      setIsSyncingSkills(false);
    }
  }, []);

  const handleEditSkill = useCallback((skill: SkillWithSource | null) => {
    if (skill === null) {
      setIsCreatingSkill(true);
      setEditingSkill(null);
      return;
    }
    setIsCreatingSkill(false);
    setEditingSkill(skill);
  }, []);

  const handleSaveSkill = useCallback(async (
    skillData: Omit<Skill, "id" | "createdAt" | "updatedAt">
  ) => {
    if (isCreatingSkill) {
      await createSkillFromAPI(skillData);
    } else if (editingSkill) {
      await updateSkillFromAPI(editingSkill.id, skillData);
    }

    const updated = await getAllSkills();
    setSkills(updated);
    setEditingSkill(null);
    setIsCreatingSkill(false);
  }, [isCreatingSkill, editingSkill]);

  const handleYoloModeToggle = useCallback((checked: boolean) => {
    setYoloMode(checked);
  }, [setYoloMode]);

  const handleSetApproval = useCallback((scenario: string, toolName: string, override: ApprovalOverride) => {
    setApproval(scenario, toolName, override);
  }, [setApproval]);

  const handleRunTool = useCallback((tool: EffectiveTool) => {
    setSelectedToolForRun(tool);
  }, []);

  const handleSaveSuggestions = useCallback(async () => {
    setIsSavingSuggestions(true);
    setSuggestionsSaveError(null);
    try {
      await updateAutoSuggest(suggestionsDraft);
    } catch (error) {
      setSuggestionsSaveError(error instanceof Error ? error.message : "Failed to save suggestions settings");
    } finally {
      setIsSavingSuggestions(false);
    }
  }, [suggestionsDraft, updateAutoSuggest]);

  // Keep tab selection in sync with open intent from caller
  useEffect(() => {
    if (open) {
      setActiveTab(initialTab ?? "general");
    }
  }, [open, initialTab]);

  // Apply theme class to document
  useEffect(() => {
    const root = document.documentElement;
    if (theme === "light") {
      root.classList.add("light-theme");
    } else {
      root.classList.remove("light-theme");
    }
    localStorage.setItem("theme", theme);
  }, [theme]);

  // Reset delete confirm when switching tabs
  useEffect(() => {
    setShowDeleteConfirm(false);
    setDeleteConfirmText("");
    setShowClearArchivedConfirm(false);
  }, [activeTab]);

  const handleThemeChange = useCallback((newTheme: Theme) => {
    setTheme(newTheme);
  }, []);

  const handleDefaultModelChange = useCallback((modelId: string) => {
    setDefaultModelState(modelId);
    setDefaultModel(modelId);
  }, []);

  const handleDeleteAll = useCallback(async () => {
    if (deleteConfirmText !== "delete all") return;
    await onDeleteAllChats();
    setShowDeleteConfirm(false);
    setDeleteConfirmText("");
  }, [deleteConfirmText, onDeleteAllChats]);

  const handleCancelDelete = useCallback(() => {
    setShowDeleteConfirm(false);
    setDeleteConfirmText("");
  }, []);

  const handleClearArchived = useCallback(async () => {
    if (!onClearArchived) return;
    await onClearArchived();
    setShowClearArchivedConfirm(false);
  }, [onClearArchived]);

  const handleMarkAllAsRead = useCallback(async () => {
    if (!onMarkAllAsRead) return;
    await onMarkAllAsRead();
  }, [onMarkAllAsRead]);

  const activeTabDef = useMemo(
    () => SETTINGS_TABS.find((tab) => tab.value === activeTab) ?? FALLBACK_TAB,
    [activeTab]
  );

  const renderTabContent = () => {
    switch (activeTab) {
      case "general":
        return (
          <div className="space-y-6">
            <SettingsSection title="Appearance">
              <div className="flex gap-2">
                <ChoiceButton
                  label="Dark"
                  icon={<Moon className="h-4 w-4" />}
                  selected={theme === "dark"}
                  onClick={() => handleThemeChange("dark")}
                  testId="theme-dark-button"
                />
                <ChoiceButton
                  label="Light"
                  icon={<Sun className="h-4 w-4" />}
                  selected={theme === "light"}
                  onClick={() => handleThemeChange("light")}
                  testId="theme-light-button"
                />
              </div>
            </SettingsSection>

            <SettingsSection
              title="Chat View"
              description="Choose how messages are displayed in conversations"
            >
              <div className="flex gap-2">
                <ChoiceButton
                  label="Bubble"
                  icon={<MessageCircle className="h-4 w-4" />}
                  selected={viewMode === "bubble"}
                  onClick={() => onViewModeChange("bubble")}
                  testId="view-mode-bubble-button"
                />
                <ChoiceButton
                  label="Compact"
                  icon={<AlignLeft className="h-4 w-4" />}
                  selected={viewMode === "compact"}
                  onClick={() => onViewModeChange("compact")}
                  testId="view-mode-compact-button"
                />
              </div>
              <p className="text-xs text-slate-600 mt-2">
                Compact mode uses full width, ideal for code-heavy conversations
              </p>
            </SettingsSection>

            <SettingsSection title="Keyboard">
              <Button
                variant="secondary"
                onClick={() => {
                  onShowKeyboardShortcuts();
                  onClose();
                }}
                className="w-full justify-start gap-2"
                data-testid="keyboard-shortcuts-button"
              >
                <Keyboard className="h-4 w-4" />
                View Keyboard Shortcuts
              </Button>
            </SettingsSection>
          </div>
        );

      case "ai":
        return (
          <SettingsSection
            title="Default Model"
            description="New chats will use this model by default"
          >
            <ModelSelector
              models={models}
              selectedModel={defaultModel}
              onSelectModel={handleDefaultModelChange}
            />
          </SettingsSection>
        );

      case "agent":
        return (
          <AgentModeSettings
            settings={agentSettings}
            onSettingsChange={setAgentSettings}
            onReset={resetAgentSettings}
          />
        );

      case "tools":
        return (
          <div className="space-y-4">
            <SettingsSection title="YOLO Mode">
              <SettingsSwitchRow
                title="Execute Without Approval"
                description="Execute all tools without asking for approval"
                checked={yoloMode}
                onCheckedChange={handleYoloModeToggle}
                disabled={isLoadingYoloMode || isUpdatingYoloMode}
                tone="yellow"
                testId="yolo-mode-toggle"
              />
              {yoloMode && (
                <div className="mt-3 p-3 rounded-lg bg-yellow-500/10 border border-yellow-500/20">
                  <p className="text-xs text-yellow-400 flex items-center gap-2">
                    <Zap className="h-3.5 w-3.5" />
                    Tools will execute automatically without confirmation
                  </p>
                </div>
              )}
            </SettingsSection>

            <ToolConfiguration
              toolsByScenario={toolsByScenario}
              categories={toolSet?.categories ?? []}
              scenarioStatuses={scenarios}
              isLoading={isLoadingTools}
              isSyncing={isSyncingTools}
              isUpdating={isUpdatingTools}
              error={toolsError?.message}
              onToggleTool={toggleTool}
              onSetApproval={handleSetApproval}
              onSyncTools={syncDiscoveredTools}
              yoloMode={yoloMode}
              onRunTool={handleRunTool}
              enabledCount={enabledTools.length}
              totalCount={toolSet?.tools.length ?? 0}
            />
          </div>
        );

      case "templates":
        return (
          <TemplatesSettingsTab
            templates={templates}
            onEditTemplate={handleEditTemplate}
            onDeleteTemplate={handleDeleteTemplate}
            onResetTemplate={handleResetTemplate}
            modeHistory={modeHistory}
            onClearHistory={clearModeHistory}
            isLoading={isLoadingTemplates}
          />
        );

      case "suggestions":
        return (
          <div className="space-y-6">
            <SettingsSection title="Suggestions Panel">
              <SettingsSwitchRow
                title="Show Suggestions"
                description="Display template suggestions above message input"
                checked={suggestionsVisible}
                onCheckedChange={setSuggestionsVisible}
              />
            </SettingsSection>

            <SettingsSection
              title="AI Merge Model"
              description="Model used when merging your message with a template"
            >
              <ModelSelector
                models={models}
                selectedModel={mergeModel}
                onSelectModel={setMergeModel}
                label="Merge model"
                compact
              />
            </SettingsSection>

            <SettingsSection title="Auto Skill Suggestion">
              {autoSuggestError && (
                <p className="text-xs text-amber-400 mb-3">
                  Could not load persisted settings: {autoSuggestError}
                </p>
              )}
              {suggestionsSaveError && (
                <p className="text-xs text-red-400 mb-3">{suggestionsSaveError}</p>
              )}

              <SettingsSwitchRow
                title="Enable Auto Suggest"
                description="Suggest relevant skills while typing"
                checked={suggestionsDraft.enabled}
                onCheckedChange={(checked) => setSuggestionsDraft((prev) => ({ ...prev, enabled: checked }))}
                disabled={autoSuggestLoading}
              />

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mt-3">
                <SettingsNumberField
                  label="Debounce (ms)"
                  value={suggestionsDraft.debounceMs}
                  min={100}
                  max={10000}
                  disabled={autoSuggestLoading}
                  onChange={(next) => setSuggestionsDraft((prev) => ({ ...prev, debounceMs: next }))}
                />
                <SettingsNumberField
                  label="Throttle (ms)"
                  value={suggestionsDraft.throttleMs}
                  min={1000}
                  max={120000}
                  disabled={autoSuggestLoading}
                  onChange={(next) => setSuggestionsDraft((prev) => ({ ...prev, throttleMs: next }))}
                />
                <SettingsNumberField
                  label="Min input length"
                  value={suggestionsDraft.minInputLength}
                  min={1}
                  max={200}
                  disabled={autoSuggestLoading}
                  onChange={(next) => setSuggestionsDraft((prev) => ({ ...prev, minInputLength: next }))}
                />
                <SettingsNumberField
                  label="Min score (%)"
                  value={suggestionsDraft.minScorePercent}
                  min={0}
                  max={100}
                  disabled={autoSuggestLoading}
                  onChange={(next) => setSuggestionsDraft((prev) => ({ ...prev, minScorePercent: next }))}
                />
                <div className="sm:col-span-2">
                  <SettingsNumberField
                    label="Max suggestions"
                    value={suggestionsDraft.maxSuggestions}
                    min={1}
                    max={20}
                    disabled={autoSuggestLoading}
                    onChange={(next) => setSuggestionsDraft((prev) => ({ ...prev, maxSuggestions: next }))}
                  />
                </div>
              </div>

              <div className="flex items-center justify-end mt-4">
                <Button
                  variant="secondary"
                  onClick={handleSaveSuggestions}
                  disabled={isSavingSuggestions || autoSuggestLoading}
                  data-testid="save-suggestions-settings-button"
                >
                  {isSavingSuggestions ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Saving...
                    </>
                  ) : "Save suggestions settings"}
                </Button>
              </div>
            </SettingsSection>
          </div>
        );

      case "skills":
        return (
          <SkillsSettingsTab
            skills={skills}
            onEditSkill={handleEditSkill}
            onDeleteSkill={handleDeleteSkill}
            isLoading={isLoadingSkills}
            onSyncSkills={handleSyncSkills}
            isSyncing={isSyncingSkills}
          />
        );

      case "data":
        return (
          <div className="space-y-6">
            <SettingsSection
              title="Usage Statistics"
              description="View token usage, costs, and activity across your chats"
            >
              <Button
                variant="secondary"
                onClick={() => {
                  onShowUsageStats();
                  onClose();
                }}
                className="w-full justify-start gap-2"
                data-testid="usage-stats-button"
              >
                <BarChart3 className="h-4 w-4" />
                View Usage Statistics
              </Button>
            </SettingsSection>

            {onMarkAllAsRead && (
              <SettingsSection title="Quick Actions">
                <Button
                  variant="secondary"
                  onClick={handleMarkAllAsRead}
                  disabled={isMarkingAllAsRead}
                  className="w-full justify-start gap-2"
                  data-testid="mark-all-read-button"
                >
                  {isMarkingAllAsRead ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <MailCheck className="h-4 w-4" />
                  )}
                  Mark All as Read
                </Button>
              </SettingsSection>
            )}

            {onClearArchived && (
              <SettingsSection title="Archived Chats">
                <div className="p-4 rounded-lg border border-white/10 bg-white/5">
                  {!showClearArchivedConfirm ? (
                    <div className="flex items-center justify-between gap-4">
                      <div>
                        <p className="text-sm font-medium text-white">Clear Archived Chats</p>
                        <p className="text-xs text-slate-500 mt-1">
                          Permanently delete all archived chats.
                        </p>
                      </div>
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => setShowClearArchivedConfirm(true)}
                        data-testid="clear-archived-settings-button"
                      >
                        <Archive className="h-4 w-4" />
                      </Button>
                    </div>
                  ) : (
                    <div className="space-y-3">
                      <p className="text-sm text-slate-300">
                        Are you sure you want to delete all archived chats?
                      </p>
                      <div className="flex gap-2">
                        <Button
                          variant="secondary"
                          size="sm"
                          onClick={() => setShowClearArchivedConfirm(false)}
                          className="flex-1"
                        >
                          Cancel
                        </Button>
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={handleClearArchived}
                          disabled={isClearingArchived}
                          className="flex-1"
                          data-testid="confirm-clear-archived-settings-button"
                        >
                          {isClearingArchived ? (
                            <Loader2 className="h-4 w-4 animate-spin mr-2" />
                          ) : (
                            <Trash2 className="h-4 w-4 mr-2" />
                          )}
                          Delete
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              </SettingsSection>
            )}

            <section>
              <h3 className="text-sm font-medium text-red-400 mb-3 flex items-center gap-2">
                <AlertTriangle className="h-4 w-4" />
                Danger Zone
              </h3>
              <div className="p-4 rounded-lg border border-red-500/20 bg-red-500/5">
                {!showDeleteConfirm ? (
                  <div className="flex items-center justify-between gap-4">
                    <div>
                      <p className="text-sm font-medium text-white">Delete All Chats</p>
                      <p className="text-xs text-slate-500 mt-1">
                        Permanently delete all chats and messages. This cannot be undone.
                      </p>
                    </div>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => setShowDeleteConfirm(true)}
                      data-testid="delete-all-button"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                ) : (
                  <div className="space-y-3">
                    <p className="text-sm text-slate-300">
                      Type <span className="font-mono text-red-400">delete all</span> to confirm:
                    </p>
                    <Input
                      type="text"
                      value={deleteConfirmText}
                      onChange={(e) => setDeleteConfirmText(e.target.value)}
                      placeholder="delete all"
                      autoFocus
                      className="focus:ring-red-500/50"
                      data-testid="delete-confirm-input"
                    />
                    <div className="flex gap-2">
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={handleCancelDelete}
                        className="flex-1"
                      >
                        Cancel
                      </Button>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={handleDeleteAll}
                        disabled={deleteConfirmText !== "delete all" || isDeletingAll}
                        className="flex-1"
                        data-testid="confirm-delete-all-button"
                      >
                        {isDeletingAll ? "Deleting..." : "Delete All"}
                      </Button>
                    </div>
                  </div>
                )}
              </div>
            </section>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <>
      <Dialog
        open={open}
        onClose={onClose}
        className="max-w-5xl"
        disableEscape={isCreatingSkill || editingSkill !== null}
      >
        <DialogHeader onClose={onClose}>Settings</DialogHeader>
        <DialogBody className="space-y-4">
          <div className="lg:hidden">
            <Dropdown
              className="w-full"
              trigger={(
                <button
                  type="button"
                  className="w-full flex items-center justify-between gap-2 px-3 py-2 bg-white/5 border border-white/10 rounded-lg text-slate-200"
                  data-testid="settings-mobile-tab-selector"
                >
                  <span className="flex items-center gap-2">
                    <activeTabDef.icon className="h-4 w-4" />
                    {activeTabDef.label}
                  </span>
                  <ChevronDown className="h-4 w-4 text-slate-400" />
                </button>
              )}
            >
              {SETTINGS_TABS.map((tab) => (
                <DropdownItem
                  key={tab.value}
                  onClick={() => setActiveTab(tab.value)}
                  className={tab.value === activeTab ? "bg-white/10 text-white" : ""}
                >
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
                  <button
                    key={tab.value}
                    onClick={() => setActiveTab(tab.value)}
                    className={`w-full flex items-center gap-2 px-3 py-2 rounded-md text-sm text-left transition-colors ${
                      isActive
                        ? "bg-indigo-500/20 text-white"
                        : "text-slate-400 hover:text-white hover:bg-white/5"
                    }`}
                    data-testid={`settings-nav-${tab.value}`}
                  >
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

          {/* Manual tool execution dialog (standalone, no chat context) */}
          {selectedToolForRun && (
            <ManualToolDialog
              open={!!selectedToolForRun}
              onClose={() => setSelectedToolForRun(null)}
              tool={selectedToolForRun}
            />
          )}
        </DialogBody>
      </Dialog>

      {/* Skill editor modal - rendered outside Dialog to avoid overflow clipping and z-index issues */}
      <SkillEditorModal
        open={isCreatingSkill || editingSkill !== null}
        onClose={() => {
          setEditingSkill(null);
          setIsCreatingSkill(false);
        }}
        skill={editingSkill ?? undefined}
        onSave={handleSaveSkill}
        // Multi-item mode props for sidebar navigation
        allSkills={skills}
        onSelectSkill={(skill) => {
          setEditingSkill(skill);
          setIsCreatingSkill(false);
        }}
        onSaveAll={async (updates) => {
          const { updateSkills } = await import("../../data/skills");
          await updateSkills(updates);
          const updated = await getAllSkills();
          setSkills(updated);
        }}
      />
    </>
  );
}
