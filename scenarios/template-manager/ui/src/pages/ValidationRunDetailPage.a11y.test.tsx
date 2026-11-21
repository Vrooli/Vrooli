import { screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

import { ValidationRunDetailPage } from "./ValidationRunDetailPage";

const fetchValidationRun = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchValidationRun: () => fetchValidationRun(),
}));

describe("ValidationRunDetailPage accessibility", () => {
  it("renders the loaded run without axe violations", async () => {
    fetchValidationRun.mockResolvedValueOnce({
      id: "validation-1",
      templateId: "react-vite",
      mode: 2,
      target: "fleet",
      status: "passed",
      trigger: "monitor",
      phaseResults: [{ phase: "shallow", status: "passed", findingCount: 0 }],
      findings: [{ key: "react-vite.aria", severity: "medium", summary: "Missing aria label", source: "ui-health" }],
    });

    const { container } = renderWithProviders(
      <MemoryRouter initialEntries={["/runs/validation-1"]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Routes>
          <Route path="/runs/:runId" element={<ValidationRunDetailPage />} />
        </Routes>
      </MemoryRouter>,
      { withoutRouter: true },
    );

    await screen.findByTestId(selectors.pages.runDetail);
    await expectNoA11yViolations(container);
  });
});
