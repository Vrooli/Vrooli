import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from "vitest";
import { screen, waitFor, cleanup, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import * as React from "react";
import { FeedbackPanel } from "./feedback-panel";
import { selectors } from "../../consts/selectors";
import type { FeedbackRound, LockStatusResponse } from "../../types";
import {
  createTestQueryClient,
  installMatchMediaMock,
  installResizeObserverMock,
  renderWithProviders,
} from "../../test-utils";

beforeAll(() => {
  installMatchMediaMock();
  installResizeObserverMock();
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
      lockStatus: vi.fn(async () => ({ locked: false }) as LockStatusResponse),
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
const mockLockStatus = vi.mocked(feedbackService.lockStatus);

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

function renderPanel(props: Partial<React.ComponentProps<typeof FeedbackPanel>> = {}) {
  return renderWithProviders(
    <FeedbackPanel initiativeName="init" {...props} />,
    {
      queryClient: createTestQueryClient({
        defaultOptions: { queries: { refetchOnMount: false } },
      }),
      withRouter: false,
    },
  );
}

beforeEach(() => {
  mockList.mockReset();
  mockDecide.mockReset();
  mockContinue.mockReset();
  mockLockStatus.mockReset();
  mockLockStatus.mockResolvedValue({ locked: false });
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

  it("shows the delete action for awaiting_user rounds", async () => {
    mockList.mockResolvedValue([round()]);
    renderPanel();
    await screen.findByTestId(selectors.feedback.panelRoundCard);
    await userEvent.click(screen.getByTestId(selectors.feedback.panelRoundExpand));
    expect(screen.getByTestId(selectors.feedback.deleteButton)).toBeInTheDocument();
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

  it("routes dismiss through the decide endpoint with kind=dismiss", async () => {
    mockList.mockResolvedValue([round()]);
    mockDecide.mockResolvedValue({
      round: round({ status: "dismissed", decision: { kind: "dismiss", decided_at: "2026-04-23T00:05:00Z" } }),
      apply_result: { outcomes: [], applied: 0, failed: 0, skipped: 0 },
    });
    renderPanel();
    await screen.findByTestId(selectors.feedback.panelRoundCard);
    await userEvent.click(screen.getByTestId(selectors.feedback.panelRoundExpand));

    await userEvent.click(screen.getByTestId(selectors.feedback.proposalDismiss));

    await waitFor(() => expect(mockDecide).toHaveBeenCalled());
    expect(mockDecide.mock.calls[0]![2]).toMatchObject({ kind: "dismiss" });
    // Dismiss must never propose mutation IDs — nothing to apply.
    expect(mockDecide.mock.calls[0]![2].acceptedMutationIds ?? []).toEqual([]);
  });

  it("renders the parse-error notice when the round needs a revision and has no proposal", async () => {
    mockList.mockResolvedValue([
      round({
        proposals: [],
        current_proposal_id: undefined,
        needs_revision: true,
        last_parse_warnings: [
          "agent output did not contain a parseable proposal JSON block",
          "stray markdown fence before the JSON",
        ],
      }),
    ]);
    renderPanel();
    await screen.findByTestId(selectors.feedback.panelRoundCard);
    await userEvent.click(screen.getByTestId(selectors.feedback.panelRoundExpand));

    // The dedicated notice is visible, guiding the user to ask for a revision
    // rather than leaving them looking at a silent empty proposal pane.
    const notice = screen.getByTestId(selectors.feedback.parseErrorNotice);
    expect(notice).toBeInTheDocument();
    expect(notice).toHaveTextContent(/readable proposal/i);
    // Warnings render as list items so the user can see *why* it failed.
    expect(within(notice).getByText(/parseable proposal JSON block/)).toBeInTheDocument();
    expect(within(notice).getByText(/stray markdown fence/)).toBeInTheDocument();
    // The revise form still renders so the user can send the follow-up.
    expect(screen.getByTestId(selectors.feedback.threadReviseInput)).toBeInTheDocument();
    // The proposal-review panel must NOT render — there is no proposal to review.
    expect(screen.queryByTestId(selectors.feedback.proposalReview)).toBeNull();
  });

  it("does not render the parse-error notice when a proposal exists", async () => {
    mockList.mockResolvedValue([
      round({
        // A proposal is present, but needs_revision flag is spuriously set.
        // The notice must still stay hidden — the proposal panel is the
        // primary surface when one exists.
        needs_revision: true,
        last_parse_warnings: ["a stale warning"],
      }),
    ]);
    renderPanel();
    await screen.findByTestId(selectors.feedback.panelRoundCard);
    await userEvent.click(screen.getByTestId(selectors.feedback.panelRoundExpand));

    expect(screen.queryByTestId(selectors.feedback.parseErrorNotice)).toBeNull();
    expect(screen.getByTestId(selectors.feedback.proposalReview)).toBeInTheDocument();
  });

  it("renders the invalid-proposal notice and revise form when the latest proposal failed validation", async () => {
    mockList.mockResolvedValue([
      round({
        proposals: [
          {
            id: "p1",
            message_index: 1,
            created_at: "2026-04-23T00:01:00Z",
            validation_errors: ["mutations[0]: op update_item requires patch"],
            raw_proposal_text:
              "{\"form\":\"mutation_list\",\"mutations\":[{\"id\":\"m1\",\"op\":\"update_item\",\"target\":\"execute/x\",\"title\":\"bad\"}]}",
            proposal: {
              form: "mutation_list",
              mutations: [
                { id: "m1", op: "update_item", target: "execute/x" },
              ],
            },
          },
        ],
        current_proposal_id: undefined,
        needs_revision: true,
        last_validation_errors: ["mutations[0]: op update_item requires patch"],
      }),
    ]);
    renderPanel();
    await screen.findByTestId(selectors.feedback.panelRoundCard);
    await userEvent.click(screen.getByTestId(selectors.feedback.panelRoundExpand));

    const notice = screen.getByTestId(selectors.feedback.invalidProposalNotice);
    expect(notice).toBeInTheDocument();
    expect(notice).toHaveTextContent(/invalid and must be revised/i);
    expect(within(notice).getByText(/requires patch/i)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.feedback.threadReviseInput)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.feedback.proposalReview)).toBeNull();
    expect(screen.queryByTestId(selectors.feedback.parseErrorNotice)).toBeNull();
  });

  it("reopens the revise form after reject so the user can try a different angle", async () => {
    // Round starts awaiting_user with a proposal. User rejects; the server
    // transitions the round to `rejected`. We then refetch and the user
    // submits a fresh feedback round — but this test focuses on the
    // immediate post-decision surface: the ProposalReview should become
    // read-only, no more Accept/Reject/Dismiss buttons.
    const initial = round();
    const terminal = round({
      status: "rejected",
      decision: {
        kind: "reject",
        rationale: "wrong direction",
        decided_at: "2026-04-23T00:05:00Z",
      },
    });

    mockList.mockResolvedValueOnce([initial]).mockResolvedValueOnce([terminal]);
    mockDecide.mockResolvedValue({
      round: terminal,
      apply_result: { outcomes: [], applied: 0, failed: 0, skipped: 0 },
    });

    renderPanel();
    await screen.findByTestId(selectors.feedback.panelRoundCard);
    await userEvent.click(screen.getByTestId(selectors.feedback.panelRoundExpand));

    // Type a rationale, reject.
    const rationale = screen.getByPlaceholderText(/rationale/i);
    await userEvent.type(rationale, "wrong direction");
    await userEvent.click(screen.getByTestId(selectors.feedback.proposalReject));

    await waitFor(() => expect(mockDecide).toHaveBeenCalled());
    expect(mockDecide.mock.calls[0]![2]).toMatchObject({
      kind: "reject",
      rationale: "wrong direction",
    });
    expect(mockDecide.mock.calls[0]![2].acceptedMutationIds ?? []).toEqual([]);
  });

  it("routes a partial accept (some mutations only) through the decide endpoint", async () => {
    // Multi-mutation round so the user can deselect one.
    const multi = round({
      proposals: [
        {
          id: "p1",
          message_index: 1,
          created_at: "2026-04-23T00:01:00Z",
          proposal: {
            form: "mutation_list",
            mutations: [
              { id: "m1", op: "change_status", target: "execute/x", status: "ready" },
              { id: "m2", op: "archive_item", target: "execute/y" },
            ],
          },
        },
      ],
    });
    mockList.mockResolvedValue([multi]);
    mockDecide.mockResolvedValue({
      round: round({ status: "applied" }),
      apply_result: {
        outcomes: [
          { mutation_id: "m1", op: "change_status", applied: true },
          { mutation_id: "m2", op: "archive_item", applied: false, skipped: true },
        ],
        applied: 1,
        failed: 0,
        skipped: 1,
      },
    });

    renderPanel();
    await screen.findByTestId(selectors.feedback.panelRoundCard);
    await userEvent.click(screen.getByTestId(selectors.feedback.panelRoundExpand));

    // Deselect m2 — only m1 should go into the accept payload.
    const cards = screen.getAllByTestId(selectors.feedback.proposalMutation);
    const m2Card = cards.find((c) => c.getAttribute("data-mutation-id") === "m2")!;
    const m2Check = within(m2Card).getByTestId(selectors.feedback.proposalMutationToggle);
    await userEvent.click(m2Check);

    await userEvent.click(screen.getByTestId(selectors.feedback.proposalAccept));
    await waitFor(() => expect(mockDecide).toHaveBeenCalled());
    // kind is partial_accept because not every mutation is included.
    expect(mockDecide.mock.calls[0]![2]).toMatchObject({
      kind: "partial_accept",
      acceptedMutationIds: ["m1"],
    });
    expect(mockDecide.mock.calls[0]![2].acceptedMutationIds).not.toContain("m2");
  });

  it("forwards previewItems to the feedback dialog so the target picker renders", async () => {
    // Regression: FeedbackPanel must wire previewItems through to FeedbackDialog
    // as `items`. Otherwise the picker + quick-action buttons stay hidden and
    // operators only see the help block.
    mockList.mockResolvedValue([]);
    renderPanel({
      previewItems: [
        { kind: "execute", name: "alpha", title: "Alpha", status: "backlog", dependsOn: [] },
        { kind: "execute", name: "beta", title: "Beta", status: "backlog", dependsOn: [] },
      ],
    });
    await userEvent.click(await screen.findByRole("button", { name: /add feedback/i }));
    expect(await screen.findByTestId(selectors.feedback.dialogTargetPicker)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.feedback.dialogQuickActionSplit)).toBeInTheDocument();
  });

  it("filters archived and missing items before forwarding to the dialog", async () => {
    mockList.mockResolvedValue([]);
    renderPanel({
      previewItems: [
        { kind: "execute", name: "alpha", title: "Alpha", status: "backlog", dependsOn: [] },
        { kind: "execute", name: "archived", title: "Archived", status: "backlog", dependsOn: [], archivedAt: "2026-04-01T00:00:00Z" },
        { kind: "execute", name: "missing", title: "Missing", status: "backlog", dependsOn: [], missing: true },
      ],
    });
    await userEvent.click(await screen.findByRole("button", { name: /add feedback/i }));
    // Open the picker so the item rows render.
    await userEvent.click(await screen.findByTestId(selectors.feedback.dialogTargetPickerToggle));
    const items = await screen.findAllByTestId(selectors.feedback.dialogTargetPickerItem);
    expect(items).toHaveLength(1);
    expect(items[0]!.textContent).toContain("Alpha");
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
