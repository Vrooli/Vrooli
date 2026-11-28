import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

import { DebtListPage } from "./DebtListPage";

const fetchDebtLedger = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchDebtLedger: () => fetchDebtLedger(),
}));

const ledgerFixture = {
  entries: [
    { key: "react-vite.aria", templateId: "react-vite", severity: "medium", status: "open", title: "Missing aria label", source: "ui-health" },
    { key: "minimal.lint", templateId: "minimal-resource", severity: "low", status: "resolved", title: "Lint warning", source: "quality-health" },
  ],
  templates: [
    { id: "react-vite", displayName: "React + Vite", kind: 1, version: "1.6.0", status: "active", tags: [], manifestPath: "", sourcePath: "" },
    { id: "minimal-resource", displayName: "Minimal Resource", kind: 3, version: "1.0.0", status: "active", tags: [], manifestPath: "", sourcePath: "" },
  ],
};

describe("DebtListPage", () => {
  it("lists every debt entry with links to its detail view", async () => {
    fetchDebtLedger.mockResolvedValueOnce(ledgerFixture);

    renderWithProviders(<DebtListPage />);

    await screen.findByTestId(selectors.debtList.root);
    const row = screen.getByTestId(selectors.debtList.row({ key: "react-vite.aria" }));
    expect(row).toHaveAttribute("href", "/debt/react-vite.aria");
    expect(row).toHaveTextContent("Missing aria label");
    expect(screen.getByTestId(selectors.debtList.row({ key: "minimal.lint" }))).toHaveTextContent("Lint warning");
  });

  it("filters by template", async () => {
    fetchDebtLedger.mockResolvedValueOnce(ledgerFixture);

    renderWithProviders(<DebtListPage />);

    await screen.findByTestId(selectors.debtList.root);
    const filter = screen.getAllByTestId(selectors.debtList.templateFilter).find((element) => element.tagName === "BUTTON");
    expect(filter).toBeDefined();
    await userEvent.click(filter!);
    await userEvent.click(await screen.findByRole("option", { name: "Minimal Resource" }));

    await waitFor(() =>
      expect(screen.queryByTestId(selectors.debtList.row({ key: "react-vite.aria" }))).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.debtList.row({ key: "minimal.lint" }))).toBeInTheDocument();
  });

  it("filters by status", async () => {
    fetchDebtLedger.mockResolvedValueOnce(ledgerFixture);

    renderWithProviders(<DebtListPage />);

    await screen.findByTestId(selectors.debtList.root);
    const filter = screen.getAllByTestId(selectors.debtList.statusFilter).find((element) => element.tagName === "BUTTON");
    expect(filter).toBeDefined();
    await userEvent.click(filter!);
    await userEvent.click(await screen.findByRole("option", { name: "open" }));

    await waitFor(() =>
      expect(screen.queryByTestId(selectors.debtList.row({ key: "minimal.lint" }))).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.debtList.row({ key: "react-vite.aria" }))).toBeInTheDocument();
  });

  it("offers a template option for every referenced template", async () => {
    fetchDebtLedger.mockResolvedValueOnce(ledgerFixture);

    renderWithProviders(<DebtListPage />);

    await screen.findByTestId(selectors.debtList.root);
    const templateFilter = screen.getAllByTestId(selectors.debtList.templateFilter).find((element) => element.tagName === "BUTTON");
    expect(templateFilter).toBeDefined();
    await userEvent.click(templateFilter!);
    expect(await screen.findByRole("option", { name: "React + Vite" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Minimal Resource" })).toBeInTheDocument();
  });

  it("renders the loading state", () => {
    fetchDebtLedger.mockReturnValueOnce(new Promise(() => {}));

    renderWithProviders(<DebtListPage />);

    expect(screen.getByTestId(selectors.debtList.loading)).toBeInTheDocument();
  });

  it("renders the error state", async () => {
    fetchDebtLedger.mockRejectedValueOnce(new Error("boom"));

    renderWithProviders(<DebtListPage />);

    await waitFor(() => expect(screen.getByTestId(selectors.debtList.error)).toBeInTheDocument());
  });
});
