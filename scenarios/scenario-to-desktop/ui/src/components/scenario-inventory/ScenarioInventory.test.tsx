import { fireEvent, render, screen, waitFor } from "@/test-utils";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ScenarioInventory } from "./ScenarioInventory";

const mocks = vi.hoisted(() => ({ fetchScenarioDesktopStatus: vi.fn() }));

vi.mock("../../lib/api", () => ({
  fetchScenarioDesktopStatus: mocks.fetchScenarioDesktopStatus,
}));
vi.mock("./ScenarioCard", () => ({
  ScenarioCard: ({ scenario, onSelect }: { scenario: { name: string }; onSelect: (scenario: { name: string }) => void }) => (
    <button type="button" onClick={() => { onSelect(scenario); }}>
      Open {scenario.name}
    </button>
  ),
}));

const response = {
  scenarios: [
    { name: "canvas-lab", display_name: "Canvas Lab", has_desktop: true },
    { name: "bridge", display_name: "Bridge", has_desktop: false },
  ],
  stats: { total: 2, with_desktop: 1, built: 1, web_only: 1 },
};

describe("ScenarioInventory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows a loading state and the server error", async () => {
    mocks.fetchScenarioDesktopStatus.mockRejectedValue(new Error("catalog unavailable"));
    render(<ScenarioInventory />);
    expect(screen.getByText("Loading scenarios...")).toBeInTheDocument();
    expect(await screen.findByText("Failed to load scenarios")).toBeInTheDocument();
    expect(screen.getByText("catalog unavailable")).toBeInTheDocument();
  });

  it("reports statistics, filters scenarios, and launches the selected one", async () => {
    mocks.fetchScenarioDesktopStatus.mockResolvedValue(response);
    const onScenarioLaunch = vi.fn();
    render(<ScenarioInventory onScenarioLaunch={onScenarioLaunch} />);

    expect(await screen.findByText("Total Scenarios")).toBeInTheDocument();
    expect(screen.getByText("With Desktop")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Desktop" }));
    expect(screen.getByRole("button", { name: "Open canvas-lab" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Open bridge" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Open canvas-lab" }));
    expect(onScenarioLaunch).toHaveBeenCalledWith(response.scenarios[0]);
  });

  it("searches by display name and explains an empty filtered result", async () => {
    mocks.fetchScenarioDesktopStatus.mockResolvedValue(response);
    render(<ScenarioInventory />);
    await screen.findByRole("button", { name: "Open canvas-lab" });

    fireEvent.change(screen.getByRole("textbox", { name: "Search scenarios" }), {
      target: { value: "Bridge" },
    });
    expect(screen.getByRole("button", { name: "Open bridge" })).toBeInTheDocument();
    fireEvent.change(screen.getByRole("textbox", { name: "Search scenarios" }), {
      target: { value: "missing" },
    });
    await waitFor(() => {
      expect(screen.getByText("No scenarios found matching your filters")).toBeInTheDocument();
    });
  });
});
