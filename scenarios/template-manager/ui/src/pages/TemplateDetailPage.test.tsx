import { fireEvent, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";

import { TemplateDetailPage } from "./TemplateDetailPage";

const fetchTemplateDetail = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchTemplateDetail: () => fetchTemplateDetail(),
}));

function renderAt(path = "/templates/react-vite") {
  return renderWithProviders(
    <MemoryRouter initialEntries={[path]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
      <Routes>
        <Route path="/templates/:templateId" element={<TemplateDetailPage />} />
      </Routes>
    </MemoryRouter>,
    { withoutRouter: true },
  );
}

const detailFixture = {
  template: {
    id: "react-vite",
    kind: 1,
    displayName: "React + Vite",
    version: "1.6.0",
    status: "active",
    manifestPath: "templates/react-vite/manifest.json",
    sourcePath: "templates/react-vite",
    tags: ["frontend", "vite"],
    versionLag: { currentVersion: "1.6.0", latestVersion: "1.7.0", lagCount: 1 },
  },
  runs: [{ id: "validation-1", templateId: "react-vite", mode: 2, status: "passed", trigger: "monitor", findings: [] }],
  drift: [{ id: "drift-1", templateId: "react-vite", target: "fleet", status: "drifted", driftCount: 4 }],
  debt: [{ key: "react-vite.aria", templateId: "react-vite", severity: "medium", status: "open", title: "Missing aria label" }],
};

describe("TemplateDetailPage", () => {
  it("renders template overview and linked runs, drift, and debt", async () => {
    fetchTemplateDetail.mockResolvedValueOnce(detailFixture);

    renderAt();

    const page = await screen.findByTestId(selectors.pages.templateDetail);
    expect(page).toHaveTextContent("React + Vite");
    expect(page).toHaveTextContent("react-vite");
    expect(page).toHaveTextContent("frontend, vite");

    // Runs/drift/debt sections start collapsed; expand before asserting links.
    fireEvent.click(screen.getByTestId(`${selectors.templateDetail.runs}-toggle`));
    fireEvent.click(screen.getByTestId(`${selectors.templateDetail.debt}-toggle`));
    const runLink = screen.getByTestId(selectors.templateDetail.runLink({ id: "validation-1" }));
    expect(runLink).toHaveAttribute("href", "/runs/validation-1");
    const debtLink = screen.getByTestId(selectors.templateDetail.debtLink({ key: "react-vite.aria" }));
    expect(debtLink).toHaveAttribute("href", "/debt/react-vite.aria");
  });

  it("renders empty section states", async () => {
    fetchTemplateDetail.mockResolvedValueOnce({ ...detailFixture, runs: [], drift: [], debt: [] });

    renderAt();

    await screen.findByTestId(selectors.pages.templateDetail);
    fireEvent.click(screen.getByTestId(`${selectors.templateDetail.runs}-toggle`));
    fireEvent.click(screen.getByTestId(`${selectors.templateDetail.drift}-toggle`));
    fireEvent.click(screen.getByTestId(`${selectors.templateDetail.debt}-toggle`));
    expect(screen.getByText(strings.templateDetail.runsEmpty)).toBeInTheDocument();
    expect(screen.getByText(strings.templateDetail.driftEmpty)).toBeInTheDocument();
    expect(screen.getByText(strings.templateDetail.debtEmpty)).toBeInTheDocument();
  });

  it("renders the loading state", () => {
    fetchTemplateDetail.mockReturnValueOnce(new Promise(() => {}));

    renderAt();

    expect(screen.getByTestId(selectors.templateDetail.loading)).toBeInTheDocument();
  });

  it("renders the error state", async () => {
    fetchTemplateDetail.mockRejectedValueOnce(new Error("boom"));

    renderAt();

    await waitFor(() => expect(screen.getByTestId(selectors.templateDetail.error)).toBeInTheDocument());
  });
});
