import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { FreshnessState } from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { proposalFixture } from "../../test-utils/proposals";
import { ProposalQueue } from "./ProposalQueue";

describe("ProposalQueue", () => {
  it("shows each proposal's subject, work chips, file count, flags and freshness", () => {
    const drifted = proposalFixture({ id: "gctp-ffffffffffff", message: { subject: "bas: second epoch (E28)" }, work: { effortRef: "effort:bas", epoch: "E28" }, freshness: { state: FreshnessState.DRIFTED } });
    renderWithProviders(<ProposalQueue proposals={[proposalFixture(), drifted]} openCount={2} onOpen={vi.fn()} />);

    expect(screen.getByRole("button", { name: /proposals \(2\)/i })).toBeInTheDocument();
    const cards = screen.getAllByTestId("proposal-card");
    expect(cards).toHaveLength(2);
    expect(cards[0]).toHaveTextContent("bas: one ReplaySpec renderer (E27)");
    expect(cards[0]).toHaveTextContent("E27");
    expect(cards[0]).toHaveTextContent("browser-automation-studio-rehabilitation");
    expect(cards[0]).toHaveTextContent("2 files");
    expect(cards[0]).toHaveTextContent("mixed 1");
    expect(cards[0]).toHaveTextContent("Fresh");
    expect(cards[1]).toHaveTextContent("Drifted");
  });

  it("opens a proposal from its card", () => {
    const onOpen = vi.fn();
    renderWithProviders(<ProposalQueue proposals={[proposalFixture()]} openCount={1} onOpen={onOpen} />);
    fireEvent.click(screen.getByTestId("proposal-card"));
    expect(onOpen).toHaveBeenCalledWith("gctp-0123456789ab");
  });

  it("stays out of the way when nothing is proposed", () => {
    renderWithProviders(<ProposalQueue proposals={[]} openCount={0} onOpen={vi.fn()} />);
    expect(screen.queryByTestId("proposal-queue")).not.toBeInTheDocument();
  });
});
