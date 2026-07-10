import { useCallback, useEffect, useRef, useState } from "react";
import {
  ArrowLeft,
  ArrowRight,
  Bot,
  Check,
  ClipboardList,
  FolderOpen,
  Paperclip,
  Play,
  Rocket,
  Settings2,
} from "lucide-react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { Input } from "./ui/input";
import { Label } from "./ui/label";
import { ModelConfigSelector, type ModelSelectionMode } from "./ModelConfigSelector";
import { AttachmentPreview } from "./AttachmentPreview";
import { ScopePathsManager } from "./ScopePathsManager";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./ui/tabs";
import { Textarea } from "./ui/textarea";
import { useAttachments, type PersistedAttachment } from "../hooks/useAttachments";
import { usePersistedFormState } from "../hooks/usePersistedFormState";
import { cn, networkAccessLabel, runnerTypeLabel, runnerTypeToSlug, sandboxModeLabel } from "../lib/utils";
import { catalogInventoryForRunner, policyOptionsForRunner } from "../lib/modelPolicyCatalog";
import type { ModelPolicyCatalog } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";
import type {
  AgentProfile,
  Run,
  RunFormData,
  RunnerStatus,
  RunnerType,
  Task,
  TaskFormData,
} from "../types";
import { RunMode, SandboxMode, RunnerType as RunnerTypeEnum } from "../types";

const RUNNER_TYPES: RunnerType[] = [
  RunnerTypeEnum.CLAUDE_CODE,
  RunnerTypeEnum.CODEX,
  RunnerTypeEnum.OPENCODE,
  RunnerTypeEnum.GROK,
];

interface QuickRunDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  profiles: AgentProfile[];
  runners?: Record<string, RunnerStatus>;
  modelPolicyCatalog?: ModelPolicyCatalog;
  defaultProjectRoot?: string;
  onCreateTask: (task: TaskFormData) => Promise<Task>;
  onCreateRun: (run: RunFormData) => Promise<Run>;
  onRunCreated?: (run: Run) => void;
}

const DEFAULT_MAX_TURNS = 500;
const DEFAULT_TIMEOUT_MINUTES = 120;

interface AgentConfigData {
  mode: "profile" | "custom";
  profileId: string;
  runnerType: RunnerType;
  model: string;
  policyRef: string;
  modelMode: ModelSelectionMode;
  maxTurns: number | string;
  timeoutMinutes: number | string;
  runMode: RunMode;
  skipPermissionPrompt: boolean;
  networkAccess: "none" | "localhost" | "full";
  features?: {
    enableBrowser?: boolean;
  };
  extraFlags?: Record<string, string[]>;
}

type Step = 1 | 2 | 3;

const STEPS: { num: Step; label: string; icon: React.ReactNode }[] = [
  { num: 1, label: "Task", icon: <ClipboardList className="h-4 w-4" /> },
  { num: 2, label: "Agent", icon: <Bot className="h-4 w-4" /> },
  { num: 3, label: "Review", icon: <Rocket className="h-4 w-4" /> },
];

const getModelId = (model: string | { id: string }): string => {
  return typeof model === "string" ? model : model.id;
};

export function QuickRunDialog({
  open,
  onOpenChange,
  profiles,
  runners,
  modelPolicyCatalog,
  defaultProjectRoot,
  onCreateTask,
  onCreateRun,
  onRunCreated,
}: QuickRunDialogProps) {
  const [currentStep, setCurrentStep, clearStepStorage] = usePersistedFormState<Step>("quick-run-step", 1);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const { attachments, addAttachment, removeAttachment, clearAttachments, restoreAttachments, getPersistedAttachments, isUploading, getUploadedIds } = useAttachments();
  const imageInputRef = useRef<HTMLInputElement>(null);

  // Step 1: Task data
  const [taskData, setTaskData, clearTaskStorage] = usePersistedFormState<{
    title: string;
    description: string;
    projectRoot: string;
  }>("quick-run-task", {
    title: "",
    description: "",
    projectRoot: defaultProjectRoot ?? "",
  });
  const [scopePaths, setScopePaths, clearScopeStorage] = usePersistedFormState<string[]>("quick-run-scope", ["."]);

  // Persisted attachment metadata
  const [persistedAttachments, setPersistedAttachments, clearAttachmentStorage] = usePersistedFormState<PersistedAttachment[]>("quick-run-attachments", []);

  // Restore attachments from localStorage on mount
  const attachmentsRestoredRef = useRef(false);
  useEffect(() => {
    if (!attachmentsRestoredRef.current && persistedAttachments.length > 0 && attachments.length === 0) {
      attachmentsRestoredRef.current = true;
      restoreAttachments(persistedAttachments);
    }
  }, [persistedAttachments, attachments.length, restoreAttachments]);

  // Persist attachment metadata whenever attachments change
  useEffect(() => {
    const uploaded = getPersistedAttachments();
    // Only update if there's a meaningful change to avoid infinite loops
    if (JSON.stringify(uploaded) !== JSON.stringify(persistedAttachments)) {
      setPersistedAttachments(uploaded);
    }
  }, [attachments]); // eslint-disable-line react-hooks/exhaustive-deps

  // Sync defaultProjectRoot when it arrives async (e.g. from health fetch)
  useEffect(() => {
    if (defaultProjectRoot && taskData.projectRoot === "") {
      setTaskData((prev) => ({ ...prev, projectRoot: defaultProjectRoot }));
    }
  }, [defaultProjectRoot]); // eslint-disable-line react-hooks/exhaustive-deps

  // Step 2: Agent config
  const [agentConfig, setAgentConfig, clearConfigStorage] = usePersistedFormState<AgentConfigData>("quick-run-config", {
    mode: "custom",
    profileId: "",
    runnerType: RunnerTypeEnum.CLAUDE_CODE,
    model: "",
    policyRef: "",
    modelMode: "default",
    maxTurns: DEFAULT_MAX_TURNS,
    timeoutMinutes: DEFAULT_TIMEOUT_MINUTES,
    runMode: RunMode.SANDBOXED,
    skipPermissionPrompt: true,
    networkAccess: "localhost",
    features: { enableBrowser: false },
    extraFlags: {},
  });
  const [existingSandboxId, setExistingSandboxId, clearSandboxStorage] = usePersistedFormState("quick-run-sandbox", "");

  const getInventoryForRunner = (runnerType: RunnerType) => {
    return catalogInventoryForRunner(modelPolicyCatalog, runnerType);
  };

  const getModelsForRunner = (runnerType: RunnerType) => {
    const inventory = getInventoryForRunner(runnerType);
    if (inventory?.models?.length) {
      return inventory.models;
    }
    const runner = runners?.[runnerType];
    return runner?.supportedModels ?? [];
  };

  const getPoliciesForRunner = (runnerType: RunnerType) => {
    return policyOptionsForRunner(modelPolicyCatalog, runnerType);
  };

  const getSelectedProfile = (): AgentProfile | undefined => {
    return profiles.find((p) => p.id === agentConfig.profileId);
  };

  const clearAllPersistedState = useCallback(() => {
    clearStepStorage();
    clearTaskStorage();
    clearScopeStorage();
    clearConfigStorage();
    clearSandboxStorage();
    clearAttachmentStorage();
    attachmentsRestoredRef.current = false;
  }, [clearStepStorage, clearTaskStorage, clearScopeStorage, clearConfigStorage, clearSandboxStorage, clearAttachmentStorage]);

  const resetForm = () => {
    setCurrentStep(1);
    setError(null);
    setExistingSandboxId("");
    clearAttachments();
    setTaskData({
      title: "",
      description: "",
      projectRoot: defaultProjectRoot ?? "",
    });
    setScopePaths(["."]);
    setAgentConfig({
      mode: "custom",
      profileId: "",
      runnerType: RunnerTypeEnum.CLAUDE_CODE,
      model: "",
      policyRef: "",
      modelMode: "default",
      maxTurns: DEFAULT_MAX_TURNS,
      timeoutMinutes: DEFAULT_TIMEOUT_MINUTES,
      runMode: RunMode.SANDBOXED,
      skipPermissionPrompt: true,
      networkAccess: "localhost",
      features: { enableBrowser: false },
      extraFlags: {},
    });
    setPersistedAttachments([]);
    clearAllPersistedState();
  };

  const handleClose = useCallback(() => {
    // Only reset transient state — persisted draft is preserved for next open
    setError(null);
    setSubmitting(false);
    onOpenChange(false);
  }, [onOpenChange]);

  const canProceedStep1 = (): boolean => {
    // Project root is required; title auto-generated if empty
    return taskData.projectRoot.trim().length > 0;
  };

  const canProceedStep2 = (): boolean => {
    if (agentConfig.mode === "profile") {
      return agentConfig.profileId.length > 0;
    }
    return agentConfig.runnerType !== RunnerTypeEnum.UNSPECIFIED;
  };

  const handleNext = () => {
    if (currentStep === 1 && canProceedStep1()) {
      setCurrentStep(2);
    } else if (currentStep === 2 && canProceedStep2()) {
      setCurrentStep(3);
    }
  };

  const handleBack = () => {
    if (currentStep === 2) {
      setCurrentStep(1);
    } else if (currentStep === 3) {
      setCurrentStep(2);
    }
  };

  const handleStepClick = (step: Step) => {
    setCurrentStep(step);
  };

  const handleImageSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      addAttachment(file);
      e.target.value = "";
    }
  };

  const generateTitle = useCallback((): string => {
    if (taskData.title.trim()) return taskData.title.trim();
    if (taskData.description.trim()) {
      const desc = taskData.description.trim();
      if (desc.length <= 60) return desc;
      const truncated = desc.slice(0, 60);
      const lastSpace = truncated.lastIndexOf(" ");
      return (lastSpace > 20 ? truncated.slice(0, lastSpace) : truncated) + "...";
    }
    const now = new Date();
    const pad = (n: number) => String(n).padStart(2, "0");
    return `Quick run ${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}`;
  }, [taskData.title, taskData.description]);

  const handleStartRun = useCallback(async () => {
    setSubmitting(true);
    setError(null);

    try {
      // Step 1: Create the task
      // Join scope paths with ":" for the backend (which expects a single string)
      // Empty array means "." (current directory / read-only)
      const scopePath = scopePaths.length > 0 ? scopePaths.join(":") : ".";
      const title = generateTitle();

      // Include uploaded image attachments as context attachments
      const uploadedIds = getUploadedIds();
      const imageAttachments = uploadedIds.map((id) => ({
        type: "image",
        attachment_id: id,
        label: "Uploaded image",
      }));

      const task = await onCreateTask({
        title,
        description: taskData.description || undefined,
        scopePath,
        projectRoot: taskData.projectRoot || undefined,
        ...(imageAttachments.length > 0 ? { contextAttachments: imageAttachments } : {}),
      });

      // Step 2: Create the run
      const runRequest: RunFormData = {
        taskId: task.id,
      };

      if (agentConfig.mode === "profile") {
        runRequest.agentProfileId = agentConfig.profileId;
      } else {
        runRequest.runnerType = agentConfig.runnerType;
        if (agentConfig.modelMode === "model" && agentConfig.model.trim() !== "") {
          runRequest.model = agentConfig.model;
        }
        if (agentConfig.modelMode === "policy") {
          runRequest.policyRef = agentConfig.policyRef;
        }
        runRequest.maxTurns = typeof agentConfig.maxTurns === "number" ? agentConfig.maxTurns : DEFAULT_MAX_TURNS;
        runRequest.timeoutMinutes = typeof agentConfig.timeoutMinutes === "number" ? agentConfig.timeoutMinutes : DEFAULT_TIMEOUT_MINUTES;
        runRequest.runMode = agentConfig.runMode;
        runRequest.skipPermissionPrompt = agentConfig.skipPermissionPrompt;
        runRequest.networkAccess = agentConfig.networkAccess;
        if (agentConfig.features?.enableBrowser) {
          runRequest.features = { enableBrowser: true };
        }
        if (agentConfig.extraFlags && Object.keys(agentConfig.extraFlags).length > 0) {
          runRequest.extraFlags = agentConfig.extraFlags;
        }
      }
      if (existingSandboxId.trim() !== "") {
        runRequest.existingSandboxId = existingSandboxId.trim();
      }

      const run = await onCreateRun(runRequest);

      // Success - reset form (clears localStorage draft), close dialog, and notify
      resetForm();
      onOpenChange(false);
      onRunCreated?.(run);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSubmitting(false);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [generateTitle, scopePaths, taskData, agentConfig, existingSandboxId, getUploadedIds, onCreateTask, onCreateRun, onRunCreated, onOpenChange]);

  // Ctrl+Enter shortcut to start run
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
        e.preventDefault();
        handleStartRun();
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, handleStartRun]);

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && handleClose()} fullScreenMobile>
      <DialogContent fullScreenMobile className="sm:max-w-2xl">
        <DialogHeader onClose={handleClose}>
          <DialogTitle className="flex items-center gap-2">
            <Play className="h-5 w-5 text-primary" />
            Quick Run
          </DialogTitle>
          <DialogDescription>
            Create a task and start an agent run in one flow
          </DialogDescription>
        </DialogHeader>

        {/* Stepper */}
        <div className="flex items-center justify-center gap-2 py-4">
          {STEPS.map((step, index) => (
            <div key={step.num} className="flex items-center">
              <button
                type="button"
                onClick={() => handleStepClick(step.num)}
                className={cn(
                  "flex items-center gap-2 rounded-full px-4 py-2 text-sm font-medium transition-colors cursor-pointer",
                  currentStep === step.num
                    ? "bg-primary text-primary-foreground"
                    : currentStep > step.num
                    ? "bg-primary/20 text-primary hover:bg-primary/30"
                    : "bg-muted text-muted-foreground hover:bg-muted/80"
                )}
              >
                {currentStep > step.num ? (
                  <Check className="h-4 w-4" />
                ) : (
                  step.icon
                )}
                <span className="hidden sm:inline">{step.label}</span>
              </button>
              {index < STEPS.length - 1 && (
                <div
                  className={cn(
                    "mx-2 h-0.5 w-8 transition-colors",
                    currentStep > step.num ? "bg-primary" : "bg-muted"
                  )}
                />
              )}
            </div>
          ))}
        </div>

        <DialogBody className="min-h-[320px]">
          {error && (
            <div className="mb-4 rounded-lg border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive">
              {error}
            </div>
          )}

          {/* Step 1: Task */}
          {currentStep === 1 && (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="description">What should the agent do?</Label>
                <Textarea
                  id="description"
                  value={taskData.description}
                  onChange={(e) =>
                    setTaskData({ ...taskData, description: e.target.value })
                  }
                  placeholder="Describe the task for the agent..."
                  rows={4}
                  autoFocus
                />
              </div>

              {/* Image attachments */}
              <div className="space-y-2">
                <label className="text-sm font-medium">Image Attachments</label>
                {attachments.length > 0 && (
                  <AttachmentPreview
                    attachments={attachments}
                    onRemove={removeAttachment}
                    isUploading={isUploading}
                  />
                )}
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => imageInputRef.current?.click()}
                >
                  <Paperclip className="h-4 w-4 mr-2" />
                  Attach Image
                </Button>
                <input
                  ref={imageInputRef}
                  type="file"
                  accept="image/jpeg,image/png,image/gif,image/webp"
                  onChange={handleImageSelect}
                  className="hidden"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="title">Task Title</Label>
                <Input
                  id="title"
                  value={taskData.title}
                  onChange={(e) =>
                    setTaskData({ ...taskData, title: e.target.value })
                  }
                  placeholder={taskData.description ? generateTitle() : "Auto-generated from description"}
                />
                <p className="text-xs text-muted-foreground">
                  Leave blank to auto-generate from description
                </p>
              </div>

              <ScopePathsManager
                projectRoot={taskData.projectRoot}
                onProjectRootChange={(value) =>
                  setTaskData({ ...taskData, projectRoot: value })
                }
                scopePaths={scopePaths}
                onScopePathsChange={setScopePaths}
                defaultProjectRoot={defaultProjectRoot}
                scopePathsHelp="Directories where the agent can make changes. Leave empty for read-only access."
              />

            </div>
          )}

          {/* Step 2: Agent Config */}
          {currentStep === 2 && (
            <div className="space-y-4">
              <Tabs
                value={agentConfig.mode}
                onValueChange={(v) =>
                  setAgentConfig({
                    ...agentConfig,
                    mode: v as "profile" | "custom",
                  })
                }
              >
                <TabsList className="grid w-full grid-cols-2">
                  <TabsTrigger value="profile" className="gap-2">
                    <Settings2 className="h-4 w-4" />
                    Use Profile
                  </TabsTrigger>
                  <TabsTrigger value="custom" className="gap-2">
                    <Bot className="h-4 w-4" />
                    Custom Config
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="profile" className="mt-4 space-y-4">
                  {profiles.length === 0 ? (
                    <div className="rounded-lg border border-dashed border-border p-6 text-center">
                      <Bot className="mx-auto h-10 w-10 text-muted-foreground opacity-50" />
                      <p className="mt-2 text-sm text-muted-foreground">
                        No profiles available. Switch to Custom Config to
                        configure the agent manually.
                      </p>
                    </div>
                  ) : (
                    <div className="space-y-2">
                      <Label htmlFor="profile">Agent Profile *</Label>
                      <select
                        id="profile"
                        value={agentConfig.profileId}
                        onChange={(e) =>
                          setAgentConfig({
                            ...agentConfig,
                            profileId: e.target.value,
                          })
                        }
                        className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                      >
                        <option value="">Select a profile...</option>
                        {profiles.map((profile) => (
                          <option key={profile.id} value={profile.id}>
                            {profile.name} ({runnerTypeLabel(profile.runnerType)})
                          </option>
                        ))}
                      </select>

                      {agentConfig.profileId && (
                        <div className="mt-4 rounded-lg border border-border bg-muted/30 p-4">
                          <h4 className="text-sm font-medium mb-2">
                            Profile Details
                          </h4>
                          {(() => {
                            const profile = getSelectedProfile();
                            if (!profile) return null;
                            return (
                              <div className="space-y-2 text-sm">
                                <div className="flex flex-wrap gap-2">
                                  <Badge variant="secondary">
                                    {runnerTypeLabel(profile.runnerType)}
                                  </Badge>
                                  {profile.model && (
                                    <Badge variant="outline">
                                      {profile.model}
                                    </Badge>
                                  )}
                                  {profile.sandboxConfig?.mode != null && profile.sandboxConfig.mode !== SandboxMode.UNSPECIFIED && (
                                    <Badge variant="outline">Sandbox: {sandboxModeLabel(profile.sandboxConfig.mode)}</Badge>
                                  )}
                                  {profile.sandboxConfig?.manualReview && (
                                    <Badge variant="outline">Manual Review</Badge>
                                  )}
                                  {profile.networkAccess != null && (
                                    <Badge variant="outline">Net: {networkAccessLabel(profile.networkAccess)}</Badge>
                                  )}
                                  {profile.features?.enableBrowser && (
                                    <Badge variant="outline">Browser</Badge>
                                  )}
                                </div>
                                {profile.description && (
                                  <p className="text-muted-foreground">
                                    {profile.description}
                                  </p>
                                )}
                              </div>
                            );
                          })()}
                        </div>
                      )}
                    </div>
                  )}
                </TabsContent>

                <TabsContent value="custom" className="mt-4 space-y-4">
                  <div className="space-y-2">
                  <Label htmlFor="runnerType">Runner Type *</Label>
                  <select
                    id="runnerType"
                    value={String(agentConfig.runnerType)}
                    onChange={(e) => {
                      const newRunnerType = Number(e.target.value) as RunnerType;
                      const availableModels = getModelsForRunner(newRunnerType);
                      const firstModel = availableModels.length > 0 ? getModelId(availableModels[0] ?? "") : "";
                      const firstPolicy = getPoliciesForRunner(newRunnerType)[0]?.ref ?? "";
                      setAgentConfig({
                        ...agentConfig,
                        runnerType: newRunnerType,
                        model:
                          agentConfig.modelMode === "model"
                            ? firstModel
                            : agentConfig.model,
                        policyRef: agentConfig.modelMode === "policy" ? firstPolicy : agentConfig.policyRef,
                      });
                    }}
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    {RUNNER_TYPES.map((type) => (
                      <option key={type} value={type}>
                        {runnerTypeLabel(type)}
                      </option>
                    ))}
                  </select>
                  </div>

                  <ModelConfigSelector
                    value={{
                      mode: agentConfig.modelMode,
                      model: agentConfig.model,
                      policyRef: agentConfig.policyRef,
                    }}
                    onChange={(selection) =>
                      setAgentConfig({
                        ...agentConfig,
                        modelMode: selection.mode,
                        model: selection.model,
                        policyRef: selection.policyRef,
                      })
                    }
                    models={getModelsForRunner(agentConfig.runnerType)}
                    policies={getPoliciesForRunner(agentConfig.runnerType)}
                    label="Model Selection"
                  />

                  <div className="grid gap-4 grid-cols-2">
                    <div className="space-y-2">
                      <Label htmlFor="maxTurns">Max Turns</Label>
                      <Input
                        id="maxTurns"
                        type="number"
                        value={agentConfig.maxTurns}
                        onChange={(e) =>
                          setAgentConfig({
                            ...agentConfig,
                            maxTurns: e.target.value === "" ? "" : parseInt(e.target.value),
                          })
                        }
                        onBlur={() =>
                          setAgentConfig((prev) => ({
                            ...prev,
                            maxTurns:
                              typeof prev.maxTurns === "number" && prev.maxTurns >= 1
                                ? Math.min(prev.maxTurns, 10000)
                                : DEFAULT_MAX_TURNS,
                          }))
                        }
                        min={1}
                        max={10000}
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="timeoutMinutes">Timeout (minutes)</Label>
                      <Input
                        id="timeoutMinutes"
                        type="number"
                        value={agentConfig.timeoutMinutes}
                        onChange={(e) =>
                          setAgentConfig({
                            ...agentConfig,
                            timeoutMinutes: e.target.value === "" ? "" : parseInt(e.target.value),
                          })
                        }
                        onBlur={() =>
                          setAgentConfig((prev) => ({
                            ...prev,
                            timeoutMinutes:
                              typeof prev.timeoutMinutes === "number" && prev.timeoutMinutes >= 1
                                ? Math.min(prev.timeoutMinutes, 1440)
                                : DEFAULT_TIMEOUT_MINUTES,
                          }))
                        }
                        min={1}
                        max={1440}
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                  <Label htmlFor="runMode">Run Mode *</Label>
                  <select
                    id="runMode"
                    value={String(agentConfig.runMode)}
                    onChange={(e) =>
                      setAgentConfig({
                        ...agentConfig,
                        runMode: Number(e.target.value) as RunMode,
                      })
                    }
                    className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                  >
                    <option value={RunMode.SANDBOXED}>Sandboxed — normal audit path (recommended)</option>
                    <option value={RunMode.IN_PLACE}>In-place — operator escape hatch (bypasses provenance + review queue)</option>
                  </select>
                  </div>

                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={agentConfig.skipPermissionPrompt}
                      onChange={(e) =>
                        setAgentConfig({
                          ...agentConfig,
                          skipPermissionPrompt: e.target.checked,
                        })
                      }
                      className="h-4 w-4 rounded border-input"
                    />
                    <span className="text-sm">Skip Permission Prompts</span>
                  </label>

                  <label className="flex items-center gap-2">
                    <span className="text-sm">Network Access</span>
                    <select
                      value={agentConfig.networkAccess}
                      onChange={(e) =>
                        setAgentConfig({
                          ...agentConfig,
                          networkAccess: e.target.value as "none" | "localhost" | "full",
                        })
                      }
                      className="h-8 rounded border border-input bg-background px-2 text-sm"
                    >
                      <option value="none">None</option>
                      <option value="localhost">Localhost</option>
                      <option value="full">Full</option>
                    </select>
                  </label>

                  {agentConfig.runnerType === RunnerTypeEnum.CLAUDE_CODE && (
                    <label className="flex items-center gap-2 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={agentConfig.features?.enableBrowser ?? false}
                        onChange={(e) =>
                          setAgentConfig({
                            ...agentConfig,
                            features: { ...agentConfig.features, enableBrowser: e.target.checked },
                          })
                        }
                        className="h-4 w-4 rounded border-input"
                      />
                      <span className="text-sm">Browser automation (--chrome)</span>
                    </label>
                  )}

                  <div className="space-y-2">
                    <Label>Extra CLI Flags</Label>
                    <Input
                      placeholder="--verbose, --allowedTools"
                      value={(agentConfig.extraFlags?.[runnerTypeToSlug(agentConfig.runnerType)] ?? []).join(", ")}
                      onChange={(e) => {
                        const flags = e.target.value.split(",").map(s => s.trim()).filter(Boolean);
                        setAgentConfig(prev => ({
                          ...prev,
                          extraFlags: { ...prev.extraFlags, [runnerTypeToSlug(prev.runnerType)]: flags },
                        }));
                      }}
                    />
                    <p className="text-xs text-muted-foreground">
                      Flags validated against runner allowlist on save
                    </p>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="existingSandboxId">Reuse Sandbox ID (optional)</Label>
                    <Input
                      id="existingSandboxId"
                      value={existingSandboxId}
                      onChange={(e) => setExistingSandboxId(e.target.value)}
                      placeholder="UUID of an existing sandbox to reuse"
                    />
                    <p className="text-xs text-muted-foreground">
                      Only applies to sandboxed runs. The sandbox must match the task scope.
                    </p>
                  </div>
                </TabsContent>
              </Tabs>
            </div>
          )}

          {/* Step 3: Review */}
          {currentStep === 3 && (
            <div className="space-y-6">
              {/* Task Summary */}
              <div className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center gap-2 mb-3">
                  <ClipboardList className="h-4 w-4 text-primary" />
                  <h4 className="font-medium">Task</h4>
                </div>
                <div className="space-y-2 text-sm">
                  <div className="flex items-center gap-2">
                    <span className="text-muted-foreground">Title: </span>
                    <span className="font-medium">{generateTitle()}</span>
                    {!taskData.title.trim() && (
                      <Badge variant="outline" className="text-xs">auto-generated</Badge>
                    )}
                  </div>
                  {taskData.description && (
                    <div>
                      <span className="text-muted-foreground">Description: </span>
                      <span className="text-muted-foreground line-clamp-2">
                        {taskData.description}
                      </span>
                    </div>
                  )}
                  {taskData.projectRoot && (
                    <div className="flex items-center gap-1">
                      <FolderOpen className="h-3 w-3 text-muted-foreground" />
                      <span className="text-muted-foreground">Project Root: </span>
                      <code className="text-xs bg-muted px-1 py-0.5 rounded">
                        {taskData.projectRoot}
                      </code>
                    </div>
                  )}
                  <div>
                    <span className="text-muted-foreground">Scope Paths: </span>
                    {scopePaths.length > 0 ? (
                      <div className="flex flex-wrap gap-1 mt-1">
                        {scopePaths.map((path) => (
                          <code key={path} className="text-xs bg-muted px-1 py-0.5 rounded">
                            {path}
                          </code>
                        ))}
                      </div>
                    ) : (
                      <span className="text-xs text-muted-foreground italic">Read-only (no write access)</span>
                    )}
                  </div>
                  {attachments.length > 0 && (
                    <div className="flex items-center gap-1">
                      <Paperclip className="h-3 w-3 text-muted-foreground" />
                      <span className="text-muted-foreground">Attachments: </span>
                      <span>{attachments.length} image{attachments.length !== 1 ? "s" : ""}</span>
                      {isUploading && (
                        <Badge variant="outline" className="text-xs">uploading...</Badge>
                      )}
                    </div>
                  )}
                </div>
              </div>

              {/* Agent Summary */}
              <div className="rounded-lg border border-border bg-card p-4">
                <div className="flex items-center gap-2 mb-3">
                  <Bot className="h-4 w-4 text-primary" />
                  <h4 className="font-medium">Agent Configuration</h4>
                </div>
                <div className="space-y-2 text-sm">
                  {agentConfig.mode === "profile" ? (
                    <>
                      <div>
                        <span className="text-muted-foreground">Mode: </span>
                        <Badge variant="secondary">Using Profile</Badge>
                      </div>
                      {(() => {
                        const profile = getSelectedProfile();
                        if (!profile) return null;
                        return (
                          <>
                            <div>
                              <span className="text-muted-foreground">Profile: </span>
                              <span className="font-medium">{profile.name}</span>
                            </div>
                            <div className="flex flex-wrap gap-2 mt-2">
                              <Badge variant="outline">{runnerTypeLabel(profile.runnerType)}</Badge>
                              {profile.model && (
                                <Badge variant="outline">{profile.model}</Badge>
                              )}
                              {profile.sandboxConfig?.mode != null && profile.sandboxConfig.mode !== SandboxMode.UNSPECIFIED && (
                                <Badge variant="outline">Sandbox: {sandboxModeLabel(profile.sandboxConfig.mode)}</Badge>
                              )}
                              {profile.sandboxConfig?.manualReview && (
                                <Badge variant="outline">Manual Review</Badge>
                              )}
                              {profile.networkAccess != null && (
                                <Badge variant="outline">Net: {networkAccessLabel(profile.networkAccess)}</Badge>
                              )}
                              {profile.features?.enableBrowser && (
                                <Badge variant="outline">Browser</Badge>
                              )}
                              {profile.extraFlags && Object.entries(profile.extraFlags).map(([rt, flagList]) =>
                                flagList.flags?.map((flag, i) => (
                                  <Badge key={`${rt}-${i}`} variant="outline">{rt}: {flag}</Badge>
                                ))
                              )}
                            </div>
                          </>
                        );
                      })()}
                    </>
                  ) : (
                    <>
                      <div>
                        <span className="text-muted-foreground">Mode: </span>
                        <Badge variant="secondary">Custom Config</Badge>
                      </div>
                      <div className="flex flex-wrap gap-2 mt-2">
                        <Badge variant="outline">{runnerTypeLabel(agentConfig.runnerType)}</Badge>
                        <Badge variant="outline">{agentConfig.model}</Badge>
                        <Badge variant="outline">
                          {agentConfig.runMode === RunMode.SANDBOXED
                            ? "Sandboxed"
                            : "In-place"}
                        </Badge>
                        <Badge variant="outline">
                          Max {agentConfig.maxTurns || DEFAULT_MAX_TURNS} turns
                        </Badge>
                        <Badge variant="outline">
                          {agentConfig.timeoutMinutes || DEFAULT_TIMEOUT_MINUTES}min timeout
                        </Badge>
                        {agentConfig.features?.enableBrowser && (
                          <Badge variant="outline">Browser</Badge>
                        )}
                        {agentConfig.extraFlags && Object.entries(agentConfig.extraFlags).map(([rt, flags]) =>
                          flags.map((flag, i) => (
                            <Badge key={`${rt}-${i}`} variant="outline">{rt}: {flag}</Badge>
                          ))
                        )}
                      </div>
                    </>
                  )}
                  {existingSandboxId.trim() !== "" && (
                    <div className="flex items-center gap-1">
                      <span className="text-muted-foreground">Sandbox: </span>
                      <code className="text-xs bg-muted px-1 py-0.5 rounded">
                        {existingSandboxId.trim()}
                      </code>
                    </div>
                  )}
                </div>
              </div>

              <div className="rounded-lg border border-primary/50 bg-primary/5 p-4">
                <p className="text-sm text-center">
                  Ready to start the agent run. Click{" "}
                  <span className="font-medium text-primary">Start Run</span> to
                  begin.
                </p>
              </div>
            </div>
          )}
        </DialogBody>

        <DialogFooter>
          {currentStep > 1 && (
            <Button
              type="button"
              variant="outline"
              onClick={handleBack}
              disabled={submitting}
              className="gap-2"
            >
              <ArrowLeft className="h-4 w-4" />
              Back
            </Button>
          )}
          <div className="flex-1" />
          <div className="flex items-center gap-2">
            {currentStep < 3 && (
              <>
                <Button
                  type="button"
                  variant="secondary"
                  onClick={handleStartRun}
                  disabled={submitting || isUploading || !canProceedStep1()}
                  className="gap-2"
                >
                  <Play className="h-4 w-4" />
                  {submitting ? "Starting..." : "Start Run"}
                </Button>
                <Button
                  type="button"
                  onClick={handleNext}
                  disabled={
                    (currentStep === 1 && !canProceedStep1()) ||
                    (currentStep === 2 && !canProceedStep2())
                  }
                  className="gap-2"
                >
                  Next
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </>
            )}
            {currentStep === 3 && (
              <Button
                type="button"
                onClick={handleStartRun}
                disabled={submitting || isUploading}
                className="gap-2"
              >
                <Play className="h-4 w-4" />
                {submitting ? "Starting..." : "Start Run"}
              </Button>
            )}
            <span className="text-xs text-muted-foreground hidden sm:inline">Ctrl+Enter</span>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
