import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

import { RunListPage } from "./RunListPage";

const fetchValidationRunList = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchValidationRunList: () => fetchValidationRunList(),
}));

const runs = [
  { id: "validation-1", templateId: "react-vite", mode: 2, status: "passed", trigger: "monitor", findings: [] },
  { id: "validation-2", templateId: "minimal-resource", mode: 1, status: "failed", trigger: "manual", findings: [{}] },
];

describe("RunListPage", () => {
  it("lists validation runs with links to their detail views", async () => {
    fetchValidationRunList.mockResolvedValueOnce(runs);

    renderWithProviders(<RunListPage />);

    await screen.findByTestId(selectors.runList.root);
    const row = screen.getByTestId(selectors.runList.row({ id: "validation-1" }));
    expect(row).toHaveAttribute("href", "/runs/validation-1");
    expect(screen.getByTestId(selectors.runList.row({ id: "validation-2" }))).toBeInTheDocument();
  });

  it("filters by status", async () => {
    fetchValidationRunList.mockResolvedValueOnce(runs);

    renderWithProviders(<RunListPage />);

    await screen.findByTestId(selectors.runList.root);
    fireEvent.change(screen.getByTestId(selectors.runList.statusFilter), { target: { value: "failed" } });

    await waitFor(() =>
      expect(screen.queryByTestId(selectors.runList.row({ id: "validation-1" }))).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.runList.row({ id: "validation-2" }))).toBeInTheDocument();
  });

  it("renders the loading state", () => {
    fetchValidationRunList.mockReturnValueOnce(new Promise(() => {}));

    renderWithProviders(<RunListPage />);

    expect(screen.getByTestId(selectors.runList.loading)).toBeInTheDocument();
  });

  it("renders the error state", async () => {
    fetchValidationRunList.mockRejectedValueOnce(new Error("boom"));

    renderWithProviders(<RunListPage />);

    await waitFor(() => expect(screen.getByTestId(selectors.runList.error)).toBeInTheDocument());
  });
});
