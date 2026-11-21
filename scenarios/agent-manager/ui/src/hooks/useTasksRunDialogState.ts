// Shared Tasks page state for run/profile dialog flows.
// AI_CHECK: react_coherence=1 | LAST: 2026-02-06

import { useState } from "react";
import type { ProfileFormData, Task } from "../types";
import { RunMode } from "../types";

export interface InlineRunConfig {
  roleRef: string;
  maxTurns: number;
  timeoutMinutes: number;
	 effort: "" | "low" | "medium" | "high" | "xhigh" | "max";
  runMode: RunMode;
  skipPermissionPrompt: boolean;
}

export type ProfileFormState = ProfileFormData;

function createInitialInlineConfig(): InlineRunConfig {
  return {
    roleRef: "code.default",
    maxTurns: 100,
    timeoutMinutes: 30,
	 effort: "",
    runMode: RunMode.SANDBOXED,
    skipPermissionPrompt: true,
  };
}

function createInitialProfileFormData(): ProfileFormState {
  return {
    name: "",
    profileKey: "",
    description: "",
    roleRef: "code.default",
    maxTurns: 100,
    sandboxMode: "protected" as const,
    networkAccess: "localhost" as const,
    timeoutMinutes: 30,
	 effort: "",
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
  };
}
