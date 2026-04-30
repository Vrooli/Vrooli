// Shared Tasks page state for run/profile dialog flows.
// AI_CHECK: react_coherence=1 | LAST: 2026-02-06

import { useState } from "react";
import type { ModelSelectionMode } from "../components/ModelConfigSelector";
import type { ProfileFormData, RunnerType, Task } from "../types";
import { ModelPreset, RunMode, RunnerType as RunnerTypeEnum } from "../types";

export interface InlineRunConfig {
  runnerType: RunnerType;
  model: string;
  modelPreset: ModelPreset;
  modelMode: ModelSelectionMode;
  maxTurns: number;
  timeoutMinutes: number;
  runMode: RunMode;
  skipPermissionPrompt: boolean;
  fallbackRunnerTypes: RunnerType[];
}

export type ProfileFormState = ProfileFormData & {
  modelMode: ModelSelectionMode;
};

function createInitialInlineConfig(): InlineRunConfig {
  return {
    runnerType: RunnerTypeEnum.CLAUDE_CODE,
    model: "",
    modelPreset: ModelPreset.UNSPECIFIED,
    modelMode: "default",
    maxTurns: 100,
    timeoutMinutes: 30,
    runMode: RunMode.SANDBOXED,
    skipPermissionPrompt: true,
    fallbackRunnerTypes: [],
  };
}

function createInitialProfileFormData(): ProfileFormState {
  return {
    name: "",
    profileKey: "",
    description: "",
    runnerType: RunnerTypeEnum.CLAUDE_CODE,
    model: "",
    modelPreset: ModelPreset.UNSPECIFIED,
    modelMode: "default",
    maxTurns: 100,
    sandboxMode: "protected" as const,
    networkAccess: "localhost" as const,
    timeoutMinutes: 30,
    fallbackRunnerTypes: [],
  };
}

export function useTasksRunDialogState() {
  const [showRunDialog, setShowRunDialog] = useState<Task | null>(null);
  const [showProfileDialog, setShowProfileDialog] = useState(false);
  const [selectedProfileId, setSelectedProfileId] = useState("");
  const [runConfigMode, setRunConfigMode] = useState<"profile" | "custom">("profile");
  const [existingSandboxId, setExistingSandboxId] = useState("");
  const [inlineConfig, setInlineConfig] = useState<InlineRunConfig>(createInitialInlineConfig);
  const [profileFormData, setProfileFormData] = useState<ProfileFormState>(createInitialProfileFormData);
  const [profileFormError, setProfileFormError] = useState<string | null>(null);

  const resetProfileForm = () => {
    setProfileFormData(createInitialProfileFormData());
    setShowProfileDialog(false);
    setProfileFormError(null);
  };

  const resetRunDialog = () => {
    setShowRunDialog(null);
    setSelectedProfileId("");
    setRunConfigMode("profile");
    setExistingSandboxId("");
  };

  const handleAddInlineFallback = () => {
    setInlineConfig((prev) => ({
      ...prev,
      fallbackRunnerTypes: [...prev.fallbackRunnerTypes, RunnerTypeEnum.CLAUDE_CODE],
    }));
  };

  const handleInlineFallbackChange = (index: number, value: string) => {
    const parsed = Number(value) as RunnerType;
    setInlineConfig((prev) => {
      const fallback = [...prev.fallbackRunnerTypes];
      fallback[index] = parsed;
      return { ...prev, fallbackRunnerTypes: fallback };
    });
  };

  const handleRemoveInlineFallback = (index: number) => {
    setInlineConfig((prev) => {
      const fallback = [...prev.fallbackRunnerTypes];
      fallback.splice(index, 1);
      return { ...prev, fallbackRunnerTypes: fallback };
    });
  };

  const handleAddProfileFallback = () => {
    setProfileFormData((prev) => ({
      ...prev,
      fallbackRunnerTypes: [...(prev.fallbackRunnerTypes ?? []), RunnerTypeEnum.CLAUDE_CODE],
    }));
  };

  const handleProfileFallbackChange = (index: number, value: string) => {
    const parsed = Number(value) as RunnerType;
    setProfileFormData((prev) => {
      const fallback = [...(prev.fallbackRunnerTypes ?? [])];
      fallback[index] = parsed;
      return { ...prev, fallbackRunnerTypes: fallback };
    });
  };

  const handleRemoveProfileFallback = (index: number) => {
    setProfileFormData((prev) => {
      const fallback = [...(prev.fallbackRunnerTypes ?? [])];
      fallback.splice(index, 1);
      return { ...prev, fallbackRunnerTypes: fallback };
    });
  };

  return {
    showRunDialog,
    setShowRunDialog,
    showProfileDialog,
    setShowProfileDialog,
    selectedProfileId,
    setSelectedProfileId,
    runConfigMode,
    setRunConfigMode,
    existingSandboxId,
    setExistingSandboxId,
    inlineConfig,
    setInlineConfig,
    profileFormData,
    setProfileFormData,
    profileFormError,
    setProfileFormError,
    resetProfileForm,
    resetRunDialog,
    handleAddInlineFallback,
    handleInlineFallbackChange,
    handleRemoveInlineFallback,
    handleAddProfileFallback,
    handleProfileFallbackChange,
    handleRemoveProfileFallback,
  };
}
