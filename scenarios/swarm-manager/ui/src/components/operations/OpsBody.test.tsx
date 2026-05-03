import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";
import { OpsBody } from "./OpsBody";
import { selectors } from "../../consts/selectors";
import type {
  ActivityRow,
  OperationsViewMode,
} from "../../types/operations";

function row(overrides: Partial<ActivityRow>): ActivityRow {
  return {
    activityId: "a-x",
    runId: "run-x",
    ownerType: "backlog",
    ownerName: "fix-foo",
    purpose: "process",
    status: "running",
    requestedAt: "2026-05-02T01:00:00Z",
    ...overrides,
  };
}

function renderBody(
  props: Partial<{
    view: OperationsViewMode;
    onViewChange: (next: OperationsViewMode) => void;
    activities: ActivityRow[];
    recentlyFinished: ActivityRow[];
    enableByPhaseView: boolean;
    selectionMode: boolean;
    onSelectionModeToggle: () => void;
  }> = {},
) {
  const onViewChange = props.onViewChange ?? vi.fn();
  const result = render(
    <MemoryRouter>
      <OpsBody
        view={props.view ?? "by-initiative"}
        onViewChange={onViewChange}
        activities={props.activities ?? []}
        recentlyFinished={props.recentlyFinished ?? []}
        enableByPhaseView={props.enableByPhaseView ?? true}
        selectionMode={props.selectionMode}
        onSelectionModeToggle={props.onSelectionModeToggle}
      />
    </MemoryRouter>,
  );
  return { ...result, onViewChange };
}

describe("OpsBody", () => {
  it("renders the by-initiative view by default", () => {
    renderBody({
      activities: [
        row({
          activityId: "a-1",
          ownerType: "initiative",
          ownerName: "auth-rewrite",
          ownerTitle: "Auth Rewrite",
          initiativeName: "auth-rewrite",
          lane: "execute",
        }),
      ],
    });
    expect(
      screen.getByTestId(selectors.operationsCenter.initiativeCard),
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId(selectors.operationsCenter.byPhaseBoard),
    ).toBeNull();
  });

  it("renders the by-phase board when view='by-phase' and the toggle is enabled", () => {
    renderBody({
      view: "by-phase",
      enableByPhaseView: true,
      activities: [
        row({
          activityId: "a-1",
          lane: "execute",
          ownerTitle: "Backlog item",
        }),
      ],
    });
    expect(
      screen.getByTestId(selectors.operationsCenter.byPhaseBoard),
    ).toBeInTheDocument();
    expect(
      screen.queryByTestId(selectors.operationsCenter.initiativeCard),
    ).toBeNull();
  });

  it("falls back to by-initiative when view='by-phase' but the toggle is disabled", () => {
    renderBody({
      view: "by-phase",
      enableByPhaseView: false,
      activities: [
        row({
          activityId: "a-1",
          ownerType: "initiative",
          ownerName: "auth-rewrite",
          ownerTitle: "Auth Rewrite",
          initiativeName: "auth-rewrite",
        }),
      ],
    });
    expect(
      screen.queryByTestId(selectors.operationsCenter.byPhaseBoard),
    ).toBeNull();
    expect(
      screen.getByTestId(selectors.operationsCenter.initiativeCard),
    ).toBeInTheDocument();
  });

  it("calls onViewChange('by-phase') when the by-phase tab is clicked and enabled", async () => {
    const onViewChange = vi.fn();
    renderBody({ onViewChange, enableByPhaseView: true });
    await userEvent.click(
      screen.getByTestId(selectors.operationsCenter.viewToggleByPhase),
    );
    expect(onViewChange).toHaveBeenCalledWith("by-phase");
  });

  it("does not call onViewChange when the by-phase tab is disabled", async () => {
    const onViewChange = vi.fn();
    renderBody({ onViewChange, enableByPhaseView: false });
    const tab = screen.getByTestId(
      selectors.operationsCenter.viewToggleByPhase,
    );
    expect(tab).toBeDisabled();
    // userEvent skips disabled buttons; double-check by direct click too.
    await userEvent.click(tab).catch(() => undefined);
    expect(onViewChange).not.toHaveBeenCalled();
  });

  it("marks the active tab via aria-selected", () => {
    const { rerender } = renderBody({
      view: "by-initiative",
      enableByPhaseView: true,
    });
    expect(
      screen.getByTestId(selectors.operationsCenter.viewToggleByInitiative),
    ).toHaveAttribute("aria-selected", "true");
    expect(
      screen.getByTestId(selectors.operationsCenter.viewToggleByPhase),
    ).toHaveAttribute("aria-selected", "false");

    rerender(
      <MemoryRouter>
        <OpsBody
          view="by-phase"
          onViewChange={vi.fn()}
          activities={[]}
          recentlyFinished={[]}
          enableByPhaseView={true}
        />
      </MemoryRouter>,
    );
    expect(
      screen.getByTestId(selectors.operationsCenter.viewToggleByPhase),
    ).toHaveAttribute("aria-selected", "true");
    expect(
      screen.getByTestId(selectors.operationsCenter.viewToggleByInitiative),
    ).toHaveAttribute("aria-selected", "false");
  });

  it("renders the recently-finished tail when present", () => {
    renderBody({
      recentlyFinished: [
        row({
          activityId: "f-1",
          runId: "run-f-1",
          status: "complete",
          ownerTitle: "Finished item",
        }),
      ],
    });
    expect(screen.getByText(/Recently finished/i)).toBeInTheDocument();
    expect(screen.getByText("Finished item")).toBeInTheDocument();
  });

  it("hides the recently-finished tail when empty", () => {
    renderBody({ recentlyFinished: [] });
    expect(screen.queryByText(/Recently finished/i)).toBeNull();
  });

  describe("selection-mode toggle", () => {
    it("does not render the toggle when no handler is provided", () => {
      renderBody();
      expect(
        screen.queryByTestId(selectors.operationsCenter.selectionModeToggle),
      ).toBeNull();
    });

    it("renders the toggle when an onSelectionModeToggle handler is supplied", () => {
      renderBody({ onSelectionModeToggle: vi.fn() });
      expect(
        screen.getByTestId(selectors.operationsCenter.selectionModeToggle),
      ).toBeInTheDocument();
    });

    it("reflects the selectionMode prop via aria-pressed and label", () => {
      const { rerender } = renderBody({
        selectionMode: false,
        onSelectionModeToggle: vi.fn(),
      });
      const toggle = screen.getByTestId(
        selectors.operationsCenter.selectionModeToggle,
      );
      expect(toggle).toHaveAttribute("aria-pressed", "false");
      expect(toggle).toHaveTextContent(/^Select$/);

      rerender(
        <MemoryRouter>
          <OpsBody
            view="by-initiative"
            onViewChange={vi.fn()}
            activities={[]}
            recentlyFinished={[]}
            enableByPhaseView
            selectionMode
            onSelectionModeToggle={vi.fn()}
          />
        </MemoryRouter>,
      );
      const onToggle = screen.getByTestId(
        selectors.operationsCenter.selectionModeToggle,
      );
      expect(onToggle).toHaveAttribute("aria-pressed", "true");
      expect(onToggle).toHaveTextContent(/Selecting/);
    });

    it("calls onSelectionModeToggle when clicked", async () => {
      const onSelectionModeToggle = vi.fn();
      renderBody({ onSelectionModeToggle });
      await userEvent.click(
        screen.getByTestId(selectors.operationsCenter.selectionModeToggle),
      );
      expect(onSelectionModeToggle).toHaveBeenCalledTimes(1);
    });
  });
});
