import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { EarningPage } from "./EarningPage";

vi.mock("../../api/tokenEconomy", () => ({
  nextIdempotencyKey: vi.fn(() => "test-key"),
  earningClient: {
    listEarnings: vi.fn().mockResolvedValue({ submissions: [
      { id: "earned-1", holderId: "sam", tokenTypeId: "chores", amountMinor: 4n, reason: "Cleared the table", adapterIdentity: "operator", replayed: false },
      { id: "earned-2", holderId: "lee", tokenTypeId: "chores", amountMinor: 2n, reason: "Fed the dog", adapterIdentity: "", replayed: true },
    ] }),
    submitEarning: vi.fn().mockResolvedValue({}),
  },
}));

describe("EarningPage", () => {
  afterEach(() => cleanup());
  it("renders operator entry through the ordinary earning surface accessibly", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<EarningPage />);
    await waitFor(() => expect(screen.getByTestId("earning-table")).toBeInTheDocument());
    expect(screen.getByTestId("earning-submit")).toBeInTheDocument();
    await user.type(screen.getByTestId("earning-holder"), "sam");
    await user.type(screen.getByTestId("earning-token-type"), "chores");
    await user.clear(screen.getByTestId("earning-amount"));
    await user.type(screen.getByTestId("earning-amount"), "4");
    await user.type(screen.getByTestId("earning-reason"), "Cleared the table");
    await user.click(screen.getByTestId("earning-submit"));
    const { earningClient } = await import("../../api/tokenEconomy");
    await waitFor(() => expect(earningClient.submitEarning).toHaveBeenCalledOnce());
    await expectNoA11yViolations(container);
  });
});
