import { describe, it, expect, vi, beforeAll, afterEach } from "vitest";
import { render, screen, cleanup, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ProposalReview } from "./proposal-review";
import { selectors } from "../../consts/selectors";
import type { ProposalRevision } from "../../types";

beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    }),
  });
});

// The graph preview uses ReactFlow which needs a ResizeObserver; skip the
// preview path in these unit tests by leaving previewItems undefined.
afterEach(() => cleanup());

function makeRevision(): ProposalRevision {
  return {
    id: "p1",
    message_index: 0,
    created_at: "2026-04-23T00:00:00Z",
    proposal: {
      form: "mutation_list",
      rationale: "clean up",
      mutations: [
        { id: "m1", op: "archive_item", target: "execute/a", rationale: "stale" },
        { id: "m2", op: "change_status", target: "execute/b", status: "ready" },
        { id: "m3", op: "add_edge", from: "execute/b", to: "execute/c", rationale: "order" },
      ],
    },
  };
}

describe("ProposalReview", () => {
  it("renders every mutation and defaults to all-selected", () => {
    render(
      <ProposalReview
        revision={makeRevision()}
        onAccept={vi.fn()}
        onReject={vi.fn()}
        onDismiss={vi.fn()}
      />,
    );
    const cards = screen.getAllByTestId(selectors.feedback.proposalMutation);
    expect(cards).toHaveLength(3);
    for (const card of cards) {
      const toggle = within(card).getByTestId(selectors.feedback.proposalMutationToggle);
      expect(toggle).toBeChecked();
    }
    expect(screen.getByTestId(selectors.feedback.proposalAccept)).toHaveTextContent(/Accept all/i);
  });

  it("tracks per-mutation toggle state and routes partial accepts upward", async () => {
    const onAccept = vi.fn();
    render(
      <ProposalReview
        revision={makeRevision()}
        onAccept={onAccept}
        onReject={vi.fn()}
        onDismiss={vi.fn()}
      />,
    );
    const cards = screen.getAllByTestId(selectors.feedback.proposalMutation);
    // Uncheck the middle mutation (m2).
    const m2Toggle = within(cards[1]!).getByTestId(selectors.feedback.proposalMutationToggle);
    await userEvent.click(m2Toggle);
    expect(m2Toggle).not.toBeChecked();
    expect(screen.getByTestId(selectors.feedback.proposalAccept)).toHaveTextContent(/Accept 2 of 3/);

    await userEvent.click(screen.getByTestId(selectors.feedback.proposalAccept));
    expect(onAccept).toHaveBeenCalledWith(expect.arrayContaining(["m1", "m3"]), expect.any(String));
    expect(onAccept.mock.calls[0]![0]).not.toContain("m2");
  });

  it("routes reject and dismiss with the typed rationale", async () => {
    const onReject = vi.fn();
    const onDismiss = vi.fn();
    render(
      <ProposalReview
        revision={makeRevision()}
        onAccept={vi.fn()}
        onReject={onReject}
        onDismiss={onDismiss}
      />,
    );
    const rationaleInput = screen.getByPlaceholderText(/rationale/i);
    await userEvent.type(rationaleInput, "not what we want");
    await userEvent.click(screen.getByTestId(selectors.feedback.proposalReject));
    expect(onReject).toHaveBeenCalledWith("not what we want");
    await userEvent.click(screen.getByTestId(selectors.feedback.proposalDismiss));
    expect(onDismiss).toHaveBeenCalledWith("not what we want");
  });

  it("renders read-only when readOnly=true (no action buttons)", () => {
    render(
      <ProposalReview
        revision={makeRevision()}
        onAccept={vi.fn()}
        onReject={vi.fn()}
        onDismiss={vi.fn()}
        readOnly
      />,
    );
    expect(screen.queryByTestId(selectors.feedback.proposalAccept)).toBeNull();
    expect(screen.queryByTestId(selectors.feedback.proposalReject)).toBeNull();
  });

  it("renders apply result badges next to each mutation", () => {
    render(
      <ProposalReview
        revision={makeRevision()}
        onAccept={vi.fn()}
        onReject={vi.fn()}
        onDismiss={vi.fn()}
        readOnly
        applyResult={{
          applied: 2,
          failed: 1,
          skipped: 0,
          outcomes: [
            { mutation_id: "m1", op: "archive_item", applied: true },
            { mutation_id: "m2", op: "change_status", applied: true },
            { mutation_id: "m3", op: "add_edge", applied: false, error: "cycle" },
          ],
        }}
      />,
    );
    expect(screen.getAllByText(/Applied/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Failed/i).length).toBeGreaterThan(0);
    expect(screen.getByText(/cycle/)).toBeInTheDocument();
  });

  it("refuses to render accept flow for full_graph proposals", () => {
    const fg: ProposalRevision = {
      id: "p1",
      message_index: 0,
      created_at: "2026-04-23T00:00:00Z",
      proposal: {
        form: "full_graph",
        graph: { nodes: [], edges: [] },
      },
    };
    render(
      <ProposalReview
        revision={fg}
        onAccept={vi.fn()}
        onReject={vi.fn()}
        onDismiss={vi.fn()}
      />,
    );
    expect(screen.getByText(/full-graph proposals/i)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.feedback.proposalAccept)).toBeNull();
  });
});
