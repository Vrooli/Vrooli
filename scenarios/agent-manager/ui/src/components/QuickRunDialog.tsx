import { useState } from "react";
import {
  ArrowLeft,
  ArrowRight,
  Bot,
  Check,
  ClipboardList,
  FolderOpen,
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
import { ModelSelector } from "./ModelSelector";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./ui/tabs";
import { Textarea } from "./ui/textarea";
import { cn } from "../lib/utils";
import type {
  AgentProfile,
  CreateRunRequest,
  CreateTaskRequest,
  Run,
  RunnerStatus,
  RunnerType,
  Task,
} from "../types";

const RUNNER_TYPES: RunnerType[] = ["claude-code", "codex", "opencode"];

const DEFAULT_MODELS: Record<RunnerType, string> = {
  "claude-code": "sonnet",
  "codex": "o4-mini",
  "opencode": "anthropic/claude-sonnet-4-5",
};

// Convert minutes to nanoseconds for Go's time.Duration
const minutesToNanoseconds = (minutes: number): number =>
  minutes * 60 * 1_000_000_000;

interface QuickRunDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  profiles: AgentProfile[];
  runners?: Record<string, RunnerStatus>;
  onCreateTask: (task: CreateTaskRequest) => Promise<Task>;
  onCreateRun: (run: CreateRunRequest) => Promise<Run>;
  onRunCreated?: (run: Run) => void;
}

interface TaskFormData {
  title: string;
  description: string;
  scopePath: string;
  projectRoot: string;
}

interface AgentConfigData {
  mode: "profile" | "custom";
  profileId: string;
  runnerType: RunnerType;
  model: string;
  maxTurns: number;
  timeoutMinutes: number;
  runMode: "sandboxed" | "in_place";
  skipPermissionPrompt: boolean;
}

type Step = 1 | 2 | 3;

const STEPS: { num: Step; label: string; icon: React.ReactNode }[] = [
  { num: 1, label: "Task", icon: <ClipboardList className="h-4 w-4" /> },
  { num: 2, label: "Agent", icon: <Bot className="h-4 w-4" /> },
  { num: 3, label: "Review", icon: <Rocket className="h-4 w-4" /> },
];

export function QuickRunDialog({
  open,
  onOpenChange,
  profiles,
  runners,
  onCreateTask,
  onCreateRun,
  onRunCreated,
}: QuickRunDialogProps) {
  const [currentStep, setCurrentStep] = useState<Step>(1);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Step 1: Task data
  const [taskData, setTaskData] = useState<TaskFormData>({
    title: "",
    description: "",
    scopePath: ".",
    projectRoot: "",
  });

  // Step 2: Agent config
  const [agentConfig, setAgentConfig] = useState<AgentConfigData>({
    mode: profiles.length > 0 ? "profile" : "custom",
    profileId: "",
    runnerType: "claude-code",
    model: "sonnet",
    maxTurns: 100,
    timeoutMinutes: 30,
    runMode: "sandboxed",
    skipPermissionPrompt: true,
  });

  const getModelsForRunner = (runnerType: RunnerType): string[] => {
    const runner = runners?.[runnerType];
    return runner?.capabilities?.SupportedModels ?? [];
  };

  const getDefaultModelForRunner = (runnerType: RunnerType): string => {
    return DEFAULT_MODELS[runnerType] ?? "";
  };

  const getSelectedProfile = (): AgentProfile | undefined => {
    return profiles.find((p) => p.id === agentConfig.profileId);
  };

  const resetForm = () => {
    setCurrentStep(1);
    setError(null);
    setTaskData({
      title: "",
      description: "",
      scopePath: ".",
      projectRoot: "",
    });
    setAgentConfig({
      mode: profiles.length > 0 ? "profile" : "custom",
      profileId: "",
      runnerType: "claude-code",
      model: "sonnet",
      maxTurns: 100,
      timeoutMinutes: 30,
      runMode: "sandboxed",
      skipPermissionPrompt: true,
    });
  };

  const handleClose = () => {
    resetForm();
    onOpenChange(false);
  };

  const canProceedStep1 = (): boolean => {
    return taskData.title.trim().length > 0 && taskData.scopePath.trim().length > 0;
  };

  const canProceedStep2 = (): boolean => {
    if (agentConfig.mode === "profile") {
      return agentConfig.profileId.length > 0;
    }
    return agentConfig.runnerType.length > 0;
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
    // Allow clicking to go back to previous steps
    if (step < currentStep) {
      setCurrentStep(step);
    }
    // Allow clicking to go forward only if current step is valid
    if (step === 2 && currentStep === 1 && canProceedStep1()) {
      setCurrentStep(2);
    }
    if (step === 3 && currentStep === 2 && canProceedStep2()) {
      setCurrentStep(3);
    }
  };

  const handleStartRun = async () => {
    setSubmitting(true);
    setError(null);

    try {
      // Step 1: Create the task
      const task = await onCreateTask({
        title: taskData.title,
        description: taskData.description || undefined,
        scopePath: taskData.scopePath,
        projectRoot: taskData.projectRoot || undefined,
      });

      // Step 2: Create the run
      const runRequest: CreateRunRequest = {
        taskId: task.id,
      };

      if (agentConfig.mode === "profile") {
        runRequest.agentProfileId = agentConfig.profileId;
      } else {
        runRequest.runnerType = agentConfig.runnerType;
        runRequest.model = agentConfig.model;
        runRequest.maxTurns = agentConfig.maxTurns;
        runRequest.timeout = minutesToNanoseconds(agentConfig.timeoutMinutes);
        runRequest.runMode = agentConfig.runMode;
        runRequest.skipPermissionPrompt = agentConfig.skipPermissionPrompt;
      }

      const run = await onCreateRun(runRequest);

      // Success - close dialog and notify
      handleClose();
      onRunCreated?.(run);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(isOpen) => !isOpen && handleClose()}>
      <DialogContent className="max-w-2xl">
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
                  "flex items-center gap-2 rounded-full px-4 py-2 text-sm font-medium transition-colors",
                  currentStep === step.num
                    ? "bg-primary text-primary-foreground"
                    : currentStep > step.num
                    ? "bg-primary/20 text-primary cursor-pointer hover:bg-primary/30"
                    : "bg-muted text-muted-foreground"
                )}
                disabled={step.num > currentStep}
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
                <Label htmlFor="title">Task Title *</Label>
                <Input
                  id="title"
                  value={taskData.title}
                  onChange={(e) =>
                    setTaskData({ ...taskData, title: e.target.value })
                  }
                  placeholder="e.g., Fix authentication bug"
                  autoFocus
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={taskData.description}
                  onChange={(e) =>
                    setTaskData({ ...taskData, description: e.target.value })
                  }
                  placeholder="Detailed instructions for the agent..."
                  rows={4}
                />
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="scopePath">Scope Path *</Label>
                  <Input
                    id="scopePath"
                    value={taskData.scopePath}
                    onChange={(e) =>
                      setTaskData({ ...taskData, scopePath: e.target.value })
                    }
                    placeholder="e.g., src/auth"
                  />
                  <p className="text-xs text-muted-foreground">
                    Directory scope where the agent can operate
                  </p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="projectRoot">Project Root</Label>
                  <Input
                    id="projectRoot"
                    value={taskData.projectRoot}
                    onChange={(e) =>
                      setTaskData({ ...taskData, projectRoot: e.target.value })
                    }
                    placeholder="Optional: /path/to/project"
                  />
                </div>
              </div>
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
                            {profile.name} ({profile.runnerType})
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
                                    {profile.runnerType}
                                  </Badge>
                                  {profile.model && (
                                    <Badge variant="outline">
                                      {profile.model}
                                    </Badge>
                                  )}
                                  {profile.requiresSandbox && (
                                    <Badge variant="outline">Sandbox</Badge>
                                  )}
                                  {profile.requiresApproval && (
                                    <Badge variant="outline">Approval</Badge>
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
                      value={agentConfig.runnerType}
                      onChange={(e) => {
                        const newRunnerType = e.target.value as RunnerType;
                        setAgentConfig({
                          ...agentConfig,
                          runnerType: newRunnerType,
                          model: getDefaultModelForRunner(newRunnerType),
                        });
                      }}
                      className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                    >
                      {RUNNER_TYPES.map((type) => (
                        <option key={type} value={type}>
                          {type}
                        </option>
                      ))}
                    </select>
                  </div>

                  <ModelSelector
                    value={agentConfig.model}
                    onChange={(model) =>
                      setAgentConfig({ ...agentConfig, model })
                    }
                    models={getModelsForRunner(agentConfig.runnerType)}
                    label="Model"
                    placeholder="Enter custom model..."
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
                            maxTurns: parseInt(e.target.value) || 100,
                          })
                        }
                        min={1}
                        max={1000}
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
                            timeoutMinutes: parseInt(e.target.value) || 30,
                          })
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
                      value={agentConfig.runMode}
                      onChange={(e) =>
                        setAgentConfig({
                          ...agentConfig,
                          runMode: e.target.value as "sandboxed" | "in_place",
                        })
                      }
                      className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                    >
                      <option value="sandboxed">Sandboxed (isolated copy)</option>
                      <option value="in_place">In-place (direct changes)</option>
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
                  <div>
                    <span className="text-muted-foreground">Title: </span>
                    <span className="font-medium">{taskData.title}</span>
                  </div>
                  {taskData.description && (
                    <div>
                      <span className="text-muted-foreground">Description: </span>
                      <span className="text-muted-foreground line-clamp-2">
                        {taskData.description}
                      </span>
                    </div>
                  )}
                  <div className="flex items-center gap-1">
                    <FolderOpen className="h-3 w-3 text-muted-foreground" />
                    <span className="text-muted-foreground">Scope: </span>
                    <code className="text-xs bg-muted px-1 py-0.5 rounded">
                      {taskData.scopePath}
                    </code>
                  </div>
                  {taskData.projectRoot && (
                    <div>
                      <span className="text-muted-foreground">Project Root: </span>
                      <code className="text-xs bg-muted px-1 py-0.5 rounded">
                        {taskData.projectRoot}
                      </code>
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
                              <Badge variant="outline">{profile.runnerType}</Badge>
                              {profile.model && (
                                <Badge variant="outline">{profile.model}</Badge>
                              )}
                              {profile.requiresSandbox && (
                                <Badge variant="outline">Sandbox</Badge>
                              )}
                              {profile.requiresApproval && (
                                <Badge variant="outline">Approval Required</Badge>
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
                        <Badge variant="outline">{agentConfig.runnerType}</Badge>
                        <Badge variant="outline">{agentConfig.model}</Badge>
                        <Badge variant="outline">
                          {agentConfig.runMode === "sandboxed"
                            ? "Sandboxed"
                            : "In-place"}
                        </Badge>
                        <Badge variant="outline">
                          Max {agentConfig.maxTurns} turns
                        </Badge>
                        <Badge variant="outline">
                          {agentConfig.timeoutMinutes}min timeout
                        </Badge>
                      </div>
                    </>
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
          {currentStep < 3 ? (
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
          ) : (
            <Button
              type="button"
              onClick={handleStartRun}
              disabled={submitting}
              className="gap-2"
            >
              <Play className="h-4 w-4" />
              {submitting ? "Starting..." : "Start Run"}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
