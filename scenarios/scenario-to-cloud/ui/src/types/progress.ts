/**
 * Progress tracking types for deployment SSE streaming.
 */

import type { PreflightResponse } from "../lib/api";

export type ProgressEventType =
  | "step_started"
  | "step_completed"
  | "progress_update"
  | "deployment_error"
  | "preflight_result"
  | "completed";

export interface ProgressEvent {
  type: ProgressEventType;
  step: string;
  step_title: string;
  progress: number; // 0-100
  message?: string;
  error?: string;
  preflight_result?: PreflightResponse;
  timestamp: string;
}

export type StepStatus = "pending" | "running" | "completed" | "failed";

export interface DeploymentStep {
  id: string;
  title: string;
  status: StepStatus;
}

export interface DeploymentProgress {
  currentStep: string;
  currentStepTitle: string;
  progress: number;
  steps: DeploymentStep[];
  error?: string;
  preflightResult?: PreflightResponse;
  isComplete: boolean;
}

/**
 * All deployment steps in order with their display titles.
 */
export const DEPLOYMENT_STEPS: { id: string; title: string }[] = [
  { id: "bundle_build", title: "Building bundle" },
  { id: "mkdir", title: "Creating directories" },
  { id: "bootstrap", title: "Installing prerequisites" },
  { id: "upload", title: "Uploading bundle" },
  { id: "extract", title: "Extracting bundle" },
  { id: "setup", title: "Running setup" },
  { id: "autoheal", title: "Configuring autoheal" },
  { id: "verify_setup", title: "Verifying installation" },
  { id: "scenario_stop", title: "Stopping existing scenario" },
  { id: "caddy_install", title: "Installing Caddy" },
  { id: "caddy_config", title: "Configuring Caddy" },
  { id: "secrets_provision", title: "Provisioning secrets" },
  { id: "resource_start", title: "Starting resources" },
  { id: "scenario_deps", title: "Starting dependencies" },
  { id: "scenario_target", title: "Starting scenario" },
  { id: "wait_for_ui", title: "Waiting for UI to listen" },
  { id: "verify_local", title: "Verifying local health" },
  { id: "verify_https", title: "Verifying HTTPS" },
  { id: "verify_origin", title: "Verifying origin reachability" },
  { id: "verify_public", title: "Verifying public reachability" },
];

/**
 * Get the initial step list with all steps pending.
 */
export function getInitialSteps(): DeploymentStep[] {
  return DEPLOYMENT_STEPS.map((s) => ({
    id: s.id,
    title: s.title,
    status: "pending" as StepStatus,
  }));
}

/**
 * Update a step's status in the steps array.
 */
export function updateStepStatus(
  steps: DeploymentStep[] | undefined,
  stepId: string,
  status: StepStatus,
): DeploymentStep[] {
  if (!steps) {
    return getInitialSteps().map((s) =>
      s.id === stepId ? { ...s, status } : s,
    );
  }
  return steps.map((s) => (s.id === stepId ? { ...s, status } : s));
}
