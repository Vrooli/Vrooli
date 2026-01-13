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
} from "lucide-react";
import { Dialog, DialogHeader, DialogBody } from "../ui/dialog";
import { Button } from "../ui/button";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../ui/tabs";
import { ModelSelector } from "./ModelSelector";
import { ToolConfiguration } from "./ToolConfiguration";
import { TemplatesSettingsTab } from "./TemplatesSettingsTab";
import { SkillsSettingsTab } from "./SkillsSettingsTab";
import { SkillEditorModal } from "./SkillEditorModal";
import { ManualToolDialog } from "../tools/ManualToolDialog";
import { useTools } from "../../hooks/useTools";
import { useYoloMode } from "../../hooks/useSettings";
import { useSuggestionsSettings } from "../../hooks/useSuggestionsSettings";
import { useModeHistory } from "../../hooks/useModeHistory";
import {
  getAllTemplates,
  deleteTemplate as deleteTemplateFromAPI,
  resetTemplate as resetTemplateFromAPI,
} from "../../data/templates";
import {
  getAllSkills,
  deleteSkill as deleteSkillFromAPI,
  resetSkill as resetSkillFromAPI,
  createSkill as createSkillFromAPI,
  updateSkill as updateSkillFromAPI,
  invalidateSkillsCache,
} from "../../data/skills";
import {
  updateDefaultSkill as updateDefaultSkillFromAPI,
} from "../../lib/api";
import type { Model, ApprovalOverride, EffectiveTool } from "../../lib/api";
import type { TemplateWithSource, SkillWithSource, Skill, SkillSource } from "../../lib/types/templates";

export type Theme = "dark" | "light";
export type ViewMode = "bubble" | "compact";
export type SettingsTab = "general" | "ai" | "templates" | "skills" | "data";

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
  onShowKeyboardShortcuts: () => void;
  onShowUsageStats: () => void;
  models: Model[];
  viewMode: ViewMode;
  onViewModeChange: (mode: ViewMode) => void;
  onEditTemplate?: (template: TemplateWithSource) => void;
}

export function Settings({
  open,
  onClose,
  onDeleteAllChats,
  isDeletingAll,
  onShowKeyboardShortcuts,
  onShowUsageStats,
  models,
  viewMode,
  onViewModeChange,
  onEditTemplate,
}: SettingsProps) {
  const [activeTab, setActiveTab] = useState<SettingsTab>("general");
  const [theme, setTheme] = useState<Theme>(() => {
    if (typeof window !== "undefined") {
      return (localStorage.getItem("theme") as Theme) || "dark";
    }
    return "dark";
  });
  const [defaultModel, setDefaultModelState] = useState<string>(getDefaultModel);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteConfirmText, setDeleteConfirmText] = useState("");
  const [showTools, setShowTools] = useState(false);
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
    isRefreshing: isRefreshingTools,
    isUpdating: isUpdatingTools,
    error: toolsError,
    toggleTool,
    setApproval,
    refreshToolRegistry,
  } = useTools({ enabled: open && activeTab === "ai" });

  // Suggestions and templates settings
  const {
    visible: suggestionsVisible,
    setVisible: setSuggestionsVisible,
    mergeModel,
    setMergeModel,
  } = useSuggestionsSettings();

  const { history: modeHistory, clearHistory: clearModeHistory } = useModeHistory();

  // Templates state - refresh when tab changes to templates
  const [templates, setTemplates] = useState<TemplateWithSource[]>([]);
  const [isLoadingTemplates, setIsLoadingTemplates] = useState(false);

  // Skills state - refresh when tab changes to skills
  const [skills, setSkills] = useState<SkillWithSource[]>([]);
  const [isLoadingSkills, setIsLoadingSkills] = useState(false);
  const [editingSkill, setEditingSkill] = useState<SkillWithSource | null>(null);
  const [isCreatingSkill, setIsCreatingSkill] = useState(false);

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
    onEditTemplate?.(template);
    onClose();
  }, [onEditTemplate, onClose]);

  // Skill handlers
  const handleDeleteSkill = useCallback(async (skillId: string) => {
    await deleteSkillFromAPI(skillId);
    const updated = await getAllSkills();
    setSkills(updated);
  }, []);

  const handleResetSkill = useCallback(async (skillId: string) => {
    await resetSkillFromAPI(skillId);
    const updated = await getAllSkills();
    setSkills(updated);
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
    skillData: Omit<Skill, "id" | "createdAt" | "updatedAt">,
    options?: { applyToDefault?: boolean }
  ) => {
    if (isCreatingSkill) {
      // Create new skill
      await createSkillFromAPI(skillData);
    } else if (editingSkill) {
      // Update existing skill
      if (options?.applyToDefault && editingSkill.source === "default") {
        // Apply changes to the default skill
        await updateDefaultSkillFromAPI(editingSkill.id, skillData);
        invalidateSkillsCache();
      } else {
        // Create user override or update user skill
        await updateSkillFromAPI(editingSkill.id, skillData);
      }
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

  return (
    <>
    <Dialog open={open} onClose={onClose} className="max-w-lg">
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
            <TabsTrigger value="templates">
              <span className="flex items-center gap-2">
                <Lightbulb className="h-4 w-4" />
                Templates
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

            {/* YOLO Mode Section */}
            <section>
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

            {/* Tools Configuration Section */}
            <section>
              <div className="flex items-center justify-between mb-3">
                <h3 className="text-sm font-medium text-slate-300">AI Tools</h3>
                <span className="text-xs text-slate-500">
                  {enabledTools.length} of {toolSet?.tools.length ?? 0} enabled
                </span>
              </div>
              {!showTools ? (
                <Button
                  variant="secondary"
                  onClick={() => setShowTools(true)}
                  className="w-full justify-start gap-2"
                  data-testid="configure-tools-button"
                >
                  <Wrench className="h-4 w-4" />
                  Configure Default Tools
                </Button>
              ) : (
                <div className="space-y-3">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setShowTools(false)}
                    className="text-slate-400"
                  >
                    Hide tool configuration
                  </Button>
                  <ToolConfiguration
                    toolsByScenario={toolsByScenario}
                    categories={toolSet?.categories ?? []}
                    scenarioStatuses={scenarios}
                    isLoading={isLoadingTools}
                    isRefreshing={isRefreshingTools}
                    isUpdating={isUpdatingTools}
                    error={toolsError?.message}
                    onToggleTool={toggleTool}
                    onSetApproval={handleSetApproval}
                    onRefresh={refreshToolRegistry}
                    yoloMode={yoloMode}
                    onRunTool={handleRunTool}
                  />
                </div>
              )}
            </section>
          </TabsContent>

          {/* Templates Tab */}
          <TabsContent value="templates" className="space-y-6">
            <TemplatesSettingsTab
              suggestionsVisible={suggestionsVisible}
              onToggleSuggestions={setSuggestionsVisible}
              mergeModel={mergeModel}
              onMergeModelChange={setMergeModel}
              templates={templates}
              onEditTemplate={handleEditTemplate}
              onDeleteTemplate={handleDeleteTemplate}
              onResetTemplate={handleResetTemplate}
              modeHistory={modeHistory}
              onClearHistory={clearModeHistory}
              models={models}
              isLoading={isLoadingTemplates}
            />
          </TabsContent>

          {/* Skills Tab */}
          <TabsContent value="skills" className="space-y-6">
            <SkillsSettingsTab
              skills={skills}
              onEditSkill={handleEditSkill}
              onDeleteSkill={handleDeleteSkill}
              onResetSkill={handleResetSkill}
              isLoading={isLoadingSkills}
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
      skillSource={editingSkill?.source}
      onSave={handleSaveSkill}
    />
  </>
  );
}
