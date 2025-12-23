import { useState } from "react";
import {
  AlertCircle,
  Bot,
  Edit,
  Plus,
  RefreshCw,
  Settings2,
  Trash2,
} from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { ModelConfigSelector, type ModelSelectionMode } from "../components/ModelConfigSelector";
import { ScrollArea } from "../components/ui/scroll-area";
import { Textarea } from "../components/ui/textarea";
import { durationMs, type Duration } from "@bufbuild/protobuf/wkt";
import { formatDate, runnerTypeLabel } from "../lib/utils";
import type { AgentProfile, ModelRegistry, ProfileFormData, RunnerStatus, RunnerType } from "../types";
import { ModelPreset, RunnerType as RunnerTypeEnum } from "../types";
import { runnerTypeToSlug } from "../lib/utils";

interface ProfilesPageProps {
  profiles: AgentProfile[];
  loading: boolean;
  error: string | null;
  onCreateProfile: (profile: ProfileFormData) => Promise<AgentProfile>;
  onUpdateProfile: (id: string, profile: ProfileFormData) => Promise<AgentProfile>;
  onDeleteProfile: (id: string) => Promise<void>;
  onRefresh: () => void;
  runners?: Record<string, RunnerStatus>;
  modelRegistry?: ModelRegistry;
}

const RUNNER_TYPES: RunnerType[] = [
  RunnerTypeEnum.CLAUDE_CODE,
  RunnerTypeEnum.CODEX,
  RunnerTypeEnum.OPENCODE,
];

const durationToMinutes = (duration: Duration | undefined): number => {
  if (!duration) return 30;
  const ms = durationMs(duration);
  return Math.max(1, Math.round(ms / 60_000));
};

const resolveModelMode = (model: string | undefined, preset: ModelPreset | undefined): ModelSelectionMode => {
  if (preset !== undefined && preset !== ModelPreset.UNSPECIFIED) {
    return "preset";
  }
  if (model && model.trim() !== "") {
    return "model";
  }
  return "default";
};

const getModelId = (model: string | { id: string }): string => {
  return typeof model === "string" ? model : model.id;
};

type ProfileFormState = ProfileFormData & {
  modelMode: ModelSelectionMode;
};

const modelPresetLabel = (preset?: ModelPreset) => {
  switch (preset) {
    case ModelPreset.FAST:
      return "Fast";
    case ModelPreset.CHEAP:
      return "Cheap";
    case ModelPreset.SMART:
      return "Smart";
    default:
      return "";
  }
};

export function ProfilesPage({
  profiles,
  loading,
  error,
  onCreateProfile,
  onUpdateProfile,
  onDeleteProfile,
  onRefresh,
  runners,
  modelRegistry,
}: ProfilesPageProps) {
  const getRegistryForRunner = (runnerType: RunnerType) => {
    return modelRegistry?.runners?.[runnerTypeToSlug(runnerType)];
  };

  // Helper to get models for a runner type from registry, fallback to capabilities
  const getModelsForRunner = (runnerType: RunnerType) => {
    const registry = getRegistryForRunner(runnerType);
    if (registry?.models?.length) {
      return registry.models;
    }
    const runner = runners?.[runnerType];
    return runner?.supportedModels ?? [];
  };

  const getPresetMapForRunner = (runnerType: RunnerType) => {
    return getRegistryForRunner(runnerType)?.presets ?? {};
  };

  const [showForm, setShowForm] = useState(false);
  const [editingProfile, setEditingProfile] = useState<AgentProfile | null>(null);
  const [formData, setFormData] = useState<ProfileFormState>({
    name: "",
    profileKey: "",
    description: "",
    runnerType: RunnerTypeEnum.CLAUDE_CODE,
    model: "",
    modelPreset: ModelPreset.UNSPECIFIED,
    modelMode: "default",
    maxTurns: 100,
    requiresSandbox: true,
    requiresApproval: true,
    timeoutMinutes: 30,
    fallbackRunnerTypes: [],
  });
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const resetForm = () => {
    setFormData({
      name: "",
      profileKey: "",
      description: "",
      runnerType: RunnerTypeEnum.CLAUDE_CODE,
      model: "",
      modelPreset: ModelPreset.UNSPECIFIED,
      modelMode: "default",
      maxTurns: 100,
      requiresSandbox: true,
      requiresApproval: true,
      timeoutMinutes: 30,
      fallbackRunnerTypes: [],
    });
    setEditingProfile(null);
    setShowForm(false);
    setFormError(null);
  };

  const handleEdit = (profile: AgentProfile) => {
    setEditingProfile(profile);
    setFormData({
      name: profile.name,
      profileKey: profile.profileKey || "",
      description: profile.description || "",
      runnerType: profile.runnerType,
      model: profile.model || "",
      modelPreset: profile.modelPreset ?? ModelPreset.UNSPECIFIED,
      modelMode: resolveModelMode(profile.model, profile.modelPreset),
      maxTurns: profile.maxTurns || 100,
      requiresSandbox: profile.requiresSandbox,
      requiresApproval: profile.requiresApproval,
      allowedTools: profile.allowedTools,
      deniedTools: profile.deniedTools,
      timeoutMinutes: durationToMinutes(profile.timeout),
      fallbackRunnerTypes: profile.fallbackRunnerTypes ?? [],
    });
    setShowForm(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setFormError(null);
    try {
      const normalizedProfile: ProfileFormData = {
        ...formData,
        model:
          formData.modelMode === "model"
            ? formData.model?.trim() ?? ""
            : "",
        modelPreset:
          formData.modelMode === "preset"
            ? formData.modelPreset ?? ModelPreset.FAST
            : ModelPreset.UNSPECIFIED,
        timeoutMinutes: formData.timeoutMinutes ?? 30,
        fallbackRunnerTypes: formData.fallbackRunnerTypes ?? [],
      };
      if (editingProfile) {
        await onUpdateProfile(editingProfile.id, normalizedProfile);
      } else {
        await onCreateProfile(normalizedProfile);
      }
      resetForm();
    } catch (err) {
      setFormError((err as Error).message);
      console.error("Failed to save profile:", err);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this profile?")) return;
    try {
      await onDeleteProfile(id);
    } catch (err) {
      console.error("Failed to delete profile:", err);
    }
  };

  const handleAddFallbackRunner = () => {
    setFormData((prev) => ({
      ...prev,
      fallbackRunnerTypes: [...(prev.fallbackRunnerTypes ?? []), RunnerTypeEnum.CLAUDE_CODE],
    }));
  };

  const handleFallbackRunnerChange = (index: number, value: string) => {
    const parsed = Number(value) as RunnerType;
    setFormData((prev) => {
      const fallback = [...(prev.fallbackRunnerTypes ?? [])];
      fallback[index] = parsed;
      return { ...prev, fallbackRunnerTypes: fallback };
    });
  };

  const handleRemoveFallbackRunner = (index: number) => {
    setFormData((prev) => {
      const fallback = [...(prev.fallbackRunnerTypes ?? [])];
      fallback.splice(index, 1);
      return { ...prev, fallbackRunnerTypes: fallback };
    });
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Agent Profiles</h2>
          <p className="text-sm text-muted-foreground">
            Configure how agents execute tasks
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={onRefresh} className="gap-2">
            <RefreshCw className="h-4 w-4" />
            Refresh
          </Button>
          <Button size="sm" onClick={() => setShowForm(true)} className="gap-2">
            <Plus className="h-4 w-4" />
            New Profile
          </Button>
        </div>
      </div>

      {error && (
        <Card className="border-destructive/50 bg-destructive/10">
          <CardContent className="flex items-center gap-3 py-4">
            <AlertCircle className="h-4 w-4 text-destructive" />
            <p className="text-sm text-destructive">{error}</p>
          </CardContent>
        </Card>
      )}

      {/* Create/Edit Profile Modal */}
      <Dialog open={showForm} onOpenChange={(open) => !open && resetForm()}>
        <DialogContent>
          <DialogHeader onClose={resetForm}>
            <DialogTitle>
              {editingProfile ? "Edit Profile" : "Create New Profile"}
            </DialogTitle>
            <DialogDescription>
              {editingProfile
                ? "Update the agent profile configuration"
                : "Define how the agent should execute tasks"}
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={handleSubmit}>
            <DialogBody className="space-y-4">
              {formError && (
                <Card className="border-destructive/50 bg-destructive/10">
                  <CardContent className="flex items-center gap-3 py-3">
                    <AlertCircle className="h-4 w-4 text-destructive" />
                    <p className="text-sm text-destructive">{formError}</p>
                  </CardContent>
                </Card>
              )}
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="name">Name *</Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) =>
                      setFormData({ ...formData, name: e.target.value })
                    }
                    placeholder="e.g., Claude Code Default"
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="runnerType">Runner Type *</Label>
                  <select
                    id="runnerType"
                    value={String(formData.runnerType)}
                    onChange={(e) => {
                      const newRunnerType = Number(e.target.value) as RunnerType;
                      const availableModels = getModelsForRunner(newRunnerType);
                      const firstModel = availableModels.length > 0 ? getModelId(availableModels[0]) : "";
                      setFormData({
                        ...formData,
                        runnerType: newRunnerType,
                        model:
                          formData.modelMode === "model"
                            ? firstModel
                            : formData.model,
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
              </div>

              <div className="space-y-2">
                <Label htmlFor="profileKey">Profile Key</Label>
                <Input
                  id="profileKey"
                  value={formData.profileKey ?? ""}
                  onChange={(e) =>
                    setFormData({ ...formData, profileKey: e.target.value })
                  }
                  placeholder="auto-generated from name if left blank"
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Description</Label>
                <Textarea
                  id="description"
                  value={formData.description}
                  onChange={(e) =>
                    setFormData({ ...formData, description: e.target.value })
                  }
                  placeholder="Describe what this profile is for..."
                  rows={2}
                />
              </div>

              <ModelConfigSelector
                value={{
                  mode: formData.modelMode,
                  model: formData.model ?? "",
                  preset: formData.modelPreset ?? ModelPreset.UNSPECIFIED,
                }}
                onChange={(selection) =>
                  setFormData({
                    ...formData,
                    modelMode: selection.mode,
                    model: selection.model,
                    modelPreset: selection.preset,
                  })
                }
                models={getModelsForRunner(formData.runnerType)}
                presetMap={getPresetMapForRunner(formData.runnerType)}
                label="Model Selection"
              />

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <Label>Fallback Runners</Label>
                  <Button type="button" variant="outline" size="sm" onClick={handleAddFallbackRunner}>
                    Add
                  </Button>
                </div>
                <p className="text-xs text-muted-foreground">
                  Ordered runners to try if the primary runner is unavailable.
                </p>
                {(formData.fallbackRunnerTypes ?? []).length === 0 ? (
                  <p className="text-xs text-muted-foreground">No fallback runners configured.</p>
                ) : (
                  <div className="space-y-2">
                    {(formData.fallbackRunnerTypes ?? []).map((runnerType, index) => (
                      <div key={`fallback-${index}`} className="flex items-center gap-2">
                        <select
                          value={String(runnerType)}
                          onChange={(e) => handleFallbackRunnerChange(index, e.target.value)}
                          className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                        >
                          {RUNNER_TYPES.map((type) => (
                            <option key={type} value={type}>
                              {runnerTypeLabel(type)}
                            </option>
                          ))}
                        </select>
                        <Button
                          type="button"
                          variant="ghost"
                          size="sm"
                          onClick={() => handleRemoveFallbackRunner(index)}
                        >
                          Remove
                        </Button>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="maxTurns">Max Turns</Label>
                  <Input
                    id="maxTurns"
                    type="number"
                    value={formData.maxTurns}
                    onChange={(e) =>
                      setFormData({ ...formData, maxTurns: parseInt(e.target.value) || 100 })
                    }
                    min={1}
                    max={1000}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="timeout">Timeout (minutes)</Label>
                  <Input
                    id="timeout"
                    type="number"
                    value={formData.timeoutMinutes ?? 30}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        timeoutMinutes: parseInt(e.target.value) || 30,
                      })
                    }
                    min={1}
                    max={1440}
                  />
                </div>
              </div>

              <div className="flex gap-6">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.requiresSandbox}
                    onChange={(e) =>
                      setFormData({ ...formData, requiresSandbox: e.target.checked })
                    }
                    className="h-4 w-4 rounded border-input"
                  />
                  <span className="text-sm">Require Sandbox</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.requiresApproval}
                    onChange={(e) =>
                      setFormData({ ...formData, requiresApproval: e.target.checked })
                    }
                    className="h-4 w-4 rounded border-input"
                  />
                  <span className="text-sm">Require Approval</span>
                </label>
              </div>
            </DialogBody>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={resetForm}>
                Cancel
              </Button>
              <Button type="submit" disabled={submitting}>
                {submitting
                  ? "Saving..."
                  : editingProfile
                  ? "Update Profile"
                  : "Create Profile"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      {/* Profiles List */}
      {loading ? (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            Loading profiles...
          </CardContent>
        </Card>
      ) : profiles.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12 text-muted-foreground">
            <Bot className="h-16 w-16 mb-4 opacity-50" />
            <h3 className="text-lg font-semibold mb-1">No Agent Profiles</h3>
            <p className="text-sm mb-4">Create your first profile to get started</p>
            <Button onClick={() => setShowForm(true)} className="gap-2">
              <Plus className="h-4 w-4" />
              Create Profile
            </Button>
          </CardContent>
        </Card>
      ) : (
        <ScrollArea className="h-[600px]">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {profiles.map((profile) => (
              <Card key={profile.id} className="relative">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-2">
                      <Settings2 className="h-5 w-5 text-primary" />
                      <CardTitle className="text-base">{profile.name}</CardTitle>
                    </div>
                    <div className="flex gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        aria-label={`Edit ${profile.name}`}
                        onClick={() => handleEdit(profile)}
                      >
                        <Edit className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 text-destructive hover:text-destructive"
                        aria-label={`Delete ${profile.name}`}
                        onClick={() => handleDelete(profile.id)}
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </div>
                  <CardDescription className="line-clamp-2">
                    {profile.description || "No description"}
                  </CardDescription>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="flex flex-wrap gap-2">
                    <Badge variant="secondary">{runnerTypeLabel(profile.runnerType)}</Badge>
                    {profile.modelPreset !== ModelPreset.UNSPECIFIED && (
                        <Badge variant="outline" className="text-xs">
                          Preset: {modelPresetLabel(profile.modelPreset)}
                        </Badge>
                      )}
                    {profile.model && profile.model.trim() !== "" && (
                      <Badge variant="outline" className="text-xs">
                        {profile.model}
                      </Badge>
                    )}
                  </div>
                  <div className="flex gap-2 text-xs">
                    {profile.requiresSandbox && (
                      <Badge variant="outline" className="text-[10px]">Sandbox</Badge>
                    )}
                    {profile.requiresApproval && (
                      <Badge variant="outline" className="text-[10px]">Approval</Badge>
                    )}
                    {profile.maxTurns && (
                      <Badge variant="outline" className="text-[10px]">
                        Max {profile.maxTurns} turns
                      </Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Created {formatDate(profile.createdAt)}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </ScrollArea>
      )}
    </div>
  );
}
