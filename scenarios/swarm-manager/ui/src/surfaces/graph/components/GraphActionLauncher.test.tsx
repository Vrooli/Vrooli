import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { GraphActionLauncher } from "./GraphActionLauncher";
// Menu entries are addressed through the shared label constants so that
// rewording a launcher card does not silently break these tests.
import { SESSION_KIND_DESCRIPTIONS, SESSION_KIND_LAUNCHER_LABELS } from "../../../components/session/session-view-model";

describe("GraphActionLauncher", () => {
  it("opens a slide-up action sheet and dispatches launcher actions", () => {
    const onQuickCapture = vi.fn();
    const onPlanWork = vi.fn();
    const onManageSwarm = vi.fn();
	const onAuthorWorkflow = vi.fn();
    const onCreateFromPlan = vi.fn();

    render(
      <GraphActionLauncher
        onQuickCapture={onQuickCapture}
        onPlanWork={onPlanWork}
        onManageSwarm={onManageSwarm}
		onAuthorWorkflow={onAuthorWorkflow}
        onCreateFromPlan={onCreateFromPlan}
      />,
    );

    fireEvent.click(screen.getByTestId("graph-action-fab"));

    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByRole("menu")).toBeInTheDocument();
    expect(screen.getByText("Capture a note, task, dependency, or relationship without starting an agent session.")).toBeInTheDocument();
    expect(screen.getByText(SESSION_KIND_DESCRIPTIONS.swarm_operations)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("menuitem", { name: "Quick Capture" }));
    expect(onQuickCapture).toHaveBeenCalledTimes(1);
    expect(screen.queryByRole("menu")).toBeNull();

    fireEvent.click(screen.getByTestId("graph-action-fab"));
    fireEvent.click(screen.getByRole("menuitem", { name: "Plan Work With Agent" }));
    expect(onPlanWork).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("graph-action-fab"));
    fireEvent.click(screen.getByRole("menuitem", { name: "Manage Swarm" }));
    expect(onManageSwarm).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("graph-action-fab"));
	fireEvent.click(screen.getByRole("menuitem", { name: SESSION_KIND_LAUNCHER_LABELS.workflow_authoring }));
	expect(onAuthorWorkflow).toHaveBeenCalledTimes(1);

	fireEvent.click(screen.getByTestId("graph-action-fab"));
    fireEvent.click(screen.getByRole("menuitem", { name: "Create from plan" }));
    expect(onCreateFromPlan).toHaveBeenCalledTimes(1);
  });

  it("disables session actions while busy and shows launcher status outside the menu", () => {
    render(
      <GraphActionLauncher
        isBusy
        status="Starting session..."
        onQuickCapture={vi.fn()}
        onPlanWork={vi.fn()}
        onManageSwarm={vi.fn()}
		onAuthorWorkflow={vi.fn()}
        onCreateFromPlan={vi.fn()}
      />,
    );

    expect(screen.getByRole("status")).toHaveTextContent("Starting session...");

    fireEvent.click(screen.getByTestId("graph-action-fab"));

    expect(screen.getByRole("menuitem", { name: "Quick Capture" })).toBeEnabled();
    expect(screen.getByRole("menuitem", { name: "Plan Work With Agent" })).toBeDisabled();
    expect(screen.getByRole("menuitem", { name: "Manage Swarm" })).toBeDisabled();
	expect(screen.getByRole("menuitem", { name: SESSION_KIND_LAUNCHER_LABELS.workflow_authoring })).toBeDisabled();
  });

  it("shows dismissible launcher errors outside the closed menu", () => {
    const onDismissError = vi.fn();

    render(
      <GraphActionLauncher
        error="Unable to start session."
        onDismissError={onDismissError}
        onQuickCapture={vi.fn()}
        onPlanWork={vi.fn()}
        onManageSwarm={vi.fn()}
		onAuthorWorkflow={vi.fn()}
        onCreateFromPlan={vi.fn()}
      />,
    );

    expect(screen.queryByRole("menu")).toBeNull();
    expect(screen.getByRole("alert")).toHaveTextContent("Unable to start session.");
    fireEvent.click(screen.getByRole("button", { name: "Dismiss error" }));
    expect(onDismissError).toHaveBeenCalledTimes(1);
  });

  it("closes on Escape", () => {
    render(
      <GraphActionLauncher
        onQuickCapture={vi.fn()}
        onPlanWork={vi.fn()}
        onManageSwarm={vi.fn()}
		onAuthorWorkflow={vi.fn()}
        onCreateFromPlan={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByTestId("graph-action-fab"));
    expect(screen.getByRole("menu")).toBeInTheDocument();

    fireEvent.keyDown(document, { key: "Escape" });
    expect(screen.queryByRole("menu")).toBeNull();
  });
});
