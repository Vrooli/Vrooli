import { describe, it, vi, beforeEach } from "vitest";
import { waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { expectNoA11yViolations } from "../../test-utils/a11y";

vi.mock("../../api/validation", () => {
  const validateScenario = vi.fn((scenario: string) =>
    Promise.resolve({
      scenario,
      passed: false,
      findings: [
        { severity: "error", code: "missing-slot", location: "ui/src/pages/foo.tsx", message: "Missing slot", suggestion: "Add it." },
      ],
      summary: { errors: 1, warnings: 0, infos: 0 },
      ranAt: "2026-05-20T10:00:00.000Z",
    }),
  );
  return { validateScenario };
});

import { ValidationListPage } from "./ValidationListPage";
import { ValidationDetailPage } from "./ValidationDetailPage";

beforeEach(() => {
  window.localStorage.clear();
});

describe("Validation feature accessibility", () => {
  it("ValidationListPage has no axe violations", async () => {
    const { container } = renderWithProviders(<ValidationListPage />);
    await expectNoA11yViolations(container);
  });

  it("ValidationDetailPage has no axe violations once the result is loaded", async () => {
    const { container, findByText } = renderWithProviders(
      <Routes>
        <Route path="/validation/:scenarioId" element={<ValidationDetailPage />} />
      </Routes>,
      { routerEntries: ["/validation/ui-health"] },
    );
    await waitFor(async () => {
      await findByText("Missing slot");
    });
    await expectNoA11yViolations(container);
  });
});
