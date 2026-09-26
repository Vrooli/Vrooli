import { screen } from "@testing-library/react";
import { describe, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

import { DebtListPage } from "./DebtListPage";

const fetchDebtLedger = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchDebtLedger: () => fetchDebtLedger(),
}));

describe("DebtListPage accessibility", () => {
  it("renders the loaded ledger without axe violations", async () => {
    fetchDebtLedger.mockResolvedValueOnce({
      entries: [
        { key: "react-vite.aria", templateId: "react-vite", severity: "medium", status: "open", title: "Missing aria label", source: "ui-health" },
      ],
      templates: [
        { id: "react-vite", displayName: "React + Vite", kind: 1, version: "1.6.0", status: "active", tags: [], manifestPath: "", sourcePath: "" },
      ],
    });

    const { container } = renderWithProviders(<DebtListPage />);

    await screen.findByTestId(selectors.debtList.root);
    await expectNoA11yViolations(container);
  });
});
