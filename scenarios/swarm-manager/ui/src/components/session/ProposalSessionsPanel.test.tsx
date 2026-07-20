import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../../test-utils/render";
import { selectors } from "../../consts/selectors";
import { agentSessionStoreInitialState, useAgentSessionStore } from "../../stores";
import { ProposalSessionsPanel } from "./ProposalSessionsPanel";

vi.mock("../../services/proposal-session-service", () => ({
  proposalSessionService: {
    list: vi.fn().mockResolvedValue([]),
    create: vi.fn(),
    decide: vi.fn(),
    revise: vi.fn(),
  },
}));

describe("ProposalSessionsPanel", () => {
  beforeEach(() => {
    useAgentSessionStore.setState({
      ...agentSessionStoreInitialState,
      sessions: [],
      fetchSessions: vi.fn().mockResolvedValue(undefined),
      createSession: vi.fn(),
      isMutating: false,
    });
  });

  it("opens the proposal-focused attach sheet instead of creating a session immediately", async () => {
    renderWithProviders(
      <ProposalSessionsPanel target={{ type: "initiative", ref: "initiative-alpha", name: "Initiative Alpha" }} />,
    );

    await userEvent.click(screen.getByTestId(selectors.agentSessions.proposalStart));

    expect(screen.getByText("New proposal session")).toBeInTheDocument();
    expect(screen.getByText("Proposal sessions use managed Swarm Operations and always produce a reviewable mutation list.")).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.agentSessions.entityAttachSuggestion)).toHaveLength(6);
  });
});
