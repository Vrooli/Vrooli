import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils/render";
import { SessionStarterSuggestions } from "./SessionConversation";
import { SessionContextPicker } from "./context/SessionContextPicker";
import { selectors } from "../../consts/selectors";
import {
  agentActivitiesStoreInitialState,
  agentSessionStoreInitialState,
  backlogStoreInitialState,
  captureStoreInitialState,
  executionStoreInitialState,
  initiativeStoreInitialState,
  scenariosStoreInitialState,
  useAgentActivitiesStore,
  useAgentSessionStore,
  useBacklogStore,
  useCaptureStore,
  useExecutionStore,
  useInitiativeStore,
  useScenariosStore,
} from "../../stores";
import type { BacklogItem, ExecutionRecord, InitiativeWithRollup } from "../../types";

function makeInitiative(name: string): InitiativeWithRollup {
  return {
    initiative: {
      name,
      title: `Initiative ${name}`,
      description: "desc",
      status: "active",
      priority: 0,
      dependsOn: [],
      items: [],
      mode: "item-level",
      acceptanceCriteria: [],
      created: "2026-03-27T00:00:00Z",
      updated: "2026-03-28T00:00:00Z",
    },
    rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0 },
  };
}

function makeBacklog(name: string): BacklogItem {
  return { kind: "fix", name, title: `Fix ${name}`, status: "backlog" } as unknown as BacklogItem;
}

function makeExecution(status: string): ExecutionRecord {
  return {
    executionId: `exec-${status}-${Math.round(Math.random() * 1e6)}`,
    backlogKind: "fix",
    backlogName: "demo",
    status,
    createdAt: "2026-05-31T11:00:00Z",
    updatedAt: "2026-05-31T11:00:00Z",
  } as unknown as ExecutionRecord;
}

/** 3 actionable (failed/canceled), 2 not — age-independent so no time flakiness. */
const EXECUTIONS = [
  makeExecution("failed"),
  makeExecution("failed"),
  makeExecution("canceled"),
  makeExecution("completed"),
  makeExecution("completed"),
];
const ACTIONABLE_EXECUTIONS = 3;

function seedStores({
  executions = EXECUTIONS,
  executionStatus = "success" as const,
  backlog = [makeBacklog("a"), makeBacklog("b"), makeBacklog("c"), makeBacklog("d")],
  initiatives = [makeInitiative("x"), makeInitiative("y")],
}: {
  executions?: ExecutionRecord[];
  executionStatus?: "idle" | "loading" | "success" | "error";
  backlog?: BacklogItem[];
  initiatives?: InitiativeWithRollup[];
} = {}) {
  useBacklogStore.setState({ ...backlogStoreInitialState, items: backlog, status: "success", fetchBacklog: vi.fn() });
  useExecutionStore.setState({ ...executionStoreInitialState, items: executions, status: executionStatus, fetchExecutions: vi.fn() });
  useInitiativeStore.setState({ ...initiativeStoreInitialState, items: initiatives, status: "success", fetchInitiatives: vi.fn() });
  // Picker also fetches these on open — stub so the alignment test makes no real calls.
  useCaptureStore.setState({ ...captureStoreInitialState, fetchCaptures: vi.fn() });
  useAgentActivitiesStore.setState({ ...agentActivitiesStoreInitialState, refreshActivities: vi.fn() });
  useScenariosStore.setState({ ...scenariosStoreInitialState, fetchScenarios: vi.fn() });
  useAgentSessionStore.setState({ ...agentSessionStoreInitialState, fetchSessions: vi.fn() });
}

function renderSuggestions(overrides: Partial<Parameters<typeof SessionStarterSuggestions>[0]> = {}) {
  const props = {
    sessionKind: "swarm_operations" as const,
    pendingContext: [],
    pendingAttachmentCount: 0,
    onChooseText: vi.fn(),
    onRequestContext: vi.fn(),
    onRequestImage: vi.fn(),
    ...overrides,
  };
  renderWithProviders(<SessionStarterSuggestions {...props} />);
  return props;
}

/** Find the starter card whose text contains `needle`. */
function card(needle: string): HTMLElement {
  const match = screen
    .getAllByTestId(selectors.agentSessions.starterSuggestion)
    .find((el) => el.textContent?.includes(needle));
  if (!match) throw new Error(`No starter card matching "${needle}"`);
  return match;
}

describe("SessionStarterSuggestions count badges", () => {
  beforeEach(() => {
    seedStores();
  });

  it("shows the full backing-set count for an unfiltered required card", () => {
    renderSuggestions();
    const chip = within(card("drain workshop decisions")).getByTestId(selectors.agentSessions.starterSuggestionCount);
    expect(chip).toHaveAttribute("data-count", "4");
    expect(chip).toHaveTextContent("4 backlog items");
  });

  it("narrows the count to failed/stale runs for the operations-run card", () => {
    renderSuggestions();
    const chip = within(card("failed or stale run")).getByTestId(selectors.agentSessions.starterSuggestionCount);
    expect(chip).toHaveAttribute("data-count", String(ACTIONABLE_EXECUTIONS));
  });

  it("disables a required card whose resolved count is zero and makes the click a no-op", async () => {
    seedStores({ executions: [], executionStatus: "success" });
    const props = renderSuggestions();
    const runCard = card("failed or stale run");
    expect(within(runCard).getByTestId(selectors.agentSessions.starterSuggestionCount)).toHaveAttribute("data-count", "0");
    expect(runCard).toBeDisabled();

    await userEvent.click(runCard);
    expect(props.onRequestContext).not.toHaveBeenCalled();
  });

  it("shows a skeleton (no number) and stays enabled while the count is loading", () => {
    seedStores({ executions: [], executionStatus: "loading" });
    renderSuggestions();
    const runCard = card("failed or stale run");
    const chip = within(runCard).getByTestId(selectors.agentSessions.starterSuggestionCount);
    expect(chip).toHaveAttribute("data-loading", "true");
    expect(chip).not.toHaveAttribute("data-count");
    expect(runCard).not.toBeDisabled();
  });

  it("opens the picker with the card's filter key on a non-zero required card", async () => {
    const props = renderSuggestions();
    await userEvent.click(card("failed or stale run"));
    expect(props.onRequestContext).toHaveBeenCalledWith("execution", "execution_failed_or_stale");
  });
});

describe("badge ↔ picker alignment (anti-dead-end guarantee)", () => {
  beforeEach(() => {
    seedStores();
  });

  it("the operations-run badge count equals the picker's filtered execution list", () => {
    // Badge side.
    renderSuggestions();
    const badge = within(card("failed or stale run")).getByTestId(selectors.agentSessions.starterSuggestionCount);
    const badgeCount = Number(badge.getAttribute("data-count"));

    // Picker side, opened from that card (same filter key).
    renderWithProviders(
      <SessionContextPicker
        isOpen
        onClose={vi.fn()}
        sessionKind="swarm_operations"
        selected={[]}
        onApply={vi.fn()}
        initialType="execution"
        initialFilterKey="execution_failed_or_stale"
      />,
    );
    // In selection mode the execution tab renders one contextRow per pickable
    // run; with the execution tab active these are exactly the visible runs.
    const pickerCards = screen.getAllByTestId(selectors.agentSessions.contextRow);

    expect(badgeCount).toBe(ACTIONABLE_EXECUTIONS);
    expect(pickerCards).toHaveLength(badgeCount);
  });
});
