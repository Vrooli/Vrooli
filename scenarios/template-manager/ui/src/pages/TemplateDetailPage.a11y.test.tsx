import { screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

import { TemplateDetailPage } from "./TemplateDetailPage";

const fetchTemplateDetail = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchTemplateDetail: () => fetchTemplateDetail(),
}));

describe("TemplateDetailPage accessibility", () => {
  it("renders the loaded template without axe violations", async () => {
    fetchTemplateDetail.mockResolvedValueOnce({
      template: {
        id: "react-vite",
        kind: 1,
        displayName: "React + Vite",
        version: "1.6.0",
        status: "active",
        manifestPath: "templates/react-vite/manifest.json",
        sourcePath: "templates/react-vite",
        tags: ["frontend"],
        versionLag: { currentVersion: "1.6.0", latestVersion: "1.7.0", lagCount: 1 },
      },
      runs: [{ id: "validation-1", templateId: "react-vite", mode: 2, status: "passed", trigger: "monitor", findings: [] }],
      drift: [{ id: "drift-1", templateId: "react-vite", target: "fleet", status: "drifted", driftCount: 4 }],
      debt: [{ key: "react-vite.aria", templateId: "react-vite", severity: "medium", status: "open", title: "Missing aria label" }],
    });

    const { container } = renderWithProviders(
      <MemoryRouter initialEntries={["/templates/react-vite"]} future={{ v7_relativeSplatPath: true, v7_startTransition: true }}>
        <Routes>
          <Route path="/templates/:templateId" element={<TemplateDetailPage />} />
        </Routes>
      </MemoryRouter>,
      { withoutRouter: true },
    );

    await screen.findByTestId(selectors.pages.templateDetail);
    await expectNoA11yViolations(container);
  });
});
