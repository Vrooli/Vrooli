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

  it("distinguishes skipped mutations from applied and failed via outcome badge + card tint", () => {
    render(
      <ProposalReview
        revision={makeRevision()}
        onAccept={vi.fn()}
        onReject={vi.fn()}
        onDismiss={vi.fn()}
        readOnly
        applyResult={{
          applied: 1,
          failed: 1,
          skipped: 1,
          outcomes: [
            { mutation_id: "m1", op: "archive_item", applied: true },
            { mutation_id: "m2", op: "change_status", applied: false, skipped: true },
            { mutation_id: "m3", op: "add_edge", applied: false, error: "cycle" },
          ],
        }}
      />,
    );
    // Outcome data attribute pins the classification so future tint/
    // color changes don't silently regress the skipped-vs-failed
    // distinction the user relies on to understand partial applies.
    const cards = screen.getAllByTestId(selectors.feedback.proposalMutation);
    expect(cards[0]).toHaveAttribute("data-outcome", "applied");
    expect(cards[1]).toHaveAttribute("data-outcome", "skipped");
    expect(cards[2]).toHaveAttribute("data-outcome", "failed");
    // Each outcome renders its own badge text.
    expect(within(cards[0]!).getByText(/^Applied$/)).toBeInTheDocument();
    expect(within(cards[1]!).getByText(/^Skipped$/)).toBeInTheDocument();
    expect(within(cards[2]!).getByText(/^Failed$/)).toBeInTheDocument();
    // Skipped card must NOT carry the error-red styling that failures have —
    // guard rail against the old "applied=false means failed" conflation.
    expect(cards[1]!.className).not.toMatch(/border-red-500/);
  });

  it("summary banner surfaces failure with red styling when any mutation failed", () => {
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
    const banner = screen.getByTestId(selectors.feedback.proposalApplySummary);
    // Red border/background must be present when failed>0 so the user
    // does not read a green banner next to "Failed: 1" and assume success.
    expect(banner.className).toMatch(/border-red-500/);
    expect(banner.className).not.toMatch(/border-emerald-500/);
  });

  it("summary banner uses amber when only skipped are present (no failures)", () => {
    render(
      <ProposalReview
        revision={makeRevision()}
        onAccept={vi.fn()}
        onReject={vi.fn()}
        onDismiss={vi.fn()}
        readOnly
        applyResult={{
          applied: 2,
          failed: 0,
          skipped: 1,
          outcomes: [
            { mutation_id: "m1", op: "archive_item", applied: true },
            { mutation_id: "m2", op: "change_status", applied: false, skipped: true },
            { mutation_id: "m3", op: "add_edge", applied: true },
          ],
        }}
      />,
    );
    const banner = screen.getByTestId(selectors.feedback.proposalApplySummary);
    expect(banner.className).toMatch(/border-amber-500/);
    expect(banner.className).not.toMatch(/border-red-500/);
    expect(banner.className).not.toMatch(/border-emerald-500/);
  });

  it("summary banner stays green when everything applied cleanly", () => {
    render(
      <ProposalReview
        revision={makeRevision()}
        onAccept={vi.fn()}
        onReject={vi.fn()}
        onDismiss={vi.fn()}
        readOnly
        applyResult={{
          applied: 3,
          failed: 0,
          skipped: 0,
          outcomes: [
            { mutation_id: "m1", op: "archive_item", applied: true },
            { mutation_id: "m2", op: "change_status", applied: true },
            { mutation_id: "m3", op: "add_edge", applied: true },
          ],
        }}
      />,
    );
    const banner = screen.getByTestId(selectors.feedback.proposalApplySummary);
    expect(banner.className).toMatch(/border-emerald-500/);
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
