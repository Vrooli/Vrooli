import { screen } from "@testing-library/react";
import { describe, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

import { RunListPage } from "./RunListPage";

const fetchValidationRunList = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchValidationRunList: () => fetchValidationRunList(),
}));

describe("RunListPage accessibility", () => {
  it("renders the loaded list without axe violations", async () => {
    fetchValidationRunList.mockResolvedValueOnce([
      { id: "validation-1", templateId: "react-vite", mode: 2, status: "passed", trigger: "monitor", findings: [] },
    ]);

    const { container } = renderWithProviders(<RunListPage />);

    await screen.findByTestId(selectors.runList.root);
    await expectNoA11yViolations(container);
  });
});
