import { describe, it, expect, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { BacklogCard } from "./backlog-card";
import type { BacklogCardProps } from "./backlog-card";
import type { BacklogItem } from "../../types";
import type { ItemActions } from "../../lib/backlog-queue-utils";

const makeItem = (overrides?: Partial<BacklogItem>): BacklogItem => ({
  name: "test-item",
  title: "Test Item",
  description: "Test description",
  kind: "idea",
  status: "ready",
  priority: 2,
  tags: [],
  suggestedSkills: [],
  created: "2026-03-20T00:00:00Z",
  updated: "2026-03-20T00:00:00Z",
  ...overrides,
});

const makeActions = (overrides?: Partial<ItemActions>): ItemActions => ({
  locked: false,
  terminal: false,
  blocked: false,
  blockingDepKeys: [],
  primaryCta: "run",
  canRun: false,
  runDisabled: false,
  canWorkshop: false,
  workshopDisabled: false,
  canFinalize: false,
  finalizeDisabled: false,
  canFollowUp: false,
  canRetry: false,
  canArchive: false,
  showDecisionStepper: false,
  agentRunning: false,
  notQueueableReason: null,
  disabledReason: null,
  ...overrides,
});

const renderCard = (overrides?: Partial<BacklogCardProps>) => {
  const props: BacklogCardProps = {
    item: makeItem(),
    allItems: [],
    itemActions: makeActions(),
    attentionReasons: [],
    isStepperCompleted: false,
    onStepperCompleted: vi.fn(),
    batchMode: false,
    isSelected: false,
    onToggleSelection: vi.fn(),
    onRun: vi.fn(),
    onArchive: vi.fn(),
    onFollowUp: vi.fn(),
    onFinalize: vi.fn(),
    onWorkshop: vi.fn(),
    archivePending: false,
    finalizePending: false,
    workshopPending: false,
    ...overrides,
  };
  return render(<BacklogCard {...props} />);
};

describe("BacklogCard", () => {
  it("renders finalize as the primary action and next round as secondary", () => {
    renderCard({
      itemActions: makeActions({
        primaryCta: "finalize",
        canFinalize: true,
        canWorkshop: true,
      }),
      workshopLabel: "Next Round",
    });

    const actionRow = screen.getByTestId("backlog-card-actions");
    expect(within(actionRow).getByRole("button", { name: "Finalize" })).toBeInTheDocument();
    expect(within(actionRow).getByRole("button", { name: "Next Round" })).toBeInTheDocument();
  });

  it("renders clickable status chip when onStatusChange is provided", () => {
    renderCard({ onStatusChange: vi.fn() });
    expect(screen.getByTestId("status-chip-trigger")).toBeInTheDocument();
  });

  it("renders read-only status when onStatusChange is not provided", () => {
    renderCard();
    expect(screen.queryByTestId("status-chip-trigger")).not.toBeInTheDocument();
    expect(screen.getByText(/ready/i)).toBeInTheDocument();
  });

  it("renders read-only status when item is locked", () => {
    renderCard({
      item: makeItem({ status: "in_progress" }),
      itemActions: makeActions({ locked: true }),
      onStatusChange: vi.fn(),
    });
    expect(screen.queryByTestId("status-chip-trigger")).not.toBeInTheDocument();
  });

  it("opens popover on status chip click and calls onStatusChange", async () => {
    const user = userEvent.setup();
    const onStatusChange = vi.fn();
    renderCard({ item: makeItem({ status: "ready" }), onStatusChange });

    await user.click(screen.getByTestId("status-chip-trigger"));
    expect(screen.getByTestId("status-chip-popover")).toBeInTheDocument();

    await user.click(screen.getByTestId("status-option-backlog"));
    expect(onStatusChange).toHaveBeenCalledWith("backlog");
  });

  it("shows finalization transition copy for research items", () => {
    renderCard({
      item: makeItem({ kind: "research", status: "researching" }),
      transitionResult: {
        autoAdvance: {
          triggered: true,
          reason: "finalizing",
          nextMode: "finalize",
        },
      },
    });

    expect(screen.getByText("Finalizing conclusion...")).toBeInTheDocument();
  });
});
