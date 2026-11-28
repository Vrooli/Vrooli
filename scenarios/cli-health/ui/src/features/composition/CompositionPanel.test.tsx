import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { CompositionPanel } from "./CompositionPanel";
import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { searchComposition } from "../../api/composition";

vi.mock("../../api/composition", () => ({ searchComposition: vi.fn() }));

describe("CompositionPanel", () => {
  afterEach(() => cleanup());

  it("renders bounded source provenance and partial runtime limits", async () => {
    vi.mocked(searchComposition).mockResolvedValue({
      results: [{ id: "cli-health:cli-composition:scenario-app", path: "cli/app.go", title: "cli-health ScenarioApp and command groups", text: "Static source evidence; runtime dispatch remains unverified.", score: 0.9, attestation: "DERIVED", evidence: [{ path: "cli/app.go", line: 22, sha256: "abc123" }] }],
      degradations: [], scope: "one scenario CLI source root",
    });
    const user = userEvent.setup();
    renderWithProviders(<CompositionPanel />);
    await user.type(screen.getByTestId(selectors.composition.query), "ScenarioApp command groups");
    await user.click(screen.getByTestId(selectors.composition.submit));
    expect(await screen.findByTestId(selectors.composition.result)).toBeInTheDocument();
    expect(screen.getByText(/cli\/app\.go:22 · sha256:abc123/)).toBeInTheDocument();
    expect(searchComposition).toHaveBeenCalledWith("ScenarioApp command groups", "cli-health");
  });
});
