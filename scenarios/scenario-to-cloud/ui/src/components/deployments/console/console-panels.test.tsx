import "@testing-library/jest-dom";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { apiErrorOf, makeMatrix, makeObservation, makePlan, makeRecoveryPoint, makeStanding, FIXTURE_TARGET_KEY } from "../../../test-utils/consoleFixtures";
import { renderWithProviders } from "../../../test-utils/renderWithProviders";
import { DeniedState } from "./DeniedState";
import { HealthPanel } from "./HealthPanel";
import { IdentityHeader } from "./IdentityHeader";
import { NextActionControl } from "./NextActionControl";
import { OperationPanel } from "./OperationPanel";
import { PlanReview } from "./PlanReview";
import { RecoveryPanel } from "./RecoveryPanel";
import { ReleasePanel } from "./ReleasePanel";
import { ActionButton, CopyButton } from "./ConsolePrimitives";

// provider-free-exception: these panels accept all state via props.

describe("IdentityHeader", () => {
  // [REQ:STC-P0-038] Environment, machine, transport and id are always visible and copyable.
  it("shows environment, target key, transport, id, fence and domain with copy affordances", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, { clipboard: { writeText } });
    const onBack = vi.fn();
    render(
      <IdentityHeader
        deployment={{ id: "dep-1", name: "Demo", scenario_id: "demo", status: "deployed", environment: "staging", fence: 9, target: { machine_id: "m-1", node_id: "n-1", enrollment_generation: 2, transport: "bridge", locator: { host: "203.0.113.10" } } }}
        domain="demo.example"
        onBack={onBack}
      />,
    );
    expect(screen.getByTestId("console-identity-environment")).toHaveTextContent("staging");
    expect(screen.getByTestId("console-identity-target")).toHaveTextContent("machine:m-1");
    expect(screen.getByTestId("console-identity-target")).toHaveTextContent("gen 2");
    expect(screen.getByTestId("console-identity-transport")).toHaveTextContent("bridge");
    expect(screen.getByTestId("console-identity-transport")).toHaveTextContent("203.0.113.10");
    expect(screen.getByTestId("console-identity-id")).toHaveTextContent("dep-1");
    expect(screen.getByTestId("console-identity-fence")).toHaveTextContent("9");
    expect(screen.getByRole("link", { name: "demo.example" })).toHaveAttribute("href", "https://demo.example");
    await userEvent.click(screen.getByRole("button", { name: "Copy target key" }));
    expect(writeText).toHaveBeenCalledWith("machine:m-1");
    await waitFor(() => expect(screen.getByRole("button", { name: "target key copied" })).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("console-back"));
    expect(onBack).toHaveBeenCalled();
  });

  it("falls back to host and unbound targets and defaults the environment", () => {
    const { rerender } = render(<IdentityHeader deployment={{ id: "d", name: "n", scenario_id: "s", status: "pending", target: { transport: "ssh", locator: { host: "h.example" } } }} onBack={() => {}} />);
    expect(screen.getByTestId("console-identity-target")).toHaveTextContent("host:h.example");
    expect(screen.getByTestId("console-identity-environment")).toHaveTextContent("production");
    rerender(<IdentityHeader deployment={{ id: "d", name: "n", scenario_id: "s", status: "pending" }} onBack={() => {}} />);
    expect(screen.getByTestId("console-identity-target")).toHaveTextContent("unbound");
    expect(screen.getByTestId("console-identity-transport")).toHaveTextContent("unbound");
  });

  it("reports a failed clipboard write without crashing", async () => {
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockRejectedValue(new Error("denied")) } });
    render(<CopyButton value="x" label="thing" />);
    await userEvent.click(screen.getByRole("button", { name: "Copy thing" }));
    expect(screen.getByRole("button", { name: "Copy thing" })).toBeInTheDocument();
  });
});

describe("ReleasePanel", () => {
  // [REQ:STC-P0-038] Desired and observed releases are shown separately with a mismatch indicator.
  it("marks a desired/observed mismatch and a configuration mismatch", () => {
    const plan = makePlan({ release_digest: "sha256:desired00000", configuration_digest: "sha256:cfgA" });
    const observation = makeObservation({ observed_release_digest: "sha256:observed0000", observed_configuration_digest: "sha256:cfgB" });
    render(<ReleasePanel deploymentId="dep-1" plan={plan} planError={null} planLoading={false} observation={observation} />);
    expect(screen.getByTestId("console-release")).toHaveAttribute("data-state", "degraded");
    expect(screen.getByTestId("console-release-mismatch")).toBeInTheDocument();
    expect(screen.getByTestId("console-release-desired")).toHaveTextContent("desired00000");
    expect(screen.getByTestId("console-release-observed")).toHaveTextContent("observed0000");
    expect(screen.getByTestId("console-release-config-mismatch")).toBeInTheDocument();
    expect(screen.getByTestId("console-release-outcome")).toHaveTextContent("apply");
  });

  it("shows in sync, unknown, loading and refusal states", () => {
    const { rerender } = render(<ReleasePanel deploymentId="dep-1" plan={makePlan()} planError={null} planLoading={false} observation={makeObservation()} />);
    expect(screen.getByTestId("console-release-match")).toBeInTheDocument();
    rerender(<ReleasePanel deploymentId="dep-1" plan={null} planError={null} planLoading={true} observation={null} />);
    expect(screen.getByTestId("console-release")).toHaveAttribute("data-state", "loading");
    rerender(<ReleasePanel deploymentId="dep-1" plan={null} planError={apiErrorOf(424, "closure_unavailable", "closure missing")} planLoading={false} observation={null} />);
    expect(screen.getByTestId("console-release-plan-error")).toHaveTextContent("closure_unavailable");
    expect(screen.getByTestId("console-release-unknown")).toBeInTheDocument();
    expect(screen.getByTestId("console-release-desired")).toHaveTextContent("not compiled");
    rerender(<ReleasePanel deploymentId="dep-1" plan={null} planError={apiErrorOf(403, "forbidden_scope", "denied", { details: { required_scope: "scenario-to-cloud:read" } })} planLoading={false} observation={null} matrix={makeMatrix()} target="machine:m" />);
    expect(screen.getByTestId("console-release")).toHaveAttribute("data-state", "denied");
    expect(screen.getByTestId("console-release-denied-scope")).toHaveTextContent("scenario-to-cloud:read");
  });
});

describe("HealthPanel", () => {
  const now = () => Date.parse("2026-09-09T12:05:00Z");

  // [REQ:STC-P0-038] Verdict, freshness, age and producer are shown separately.
  it("renders a healthy current observation with age, producer and checks", () => {
    render(<HealthPanel observation={makeObservation()} loading={false} error={null} now={now} />);
    expect(screen.getByTestId("console-health")).toHaveAttribute("data-state", "ready");
    expect(screen.getByTestId("console-health-status")).toHaveTextContent("Healthy");
    expect(screen.getByTestId("console-health-freshness")).toHaveTextContent("current");
    expect(screen.getByTestId("console-health-observed-at")).toHaveTextContent("5m ago");
    expect(screen.getByTestId("console-health-producer")).toHaveTextContent("scenario-to-cloud:health:v1");
    expect(screen.getByTestId("console-health-checks").querySelectorAll("li")).toHaveLength(2);
    expect(screen.getByTestId("console-health-live")).toHaveTextContent("Health Healthy, current, observed 5m ago");
  });

  it("never renders stale, partial or unknown evidence as ready", () => {
    const { rerender } = render(<HealthPanel observation={makeObservation({ status: "HEALTH_STATUS_HEALTHY", freshness: "FRESHNESS_STALE" })} loading={false} error={null} now={now} />);
    expect(screen.getByTestId("console-health")).toHaveAttribute("data-state", "degraded");
    expect(screen.getByTestId("console-health-freshness")).toHaveTextContent("stale evidence");
    rerender(<HealthPanel observation={makeObservation({ status: "HEALTH_STATUS_UNSPECIFIED", partial: true, missing_dependencies: ["edge_dns"], checks: [{ id: "edge_dns", status: "CHECK_STATUS_UNAVAILABLE", reason_code: "dns_unresolved", detail: "no answer" }], next_actions: [{ owner: "scenario-to-cloud", kind: "operation", reference: "/x", label: "Check DNS" }] })} loading={false} error={null} now={now} />);
    expect(screen.getByTestId("console-health-status")).toHaveTextContent("Health unknown");
    expect(screen.getByTestId("console-health-partial")).toHaveTextContent("edge_dns");
    expect(screen.getByTestId("console-health-next-action-0")).toHaveTextContent("Check DNS");
    rerender(<HealthPanel observation={makeObservation({ status: "HEALTH_STATUS_UNHEALTHY", checks: [{ id: "application_readiness", status: "CHECK_STATUS_FAILED" }, { id: "x", status: "CHECK_STATUS_WARNED" }] })} loading={false} error={null} now={now} />);
    expect(screen.getByTestId("console-health-status")).toHaveTextContent("Unhealthy");
    rerender(<HealthPanel observation={makeObservation({ status: "HEALTH_STATUS_DEGRADED", observed_at: undefined, producer_ref: "" })} loading={false} error={null} now={now} />);
    expect(screen.getByTestId("console-health-observed-at")).toHaveTextContent("time unknown");
    expect(screen.getByTestId("console-health-producer")).toHaveTextContent("unknown");
  });

  it("renders loading, empty and error states", () => {
    const { rerender } = render(<HealthPanel observation={null} loading={true} error={null} />);
    expect(screen.getByTestId("console-health")).toHaveAttribute("data-state", "loading");
    expect(screen.getByTestId("console-health")).toHaveAttribute("aria-busy", "true");
    rerender(<HealthPanel observation={null} loading={false} error={null} />);
    expect(screen.getByTestId("console-health-empty")).toHaveTextContent("No health observation has been produced");
    rerender(<HealthPanel observation={null} loading={false} error={new Error("reach_unavailable")} />);
    expect(screen.getByTestId("console-health-empty")).toHaveTextContent("reach_unavailable");
  });
});

describe("OperationPanel", () => {
  // [REQ:STC-P0-038] Completed steps and the active step are listed; no percentage exists.
  it("lists completed and active steps, the next action, the reattach command and receipts", () => {
    render(<OperationPanel standing={makeStanding()} loading={false} error={null} target={FIXTURE_TARGET_KEY} liveMessage="Staging release" />);
    const steps = screen.getByTestId("console-operation-steps").querySelectorAll("li");
    expect(steps).toHaveLength(4);
    expect(steps[3]).toHaveAttribute("data-outcome", "active");
    expect(screen.getByTestId("console-operation-state")).toHaveTextContent("running");
    expect(screen.getByTestId("console-operation-next-action")).toHaveTextContent("Keep waiting for the operation");
    expect(screen.getByTestId("console-operation-reattach")).toHaveTextContent("scenario-to-cloud operation wait op-1234567890");
    expect(screen.getByTestId("console-operation-live-message")).toHaveTextContent("Staging release");
    expect(screen.getByTestId("console-operation").textContent).not.toMatch(/%/);
    fireEvent.click(screen.getByRole("button", { name: /Step receipts/ }));
    expect(screen.getByTestId("console-operation-receipts").querySelectorAll("li")).toHaveLength(3);
    expect(screen.getByTestId("console-operation-live")).toHaveTextContent("Operation running, step release.stage");
  });

  // [REQ:STC-P0-039] Interrupted and failed-recovery states move focus and announce.
  it("marks a resumed operation as interrupted and focuses its heading", () => {
    render(<OperationPanel standing={makeStanding()} loading={false} error={null} resumed target={FIXTURE_TARGET_KEY} />);
    expect(screen.getByTestId("console-operation")).toHaveAttribute("data-state", "interrupted");
    expect(screen.getByRole("heading", { name: "Operation" })).toHaveFocus();
    expect(screen.getByTestId("console-operation-live")).toHaveTextContent("resumed from durable state");
  });

  it("renders failed recovery with unknown effects, failed steps and the refusal", () => {
    const handlers = { onRecovery: vi.fn() };
    render(
      <OperationPanel
        standing={makeStanding({
          state: "failed_recovery",
          terminal: true,
          active_step: "release.activate",
          step_receipts: [{ step: "release.activate", outcome: "failed", fence: 3, source: "target", error: "activate refused", completed_at: "t" }],
          unknown_effects: [{ step: "release.activate", fence: 3, reason: "receipt missing", retry: "observe_then_replay", next_action: "inspect the target receipt", recorded_at: "t" }],
          error: { code: "fence_stale", message: "fence 2 < 3", next_action: { owner: "scenario-to-cloud", kind: "recovery", reference: "/api/v1/deployments/d/recovery" } },
          next_action: { owner: "scenario-to-cloud", kind: "recovery", reference: "/api/v1/deployments/d/recovery" },
        })}
        loading={false}
        error={null}
        target={FIXTURE_TARGET_KEY}
        handlers={handlers}
      />,
    );
    expect(screen.getByTestId("console-operation")).toHaveAttribute("data-state", "failed-recovery");
    expect(screen.getByTestId("console-operation-unknown-effects")).toHaveTextContent("observe_then_replay");
    expect(screen.getByTestId("console-operation-refusal")).toHaveTextContent("fence_stale");
    const failed = screen.getByTestId("console-operation-steps").querySelector('[data-outcome="failed"]');
    expect(failed).toHaveTextContent("release.activate");
    fireEvent.click(screen.getByTestId("console-operation-next-action"));
    expect(handlers.onRecovery).toHaveBeenCalled();
    expect(screen.getByRole("heading", { name: "Operation" })).toHaveFocus();
  });

  it("renders empty, loading, denied and plain error states", () => {
    const { rerender } = render(<OperationPanel standing={null} loading={false} error={null} target="t" emptyAction={<button type="button">Review plan</button>} />);
    expect(screen.getByTestId("console-operation")).toHaveAttribute("data-state", "empty");
    expect(screen.getByRole("button", { name: "Review plan" })).toBeInTheDocument();
    rerender(<OperationPanel standing={null} loading={true} error={null} target="t" />);
    expect(screen.getByTestId("console-operation")).toHaveAttribute("data-state", "loading");
    rerender(<OperationPanel standing={null} loading={false} error={apiErrorOf(403, "forbidden_target", "denied")} target="machine:m-1" matrix={makeMatrix()} />);
    expect(screen.getByTestId("console-operation")).toHaveAttribute("data-state", "denied");
    expect(screen.getByTestId("console-operation-denied-target")).toHaveTextContent("machine:m-1");
    rerender(<OperationPanel standing={null} loading={false} error={new Error("boom")} target="t" />);
    expect(screen.getByTestId("console-operation-error")).toHaveTextContent("boom");
    rerender(<OperationPanel standing={makeStanding({ state: "succeeded", terminal: true, active_step: undefined, next_action: undefined, result: { outcome: "succeeded", completed_steps: 3, message: "done" } })} loading={false} error={null} target="t" />);
    expect(screen.getByTestId("console-operation-result")).toHaveTextContent("done");
    expect(screen.getByTestId("console-operation-next")).toHaveTextContent("no further action was named");
  });

  // [REQ:STC-P0-039] Cancel is previewed and confirmed; refusals are typed.
  it("cancels through the preview dialog and shows the cancel refusal", async () => {
    const onCancel = vi.fn().mockResolvedValue(undefined);
    const { rerender } = render(<OperationPanel standing={makeStanding()} loading={false} error={null} target={FIXTURE_TARGET_KEY} onCancel={onCancel} />);
    await userEvent.click(screen.getByTestId("console-operation-cancel"));
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveTextContent(FIXTURE_TARGET_KEY);
    expect(dialog).toHaveTextContent("No declared data bindings");
    await userEvent.type(screen.getByTestId("console-destructive-confirm-input"), "op-12345");
    await userEvent.click(screen.getByTestId("console-destructive-confirm"));
    await waitFor(() => expect(onCancel).toHaveBeenCalledWith("op-1234567890"));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    rerender(<OperationPanel standing={makeStanding({ cancel_requested: true })} loading={false} error={null} target={FIXTURE_TARGET_KEY} onCancel={onCancel} cancelError={apiErrorOf(403, "forbidden_scope", "denied", { details: { required_scope: "scenario-to-cloud:destructive" } })} />);
    expect(screen.getByTestId("console-operation-cancel")).toBeDisabled();
    expect(screen.getByTestId("console-operation-cancel-denied-scope")).toHaveTextContent("scenario-to-cloud:destructive");
  });
});

describe("RecoveryPanel", () => {
  // [REQ:STC-P0-039] Destructive restore names the affected data and the target before confirmation.
  it("checks eligibility through the API and previews a restore naming bindings and target", async () => {
    const onCheckRollback = vi.fn().mockResolvedValue({ compatible: true, schema_strategy: "same_schema", current_schema: "12", target_schema: "12" });
    const onRestore = vi.fn().mockResolvedValue({});
    renderWithProviders(<RecoveryPanel deploymentId="0f4d2c1a-rest" target={FIXTURE_TARGET_KEY} recoveryPoints={[makeRecoveryPoint()]} loading={false} error={null} onCheckRollback={onCheckRollback} onRestore={onRestore} />);
    expect(screen.getByTestId("console-recovery")).toHaveAttribute("data-state", "ready");
    await userEvent.click(screen.getByRole("button", { name: "rp-0001" }));
    await userEvent.click(screen.getByTestId("console-recovery-check-rollback"));
    await waitFor(() => expect(screen.getByTestId("console-recovery-verdict")).toHaveAttribute("data-compatible", "true"));
    expect(onCheckRollback).toHaveBeenCalledWith({ recoveryPointId: "rp-0001", currentSchema: "12", targetSchema: "12" });
    await userEvent.click(screen.getByTestId("console-recovery-restore"));
    const dialog = screen.getByRole("dialog");
    expect(screen.getByTestId("console-destructive-target")).toHaveTextContent(FIXTURE_TARGET_KEY);
    expect(screen.getByTestId("console-destructive-data")).toHaveTextContent("postgres:main (postgres) → postgres://fixture-db:5432/app");
    expect(dialog).toHaveTextContent("rp-0001");
    await userEvent.type(screen.getByTestId("console-destructive-confirm-input"), "0f4d2c1a");
    await userEvent.click(screen.getByTestId("console-destructive-confirm"));
    await waitFor(() => expect(onRestore).toHaveBeenCalledWith({ recoveryPointId: "rp-0001", into: { "postgres:main": "postgres://fixture-db:5432/app" } }));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("disables restore on an incompatible verdict and shows typed refusals", async () => {
    const onCheckRollback = vi.fn().mockResolvedValue({ compatible: false, schema_strategy: "forward_only", current_schema: "13", target_schema: "12", reason_code: "schema_forward_only", reason: "schema 13 cannot roll back to 12", plan: { kind: "forward_repair", preconditions: [], steps: ["apply forward repair", "restore rp-0001"] } });
    const onRestore = vi.fn();
    renderWithProviders(<RecoveryPanel deploymentId="dep-1" target="t" recoveryPoints={[makeRecoveryPoint({ id: "rp-new", schema_version: "13" }), makeRecoveryPoint()]} loading={false} error={null} onCheckRollback={onCheckRollback} onRestore={onRestore} />);
    await userEvent.click(screen.getByRole("button", { name: "rp-0001" }));
    await userEvent.click(screen.getByTestId("console-recovery-check-rollback"));
    await waitFor(() => expect(screen.getByTestId("console-recovery-verdict")).toHaveAttribute("data-compatible", "false"));
    expect(screen.getByTestId("console-recovery-verdict")).toHaveTextContent("apply forward repair");
    expect(screen.getByTestId("console-recovery-restore")).toBeDisabled();
    expect(onCheckRollback).toHaveBeenCalledWith({ recoveryPointId: "rp-0001", currentSchema: "13", targetSchema: "12" });

    const refusing = vi.fn().mockRejectedValue(apiErrorOf(409, "rollback_incompatible", "not admitted", { next_action: { owner: "scenario-to-cloud", kind: "forward_repair", reference: "rp-0001", label: "run forward repair" } }));
    renderWithProviders(<RecoveryPanel deploymentId="dep-2" target="t" recoveryPoints={[makeRecoveryPoint({ id: "rp-x" })]} loading={false} error={null} onCheckRollback={refusing} onRestore={onRestore} />);
    await userEvent.click(screen.getByRole("button", { name: "rp-x" }));
    const checks = screen.getAllByTestId("console-recovery-check-rollback");
    const secondCheck = checks[1];
    if (!secondCheck) throw new Error("expected a second eligibility button");
    await userEvent.click(secondCheck);
    await waitFor(() => expect(screen.getByTestId("console-recovery-verdict-refusal")).toHaveTextContent("rollback_incompatible"));
    expect(screen.getByTestId("console-recovery-verdict-next-action")).toHaveTextContent("run forward repair");
  });

  it("shows a denied restore, a refused restore, the handoff, empty and denied listing states", async () => {
    const onRestore = vi.fn().mockRejectedValueOnce(apiErrorOf(403, "forbidden_scope", "denied", { details: { required_scope: "scenario-to-cloud:destructive" } })).mockRejectedValueOnce(apiErrorOf(409, "restore_target_not_clean", "binding has rows"));
    const { rerender } = renderWithProviders(<RecoveryPanel deploymentId="dep-1" target="t" recoveryPoints={[makeRecoveryPoint({ bindings: [] })]} loading={false} error={null} onRestore={onRestore} />);
    await userEvent.click(screen.getByRole("button", { name: "rp-0001" }));
    await userEvent.click(screen.getByTestId("console-recovery-restore"));
    expect(screen.getByTestId("console-destructive-data")).toHaveTextContent("postgres:main");
    await userEvent.type(screen.getByTestId("console-destructive-confirm-input"), "dep-1");
    await userEvent.click(screen.getByTestId("console-destructive-confirm"));
    await waitFor(() => expect(screen.getByTestId("console-recovery-restore-denied-scope")).toHaveTextContent("scenario-to-cloud:destructive"));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    await userEvent.click(screen.getByTestId("console-recovery-restore"));
    await userEvent.type(screen.getByTestId("console-destructive-confirm-input"), "dep-1");
    await userEvent.click(screen.getByTestId("console-destructive-confirm"));
    await waitFor(() => expect(screen.getByTestId("console-destructive-error")).toHaveTextContent("binding has rows"));
    await userEvent.click(screen.getByTestId("console-destructive-cancel"));
    expect(screen.getByTestId("console-recovery-restore-refusal")).toHaveTextContent("restore_target_not_clean");

    rerender(<RecoveryPanel deploymentId="dep-1" target="t" recoveryPoints={[]} loading={false} error={null} plan={makePlan({ outcome: "needs_input", handoff: { owner: "vrooli-onboarding", kind: "resume_handoff", reference: "vrooli-onboarding://deployments/dep-1/resume/x", missing: ["credential:db"] } })} />);
    expect(screen.getByTestId("console-recovery")).toHaveAttribute("data-state", "needs-input");
    expect(screen.getByTestId("console-recovery-handoff-link")).toHaveAttribute("href", "vrooli-onboarding://deployments/dep-1/resume/x");
    rerender(<RecoveryPanel deploymentId="dep-1" target="t" recoveryPoints={[]} loading={false} error={null} />);
    expect(screen.getByTestId("console-recovery-empty")).toBeInTheDocument();
    rerender(<RecoveryPanel deploymentId="dep-1" target="t" recoveryPoints={null} loading={true} error={null} />);
    expect(screen.getByTestId("console-recovery")).toHaveAttribute("data-state", "loading");
    rerender(<RecoveryPanel deploymentId="dep-1" target="machine:m" recoveryPoints={null} loading={false} error={apiErrorOf(403, "forbidden_target", "denied")} />);
    expect(screen.getByTestId("console-recovery")).toHaveAttribute("data-state", "denied");
    rerender(<RecoveryPanel deploymentId="dep-1" target="t" recoveryPoints={null} loading={false} error={new Error("network")} />);
    expect(screen.getByTestId("console-recovery-error")).toHaveTextContent("network");
  });
});

describe("PlanReview", () => {
  it("renders no-op, error, denied and loading states and confirms apply through the dialog", async () => {
    const onApply = vi.fn().mockResolvedValue(undefined);
    const onReplan = vi.fn();
    const { rerender } = render(<PlanReview deploymentId="0f4d2c1a-x" target="t" plan={makePlan({ outcome: "no_op" }, { changes: [], data_effects: [], shell_preview: [] })} loading={false} error={null} onApply={onApply} applyPending={false} applyError={null} onReplan={onReplan} />);
    expect(screen.getByTestId("console-review")).toHaveAttribute("data-state", "no-op");
    expect(screen.getByTestId("console-review-apply")).toBeDisabled();
    expect(screen.getByTestId("console-review-no-changes")).toBeInTheDocument();
    expect(screen.getByTestId("console-review-no-data-effects")).toBeInTheDocument();
    expect(screen.getByText(/Nothing would change/)).toBeInTheDocument();

    rerender(<PlanReview deploymentId="0f4d2c1a-x" target="t" plan={null} loading={false} error={apiErrorOf(424, "closure_unavailable", "no closure", { next_action: { owner: "scenario-to-cloud", kind: "closure", reference: "docs/closure.md" } })} onApply={onApply} applyPending={false} applyError={null} onReplan={onReplan} />);
    expect(screen.getByTestId("console-review-error")).toHaveTextContent("closure_unavailable");
    fireEvent.click(screen.getByTestId("console-review-retry"));
    expect(onReplan).toHaveBeenCalled();

    rerender(<PlanReview deploymentId="0f4d2c1a-x" target="machine:m" plan={null} loading={false} error={apiErrorOf(403, "forbidden_target", "denied")} onApply={onApply} applyPending={false} applyError={null} onReplan={onReplan} />);
    expect(screen.getByTestId("console-review")).toHaveAttribute("data-state", "denied");
    expect(screen.getByRole("heading", { name: "Target not granted" })).toHaveFocus();

    rerender(<PlanReview deploymentId="0f4d2c1a-x" target="t" plan={null} loading={true} error={null} onApply={onApply} applyPending={false} applyError={null} onReplan={onReplan} />);
    expect(screen.getByTestId("console-review")).toHaveAttribute("data-state", "loading");

    rerender(<PlanReview deploymentId="0f4d2c1a-x" target="t" plan={makePlan()} loading={false} error={null} onApply={onApply} applyPending={false} applyError={null} onReplan={onReplan} />);
    await userEvent.click(screen.getByTestId("console-review-apply"));
    expect(screen.getByTestId("console-destructive-data")).toHaveTextContent("postgres:main (backup_completed)");
    await userEvent.type(screen.getByTestId("console-destructive-confirm-input"), "0f4d2c1a");
    await userEvent.click(screen.getByTestId("console-destructive-confirm"));
    await waitFor(() => expect(onApply).toHaveBeenCalledWith("sha256:plan-digest-fixture"));
  });

  it("shows apply denial, needs_input refusal with handoff and other refusals", () => {
    const base = { deploymentId: "d", target: "t", plan: makePlan(), loading: false, error: null, onApply: vi.fn(), applyPending: false, onReplan: vi.fn() };
    const { rerender } = render(<PlanReview {...base} applyError={apiErrorOf(403, "forbidden_scope", "denied", { details: { required_scope: "scenario-to-cloud:destructive" } })} />);
    expect(screen.getByTestId("console-review-apply-denied-scope")).toHaveTextContent("scenario-to-cloud:destructive");
    rerender(<PlanReview {...base} applyError={apiErrorOf(428, "needs_input", "needs input", { next_action: { owner: "vrooli-onboarding", kind: "resume_handoff", reference: "vrooli-onboarding://deployments/d/resume/y", label: "Resume onboarding" } })} />);
    expect(screen.getByTestId("console-review-apply-handoff")).toHaveAttribute("href", "vrooli-onboarding://deployments/d/resume/y");
    rerender(<PlanReview {...base} applyError={apiErrorOf(409, "request_key_conflict", "conflict", { next_action: { owner: "scenario-to-cloud", kind: "operation", reference: "/api/v1/operations/op-1" } })} />);
    expect(screen.getByTestId("console-review-apply-refusal")).toHaveTextContent("request_key_conflict");
    expect(screen.getByTestId("console-review-apply-next-action")).toHaveTextContent("Inspect the operation");
    rerender(<PlanReview {...base} applyPending applyError={null} />);
    expect(screen.getByTestId("console-review-apply")).toBeDisabled();
  });
});

describe("DeniedState and NextActionControl", () => {
  it("renders nothing for non-authority errors and a sign-in link for unauthenticated", () => {
    const { container, rerender } = render(<DeniedState error={new Error("x")} />);
    expect(container).toBeEmptyDOMElement();
    rerender(<DeniedState error={apiErrorOf(401, "unauthenticated", "no identity", { next_action: { owner: "operator", kind: "sign_in", reference: "https://example/sign-in", label: "Sign in" } })} />);
    expect(screen.getByTestId("console-denied-next-action")).toHaveAttribute("href", "https://example/sign-in");
    rerender(<DeniedState error={apiErrorOf(403, "forbidden_origin", "origin")} />);
    expect(screen.getByTestId("console-denied")).toHaveAttribute("data-code", "forbidden_origin");
  });

  it("maps next_action kinds to links, buttons and text", () => {
    const handlers = { onResume: vi.fn(), onReplan: vi.fn(), onInspectOperation: vi.fn() };
    const { rerender } = render(<NextActionControl next={{ kind: "resume" }} handlers={handlers} />);
    fireEvent.click(screen.getByRole("button"));
    expect(handlers.onResume).toHaveBeenCalled();
    rerender(<NextActionControl next={{ kind: "replan" }} handlers={handlers} />);
    fireEvent.click(screen.getByRole("button"));
    expect(handlers.onReplan).toHaveBeenCalled();
    rerender(<NextActionControl next={{ kind: "operation" }} handlers={handlers} />);
    fireEvent.click(screen.getByRole("button"));
    expect(handlers.onInspectOperation).toHaveBeenCalled();
    rerender(<NextActionControl next={{ kind: "reconcile", owner: "scenario-to-cloud" }} />);
    expect(screen.getByTestId("console-next-action")).toHaveTextContent("owner: scenario-to-cloud");
    rerender(<NextActionControl next={null} />);
    expect(screen.queryByTestId("console-next-action")).not.toBeInTheDocument();
  });

  it("describes unavailable actions next to the disabled control", () => {
    render(
      <ActionButton available={false} reason="needs review">
        Apply
      </ActionButton>,
    );
    const button = screen.getByRole("button", { name: "Apply" });
    expect(button).toBeDisabled();
    expect(button).toHaveAccessibleDescription("needs review");
  });
});
