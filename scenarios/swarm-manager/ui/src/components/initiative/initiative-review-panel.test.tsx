import { describe, it, expect, vi, beforeAll, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, cleanup } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { InitiativeReviewPanel } from "./initiative-review-panel";
import { selectors } from "../../consts/selectors";
import type {
  InitiativeReviewDecision,
  InitiativeReviewRound,
  InitiativeStatus,
} from "../../types";

beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation(() => ({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  });
});

vi.mock("../../services/initiative-review-service", async () => {
  const actual = await vi.importActual<typeof import("../../services/initiative-review-service")>(
    "../../services/initiative-review-service",
  );
  return {
    ...actual,
    initiativeReviewService: {
      listRounds: vi.fn(),
      getRound: vi.fn(),
      trigger: vi.fn(),
      decide: vi.fn(),
      listDecisions: vi.fn(),
    },
  };
});

const { initiativeReviewService } = await import("../../services/initiative-review-service");
const mockListRounds = vi.mocked(initiativeReviewService.listRounds);
const mockListDecisions = vi.mocked(initiativeReviewService.listDecisions);
const mockTrigger = vi.mocked(initiativeReviewService.trigger);
const mockDecide = vi.mocked(initiativeReviewService.decide);

function renderPanel(status: InitiativeStatus) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false, refetchOnMount: false } } });
  const onDecided = vi.fn();
  return {
    onDecided,
    ...render(
      <QueryClientProvider client={qc}>
        <InitiativeReviewPanel
          initiativeName="init"
          initiativeStatus={status}
          onDecided={onDecided}
        />
      </QueryClientProvider>,
    ),
  };
}

function makeRound(overrides: Partial<InitiativeReviewRound> = {}): InitiativeReviewRound {
  return {
    round: 1,
    generated_at: "2026-04-23T00:00:00Z",
    status: "complete",
    classification: "delivered",
    agent_assessment: "Shipped the thing.",
    evidence: [],
    ...overrides,
  };
}

function makeDecision(overrides: Partial<InitiativeReviewDecision> = {}): InitiativeReviewDecision {
  return {
    verdict: "accept",
    status: "completed",
    decided_at: "2026-04-23T00:05:00Z",
    decided_by: "matt",
    prior_status: "review_pending",
    ...overrides,
  };
}

beforeEach(() => {
  mockListRounds.mockReset();
  mockListDecisions.mockReset();
  mockTrigger.mockReset();
  mockDecide.mockReset();
  mockListDecisions.mockResolvedValue([]);
});
afterEach(() => cleanup());

describe("InitiativeReviewPanel", () => {
  it("shows the trigger button enabled when initiative is active", async () => {
    mockListRounds.mockResolvedValue([]);
    renderPanel("active");
    const trigger = await screen.findByTestId(selectors.initiativeReview.triggerButton);
    expect(trigger).toBeEnabled();
  });

  it("disables the trigger button outside of 'active' status", async () => {
    mockListRounds.mockResolvedValue([makeRound({ status: "gathering" })]);
    renderPanel("in_review");
    const trigger = await screen.findByTestId(selectors.initiativeReview.triggerButton);
    expect(trigger).toBeDisabled();
  });

  it("renders decide buttons when status is review_pending", async () => {
    mockListRounds.mockResolvedValue([makeRound()]);
    renderPanel("review_pending");
    expect(await screen.findByTestId(selectors.initiativeReview.verdictAccept)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.initiativeReview.verdictFail)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.initiativeReview.verdictFollowup)).toBeInTheDocument();
  });

  it("invokes decide() with verdict + rationale and calls onDecided", async () => {
    mockListRounds.mockResolvedValue([makeRound()]);
    mockDecide.mockResolvedValue({
      initiative: "init",
      verdict: "accept",
      status: "completed",
      decided_at: "2026-04-23T00:10:00Z",
    });
    const { onDecided } = renderPanel("review_pending");
    const rationale = await screen.findByTestId(selectors.initiativeReview.rationaleInput);
    await userEvent.type(rationale, "all green");
    await userEvent.click(screen.getByTestId(selectors.initiativeReview.verdictAccept));
    await waitFor(() => expect(mockDecide).toHaveBeenCalled());
    expect(mockDecide).toHaveBeenCalledWith("init", { verdict: "accept", rationale: "all green" });
    await waitFor(() => expect(onDecided).toHaveBeenCalled());
  });

  it("triggers a manual review run", async () => {
    mockListRounds.mockResolvedValue([]);
    mockTrigger.mockResolvedValue({ started: true, round: 1 });
    renderPanel("active");
    await userEvent.click(await screen.findByTestId(selectors.initiativeReview.triggerButton));
    await waitFor(() => expect(mockTrigger).toHaveBeenCalledWith("init"));
  });

  it("renders decision history rows when present", async () => {
    mockListRounds.mockResolvedValue([]);
    mockListDecisions.mockResolvedValue([makeDecision({ rationale: "ok ship" })]);
    renderPanel("completed");
    const row = await screen.findByTestId(selectors.initiativeReview.decisionRecord);
    expect(row).toHaveAttribute("data-verdict", "accept");
    expect(row).toHaveTextContent(/ok ship/i);
  });
});
