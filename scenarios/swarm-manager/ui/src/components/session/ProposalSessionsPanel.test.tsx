import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@vrooli/api-base/testing";
import { selectors } from "../../consts/selectors";
import { agentSessionStoreInitialState, useAgentSessionStore } from "../../stores";
import { ProposalSessionsPanel } from "./ProposalSessionsPanel";
import { proposalSessionService } from "../../services/proposal-session-service";

vi.mock("../../services/proposal-session-service", () => ({
  proposalSessionService: {
    list: vi.fn().mockResolvedValue([]),
    create: vi.fn(),
    decide: vi.fn(),
    acceptKeep: vi.fn(),
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
      <ProposalSessionsPanel target={{ type: "goal", ref: "goal-alpha", name: "Goal Alpha" }} />,
    );

    await userEvent.click(screen.getByTestId(selectors.agentSessions.proposalStart));

    expect(screen.getByText("New proposal session")).toBeInTheDocument();
    expect(screen.getByText("Proposal sessions use managed Swarm Operations and always produce a reviewable mutation list.")).toBeInTheDocument();
    expect(screen.getAllByTestId(selectors.agentSessions.entityAttachSuggestion)).toHaveLength(6);
  });

  it("uses direct proposal decisions by default and reveals batch checkboxes only in selection mode", async () => {
    vi.mocked(proposalSessionService.list).mockResolvedValue([
      {
        id: "sess-1",
        title: "Schema review",
        status: "proposal_ready",
        skill_id: "skill",
        proposals: [{
          id: "prop-1",
          kind: "mutation_list",
          status: "ready",
          summary: "Update the schema",
          payload_json: JSON.stringify({ mutations: [{ id: "m1", op: "reset_artifacts", target: "fix/schema", rationale: "Refresh derived work." }] }),
          created_at: "2026-07-20T00:00:00Z",
          updated_at: "2026-07-20T00:00:00Z",
        }],
      },
      {
        id: "sess-2",
        title: "Second review",
        status: "proposal_ready",
        skill_id: "skill",
        proposals: [{ id: "prop-2", kind: "mutation_list", status: "ready", summary: "Second change", payload_json: "{}", created_at: "2026-07-20T00:00:00Z", updated_at: "2026-07-20T00:00:00Z" }],
      },
    ]);

    renderWithProviders(<ProposalSessionsPanel target={{ type: "goal", ref: "goal-alpha", name: "Goal Alpha" }} />);

    expect(await screen.findByText("Review change set (1)")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Apply proposal" })).toHaveLength(2);
    expect(screen.queryByLabelText("Select Update the schema")).toBeNull();

    await userEvent.click(screen.getByRole("button", { name: "Select proposals" }));

    expect(screen.getByLabelText("Select Update the schema")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Apply proposal" })).toBeNull();
  });

  it("renders no-change recommendations with their reasoning and no apply control", async () => {
    vi.mocked(proposalSessionService.list).mockResolvedValue([{
      id: "session-no-change",
      title: "Review stale item",
      status: "proposal_ready",
      skill_id: "swarm-manager-proposals",
      proposal_target: { type: "backlog_item", ref: "research/stale-item", name: "Stale item" },
      proposals: [{
        id: "proposal-no-change",
        // Legacy sessions recorded this as mutation_list; the UI must still
        // present the conclusion rather than an empty change set.
        kind: "mutation_list",
        status: "ready",
        summary: "Mutation proposal for stale item",
        payload_json: JSON.stringify({ form: "mutation_list", rationale: "The evidence supports keeping the item unchanged.", mutations: [] }),
        created_at: "2026-07-20T00:00:00Z",
        updated_at: "2026-07-20T00:00:00Z",
      }],
    }]);

    renderWithProviders(<ProposalSessionsPanel target={{ type: "goal", ref: "goal-alpha", name: "Goal Alpha" }} />);

    expect(await screen.findByText("No changes recommended")).toBeInTheDocument();
    expect(screen.getByText("The evidence supports keeping the item unchanged.")).toBeInTheDocument();
    expect(screen.getByText("No changes will be applied. Accept this conclusion to record a freshness review, or request a revision if you want the session to reassess it.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Apply proposal" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Accept keep recommendation" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Request revision" })).toBeInTheDocument();
  });
});
