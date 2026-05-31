import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { SessionContextPicker } from "./SessionContextPicker";
import { selectors } from "../../../consts/selectors";
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
} from "../../../stores";
import type { InitiativeWithRollup } from "../../../types";
import type { SessionContextOption } from "./session-context-refs";

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

const INITIATIVES = ["a", "b", "c", "d", "e"].map(makeInitiative);

function seedStores() {
  useBacklogStore.setState({ ...backlogStoreInitialState, fetchBacklog: vi.fn() });
  useCaptureStore.setState({ ...captureStoreInitialState, fetchCaptures: vi.fn() });
  useExecutionStore.setState({ ...executionStoreInitialState, fetchExecutions: vi.fn() });
  useAgentActivitiesStore.setState({ ...agentActivitiesStoreInitialState, refreshActivities: vi.fn() });
  useScenariosStore.setState({ ...scenariosStoreInitialState, fetchScenarios: vi.fn() });
  useAgentSessionStore.setState({ ...agentSessionStoreInitialState, fetchSessions: vi.fn() });
  useInitiativeStore.setState({ ...initiativeStoreInitialState, items: INITIATIVES, fetchInitiatives: vi.fn() });
}

function renderPicker(onApply: (items: SessionContextOption[]) => void) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <SessionContextPicker
        isOpen
        onClose={vi.fn()}
        sessionKind="meta_orchestration"
        selected={[]}
        onApply={onApply}
        initialType="initiative"
      />
    </QueryClientProvider>,
  );
}

describe("SessionContextPicker", () => {
  beforeEach(() => {
    seedStores();
  });

  it("disables remaining cards once the per-type cap is reached and emits the selected options", async () => {
    const onApply = vi.fn();
    renderPicker(onApply);

    // The initiative cap is 4. Select the first four initiatives.
    expect(screen.getAllByTestId(selectors.agentSessions.contextRow)).toHaveLength(5);
    for (let i = 0; i < 4; i++) {
      await userEvent.click(screen.getAllByTestId(selectors.agentSessions.contextRow)[i]!);
    }

    // The fifth (unselected) card is now disabled with the cap reason.
    const rows = screen.getAllByTestId(selectors.agentSessions.contextRow);
    expect(rows[4]!).toBeDisabled();
    expect(rows[4]!.getAttribute("title")).toMatch(/Initiatives allows 4 selections/);

    // Already-selected cards remain interactive (can deselect).
    expect(rows[0]!).not.toBeDisabled();

    await userEvent.click(screen.getByTestId(selectors.agentSessions.contextAttachButton));
    expect(onApply).toHaveBeenCalledTimes(1);
    const emitted = onApply.mock.calls[0]![0] as SessionContextOption[];
    expect(emitted).toHaveLength(4);
    expect(emitted.every((option) => option.type === "initiative")).toBe(true);
  });
});
