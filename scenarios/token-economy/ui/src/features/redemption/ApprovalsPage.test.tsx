import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { ApprovalsPage } from "./ApprovalsPage";

vi.mock("../../api/tokenEconomy", () => ({
  nextIdempotencyKey: vi.fn(() => "test-key"),
  minterClient: {
    listPendingRedemptions: vi.fn().mockResolvedValue({ redemptions: [
      { id: "redeem-1", holderId: "sam", catalogEntryId: "movie", amount: 5n },
      { id: "redeem-2", holderId: "lee", catalogEntryId: "story", amount: 3n },
    ] }),
    approveRedemption: vi.fn().mockResolvedValue({}),
    denyRedemption: vi.fn().mockResolvedValue({}),
  },
}));

describe("ApprovalsPage", () => {
  afterEach(() => cleanup());
  it("renders the approval queue and its empty state accessibly", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<ApprovalsPage />);
    await waitFor(() => expect(screen.getByTestId("approvals-table")).toBeInTheDocument());
    await user.type(screen.getByTestId("approvals-reason-redeem-1"), "The work was completed");
    await user.click(screen.getByTestId("approvals-approve-redeem-1"));
    await user.click(screen.getByTestId("approvals-deny-redeem-1"));
    await user.click(screen.getByTestId("approvals-approve-redeem-2"));
    await user.click(screen.getByTestId("approvals-deny-redeem-2"));
    const { minterClient } = await import("../../api/tokenEconomy");
    await waitFor(() => expect(minterClient.approveRedemption).toHaveBeenCalledTimes(2));
    expect(minterClient.denyRedemption).toHaveBeenCalledTimes(2);
    await expectNoA11yViolations(container);
  });
});
