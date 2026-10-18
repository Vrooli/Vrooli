import { afterEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";

import { conversationsClient } from "../../api/notifications";
import { renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { AsksPage } from "./AsksPage";

vi.mock("../../api/notifications", () => ({
  identityClient: { getSession: vi.fn(() => Promise.resolve({ signedIn: false })) },
  conversationsClient: { getAsk: vi.fn(), answer: vi.fn(), listAsks: vi.fn() },
}));

describe("AsksPage", () => {
  afterEach(() => vi.clearAllMocks());

  it("shows configured-recipient asks without prompting for login", async () => {
    await setLocale("en");
    vi.mocked(conversationsClient.listAsks).mockResolvedValue({
      asks: [{ id: "ask-1", question: "Review the remaining delivery gap", options: [], recommended: "", state: "pending" }],
    } as never);

    renderWithProviders(
      <Routes>
        <Route path="/asks" element={<AsksPage />} />
      </Routes>,
      { initialEntries: ["/asks"] },
    );

    expect(await screen.findByText("Review the remaining delivery gap")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.asks.list)).toBeInTheDocument();
    expect(screen.queryByText(/sign in/i)).not.toBeInTheDocument();
    expect(conversationsClient.listAsks).toHaveBeenCalledWith({ openOnly: true, limit: 50 });
  });
});
