import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { AutonomyReadyPrompt } from "./AutonomyReadyPrompt";

const mocks = vi.hoisted(() => ({ list: vi.fn(), get: vi.fn(), update: vi.fn() }));
const { list, get, update } = mocks;

vi.mock("../../services", () => ({
  transitionService: { list: mocks.list },
  settingsService: { get: mocks.get, update: mocks.update },
}));

describe("AutonomyReadyPrompt", () => {
  beforeEach(() => {
    list.mockResolvedValue([{
      key: "capture.classify",
      humanGates: [{ id: "capture-to-suggested", decides: "Accept capture", mode: "manual", readiness: "ready", acceptanceRate: 0.9, sampleSize: 20, threshold: 0.8 }],
    }]);
    get.mockResolvedValue({ autonomyGateModes: {} });
    update.mockResolvedValue({ autonomyGateModes: { "capture-to-suggested": "auto" } });
  });

  it("names a server-certified ready gate and offers flip and dismissal", async () => {
    renderWithProviders(<AutonomyReadyPrompt />);
    expect(await screen.findByTestId("autonomy-ready-prompts")).toHaveTextContent("capture-to-suggested");
    expect(screen.getByText(/Acceptance 90.0%/)).toBeInTheDocument();
    expect(screen.getByText(/attributed sample 20/)).toBeInTheDocument();
    expect(screen.getByText(/threshold 80.0%/)).toBeInTheDocument();
    expect(screen.getByTestId("autonomy-ready-flip-capture-to-suggested")).toBeInTheDocument();
    expect(screen.getByTestId("autonomy-ready-dismiss-capture-to-suggested")).toBeInTheDocument();
  });
});
