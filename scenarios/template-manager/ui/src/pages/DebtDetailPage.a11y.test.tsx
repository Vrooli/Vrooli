import { screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

import { DebtDetailPage } from "./DebtDetailPage";

const fetchDebtEntry = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchDebtEntry: () => fetchDebtEntry(),
}));

describe("DebtDetailPage accessibility", () => {
  it("renders the loaded debt entry without axe violations", async () => {
    fetchDebtEntry.mockResolvedValueOnce({
      key: "react-vite.aria",
      templateId: "react-vite",
      source: "ui-health",
      severity: "medium",
      status: "open",
      title: "Missing aria label",
      detail: "The primary CTA has no accessible name.",
    });

    const { container } = renderWithProviders(
      <MemoryRouter initialEntries={["/debt/react-vite.aria"]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Routes>
          <Route path="/debt/:debtKey" element={<DebtDetailPage />} />
        </Routes>
      </MemoryRouter>,
      { withoutRouter: true },
    );

    await screen.findByTestId(selectors.pages.debtDetail);
    await expectNoA11yViolations(container);
  });
});
