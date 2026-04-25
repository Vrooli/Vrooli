import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { BacklogActionButtons } from "./backlog-action-buttons";
import { BacklogDetailProvider, type BacklogDetailContextValue } from "../../contexts/BacklogDetailContext";
import type { ItemActions } from "../../lib/backlog-queue-utils";
import type { BacklogItem } from "../../types/domain";

const baseItem = {
  title: "test", description: "", status: "failed" as const, priority: 3, tags: [],
};

const baseActions: ItemActions = {
  locked: false,
  terminal: true,
  blocked: false,
  blockingDepKeys: [],
  primaryCta: "followUp",
  canRun: false,
  runDisabled: false,
  canWorkshop: false,
  workshopDisabled: false,
  canFinalize: false,
  finalizeDisabled: false,
  canFollowUp: true,
  canRetry: true,
  canArchive: true,
  showDecisionStepper: false,
  agentRunning: false,
  notQueueableReason: null,
  disabledReason: null,
};

function renderWithCtx(actions: ItemActions, overrides: Partial<{
  onFollowUp: () => void;
  onRetry: () => void;
}> = {}) {
  const ctxValue: BacklogDetailContextValue = {
    backlogKind: "execute",
    name: "test",
    item: { ...baseItem, kind: "execute", name: "test" } as unknown as BacklogItem,
    itemActions: actions,
    isLocked: false,
    isTerminal: true,
    agentRunIsActive: false,
    latestAgentActivity: null,
    deliverableLabel: "Plan",
    workshopActionLabel: "Workshop",
    agentRunningLabel: "Running",
    agentLabel: "Agent",
    isWorkshopFinalized: false,
    workshopBlockedDeps: [],
    isRunningAgent: false,
  };
  return render(
    <BacklogDetailProvider value={ctxValue}>
      <BacklogActionButtons
        item={baseItem}
        isUpdating={false}
        onFinalizeWorkshop={() => {}}
        onStartRun={() => {}}
        onRunWorkshop={() => {}}
        onEdit={() => {}}
        onFollowUp={overrides.onFollowUp ?? (() => {})}
        onRetry={overrides.onRetry ?? (() => {})}
        onOpenAgentDialog={() => {}}
        onArchive={() => {}}
        onStatusChange={() => {}}
        onResetWorkshop={() => {}}
        hasWorkshopRounds={false}
        onDelete={() => {}}
      />
    </BacklogDetailProvider>,
  );
}

describe("BacklogActionButtons retry", () => {
  it("renders Retry button when canRetry is true", () => {
    renderWithCtx(baseActions);
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });

  it("does not render Retry button when canRetry is false", () => {
    renderWithCtx({ ...baseActions, canRetry: false });
    expect(screen.queryByRole("button", { name: /retry/i })).toBeNull();
  });

  it("renders both Follow Up and Retry side by side when both gates pass", () => {
    renderWithCtx(baseActions);
    expect(screen.getByRole("button", { name: /follow up/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });

  it("invokes onRetry when the Retry button is clicked", () => {
    const onRetry = vi.fn();
    renderWithCtx(baseActions, { onRetry });
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
