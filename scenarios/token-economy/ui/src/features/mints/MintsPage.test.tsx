import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { MintsPage } from "./MintsPage";

vi.mock("../../api/tokenEconomy", () => ({
  minterClient: {
    listTokenTypes: vi.fn().mockResolvedValue({ tokenTypes: [
      { id: "chores", name: "Chore stars", symbol: "STAR", color: "#2563eb", mintedAmount: 8n, capAmount: 20n, retired: false },
      { id: "kindness", name: "Kindness points", symbol: "KIND", color: "#f59e0b", mintedAmount: 3n, capAmount: 0n, retired: true },
    ] }),
    createTokenType: vi.fn().mockResolvedValue({}),
    retireTokenType: vi.fn().mockResolvedValue({}),
    mintSupply: vi.fn().mockResolvedValue({}),
  },
}));

describe("MintsPage", () => {
  afterEach(() => cleanup());
  it("renders the authority form and empty table accessibly", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<MintsPage />);
    await waitFor(() => expect(screen.getByTestId("token-types-list")).toBeInTheDocument());
    expect(screen.getByTestId("token-types-create")).toBeInTheDocument();
    await user.type(screen.getByTestId("token-types-name"), "Kindness points");
    await user.type(screen.getByTestId("token-types-symbol"), "KIND");
    await user.type(screen.getByTestId("token-types-minter-subject"), "parent");
    await user.selectOptions(screen.getByTestId("token-types-supply-policy"), "2");
    await user.clear(screen.getByTestId("token-types-cap"));
    await user.type(screen.getByTestId("token-types-cap"), "100");
    await user.click(screen.getByTestId("token-types-create"));
    await user.clear(screen.getByTestId("token-types-mint-amount-chores"));
    await user.type(screen.getByTestId("token-types-mint-amount-chores"), "2");
    await user.click(screen.getByTestId("token-types-mint-chores"));
    await user.click(screen.getByTestId("token-types-retire-chores"));
    const { minterClient } = await import("../../api/tokenEconomy");
    await waitFor(() => expect(minterClient.createTokenType).toHaveBeenCalledOnce());
    expect(minterClient.mintSupply).toHaveBeenCalledOnce();
    expect(minterClient.retireTokenType).toHaveBeenCalledOnce();
    await expectNoA11yViolations(container);
  });
});
