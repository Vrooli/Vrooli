import "@testing-library/jest-dom";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import type { Investigation } from "../../types/investigation";
import type { useDeployment } from "../../hooks/useDeployment";

const hooks = vi.hoisted(() => ({
  useDeploymentInvestigation: vi.fn(),
  useDeploymentRecord: vi.fn(),
}));

vi.mock("../../hooks/useInvestigation", () => ({ useDeploymentInvestigation: hooks.useDeploymentInvestigation }));
vi.mock("../../hooks/useDeployments", () => ({ useDeployment: hooks.useDeploymentRecord }));
vi.mock("./DeploymentProgress", () => ({
  DeploymentProgress: ({ onComplete }: { onComplete: (success: boolean, error?: string) => void }) => (
    <button onClick={() => onComplete(true)} type="button">Mock deployment progress</button>
  ),
}));
vi.mock("./SpawnAgentButton", () => ({
  SpawnAgentButton: ({ onTaskStarted }: { onTaskStarted?: (taskId: string) => void }) => (
    <button onClick={() => onTaskStarted?.("task-1")} type="button">Mock spawn agent</button>
  ),
}));
vi.mock("./InvestigationProgress", () => ({
  InvestigationProgress: ({
    onStop,
    onViewReport,
    isOutdated,
  }: {
    onStop?: () => void;
    onViewReport?: (id: string) => void;
    isOutdated?: boolean;
  }) => (
    <div>
      <span>{isOutdated ? "Outdated investigation" : "Current investigation"}</span>
      <button onClick={onStop} type="button">Mock stop investigation</button>
      <button onClick={() => onViewReport?.("investigation-1")} type="button">Mock view report</button>
    </div>
  ),
}));
vi.mock("./InvestigationReport", () => ({
  InvestigationReport: ({
    onClose,
    onApplyFixes,
  }: {
    onClose: () => void;
    onApplyFixes?: (id: string, options: { immediate: boolean; permanent: boolean; prevention: boolean }) => Promise<void>;
  }) => (
    <div>
      <span>Mock investigation report</span>
      <button onClick={onClose} type="button">Mock close report</button>
      <button onClick={() => onApplyFixes?.("investigation-1", { immediate: true, permanent: false, prevention: false })} type="button">Mock apply fixes</button>
    </div>
  ),
}));

import { StepDeploy } from "./StepDeploy";
import { renderWithProviders } from "../../test-utils/renderWithProviders";

const investigation: Investigation = {
  id: "investigation-1",
  deployment_id: "deployment-1",
  deployment_run_id: "run-old",
  status: "running",
  progress: 40,
  findings: "Needs attention",
  details: { source: "agent", operation_mode: "investigate", trigger_reason: "test" },
  created_at: "2026-08-14T10:00:00.000Z",
  updated_at: "2026-08-14T10:01:00.000Z",
};

function deploymentState(overrides?: Partial<{
  deploymentStatus: "idle" | "deploying" | "success" | "failed";
  deploymentError: string | null;
  deploymentId: string | null;
  parsedManifest: { ok: true; value: { edge?: { domain?: string } } } | { ok: false; error: string };
}>) {
  const state = {
    deploymentStatus: "idle" as const,
    deploymentError: null,
    deploymentId: null,
    deploy: vi.fn(),
    parsedManifest: { ok: true as const, value: { edge: { domain: "app.example.com" } } },
    reset: vi.fn(),
    onDeploymentComplete: vi.fn(),
    ...overrides,
  };
  return state as unknown as ReturnType<typeof useDeployment>;
}

function investigationState(overrides?: Partial<ReturnType<typeof hooks.useDeploymentInvestigation>>) {
  return {
    activeInvestigation: null,
    isRunning: false,
    stop: vi.fn(),
    isStopping: false,
    viewReport: vi.fn(),
    applyFixes: vi.fn().mockResolvedValue(undefined),
    isApplyingFixes: false,
    ...overrides,
  };
}

describe("StepDeploy", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    hooks.useDeploymentInvestigation.mockReturnValue(investigationState());
    hooks.useDeploymentRecord.mockReturnValue({ data: undefined });
  });

  it("starts an idle deployment only with a valid manifest and explains the lifecycle", () => {
    const deployment = deploymentState({ parsedManifest: { ok: false, error: "invalid" } });
    const { rerender } = renderWithProviders(<StepDeploy deployment={deployment} />);
    expect(screen.getByRole("button", { name: "Deploy to VPS" })).toBeDisabled();
    expect(screen.getByText("Bundle is uploaded to the target server via SCP")).toBeInTheDocument();
    expect(screen.getByText("Caddy configures HTTPS with Let's Encrypt")).toBeInTheDocument();

    const ready = deploymentState();
    rerender(<StepDeploy deployment={ready} />);
    fireEvent.click(screen.getByRole("button", { name: "Deploy to VPS" }));
    expect(ready.deploy).toHaveBeenCalledOnce();
  });

  it("wires progress completion, retries failures, and exposes agent investigation controls", async () => {
    const deployment = deploymentState({ deploymentStatus: "deploying", deploymentId: "deployment-1" });
    const currentInvestigation = investigationState({
      activeInvestigation: investigation,
      isRunning: true,
    });
    hooks.useDeploymentInvestigation.mockReturnValue(currentInvestigation);
    hooks.useDeploymentRecord.mockReturnValue({ data: { run_id: "run-current", last_deployed_at: "2026-08-14T12:00:00.000Z" } });
    renderWithProviders(<StepDeploy deployment={deployment} />);

    fireEvent.click(screen.getByRole("button", { name: "Mock deployment progress" }));
    expect(deployment.onDeploymentComplete).toHaveBeenCalledWith(true, undefined);
    fireEvent.click(screen.getByRole("button", { name: "Mock spawn agent" }));
    fireEvent.click(screen.getByRole("button", { name: "Mock stop investigation" }));
    expect(currentInvestigation.stop).toHaveBeenCalledOnce();
    expect(screen.getByText("Outdated investigation")).toBeInTheDocument();

    const failed = deploymentState({ deploymentStatus: "failed", deploymentId: "deployment-1", deploymentError: "VPS unavailable" });
    cleanup();
    renderWithProviders(<StepDeploy deployment={failed} />);
    fireEvent.click(screen.getByRole("button", { name: "Retry Deployment" }));
    expect(failed.deploy).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: "Mock deployment progress" })).toBeInTheDocument();
  });

  it("shows successful deployment details and manages an investigation report", async () => {
    const deployment = deploymentState({ deploymentStatus: "success", deploymentId: "deployment-1" });
    const currentInvestigation = investigationState({
      activeInvestigation: { ...investigation, status: "completed", deployment_run_id: "run-current" },
      isRunning: false,
    });
    hooks.useDeploymentInvestigation.mockReturnValue(currentInvestigation);
    hooks.useDeploymentRecord.mockReturnValue({ data: { run_id: "run-current" } });
    const onViewDeployments = vi.fn();
    renderWithProviders(<StepDeploy deployment={deployment} onViewDeployments={onViewDeployments} />);

    expect(screen.getByText("Deployment Successful!")).toBeInTheDocument();
    expect(screen.getByText("ID: deployment-1")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /https:\/\/app\.example\.com/ })).toHaveAttribute("href", "https://app.example.com");
    fireEvent.click(screen.getByRole("button", { name: "View Deployments" }));
    expect(onViewDeployments).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole("button", { name: "Mock view report" }));
    expect(currentInvestigation.viewReport).toHaveBeenCalledWith("investigation-1");
    await waitFor(() => expect(screen.getByText("Mock investigation report")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Mock close report" }));
    expect(screen.queryByText("Mock investigation report")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Mock view report" }));
    await waitFor(() => expect(screen.getByText("Mock investigation report")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "Mock apply fixes" }));
    await waitFor(() => expect(currentInvestigation.applyFixes).toHaveBeenCalledWith("investigation-1", { immediate: true, permanent: false, prevention: false }));
    expect(screen.queryByText("Mock investigation report")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start New Deployment" }));
    expect(deployment.reset).toHaveBeenCalledOnce();
  });
});
