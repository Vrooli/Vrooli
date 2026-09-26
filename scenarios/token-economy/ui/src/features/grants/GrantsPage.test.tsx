import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { GrantsPage } from "./GrantsPage";

vi.mock("../../api/tokenEconomy", () => ({
  nextIdempotencyKey: vi.fn(() => "test-key"),
  minterClient: {
    listGrants: vi.fn().mockResolvedValue({ grants: [
      { id: "grant-1", holderId: "sam", amountMinor: 8n, status: 2, rules: [{ id: "rule-1", condition: 1 }] },
      { id: "grant-2", holderId: "lee", amountMinor: 2n, status: 5, rules: [] },
    ] }),
    createGrant: vi.fn().mockResolvedValue({}),
  },
}));

describe("GrantsPage", () => {
  afterEach(() => cleanup());
  it("renders rule authoring and grant evidence accessibly", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<GrantsPage />);
    await waitFor(() => expect(screen.getByTestId("grants-table")).toBeInTheDocument());
    expect(screen.getByTestId("grants-rule-editor")).toBeInTheDocument();
    expect(screen.getByTestId("grants-rule-preview")).toBeInTheDocument();
    await user.type(screen.getByTestId("grants-token-type"), "chores");
    await user.type(screen.getByTestId("grants-holder"), "sam");
    await user.clear(screen.getByTestId("grants-source"));
    await user.type(screen.getByTestId("grants-source"), "weekly-chores");
    await user.type(screen.getByTestId("grants-authorizer"), "parent");
    await user.clear(screen.getByTestId("grants-amount"));
    await user.type(screen.getByTestId("grants-amount"), "8");
    await user.type(screen.getByTestId("grants-scope"), "movie");
    await user.selectOptions(screen.getByTestId("grants-rule-condition"), "2");
    await user.type(screen.getByTestId("grants-rule-operand"), "late-night");
    await user.click(screen.getByTestId("grants-issue"));
    await user.clear(screen.getByTestId("grants-scope"));
    await user.clear(screen.getByTestId("grants-rule-operand"));
    await user.click(screen.getByTestId("grants-issue"));
    const { minterClient } = await import("../../api/tokenEconomy");
    await waitFor(() => expect(minterClient.createGrant).toHaveBeenCalledTimes(2));
    await expectNoA11yViolations(container);
  });
});
