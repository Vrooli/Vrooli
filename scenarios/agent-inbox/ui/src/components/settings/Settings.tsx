import { useState, useCallback, useEffect } from "react";
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
} from "lucide-react";
import { Dialog, DialogHeader, DialogBody } from "../ui/dialog";
import { Button } from "../ui/button";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../ui/tabs";
import { ModelSelector } from "./ModelSelector";
import { ToolConfiguration } from "./ToolConfiguration";
import { TemplatesSettingsTab } from "./TemplatesSettingsTab";
import { SkillsSettingsTab } from "./SkillsSettingsTab";
import { SkillEditorModal } from "./SkillEditorModal";
import { AgentModeSettings } from "./AgentModeSettings";
import { ManualToolDialog } from "../tools/ManualToolDialog";
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
  } = useYoloMode(open && activeTab === "ai");

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
  } = useTools({ enabled: open && (activeTab === "ai" || activeTab === "tools") });

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
      // Create new skill
      setIsCreatingSkill(true);
      setEditingSkill(null);
    } else {
      // Edit existing skill
      setIsCreatingSkill(false);
      setEditingSkill(skill);
    }
  }, []);

  const handleSaveSkill = useCallback(async (
    skillData: Omit<Skill, "id" | "createdAt" | "updatedAt">
  ) => {
    if (isCreatingSkill) {
      // Create new skill
      await createSkillFromAPI(skillData);
    } else if (editingSkill) {
      // Update existing skill - goes directly to prompt-manager
      await updateSkillFromAPI(editingSkill.id, skillData);
    }
    // Refresh skills list
    const updated = await getAllSkills();
    setSkills(updated);
    setEditingSkill(null);
    setIsCreatingSkill(false);
  }, [isCreatingSkill, editingSkill]);

  const handleYoloModeToggle = useCallback(() => {
    setYoloMode(!yoloMode);
  }, [yoloMode, setYoloMode]);

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

  return (
    <>
    <Dialog open={open} onClose={onClose} className="max-w-2xl" disableEscape={isCreatingSkill || editingSkill !== null}>
      <DialogHeader onClose={onClose}>Settings</DialogHeader>
      <DialogBody className="space-y-4">
        <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as SettingsTab)}>
          <TabsList className="mb-4">
            <TabsTrigger value="general">
              <span className="flex items-center gap-2">
                <Settings2 className="h-4 w-4" />
                General
              </span>
            </TabsTrigger>
            <TabsTrigger value="ai">
              <span className="flex items-center gap-2">
                <Cpu className="h-4 w-4" />
                AI
              </span>
            </TabsTrigger>
            <TabsTrigger value="agent">
              <span className="flex items-center gap-2">
                <Bot className="h-4 w-4" />
                Agent
              </span>
            </TabsTrigger>
            <TabsTrigger value="tools">
              <span className="flex items-center gap-2">
                <Wrench className="h-4 w-4" />
                Tools
              </span>
            </TabsTrigger>
            <TabsTrigger value="templates">
              <span className="flex items-center gap-2">
                <Lightbulb className="h-4 w-4" />
                Templates
              </span>
            </TabsTrigger>
            <TabsTrigger value="suggestions">
              <span className="flex items-center gap-2">
                <Lightbulb className="h-4 w-4" />
                Suggestions
              </span>
            </TabsTrigger>
            <TabsTrigger value="skills">
              <span className="flex items-center gap-2">
                <BookOpen className="h-4 w-4" />
                Skills
              </span>
            </TabsTrigger>
            <TabsTrigger value="data">
              <span className="flex items-center gap-2">
                <Database className="h-4 w-4" />
                Data
              </span>
            </TabsTrigger>
          </TabsList>

          {/* General Tab */}
          <TabsContent value="general" className="space-y-6">
            {/* Appearance Section */}
            <section>
              <h3 className="text-sm font-medium text-slate-300 mb-3">Appearance</h3>
              <div className="flex gap-2">
                <button
                  onClick={() => handleThemeChange("dark")}
                  className={`flex-1 flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors ${
                    theme === "dark"
                      ? "bg-indigo-500/20 border-indigo-500 text-white"
                      : "bg-white/5 border-white/10 text-slate-400 hover:text-white hover:border-white/20"
                  }`}
                  data-testid="theme-dark-button"
                >
                  <Moon className="h-4 w-4" />
                  <span className="text-sm">Dark</span>
                </button>
                <button
                  onClick={() => handleThemeChange("light")}
                  className={`flex-1 flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors ${
                    theme === "light"
                      ? "bg-indigo-500/20 border-indigo-500 text-white"
                      : "bg-white/5 border-white/10 text-slate-400 hover:text-white hover:border-white/20"
                  }`}
                  data-testid="theme-light-button"
                >
                  <Sun className="h-4 w-4" />
                  <span className="text-sm">Light</span>
                </button>
              </div>
            </section>

            {/* Chat View Section */}
            <section>
              <h3 className="text-sm font-medium text-slate-300 mb-3">Chat View</h3>
              <p className="text-xs text-slate-500 mb-3">
                Choose how messages are displayed in conversations
              </p>
              <div className="flex gap-2">
                <button
                  onClick={() => onViewModeChange("bubble")}
                  className={`flex-1 flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors ${
                    viewMode === "bubble"
                      ? "bg-indigo-500/20 border-indigo-500 text-white"
                      : "bg-white/5 border-white/10 text-slate-400 hover:text-white hover:border-white/20"
                  }`}
                  data-testid="view-mode-bubble-button"
                >
                  <MessageCircle className="h-4 w-4" />
                  <span className="text-sm">Bubble</span>
                </button>
                <button
                  onClick={() => onViewModeChange("compact")}
                  className={`flex-1 flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors ${
                    viewMode === "compact"
                      ? "bg-indigo-500/20 border-indigo-500 text-white"
                      : "bg-white/5 border-white/10 text-slate-400 hover:text-white hover:border-white/20"
                  }`}
                  data-testid="view-mode-compact-button"
                >
                  <AlignLeft className="h-4 w-4" />
                  <span className="text-sm">Compact</span>
                </button>
              </div>
              <p className="text-xs text-slate-600 mt-2">
                Compact mode uses full width, ideal for code-heavy conversations
              </p>
            </section>

            {/* Keyboard Shortcuts Section */}
            <section>
              <h3 className="text-sm font-medium text-slate-300 mb-3">Keyboard</h3>
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
            </section>
          </TabsContent>

          {/* AI Tab */}
          <TabsContent value="ai" className="space-y-6">
            {/* Default Model Section */}
            <section>
              <h3 className="text-sm font-medium text-slate-300 mb-3">Default Model</h3>
              <p className="text-xs text-slate-500 mb-3">
                New chats will use this model by default
              </p>
              <ModelSelector
                models={models}
                selectedModel={defaultModel}
                onSelectModel={handleDefaultModelChange}
              />
            </section>

          </TabsContent>

          {/* Agent Tab */}
          <TabsContent value="agent" className="space-y-6">
            <AgentModeSettings
              settings={agentSettings}
              onSettingsChange={setAgentSettings}
              onReset={resetAgentSettings}
            />
          </TabsContent>

          {/* Tools Tab */}
          <TabsContent value="tools" className="space-y-4">
            {/* YOLO Mode Section */}
            <section className="p-3 rounded-lg border border-white/10 bg-white/5">
              <div className="flex items-center justify-between">
                <div className="flex-1">
                  <h3 className="text-sm font-medium text-slate-300">YOLO Mode</h3>
                  <p className="text-xs text-slate-500 mt-1">
                    Execute all tools without asking for approval
                  </p>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={yoloMode}
                    onChange={handleYoloModeToggle}
                    disabled={isLoadingYoloMode || isUpdatingYoloMode}
                    className="sr-only peer"
                    data-testid="yolo-mode-toggle"
                  />
                  <div className="w-11 h-6 bg-slate-700 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-indigo-500/50 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-yellow-500" />
                </label>
              </div>
              {yoloMode && (
                <div className="mt-3 p-3 rounded-lg bg-yellow-500/10 border border-yellow-500/20">
                  <p className="text-xs text-yellow-400 flex items-center gap-2">
                    <Zap className="h-3.5 w-3.5" />
                    Tools will execute automatically without confirmation
                  </p>
                </div>
              )}
            </section>

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
          </TabsContent>

          {/* Templates Tab */}
          <TabsContent value="templates" className="space-y-6">
            <TemplatesSettingsTab
              templates={templates}
              onEditTemplate={handleEditTemplate}
              onDeleteTemplate={handleDeleteTemplate}
              onResetTemplate={handleResetTemplate}
              modeHistory={modeHistory}
              onClearHistory={clearModeHistory}
              isLoading={isLoadingTemplates}
            />
          </TabsContent>

          {/* Suggestions Tab */}
          <TabsContent value="suggestions" className="space-y-6">
            <section>
              <h3 className="text-sm font-medium text-slate-300 mb-3">Suggestions Panel</h3>
              <div className="flex items-center justify-between p-3 bg-white/5 border border-white/10 rounded-lg">
                <div>
                  <p className="text-sm text-white">Show Suggestions</p>
                  <p className="text-xs text-slate-500">Display template suggestions above message input</p>
                </div>
                <button
                  onClick={() => setSuggestionsVisible(!suggestionsVisible)}
                  className={`relative w-11 h-6 rounded-full transition-colors ${
                    suggestionsVisible ? "bg-indigo-600" : "bg-slate-600"
                  }`}
                >
                  <span
                    className={`absolute top-1 left-1 w-4 h-4 bg-white rounded-full transition-transform ${
                      suggestionsVisible ? "translate-x-5" : ""
                    }`}
                  />
                </button>
              </div>
            </section>

            <section>
              <h3 className="text-sm font-medium text-slate-300 mb-3">AI Merge Model</h3>
              <p className="text-xs text-slate-500 mb-3">Model used when merging your message with a template</p>
              <ModelSelector
                models={models}
                selectedModel={mergeModel}
                onSelectModel={setMergeModel}
                label="Merge model"
                compact
              />
            </section>

            <section className="space-y-3">
              <h3 className="text-sm font-medium text-slate-300">Auto Skill Suggestion</h3>
              {autoSuggestError && (
                <p className="text-xs text-amber-400">Could not load persisted settings: {autoSuggestError}</p>
              )}
              {suggestionsSaveError && (
                <p className="text-xs text-red-400">{suggestionsSaveError}</p>
              )}

              <div className="flex items-center justify-between p-3 bg-white/5 border border-white/10 rounded-lg">
                <div>
                  <p className="text-sm text-white">Enable Auto Suggest</p>
                  <p className="text-xs text-slate-500">Suggest relevant skills while typing</p>
                </div>
                <button
                  disabled={autoSuggestLoading}
                  onClick={() => setSuggestionsDraft((prev) => ({ ...prev, enabled: !prev.enabled }))}
                  className={`relative w-11 h-6 rounded-full transition-colors disabled:opacity-50 ${
                    suggestionsDraft.enabled ? "bg-indigo-600" : "bg-slate-600"
                  }`}
                >
                  <span
                    className={`absolute top-1 left-1 w-4 h-4 bg-white rounded-full transition-transform ${
                      suggestionsDraft.enabled ? "translate-x-5" : ""
                    }`}
                  />
                </button>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <label className="text-xs text-slate-400">
                  Debounce (ms)
                  <input
                    type="number"
                    min={100}
                    max={10000}
                    value={suggestionsDraft.debounceMs}
                    onChange={(e) => setSuggestionsDraft((prev) => ({ ...prev, debounceMs: Number(e.target.value) || 0 }))}
                    className="mt-1 w-full bg-slate-800 border border-white/10 rounded-lg px-3 py-2 text-sm text-white"
                  />
                </label>
                <label className="text-xs text-slate-400">
                  Throttle (ms)
                  <input
                    type="number"
                    min={1000}
                    max={120000}
                    value={suggestionsDraft.throttleMs}
                    onChange={(e) => setSuggestionsDraft((prev) => ({ ...prev, throttleMs: Number(e.target.value) || 0 }))}
                    className="mt-1 w-full bg-slate-800 border border-white/10 rounded-lg px-3 py-2 text-sm text-white"
                  />
                </label>
                <label className="text-xs text-slate-400">
                  Min input length
                  <input
                    type="number"
                    min={1}
                    max={200}
                    value={suggestionsDraft.minInputLength}
                    onChange={(e) => setSuggestionsDraft((prev) => ({ ...prev, minInputLength: Number(e.target.value) || 0 }))}
                    className="mt-1 w-full bg-slate-800 border border-white/10 rounded-lg px-3 py-2 text-sm text-white"
                  />
                </label>
                <label className="text-xs text-slate-400">
                  Min score (%)
                  <input
                    type="number"
                    min={0}
                    max={100}
                    value={suggestionsDraft.minScorePercent}
                    onChange={(e) => setSuggestionsDraft((prev) => ({ ...prev, minScorePercent: Number(e.target.value) || 0 }))}
                    className="mt-1 w-full bg-slate-800 border border-white/10 rounded-lg px-3 py-2 text-sm text-white"
                  />
                </label>
                <label className="text-xs text-slate-400 sm:col-span-2">
                  Max suggestions
                  <input
                    type="number"
                    min={1}
                    max={20}
                    value={suggestionsDraft.maxSuggestions}
                    onChange={(e) => setSuggestionsDraft((prev) => ({ ...prev, maxSuggestions: Number(e.target.value) || 0 }))}
                    className="mt-1 w-full bg-slate-800 border border-white/10 rounded-lg px-3 py-2 text-sm text-white"
                  />
                </label>
              </div>

              <div className="flex items-center justify-end">
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
            </section>
          </TabsContent>



          {/* Skills Tab */}
          <TabsContent value="skills" className="space-y-6">
            <SkillsSettingsTab
              skills={skills}
              onEditSkill={handleEditSkill}
              onDeleteSkill={handleDeleteSkill}
              isLoading={isLoadingSkills}
              onSyncSkills={handleSyncSkills}
              isSyncing={isSyncingSkills}
            />
          </TabsContent>

          {/* Data Tab */}
          <TabsContent value="data" className="space-y-6">
            {/* Usage Statistics Section */}
            <section>
              <h3 className="text-sm font-medium text-slate-300 mb-3">Usage Statistics</h3>
              <p className="text-xs text-slate-500 mb-3">
                View token usage, costs, and activity across your chats
              </p>
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
            </section>

            {/* Mark All as Read */}
            {onMarkAllAsRead && (
              <section>
                <h3 className="text-sm font-medium text-slate-300 mb-3">Quick Actions</h3>
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
              </section>
            )}

            {/* Clear Archived Chats */}
            {onClearArchived && (
              <section>
                <h3 className="text-sm font-medium text-slate-300 mb-3">Archived Chats</h3>
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
              </section>
            )}

            {/* Danger Zone */}
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
                    <input
                      type="text"
                      value={deleteConfirmText}
                      onChange={(e) => setDeleteConfirmText(e.target.value)}
                      placeholder="delete all"
                      className="w-full bg-white/5 border border-white/10 rounded-lg px-3 py-2 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-red-500/50"
                      autoFocus
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
          </TabsContent>
        </Tabs>

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
        // Import batch save function
        const { updateSkills } = await import("../../data/skills");
        await updateSkills(updates);
        // Refresh skills list
        const updated = await getAllSkills();
        setSkills(updated);
      }}
    />
  </>
  );
}
