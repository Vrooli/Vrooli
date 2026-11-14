import { afterEach, describe, it, vi } from "vitest";
import { cleanup, waitFor, screen } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";

vi.mock("../api/scenarios", () => ({
  fetchScenarios: vi.fn().mockResolvedValue({
    vrooliRoot: "/repo",
    scenarios: [
      { id: "alpha", displayName: "Alpha", description: "First", path: "/repo/alpha", flowCount: 2 },
    ],
  }),
  fetchScenarioDetail: vi.fn(),
}));

vi.mock("../api/artifacts", () => ({
  generateScenarioArtifacts: vi.fn(),
  clearScenarioArtifacts: vi.fn(),
}));

import { ScenariosPage } from "./ScenariosPage";

describe("ScenariosPage accessibility", () => {
  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(<ScenariosPage />);
    await waitFor(() => expect(screen.getByTestId("scenario-table")).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
