import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ProposalsSection } from "./ProposalsSection";
import { useProposals } from "../../lib/hooks-proposals";

vi.mock("../../lib/hooks-proposals", () => ({ useProposals: vi.fn() }));

describe("ProposalsSection", () => {
  it("offers a retry when the inbox read fails", () => {
    const refetch = vi.fn();
    vi.mocked(useProposals).mockReturnValue({
      error: new Error("service unavailable"),
      refetch,
      isFetching: false,
      data: undefined,
    } as never);

    render(<ProposalsSection repoId="repo" onOpen={vi.fn()}><div /></ProposalsSection>);

    expect(screen.getByTestId("proposal-inbox-error")).toHaveTextContent("Proposal inbox is unavailable");
    fireEvent.click(screen.getByTestId("proposal-inbox-retry"));
    expect(refetch).toHaveBeenCalled();
  });
});
