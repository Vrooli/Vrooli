import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, cleanup, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { FeedbackPanel } from "./feedback-panel";
import { selectors } from "../../consts/selectors";
import type { FeedbackRound } from "../../types";

beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation(() => ({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  });
  // ResizeObserver used by ReactFlow / auto-resize hooks.
  class ROShim {
    observe() {}
    unobserve() {}
    disconnect() {}
  }
  (globalThis as unknown as { ResizeObserver: typeof ROShim }).ResizeObserver = ROShim;
});

vi.mock("../../services/feedback-service", async () => {
  const actual = await vi.importActual<typeof import("../../services/feedback-service")>(
    "../../services/feedback-service",
  );
  return {
    ...actual,
    feedbackService: {
      list: vi.fn(),
      get: vi.fn(),
      start: vi.fn(),
      continue_: vi.fn(),
      decide: vi.fn(),
      dismiss: vi.fn(),
      lockStatus: vi.fn(),
      attachmentUrl: (name: string, round: number, id: string) =>
        `/initiatives/${name}/feedback/${round}/attachments/${id}`,
    },
  };
});

vi.mock("../../hooks/useIndexedDBAttachments", () => ({
  useIndexedDBAttachments: () => ({
    attachments: [],
    addFile: vi.fn(),
    removeFile: vi.fn(),
    clearAll: vi.fn(),
    getFiles: () => [],
  }),
}));

const { feedbackService } = await import("../../services/feedback-service");
const mockList = vi.mocked(feedbackService.list);
const mockDecide = vi.mocked(feedbackService.decide);
const mockContinue = vi.mocked(feedbackService.continue_);

function round(overrides: Partial<FeedbackRound> = {}): FeedbackRound {
  return {
    initiative_name: "init",
    number: 1,
    slug: "r1",
    type: "feedback",
    status: "awaiting_user",
    submission: { text: "something off", created_at: "2026-04-23T00:00:00Z" },
    thread: [
      { role: "user", content: "something off", created_at: "2026-04-23T00:00:00Z" },
      { role: "agent", content: "sure thing", created_at: "2026-04-23T00:01:00Z", proposal_id: "p1" },
    ],
    proposals: [
      {
        id: "p1",
        message_index: 1,
        created_at: "2026-04-23T00:01:00Z",
        proposal: {
          form: "mutation_list",
          mutations: [
            { id: "m1", op: "change_status", target: "execute/x", status: "ready" },
          ],
        },
      },
    ],
    current_proposal_id: "p1",
    created_at: "2026-04-23T00:00:00Z",
    updated_at: "2026-04-23T00:01:00Z",
    ...overrides,
  };
}

function renderPanel() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false, refetchOnMount: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <FeedbackPanel initiativeName="init" />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  mockList.mockReset();
  mockDecide.mockReset();
  mockContinue.mockReset();
});
afterEach(() => cleanup());

describe("FeedbackPanel", () => {
  it("shows an empty state when there are no rounds", async () => {
    mockList.mockResolvedValue([]);
    renderPanel();
    await waitFor(() => {
      expect(screen.getByTestId(selectors.feedback.panelEmpty)).toBeInTheDocument();
    });
  });

  it("renders a round card and expands to show thread + proposal review", async () => {
    mockList.mockResolvedValue([round()]);
    renderPanel();
    await screen.findByTestId(selectors.feedback.panelRoundCard);
    await userEvent.click(screen.getByTestId(selectors.feedback.panelRoundExpand));
    expect(screen.getByTestId(selectors.feedback.proposalReview)).toBeInTheDocument();
    // Two thread messages (user + agent).
    expect(screen.getAllByTestId(selectors.feedback.threadMessage)).toHaveLength(2);
  });

  it("submits a partial accept through the decide endpoint", async () => {
    mockList.mockResolvedValue([round()]);
    mockDecide.mockResolvedValue({
      round: round({ status: "applied" }),
      apply_result: { outcomes: [], applied: 1, failed: 0, skipped: 0 },
    });
    renderPanel();
    await screen.findByTestId(selectors.feedback.panelRoundCard);
    await userEvent.click(screen.getByTestId(selectors.feedback.panelRoundExpand));

    await userEvent.click(screen.getByTestId(selectors.feedback.proposalAccept));
    await waitFor(() => expect(mockDecide).toHaveBeenCalled());
    expect(mockDecide.mock.calls[0]![2]).toMatchObject({
      kind: "accept",
      acceptedMutationIds: ["m1"],
    });
  });

  it("sends a revise continuation with the user's text", async () => {
    mockList.mockResolvedValue([round()]);
    mockContinue.mockResolvedValue(round({ status: "agent_thinking" }));
    renderPanel();
    await screen.findByTestId(selectors.feedback.panelRoundCard);
    await userEvent.click(screen.getByTestId(selectors.feedback.panelRoundExpand));

    const revise = screen.getByTestId(selectors.feedback.threadReviseInput);
    await userEvent.type(revise, "please drop m1");
    await userEvent.click(screen.getByTestId(selectors.feedback.threadReviseSubmit));

    await waitFor(() => expect(mockContinue).toHaveBeenCalled());
    expect(mockContinue.mock.calls[0]![2]).toMatchObject({ text: "please drop m1" });
  });

  it("renders the decision block for terminal rounds", async () => {
    mockList.mockResolvedValue([
      round({
        status: "applied",
        decision: {
          kind: "accept",
          rationale: "good call",
          decided_at: "2026-04-23T00:05:00Z",
          decided_by: "matt",
        },
      }),
    ]);
    renderPanel();
    const card = await screen.findByTestId(selectors.feedback.panelRoundCard);
    await userEvent.click(within(card).getByTestId(selectors.feedback.panelRoundExpand));
    expect(screen.getByText(/good call/i)).toBeInTheDocument();
  });
});
