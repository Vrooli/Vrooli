import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import {
  AlertCircle,
  Bot,
  Plus,
  RefreshCw,
  Settings2,
} from "lucide-react";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent } from "../components/ui/card";
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
import { Textarea } from "../components/ui/textarea";
import { durationMs, type Duration } from "@bufbuild/protobuf/wkt";
import { formatDate, runnerTypeLabel } from "../lib/utils";
import type { AgentProfile, ModelRegistry, ProfileFormData, RunnerStatus, RunnerType } from "../types";
import { ModelPreset, RunnerType as RunnerTypeEnum } from "../types";
import { runnerTypeToSlug } from "../lib/utils";
import { ProfileDetail } from "../components/ProfileDetail";
import { useViewportSize } from "../hooks/useViewportSize";

import { MasterDetailLayout, ListPanel, DetailPanel } from "../components/patterns/MasterDetail";
import { SearchToolbar, type FilterConfig, type SortOption } from "../components/patterns/SearchToolbar";
import { ListItem, ListItemTitle, ListItemSubtitle } from "../components/patterns/ListItem";

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

const RUNNER_TYPE_FILTER_OPTIONS = [
  { value: String(RunnerTypeEnum.CLAUDE_CODE), label: "Claude Code" },
  { value: String(RunnerTypeEnum.CODEX), label: "Codex" },
  { value: String(RunnerTypeEnum.OPENCODE), label: "OpenCode" },
];

const SORT_OPTIONS: SortOption[] = [
  { value: "newest", label: "Newest First" },
  { value: "oldest", label: "Oldest First" },
  { value: "name", label: "Name A-Z" },
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
  const { isDesktop } = useViewportSize();
  const getRegistryForRunner = (runnerType: RunnerType) => {
    return modelRegistry?.runners?.[runnerTypeToSlug(runnerType)];
  };

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

  // Selection state
  const [selectedProfileId, setSelectedProfileId] = useState<string | null>(null);
  const [searchParams] = useSearchParams();
  const profileIdParam = searchParams.get("profileId");

  // Modal state
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

  // Filter/sort/search state
  const [searchQuery, setSearchQuery] = useState("");
  const [runnerTypeFilter, setRunnerTypeFilter] = useState<string>("all");
  const [sortBy, setSortBy] = useState<string>("newest");

  const selectedProfile = useMemo(
    () => profiles.find((p) => p.id === selectedProfileId) || null,
    [profiles, selectedProfileId]
  );

  useEffect(() => {
    if (profileIdParam) {
      setSelectedProfileId(profileIdParam);
    }
  }, [profileIdParam]);

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
      if (selectedProfileId === id) {
        setSelectedProfileId(null);
      }
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

  const filteredAndSortedProfiles = useMemo(() => {
    let result = [...profiles];

    if (runnerTypeFilter !== "all") {
      const runnerType = Number(runnerTypeFilter) as RunnerType;
      result = result.filter((p) => p.runnerType === runnerType);
    }

    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      result = result.filter(
        (p) =>
          p.name.toLowerCase().includes(query) ||
          p.description?.toLowerCase().includes(query)
      );
    }

    result.sort((a, b) => {
      if (sortBy === "name") {
        return a.name.localeCompare(b.name);
      }
      const aTime = a.createdAt ? new Date(a.createdAt.toString()).getTime() : 0;
      const bTime = b.createdAt ? new Date(b.createdAt.toString()).getTime() : 0;
      return sortBy === "newest" ? bTime - aTime : aTime - bTime;
    });

    return result;
  }, [profiles, runnerTypeFilter, searchQuery, sortBy]);

  useEffect(() => {
    if (!isDesktop) return;
    if (filteredAndSortedProfiles.length === 0) return;

    const hasSelection =
      selectedProfileId !== null &&
      filteredAndSortedProfiles.some((profile) => profile.id === selectedProfileId);

    if (!hasSelection) {
      setSelectedProfileId(filteredAndSortedProfiles[0].id);
    }
  }, [filteredAndSortedProfiles, isDesktop, selectedProfileId]);

  const filters: FilterConfig[] = [
    {
      id: "runnerType",
      label: "Filter by runner type",
      value: runnerTypeFilter,
      options: RUNNER_TYPE_FILTER_OPTIONS,
      onChange: setRunnerTypeFilter,
      allLabel: "All Runners",
    },
  ];

  const listPanel = (
    <ListPanel
      title="Agent Profiles"
      count={filteredAndSortedProfiles.length}
      loading={loading}
      headerActions={
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={onRefresh}>
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Button size="sm" onClick={() => setShowForm(true)} className="gap-1">
            <Plus className="h-4 w-4" />
            <span className="hidden sm:inline">New</span>
          </Button>
        </div>
      }
      toolbar={
        <SearchToolbar
          searchValue={searchQuery}
          onSearchChange={setSearchQuery}
          searchPlaceholder="Search profiles..."
          filters={filters}
          sortOptions={SORT_OPTIONS}
          currentSort={sortBy}
          onSortChange={setSortBy}
        />
      }
      empty={
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Bot className="h-12 w-12 mb-3 opacity-50" />
          <p className="font-medium">
            {profiles.length === 0 ? "No Agent Profiles" : "No Matching Profiles"}
          </p>
          <p className="text-sm text-center mt-1">
            {profiles.length === 0
              ? "Create your first profile to get started"
              : "Try adjusting your filters"}
          </p>
          {profiles.length === 0 && (
            <Button
              onClick={() => setShowForm(true)}
              className="gap-2 mt-4"
              size="sm"
            >
              <Plus className="h-4 w-4" />
              Create Profile
            </Button>
          )}
        </div>
      }
    >
      {filteredAndSortedProfiles.map((profile) => (
        <ListItem
          key={profile.id}
          selected={selectedProfileId === profile.id}
          onClick={() => setSelectedProfileId(profile.id)}
          icon={<Settings2 className="h-5 w-5 text-primary flex-shrink-0" />}
          actions={
            <Badge variant="secondary">{runnerTypeLabel(profile.runnerType)}</Badge>
          }
        >
          <ListItemTitle>{profile.name}</ListItemTitle>
          <ListItemSubtitle>
            {profile.description || "No description"} | {formatDate(profile.createdAt)}
          </ListItemSubtitle>
        </ListItem>
      ))}
    </ListPanel>
  );

  const detailPanel = (
    <DetailPanel
      title="Profile Details"
      hasSelection={!!selectedProfile}
      empty={
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <Settings2 className="h-12 w-12 mb-3 opacity-50" />
          <p className="text-sm">Select a profile to view details</p>
        </div>
      }
    >
      {selectedProfile && (
        <ProfileDetail
          profile={selectedProfile}
          onEdit={handleEdit}
          onDelete={handleDelete}
        />
      )}
    </DetailPanel>
  );

  // Build header content with error banner
  const headerContent = error ? (
    <Card className="border-destructive/50 bg-destructive/10">
      <CardContent className="flex items-center gap-3 py-4">
        <AlertCircle className="h-4 w-4 text-destructive" />
        <p className="text-sm text-destructive">{error}</p>
      </CardContent>
    </Card>
  ) : null;

  return (
    <>
      <MasterDetailLayout
        pageTitle="Agent Profiles"
        pageSubtitle="Configure how agents execute tasks"
        storageKey="profiles"
        headerContent={headerContent}
        listPanel={listPanel}
        detailPanel={detailPanel}
        selectedId={selectedProfileId}
        onDeselect={() => setSelectedProfileId(null)}
        detailTitle={selectedProfile?.name ?? "Profile Details"}
      />

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
    </>
  );
}
