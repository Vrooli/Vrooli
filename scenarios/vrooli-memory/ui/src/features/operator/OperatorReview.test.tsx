import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { getFrontier, listFacets, listPinProposals, resolvePinProposal } from "../../api/operator";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { OperatorReview } from "./OperatorReview";

vi.mock("../../api/operator", () => ({
  getFrontier: vi.fn(),
  listFacets: vi.fn(),
  listPinProposals: vi.fn(),
  resolvePinProposal: vi.fn(),
}));

describe("[REQ:VMEM-P1-006] OperatorReview", () => {
  afterEach(() => { cleanup(); vi.resetAllMocks(); });

  it("renders the frontier and resolves pin proposals accessibly", async () => {
    vi.mocked(getFrontier).mockResolvedValue([{ id: "summary", entryId: "entry", facetId: "episode", childIds: ["leaf"], depth: 1, span: 2 } as never]);
    vi.mocked(listFacets).mockResolvedValue([{ id: "episode", label: "Episode" } as never]);
    vi.mocked(listPinProposals).mockResolvedValue([{ id: "proposal", entryIds: ["entry"], rationale: "Review this pin" } as never]);
    vi.mocked(resolvePinProposal).mockResolvedValue(undefined);

    const user = userEvent.setup();
    const rendered = renderWithProviders(<OperatorReview />);
    await screen.findByTestId(selectors.operator.frontierList);
    await screen.findByTestId(selectors.operator.proposalList);
    expect(screen.getByText(strings.pages.operator.summary)).toBeInTheDocument();
    await expectNoA11yViolations(rendered.container);

    await user.click(screen.getByRole("button", { name: strings.pages.operator.accept }));
    await waitFor(() => expect(resolvePinProposal).toHaveBeenCalledWith("proposal", true));
    await user.click(screen.getByRole("button", { name: strings.pages.operator.reject }));
    await waitFor(() => expect(resolvePinProposal).toHaveBeenCalledWith("proposal", false));
  });

  it("renders empty and failed asynchronous surfaces", async () => {
    vi.mocked(getFrontier).mockResolvedValue([]);
    vi.mocked(listFacets).mockResolvedValue([]);
    vi.mocked(listPinProposals).mockResolvedValue([]);
    renderWithProviders(<OperatorReview />);
    await screen.findByText(strings.pages.operator.frontierEmpty);
    await screen.findByText(strings.pages.operator.proposalsEmpty);
    cleanup();

    vi.mocked(getFrontier).mockRejectedValue(new Error("frontier unavailable"));
    vi.mocked(listFacets).mockRejectedValue(new Error("facets unavailable"));
    vi.mocked(listPinProposals).mockRejectedValue(new Error("proposals unavailable"));
    renderWithProviders(<OperatorReview />);
    await waitFor(() => expect(screen.getByTestId(selectors.operator.frontierList)).toHaveAttribute("data-experience-state", "error"));
    expect(screen.getByTestId(selectors.operator.proposalList)).toHaveAttribute("data-experience-state", "error");
  });
});
