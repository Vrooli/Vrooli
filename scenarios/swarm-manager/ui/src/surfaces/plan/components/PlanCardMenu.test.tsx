import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { selectors } from "../../../consts/selectors";
import { goalsService } from "../../../services";
import { createTestQueryClient, renderWithProviders } from "../../../test-utils";
import type { GoalWithScope } from "../../../types/goal";
import { GOALS_QUERY_KEY } from "../hooks/useGoals";
import type { StableItemCallbacks } from "../../../hooks/useCommandPostItemActions";
import type { PlanCardData } from "../types";
import { PlanCardActionsContext, type PlanCardActions } from "./plan-card-actions-context";
import { PlanCardMenu } from "./PlanCardMenu";

function itemCard(overrides?: Partial<PlanCardData>): PlanCardData {
  return {
    id: "backlog-item/fix/thing",
    cardType: "item",
    action: "run",
    itemKind: "fix",
    itemName: "thing",
    title: "Thing",
    status: "ready",
    priority: 3,
    wave: 0,
    initiative: "",
    effort: "",
    gate: null,
    outcome: "",
    finishedAt: "",
    executionId: "",
    unblocks: 0,
    ...overrides,
  };
}

function makeContext(callbacks?: Partial<StableItemCallbacks>): {
  value: PlanCardActions;
  spies: {
    openDecisions: ReturnType<typeof vi.fn>;
    snoozeCard: ReturnType<typeof vi.fn>;
    callbacks: StableItemCallbacks;
  };
} {
  const cb: StableItemCallbacks = {
    onRun: vi.fn(),
    onArchive: vi.fn(),
    onFollowUp: vi.fn(),
    onStatusChange: vi.fn(),
    ...callbacks,
  };
  const openDecisions = vi.fn();
  const snoozeCard = vi.fn();
  return {
    value: {
      getCallbacks: () => cb,
      getBacklogItem: () => undefined,
      runTargets: vi.fn(),
      snoozeCard,
      openDecisions,
    },
    spies: { openDecisions, snoozeCard, callbacks: cb },
  };
}

function goal(name: string, targets: string[] = []): GoalWithScope {
  return {
    goal: {
      name,
      title: name,
      description: "",
      status: "active",
      priority: 1,
      targets,
      seeded: false,
      scopeHistory: [],
      created: "",
      updated: "",
    },
    scope: {
      targets,
      closure: targets,
      completed: [],
      ready: [],
      blocked: [],
      total: targets.length,
      completedCount: 0,
      blockedCount: 0,
      progressPct: 0,
    },
    eta: null,
  };
}

function renderMenu(card: PlanCardData, ctx: PlanCardActions, goals: GoalWithScope[] = []) {
  const queryClient = createTestQueryClient();
  queryClient.setQueryData(GOALS_QUERY_KEY, goals);
  return renderWithProviders(
    <PlanCardActionsContext.Provider value={ctx}>
      <PlanCardMenu card={card} />
    </PlanCardActionsContext.Provider>, { queryClient },
  );
}

describe("PlanCardMenu", () => {
  it("run card exposes Run and forwards to the shared callbacks", async () => {
    const { value, spies } = makeContext();
    const user = userEvent.setup();
    renderMenu(itemCard(), value);

    await user.click(screen.getByTestId("plan-card-menu-backlog-item/fix/thing"));
    await user.click(screen.getByTestId(selectors.plan.cardMenuRun));
    expect(spies.callbacks.onRun).toHaveBeenCalled();
  });

  it("decide gate exposes Answer and opens the scoped drawer", async () => {
    const { value, spies } = makeContext();
    const user = userEvent.setup();
    renderMenu(
      itemCard({
        cardType: "gate",
        action: "decide",
        gate: {
          id: "decide:backlog/fix/thing",
          kind: "decide",
          ownerType: "backlog",
          ownerKind: "fix",
          ownerName: "thing",
          ownerTitle: "Thing",
          count: 3,
          blocks: [],
          decidableSince: "",
          suggested: "",
        },
      }),
      value,
    );

    await user.click(screen.getByTestId("plan-card-menu-backlog-item/fix/thing"));
    await user.click(screen.getByTestId(selectors.plan.cardMenuAnswer));
    expect(spies.openDecisions).toHaveBeenCalledWith("fix/thing");
  });

  it("snooze preset invokes snoozeCard", async () => {
    const { value, spies } = makeContext();
    const user = userEvent.setup();
    renderMenu(itemCard(), value);

    await user.click(screen.getByTestId("plan-card-menu-backlog-item/fix/thing"));
    await user.click(screen.getByTestId("plan-card-menu-snooze-1-hour"));
    expect(spies.snoozeCard).toHaveBeenCalled();
  });

  it("status transitions map to onStatusChange", async () => {
    const { value, spies } = makeContext();
    const user = userEvent.setup();
    renderMenu(itemCard({ status: "backlog", action: "workshop" }), value);

    await user.click(screen.getByTestId("plan-card-menu-backlog-item/fix/thing"));
    await user.click(screen.getByTestId("plan-card-menu-status-ready"));
    expect(spies.callbacks.onStatusChange).toHaveBeenCalledWith("ready");
  });

  it("renders nothing without the context", () => {
    renderWithProviders(<PlanCardMenu card={itemCard()} />);
    expect(screen.queryByTestId("plan-card-menu-backlog-item/fix/thing")).toBeNull();
  });

  it("keeps the shared Set as goal action available for item cards", async () => {
    const { value } = makeContext();
    const create = vi.spyOn(goalsService, "create").mockResolvedValue(goal("thing-goal", ["fix/thing"]));
    const user = userEvent.setup();
    renderMenu(itemCard(), value);

    await user.click(screen.getByTestId("plan-card-menu-backlog-item/fix/thing"));
    await user.click(screen.getByTestId("plan-card-menu-set-goal"));
    await user.click(screen.getByTestId("set-as-goal-create"));

    expect(create).toHaveBeenCalledWith({ title: "Thing", targets: ["fix/thing"] });
  });
});
