import { screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";

import { DebtDetailPage } from "./DebtDetailPage";

const fetchDebtEntry = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchDebtEntry: () => fetchDebtEntry(),
}));

function renderAt(path = "/debt/react-vite.aria") {
  return renderWithProviders(
    <MemoryRouter initialEntries={[path]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <Routes>
        <Route path="/debt/:debtKey" element={<DebtDetailPage />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

const entryFixture = {
  key: "react-vite.aria",
  templateId: "react-vite",
  source: "ui-health",
  severity: "medium",
  status: "open",
  title: "Missing aria label",
  detail: "The primary CTA has no accessible name.",
};

describe("DebtDetailPage", () => {
  it("renders the debt entry with provenance and detail", async () => {
    fetchDebtEntry.mockResolvedValueOnce(entryFixture);

    renderAt();

    const page = await screen.findByTestId(selectors.pages.debtDetail);
    expect(page).toHaveTextContent("Missing aria label");
    expect(page).toHaveTextContent("react-vite.aria");
    expect(page).toHaveTextContent("ui-health");
    expect(page).toHaveTextContent("The primary CTA has no accessible name.");

    const templateLink = screen.getByRole("link", { name: "react-vite" });
    expect(templateLink).toHaveAttribute("href", "/templates/react-vite");
  });

  it("shows the empty-detail state when there is no message", async () => {
    fetchDebtEntry.mockResolvedValueOnce({ ...entryFixture, detail: "" });

    renderAt();

    await screen.findByTestId(selectors.pages.debtDetail);
    expect(screen.getByText(strings.debtDetail.messageEmpty)).toBeInTheDocument();
  });

  it("renders the loading state", () => {
    fetchDebtEntry.mockReturnValueOnce(new Promise(() => {}));

    renderAt();

    expect(screen.getByTestId(selectors.debtDetail.loading)).toBeInTheDocument();
  });

  it("renders the error state", async () => {
    fetchDebtEntry.mockRejectedValueOnce(new Error("boom"));

    renderAt();

    await waitFor(() => expect(screen.getByTestId(selectors.debtDetail.error)).toBeInTheDocument());
  });
});
