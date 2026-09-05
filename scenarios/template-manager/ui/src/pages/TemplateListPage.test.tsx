import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

import { TemplateListPage } from "./TemplateListPage";

const fetchTemplateList = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchTemplateList: () => fetchTemplateList(),
}));

const templates = [
  { id: "react-vite", kind: 1, displayName: "React + Vite", version: "1.6.0", status: "active", tags: [], manifestPath: "", sourcePath: "", versionLag: { lagCount: 0 } },
  { id: "minimal-resource", kind: 3, displayName: "Minimal Resource", version: "1.0.0", status: "active", tags: [], manifestPath: "", sourcePath: "", versionLag: { lagCount: 1 } },
];

describe("TemplateListPage", () => {
  it("lists templates with links to their detail views", async () => {
    fetchTemplateList.mockResolvedValueOnce(templates);

    renderWithProviders(<TemplateListPage />);

    await screen.findByTestId(selectors.templateList.root);
    const row = screen.getByTestId(selectors.templateList.row({ id: "react-vite" }));
    expect(row).toHaveAttribute("href", "/templates/react-vite");
    expect(row).toHaveTextContent("React + Vite");
    expect(screen.getByTestId(selectors.templateList.row({ id: "minimal-resource" }))).toBeInTheDocument();
  });

  it("filters by kind", async () => {
    fetchTemplateList.mockResolvedValueOnce(templates);

    renderWithProviders(<TemplateListPage />);

    await screen.findByTestId(selectors.templateList.root);
    fireEvent.change(screen.getByTestId(selectors.templateList.kindFilter), { target: { value: "3" } });

    await waitFor(() =>
      expect(screen.queryByTestId(selectors.templateList.row({ id: "react-vite" }))).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.templateList.row({ id: "minimal-resource" }))).toBeInTheDocument();
  });

  it("renders the loading state", () => {
    fetchTemplateList.mockReturnValueOnce(new Promise(() => {}));

    renderWithProviders(<TemplateListPage />);

    expect(screen.getByTestId(selectors.templateList.loading)).toBeInTheDocument();
  });

  it("renders the error state", async () => {
    fetchTemplateList.mockRejectedValueOnce(new Error("boom"));

    renderWithProviders(<TemplateListPage />);

    await waitFor(() => expect(screen.getByTestId(selectors.templateList.error)).toBeInTheDocument());
  });
});
