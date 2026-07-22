import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
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
    archivePending: false,
    ...overrides,
  };
  return render(<BacklogCard {...props} />);
};

describe("BacklogCard", () => {
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

  describe("pick mode (SessionContextPicker)", () => {
    it("renders the context row with title and suppresses all action rows", () => {
      render(
        <BacklogCard
          item={makeItem({ title: "Pickable Item" })}
          selection={{ selectionMode: true, selected: false, onToggleSelect: vi.fn() }}
        />,
      );
      expect(screen.getByText("Pickable Item")).toBeInTheDocument();
      expect(screen.queryByTestId("backlog-card-actions")).not.toBeInTheDocument();
      expect(screen.queryByTestId("status-chip-trigger")).not.toBeInTheDocument();
      expect(screen.getByTestId("session-context-row")).toHaveAttribute("aria-pressed", "false");
    });

    it("toggles on click and respects the disabled cap state", async () => {
      const onToggleSelect = vi.fn();
      const { rerender } = render(
        <BacklogCard
          item={makeItem()}
          selection={{ selectionMode: true, selected: false, onToggleSelect }}
        />,
      );
      await userEvent.click(screen.getByTestId("session-context-row"));
      expect(onToggleSelect).toHaveBeenCalledTimes(1);

      rerender(
        <BacklogCard
          item={makeItem()}
          selection={{ selectionMode: true, selected: false, disabled: true, disabledReason: "Cap reached", onToggleSelect }}
        />,
      );
      const row = screen.getByTestId("session-context-row");
      expect(row).toBeDisabled();
      expect(row).toHaveAttribute("title", "Cap reached");
      await userEvent.click(row);
      expect(onToggleSelect).toHaveBeenCalledTimes(1);
    });
  });
});
