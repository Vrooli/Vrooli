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
  { id: "preflight", title: "Running preflight checks" },
  { id: "host.prepare", title: "Preparing host" },
  { id: "edge.firewall.allow", title: "Opening inbound HTTP/HTTPS" },
  { id: "data.inventory", title: "Inventorying persistent data" },
  { id: "release.deliver", title: "Delivering release" },
  { id: "release.verify", title: "Verifying release" },
  { id: "release.stage", title: "Staging release" },
  { id: "data.backup", title: "Backing up persistent data" },
  { id: "release.activate", title: "Activating release" },
  { id: "config.apply", title: "Applying configuration" },
  { id: "workload.stop", title: "Stopping existing scenario" },
  { id: "edge.route.apply", title: "Configuring edge route" },
  { id: "credentials.provision", title: "Provisioning credentials" },
  { id: "runtime.start_dependencies", title: "Starting dependencies" },
  { id: "workload.start", title: "Starting scenario" },
  { id: "verify.readiness", title: "Verifying readiness" },
  { id: "release.retain_predecessor", title: "Retaining predecessor" },
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
