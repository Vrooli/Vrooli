import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/validation", () => {
  const validateScenario = vi.fn();
  return { validateScenario };
});

import { ValidationDetailPage } from "./ValidationDetailPage";
import { validateScenario } from "../../api/validation";
import type { ValidationResult } from "../../api/validation";

const baseResult: ValidationResult = {
  scenario: "ui-health",
  passed: false,
  findings: [
    { severity: "error", code: "missing-slot", location: "ui/src/pages/foo.tsx", message: "Missing slot", suggestion: "Add it." },
    { severity: "warning", code: "stale-key", location: "ui/manifest.json", message: "Stale key", suggestion: "" },
    { severity: "info", code: "tip", location: "", message: "Tip", suggestion: "" },
  ],
  summary: { errors: 1, warnings: 1, infos: 1 },
  ranAt: "2026-05-20T10:00:00.000Z",
};

beforeEach(() => {
  window.localStorage.clear();
  vi.mocked(validateScenario).mockReset();
});

describe("ValidationDetailPage", () => {
  it("shows the summary and findings after the initial query resolves", async () => {
    vi.mocked(validateScenario).mockResolvedValueOnce(baseResult);
    renderWithProviders(
      <Routes>
        <Route path="/validation/:scenarioId" element={<ValidationDetailPage />} />
      </Routes>,
      { routerEntries: ["/validation/ui-health"] },
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.validation.detail.summary)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.validation.detail.findings)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.validation.detail.statusBadge)).toBeInTheDocument();
    expect(screen.getAllByTestId(/validation-detail-finding-/)).toHaveLength(3);
  });

  it("filters findings by severity", async () => {
    const user = userEvent.setup();
    vi.mocked(validateScenario).mockResolvedValueOnce(baseResult);
    renderWithProviders(
      <Routes>
        <Route path="/validation/:scenarioId" element={<ValidationDetailPage />} />
      </Routes>,
      { routerEntries: ["/validation/ui-health"] },
    );
    await waitFor(() =>
      expect(screen.getAllByTestId(/validation-detail-finding-/)).toHaveLength(3),
    );
    await user.click(
      screen.getByTestId(selectors.validation.severityFilter({ severity: "error" })),
    );
    expect(screen.getAllByTestId(/validation-detail-finding-/)).toHaveLength(1);
  });

  it("revalidates on button click", async () => {
    const user = userEvent.setup();
    vi.mocked(validateScenario).mockResolvedValue(baseResult);
    renderWithProviders(
      <Routes>
        <Route path="/validation/:scenarioId" element={<ValidationDetailPage />} />
      </Routes>,
      { routerEntries: ["/validation/ui-health"] },
    );
    await waitFor(() => expect(validateScenario).toHaveBeenCalledTimes(1));
    await user.click(screen.getByTestId(selectors.validation.detail.revalidate));
    await waitFor(() => expect(validateScenario).toHaveBeenCalledTimes(2));
  });

  it("renders a 'no findings' empty state when there are no findings", async () => {
    vi.mocked(validateScenario).mockResolvedValueOnce({
      ...baseResult,
      passed: true,
      findings: [],
      summary: { errors: 0, warnings: 0, infos: 0 },
    });
    renderWithProviders(
      <Routes>
        <Route path="/validation/:scenarioId" element={<ValidationDetailPage />} />
      </Routes>,
      { routerEntries: ["/validation/ui-health"] },
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.validation.detail.empty)).toBeInTheDocument(),
    );
  });

  it("surfaces an API error in an alert", async () => {
    vi.mocked(validateScenario).mockRejectedValueOnce(new Error("boom"));
    renderWithProviders(
      <Routes>
        <Route path="/validation/:scenarioId" element={<ValidationDetailPage />} />
      </Routes>,
      { routerEntries: ["/validation/ui-health"] },
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.validation.detail.error)).toBeInTheDocument(),
    );
  });
});
