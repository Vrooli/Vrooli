import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";

import { ValidationRunDetailPage } from "./ValidationRunDetailPage";

const fetchValidationRun = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchValidationRun: () => fetchValidationRun(),
}));

function renderAt(path = "/runs/validation-1") {
  return renderWithProviders(
    <MemoryRouter initialEntries={[path]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <Routes>
        <Route path="/runs/:runId" element={<ValidationRunDetailPage />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

const runFixture = {
  id: "validation-1",
  templateId: "react-vite",
  mode: 2,
  target: "fleet",
  status: "passed",
  trigger: "monitor",
  phaseResults: [{ phase: "shallow", status: "passed", findingCount: 0 }],
  findings: [{ key: "react-vite.aria", severity: "medium", summary: "Missing aria label", source: "ui-health" }],
};

describe("ValidationRunDetailPage", () => {
  it("renders run overview, phases, and findings", async () => {
    fetchValidationRun.mockResolvedValueOnce(runFixture);

    renderAt();

    const page = await screen.findByTestId(selectors.pages.runDetail);
    expect(page).toHaveTextContent("validation-1");
    expect(page).toHaveTextContent("deep");
    expect(page).toHaveTextContent("shallow");
    // Findings start collapsed to keep the page short; expand to inspect them.
    fireEvent.click(screen.getByTestId(`${selectors.runDetail.findings}-toggle`));
    expect(page).toHaveTextContent("Missing aria label");
    expect(page).toHaveTextContent("ui-health");
  });

  it("shows the empty findings state for a clean run", async () => {
    fetchValidationRun.mockResolvedValueOnce({ ...runFixture, findings: [] });

    renderAt();

    await screen.findByTestId(selectors.pages.runDetail);
    fireEvent.click(screen.getByTestId(`${selectors.runDetail.findings}-toggle`));
    expect(screen.getByText(strings.runDetail.findingsEmpty)).toBeInTheDocument();
  });

  it("renders the loading state", () => {
    fetchValidationRun.mockReturnValueOnce(new Promise(() => {}));

    renderAt();

    expect(screen.getByTestId(selectors.runDetail.loading)).toBeInTheDocument();
  });

  it("renders the error state", async () => {
    fetchValidationRun.mockRejectedValueOnce(new Error("boom"));

    renderAt();

    await waitFor(() => expect(screen.getByTestId(selectors.runDetail.error)).toBeInTheDocument());
  });
});
