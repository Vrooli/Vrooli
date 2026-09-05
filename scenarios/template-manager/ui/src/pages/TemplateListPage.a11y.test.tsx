import { screen } from "@testing-library/react";
import { describe, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

import { TemplateListPage } from "./TemplateListPage";

const fetchTemplateList = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchTemplateList: () => fetchTemplateList(),
}));

describe("TemplateListPage accessibility", () => {
  it("renders the loaded list without axe violations", async () => {
    fetchTemplateList.mockResolvedValueOnce([
      { id: "react-vite", kind: 1, displayName: "React + Vite", version: "1.6.0", status: "active", tags: [], manifestPath: "", sourcePath: "", versionLag: { lagCount: 0 } },
    ]);

    const { container } = renderWithProviders(<TemplateListPage />);

    await screen.findByTestId(selectors.templateList.root);
    await expectNoA11yViolations(container);
  });
});
