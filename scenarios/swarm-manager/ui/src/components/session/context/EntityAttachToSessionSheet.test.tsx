/**
 * Tests for EntityAttachToSessionSheet — the attach-context drawer.
 *
 * Pins the two-mode design (new session vs. add to existing), the mode-aware
 * footer action, and the optional starter-prompt prefill: drafting with a
 * suggestion selected stages the context AND seeds the new session's composer
 * draft in localStorage.
 */
import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../../test-utils/render";
import { EntityAttachToSessionSheet } from "./EntityAttachToSessionSheet";
import { selectors } from "../../../consts/selectors";
import { agentSessionStoreInitialState, useAgentSessionStore } from "../../../stores";
import { readSessionDraft } from "../session-draft-storage";
import { peekStagedContextForSession } from "./pending-session-context";
import { proposalSessionService } from "../../../services/proposal-session-service";
import type { AgentSession } from "../../../types";
import type { SessionContextOption } from "./session-context-refs";

const BACKLOG_OPTION: SessionContextOption = {
  type: "backlog_item",
  ref: "fix/flaky-stats",
  title: "Fix flaky stats test",
  nodeId: "backlog-item/fix/flaky-stats",
};

const INITIATIVE_OPTION: SessionContextOption = {
  type: "initiative",
  ref: "initiative-alpha",
  title: "Initiative Alpha",
  nodeId: "initiative/initiative-alpha",
};

const CREATED_SESSION = { id: "sess-new", kind: "meta_orchestration", status: "draft" } as AgentSession;

function seedStore(overrides: Partial<ReturnType<typeof useAgentSessionStore.getState>> = {}) {
  useAgentSessionStore.setState({
    ...agentSessionStoreInitialState,
    sessions: [],
    fetchSessions: vi.fn().mockResolvedValue(undefined),
    createSession: vi.fn().mockResolvedValue(CREATED_SESSION),
    isMutating: false,
    ...overrides,
  });
}

function renderSheet(options: { option?: SessionContextOption; proposalMode?: boolean } = {}) {
  return renderWithProviders(
    <EntityAttachToSessionSheet
      isOpen
      onClose={vi.fn()}
      option={options.option ?? BACKLOG_OPTION}
      proposalMode={options.proposalMode}
    />,
  );
}

describe("EntityAttachToSessionSheet", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
    seedStore();
  });

  it("defaults to new-session mode: kind cards + Draft session footer, no search or Attach", () => {
    renderSheet();

    expect(screen.getByTestId(selectors.agentSessions.entityAttachModeNew)).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId(selectors.agentSessions.entityAttachKindSelect)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.agentSessions.entityAttachQuickStart)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.agentSessions.entityAttachSearch)).toBeNull();
    expect(screen.queryByTestId(selectors.agentSessions.entityAttachConfirm)).toBeNull();
  });

  it("switching to existing mode swaps the body and footer action", async () => {
    renderSheet();

    await userEvent.click(screen.getByTestId(selectors.agentSessions.entityAttachModeExisting));

    expect(screen.getByTestId(selectors.agentSessions.entityAttachSearch)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.agentSessions.entityAttachSessionList)).toHaveTextContent("No matching sessions.");
    expect(screen.getByTestId(selectors.agentSessions.entityAttachConfirm)).toBeDisabled();
    expect(screen.queryByTestId(selectors.agentSessions.entityAttachQuickStart)).toBeNull();
    expect(screen.queryByTestId(selectors.agentSessions.entityAttachKindSelect)).toBeNull();
  });

  it("offers entity-specific starter prompts, toggling selection on click", async () => {
    renderSheet();

    const cards = screen.getAllByTestId(selectors.agentSessions.entityAttachSuggestion);
    const specific = cards.find((card) => card.textContent?.includes('Plan follow-up work for "Fix flaky stats test".'));
    expect(specific).toBeDefined();

    await userEvent.click(specific!);
    expect(specific).toHaveAttribute("aria-pressed", "true");
    await userEvent.click(specific!);
    expect(specific).toHaveAttribute("aria-pressed", "false");
  });

  it("drafting with a suggestion selected stages the context and seeds the composer draft", async () => {
    renderSheet();

    const cards = screen.getAllByTestId(selectors.agentSessions.entityAttachSuggestion);
    const specific = cards.find((card) => card.textContent?.includes("Plan follow-up work"));
    await userEvent.click(specific!);
    await userEvent.click(screen.getByTestId(selectors.agentSessions.entityAttachQuickStart));

    await waitFor(() => {
      expect(peekStagedContextForSession("sess-new")).toEqual([BACKLOG_OPTION]);
    });
    expect(readSessionDraft("sess-new")).toBe('Plan follow-up work for "Fix flaky stats test".');
  });

  it("drafting without a suggestion leaves the composer draft empty", async () => {
    renderSheet();

    await userEvent.click(screen.getByTestId(selectors.agentSessions.entityAttachQuickStart));

    await waitFor(() => {
      expect(peekStagedContextForSession("sess-new")).toEqual([BACKLOG_OPTION]);
    });
    expect(readSessionDraft("sess-new")).toBe("");
  });

  it("uses the proposal flow's five mutation lenses instead of generic session suggestions", async () => {
    renderSheet({ option: INITIATIVE_OPTION, proposalMode: true });

    expect(screen.getByRole("heading", { name: "Start proposal" })).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.agentSessions.entityAttachKindSelect)).toBeNull();
    expect(screen.getAllByTestId(selectors.agentSessions.entityAttachSuggestion)).toHaveLength(5);
    expect(screen.getByText('Split oversized items in "Initiative Alpha".')).toBeInTheDocument();
    expect(screen.getByText('Merge tightly coupled items in "Initiative Alpha".')).toBeInTheDocument();
    expect(screen.getByText('Identify missing work for "Initiative Alpha".')).toBeInTheDocument();
    expect(screen.getByText('Reconcile this initiative with code drift: "Initiative Alpha".')).toBeInTheDocument();
    expect(screen.getByText('Reframe the scope and outcomes for "Initiative Alpha".')).toBeInTheDocument();
  });

  it("creates a target-bound proposal session when started from proposal mode", async () => {
    const create = vi.spyOn(proposalSessionService, "create").mockResolvedValue(CREATED_SESSION as never);
    renderSheet({ option: INITIATIVE_OPTION, proposalMode: true });

    await userEvent.click(screen.getByText('Identify missing work for "Initiative Alpha".'));
    await userEvent.click(screen.getByTestId(selectors.agentSessions.entityAttachQuickStart));

    await waitFor(() => {
      expect(create).toHaveBeenCalledWith({
        title: "Proposal for Initiative Alpha",
        target: { type: "initiative", ref: "initiative-alpha", name: "Initiative Alpha" },
      });
    });
    expect(readSessionDraft("sess-new")).toBe('Identify missing work for "Initiative Alpha".');
  });
});
