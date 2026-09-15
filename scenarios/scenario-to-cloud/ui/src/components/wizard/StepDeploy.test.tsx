import "@testing-library/jest-dom";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import type { Investigation } from "../../types/investigation";
import type { useDeployment } from "../../hooks/useDeployment";
import { makePlan, makeStanding, apiErrorOf } from "../../test-utils/consoleFixtures";

const hooks = vi.hoisted(() => ({
  useDeploymentInvestigation: vi.fn(),
  useDeploymentRecord: vi.fn(),
}));

const consoleState = vi.hoisted(() => ({
  plan: null as unknown,
  planError: null as unknown,
  standing: null as unknown,
  pointer: null as { deployment_id: string; operation_id: string; plan_digest: string } | null,
  resumed: false,
  apply: { mutateAsync: vi.fn(), isPending: false, error: null as unknown, reset: vi.fn() },
  cancel: { mutateAsync: vi.fn(), isPending: false, error: null as unknown },
  attach: vi.fn(),
  detach: vi.fn(),
}));

vi.mock("../../hooks/useInvestigation", () => ({ useDeploymentInvestigation: hooks.useDeploymentInvestigation }));
vi.mock("../../hooks/useDeployments", () => ({ useDeployment: hooks.useDeploymentRecord }));
vi.mock("../../hooks/useConsole", () => ({
  useAuthzMatrix: () => ({ data: null }),
  useCompiledPlan: () => ({ data: consoleState.plan, error: consoleState.planError, isLoading: false, isFetching: false, refetch: vi.fn() }),
  useApplyPlan: () => consoleState.apply,
  useCancelOperation: () => consoleState.cancel,
  useDurableOperation: () => ({ pointer: consoleState.pointer, resumed: consoleState.resumed, rehydrating: false, attach: consoleState.attach, detach: consoleState.detach }),
  useOperationStanding: () => ({ data: consoleState.standing, error: null, isLoading: false }),
}));
vi.mock("./SpawnAgentButton", () => ({
  SpawnAgentButton: ({ onTaskStarted }: { onTaskStarted?: (taskId: string) => void }) => (
    <button onClick={() => onTaskStarted?.("task-1")} type="button">Mock spawn agent</button>
  ),
}));
vi.mock("./InvestigationProgress", () => ({
  InvestigationProgress: ({ onStop, onViewReport, isOutdated }: { onStop?: () => void; onViewReport?: (id: string) => void; isOutdated?: boolean }) => (
    <div>
      <span>{isOutdated ? "Outdated investigation" : "Current investigation"}</span>
      <button onClick={onStop} type="button">Mock stop investigation</button>
      <button onClick={() => onViewReport?.("investigation-1")} type="button">Mock view report</button>
    </div>
  ),
}));
vi.mock("./InvestigationReport", () => ({
  InvestigationReport: ({ onClose, onApplyFixes }: { onClose: () => void; onApplyFixes?: (id: string, options: { immediate: boolean; permanent: boolean; prevention: boolean }) => Promise<void> }) => (
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
  status: "running",
  progress: 40,
  findings: "Needs attention",
  details: { source: "agent", operation_mode: "investigate", trigger_reason: "test" },
  created_at: "2026-08-14T10:00:00.000Z",
  updated_at: "2026-08-14T10:01:00.000Z",
};

function deploymentState(overrides?: Partial<{
  deploymentStatus: "idle" | "review" | "deploying" | "success" | "failed";
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
    onOperationAdmitted: vi.fn(),
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
    consoleState.plan = null;
    consoleState.planError = null;
    consoleState.standing = null;
    consoleState.pointer = null;
    consoleState.resumed = false;
    consoleState.apply.isPending = false;
    consoleState.apply.error = null;
    hooks.useDeploymentInvestigation.mockReturnValue(investigationState());
    hooks.useDeploymentRecord.mockReturnValue({ data: undefined });
  });

  it("starts the review only with a valid manifest and explains the reviewed lifecycle", () => {
    const deployment = deploymentState({ parsedManifest: { ok: false, error: "invalid" } });
    const { rerender } = renderWithProviders(<StepDeploy deployment={deployment} />);
    expect(screen.getByRole("button", { name: "Review deployment plan" })).toBeDisabled();
    expect(screen.getByText(/The executable plan is compiled and shown for review/)).toBeInTheDocument();
    expect(screen.queryByText(/percent|%/)).not.toBeInTheDocument();

    const ready = deploymentState();
    rerender(<StepDeploy deployment={ready} />);
    fireEvent.click(screen.getByRole("button", { name: "Review deployment plan" }));
    expect(ready.deploy).toHaveBeenCalledOnce();
  });

  // [REQ:STC-P0-038] The wizard reviews changes, data effects, downtime and
  // recovery before applying with the reviewed digest and a request key.
  it("reviews the compiled plan and applies it with the plan digest and a uuid request key", async () => {
    consoleState.plan = makePlan();
    consoleState.apply.mutateAsync.mockResolvedValue({ schema_version: "1", operation_id: "op-new", plan_digest: "sha256:plan-digest-fixture", state: "admitted" });
    const deployment = deploymentState({ deploymentStatus: "review", deploymentId: "deployment-1" });
    renderWithProviders(<StepDeploy deployment={deployment} />);

    expect(screen.getByTestId("console-review")).toHaveAttribute("data-state", "ready");
    expect(screen.getByTestId("console-review-changes")).toHaveTextContent("release.stage");
    expect(screen.getByTestId("console-review-data-effects")).toHaveTextContent("postgres:main");
    expect(screen.getByTestId("console-review-downtime")).toHaveTextContent("5s");
    expect(screen.getByTestId("console-review-recovery")).toHaveTextContent("rollback_release");
    expect(screen.queryByTestId("console-review-shell")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Show equivalent commands" }));
    expect(screen.getByTestId("console-review-shell")).toHaveTextContent("vrooli cloud-target release stage");

    fireEvent.click(screen.getByTestId("console-review-apply"));
    await waitFor(() => expect(consoleState.apply.mutateAsync).toHaveBeenCalledOnce());
    const [firstCall] = consoleState.apply.mutateAsync.mock.calls;
    const call = firstCall?.[0] as { deploymentId: string; planDigest: string; requestKey: string };
    expect(call.deploymentId).toBe("deployment-1");
    expect(call.planDigest).toBe("sha256:plan-digest-fixture");
    expect(call.requestKey).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
    await waitFor(() => expect(consoleState.attach).toHaveBeenCalledWith({ deployment_id: "deployment-1", operation_id: "op-new", plan_digest: "sha256:plan-digest-fixture" }));
    expect(deployment.onOperationAdmitted).toHaveBeenCalledOnce();
  });

  it("renders the onboarding handoff instead of a secret prompt when the plan needs input", () => {
    consoleState.plan = makePlan({ outcome: "needs_input", handoff: { owner: "vrooli-onboarding", kind: "resume_handoff", reference: "vrooli-onboarding://deployments/deployment-1/resume/abc", missing: ["credential:stripe_key"] } });
    const deployment = deploymentState({ deploymentStatus: "review", deploymentId: "deployment-1" });
    renderWithProviders(<StepDeploy deployment={deployment} />);
    expect(screen.getByTestId("console-review")).toHaveAttribute("data-state", "needs-input");
    const link = screen.getByTestId("console-review-handoff-link");
    expect(link).toHaveAttribute("href", "vrooli-onboarding://deployments/deployment-1/resume/abc");
    expect(screen.getByTestId("console-review-apply")).toBeDisabled();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    expect(link).toHaveFocus();
  });

  it("reports a stale plan refusal with a replan action", () => {
    consoleState.plan = makePlan();
    consoleState.apply.error = apiErrorOf(409, "plan_stale", "The plan is stale", { next_action: { owner: "scenario-to-cloud", kind: "replan", reference: "plan" } });
    const deployment = deploymentState({ deploymentStatus: "review", deploymentId: "deployment-1" });
    renderWithProviders(<StepDeploy deployment={deployment} />);
    expect(screen.getByTestId("console-review-apply-refusal")).toHaveTextContent("plan_stale");
    fireEvent.click(screen.getByTestId("console-review-replan"));
    expect(consoleState.apply.reset).toHaveBeenCalled();
  });

  // [REQ:STC-P0-039] The operation view follows the durable standing and
  // completes the wizard from the terminal state, never from a stream event.
  it("follows the durable operation to completion and exposes agent investigation controls", async () => {
    consoleState.pointer = { deployment_id: "deployment-1", operation_id: "op-1234567890", plan_digest: "sha256:plan-digest-fixture" };
    consoleState.standing = makeStanding({ state: "succeeded", terminal: true, active_step: undefined, completed_steps: ["host.prepare", "release.stage", "release.activate"], result: { outcome: "succeeded", completed_steps: 3, message: "Release activated" }, next_action: undefined });
    const deployment = deploymentState({ deploymentStatus: "deploying", deploymentId: "deployment-1" });
    const currentInvestigation = investigationState({ activeInvestigation: investigation, isRunning: true });
    hooks.useDeploymentInvestigation.mockReturnValue(currentInvestigation);
    hooks.useDeploymentRecord.mockReturnValue({ data: { last_deployed_at: "2026-08-14T12:00:00.000Z" } });
    renderWithProviders(<StepDeploy deployment={deployment} />);

    await waitFor(() => expect(deployment.onDeploymentComplete).toHaveBeenCalledWith(true, undefined));
    expect(screen.getByTestId("console-operation-state")).toHaveTextContent("succeeded");
    expect(screen.getByTestId("console-operation-steps").querySelectorAll("li")).toHaveLength(3);
    fireEvent.click(screen.getByRole("button", { name: "Mock spawn agent" }));
    fireEvent.click(screen.getByRole("button", { name: "Mock stop investigation" }));
    expect(currentInvestigation.stop).toHaveBeenCalledOnce();
    expect(screen.getByText("Outdated investigation")).toBeInTheDocument();
  });

  it("resumes an interrupted operation from durable state and lets a failed one be re-reviewed", () => {
    consoleState.pointer = { deployment_id: "deployment-1", operation_id: "op-1234567890", plan_digest: "sha256:plan-digest-fixture" };
    consoleState.resumed = true;
    consoleState.standing = makeStanding();
    const deploying = deploymentState({ deploymentStatus: "deploying", deploymentId: "deployment-1" });
    renderWithProviders(<StepDeploy deployment={deploying} />);
    expect(screen.getByTestId("console-operation")).toHaveAttribute("data-state", "interrupted");
    expect(screen.getByTestId("console-operation-resumed")).toBeInTheDocument();
    expect(screen.getByTestId("console-operation-reattach")).toHaveTextContent("scenario-to-cloud operation wait op-1234567890");
    expect(deploying.onDeploymentComplete).not.toHaveBeenCalled();
    cleanup();

    consoleState.resumed = false;
    consoleState.standing = makeStanding({ state: "failed", terminal: true, error: { code: "release_verification_failed", message: "digest mismatch" }, next_action: { owner: "scenario-to-cloud", kind: "operation", reference: "/api/v1/operations/op-1234567890" } });
    const failed = deploymentState({ deploymentStatus: "failed", deploymentId: "deployment-1", deploymentError: "digest mismatch" });
    renderWithProviders(<StepDeploy deployment={failed} />);
    expect(screen.getByTestId("console-operation-refusal")).toHaveTextContent("release_verification_failed");
    fireEvent.click(screen.getByTestId("deploy-retry-button"));
    expect(consoleState.detach).toHaveBeenCalled();
    expect(failed.deploy).toHaveBeenCalledOnce();
  });

  it("shows successful deployment details and manages an investigation report", async () => {
    const deployment = deploymentState({ deploymentStatus: "success", deploymentId: "deployment-1" });
    const currentInvestigation = investigationState({ activeInvestigation: { ...investigation, status: "completed" }, isRunning: false });
    hooks.useDeploymentInvestigation.mockReturnValue(currentInvestigation);
    hooks.useDeploymentRecord.mockReturnValue({ data: {} });
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
