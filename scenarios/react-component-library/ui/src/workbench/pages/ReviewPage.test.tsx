import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import {
  FindingSeverity,
  FindingStatus,
  type Finding,
  type ReviewSummary,
} from "../../api/review";

vi.mock("../../api/review", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/review")>();
  return {
    ...actual,
    listFindingsPage: vi.fn(),
    getReviewSummary: vi.fn(),
    recordDisposition: vi.fn(),
    assignFinding: vi.fn(),
  };
});

import * as reviewApi from "../../api/review";
import { ReviewPage } from "./ReviewPage";

const summary: ReviewSummary = {
  $typeName: "vrooli.react_component_library.v1.review.ReviewSummary",
  open: 60,
  blockers: 2,
  high: 5,
  acknowledged: 3,
  accepted: 0,
  fixed: 0,
};

function finding(id: string, overrides: Partial<Finding> = {}): Finding {
  return {
    $typeName: "vrooli.react_component_library.v1.review.Finding",
    id,
    subjectKind: "asset",
    subjectId: `asset-${id}`,
    gate: "accessibility",
    severity: FindingSeverity.HIGH,
    status: FindingStatus.OPEN,
    message: `message for ${id}`,
    remediation: "fix it",
    evidenceJson: "{}",
    docsRef: "",
    owner: "",
    createdAt: "2026-11-18T09:00:00Z",
    dispositions: [],
    ...overrides,
  };
}

describe("ReviewPage", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.mocked(reviewApi.getReviewSummary).mockResolvedValue(summary);
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("pages through the filtered queue and resets to page 1 when a filter changes", async () => {
    vi.mocked(reviewApi.listFindingsPage).mockImplementation((input = {}) =>
      Promise.resolve(
        (input.offset ?? 0) === 0
          ? { findings: [finding("a"), finding("b")], total: 60 }
          : { findings: [finding("c"), finding("d")], total: 60 },
      ),
    );

    renderWithProviders(<ReviewPage />);

    expect(await screen.findByRole("button", { name: /asset-a/ })).toBeInTheDocument();
    expect(screen.getByTestId("review-range")).toHaveTextContent("1–2 of 60");
    expect(screen.getByRole("button", { name: /Previous/ })).toBeDisabled();
    expect(reviewApi.listFindingsPage).toHaveBeenCalledWith(
      expect.objectContaining({ offset: 0, limit: 25 }),
    );

    fireEvent.click(screen.getByRole("button", { name: /Next/ }));
    await waitFor(() =>
      expect(reviewApi.listFindingsPage).toHaveBeenCalledWith(
        expect.objectContaining({ offset: 25, limit: 25 }),
      ),
    );
    expect(await screen.findByRole("button", { name: /asset-c/ })).toBeInTheDocument();
    expect(screen.getByTestId("review-range")).toHaveTextContent("26–27 of 60");

    fireEvent.change(screen.getByLabelText(/Severity/), {
      target: { value: String(FindingSeverity.BLOCKER) },
    });
    await waitFor(() =>
      expect(reviewApi.listFindingsPage).toHaveBeenLastCalledWith(
        expect.objectContaining({ severity: FindingSeverity.BLOCKER, offset: 0 }),
      ),
    );
  });

  it("shows an empty page with a way back instead of claiming the queue is empty", async () => {
    vi.mocked(reviewApi.listFindingsPage).mockImplementation((input = {}) =>
      Promise.resolve(
        (input.offset ?? 0) === 0
          ? { findings: [finding("a")], total: 60 }
          : { findings: [], total: 60 },
      ),
    );

    renderWithProviders(<ReviewPage />);
    await screen.findByRole("button", { name: /asset-a/ });

    fireEvent.click(screen.getByRole("button", { name: /Next/ }));
    expect(await screen.findByText(/This page is empty/)).toBeInTheDocument();
    expect(screen.queryByText(/Nothing needs a decision/)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Back to first page" }));
    await waitFor(() =>
      expect(reviewApi.listFindingsPage).toHaveBeenLastCalledWith(
        expect.objectContaining({ offset: 0 }),
      ),
    );
  });

  it("shows a request error as unavailable rather than an empty queue", async () => {
    vi.mocked(reviewApi.listFindingsPage).mockRejectedValue(new Error("offline"));

    renderWithProviders(<ReviewPage />);

    expect(await screen.findByText(/did not answer/)).toBeInTheDocument();
    expect(screen.queryByText(/Nothing needs a decision/)).not.toBeInTheDocument();
  });

  it("clears the decision draft when the selected finding changes", async () => {
    vi.mocked(reviewApi.listFindingsPage).mockResolvedValue({
      findings: [finding("a"), finding("b")],
      total: 2,
    });

    renderWithProviders(<ReviewPage />);
    await screen.findByRole("button", { name: /asset-a/ });

    const reason = screen.getByLabelText(/Reason/);
    fireEvent.change(reason, { target: { value: "because a" } });
    expect(reason).toHaveValue("because a");

    fireEvent.click(screen.getByRole("button", { name: /asset-b/ }));
    expect(screen.getByLabelText(/Reason/)).toHaveValue("");
  });
});
