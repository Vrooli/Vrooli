import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { HoldersPage } from "./HoldersPage";

vi.mock("../../api/tokenEconomy", () => ({
  nextIdempotencyKey: vi.fn(() => "test-key"),
  minterClient: {
    listHolders: vi.fn().mockResolvedValue({ holders: [{ id: "sam", displayName: "Sam", authenticatorSubject: "auth:sam" }] }),
    createHolder: vi.fn().mockResolvedValue({}),
  },
}));

describe("HoldersPage", () => {
  afterEach(() => cleanup());
  it("renders identity binding and the holder table accessibly", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<HoldersPage />);
    await waitFor(() => expect(screen.getByTestId("holders-table")).toBeInTheDocument());
    expect(screen.getByTestId("holders-identity-binding")).toBeInTheDocument();
    await user.type(screen.getByTestId("holders-display-name"), "Lee");
    await user.type(screen.getByTestId("holders-identity-binding"), "auth:lee");
    await user.click(screen.getByTestId("holders-add"));
    const { minterClient } = await import("../../api/tokenEconomy");
    await waitFor(() => expect(minterClient.createHolder).toHaveBeenCalledOnce());
    await expectNoA11yViolations(container);
  });
});
