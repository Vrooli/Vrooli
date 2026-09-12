import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import {
  buildBacklogActionMenuItems,
  type BacklogActionMenuDetail,
  type BacklogActionMenuOptions,
} from "./backlog-action-buttons";
import { ActionMenuSheetContent } from "../ui/action-menu";
import type { ItemActions } from "../../lib/backlog-queue-utils";

const baseActions: ItemActions = {
  locked: false,
  terminal: true,
  blocked: false,
  blockingDepKeys: [],
  primaryCta: "followUp",
  canRun: false,
  runDisabled: false,
  canFollowUp: true,
  canRetry: true,
  canArchive: true,
  showDecisionStepper: false,
  agentRunning: false,
  notQueueableReason: null,
  disabledReason: null,
};

function renderMenu(actions: ItemActions, overrides: Partial<BacklogActionMenuOptions> = {}) {
  const detail: BacklogActionMenuDetail = {
    itemActions: actions,
    isLocked: false,
    isTerminal: true,
    agentRunningLabel: "Running",
  };
  const items = buildBacklogActionMenuItems(detail, {
    isUpdating: false,
    onStartRun: () => {},
    onEdit: () => {},
    onFollowUp: () => {},
    onRetry: () => {},
    onArchive: () => {},
    onDelete: () => {},
    ...overrides,
  });
  return render(<ActionMenuSheetContent items={items} />);
}

describe("buildBacklogActionMenuItems", () => {
  it("includes a Retry action when canRetry is true", () => {
    renderMenu(baseActions);
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });

  it("omits the Retry action when canRetry is false", () => {
    renderMenu({ ...baseActions, canRetry: false });
    expect(screen.queryByRole("button", { name: /retry/i })).toBeNull();
  });

  it("includes both Follow Up and Retry when both gates pass", () => {
    renderMenu(baseActions);
    expect(screen.getByRole("button", { name: /follow up/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });

  it("invokes onRetry when the Retry action is selected", () => {
    const onRetry = vi.fn();
    renderMenu(baseActions, { onRetry });
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("excludes legacy workshop actions from the menu", () => {
    renderMenu({
      ...baseActions,
      terminal: false,
      primaryCta: "run",
      canRun: true,
      canFollowUp: false,
      canRetry: false,
    });
    expect(screen.queryByRole("button", { name: "Run" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Workshop" })).toBeNull();
  });
});
