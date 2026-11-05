import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { autoDrainService } from "../../services/auto-drain-service";
import { GoalDrainToggle } from "./GoalDrainToggle";

describe("GoalDrainToggle", () => {
  afterEach(() => vi.restoreAllMocks());

  it("renders the default-OFF section and turns it on through the service", async () => {
    vi.spyOn(autoDrainService, "get").mockResolvedValue({ enabled: false });
    const setSpy = vi.spyOn(autoDrainService, "set").mockResolvedValue({ enabled: true });

    renderWithProviders(<GoalDrainToggle />);

    expect(await screen.findByTestId("settings-goal-drain")).toBeInTheDocument();
    // Off is the visually-selected option by default (cyan class).
    const offBtn = screen.getByRole("button", { name: "Off" });
    await waitFor(() => expect(offBtn.className).toContain("text-cyan-400"));

    await userEvent.click(screen.getByRole("button", { name: "On" }));
    await waitFor(() => expect(setSpy).toHaveBeenCalledWith(true));
  });

  it("reflects a persisted ON state and can switch back off", async () => {
    vi.spyOn(autoDrainService, "get").mockResolvedValue({ enabled: true });
    const setSpy = vi.spyOn(autoDrainService, "set").mockResolvedValue({ enabled: false });
    renderWithProviders(<GoalDrainToggle />);

    const onBtn = await screen.findByRole("button", { name: "On" });
    await waitFor(() => expect(onBtn.className).toContain("text-cyan-400"));

    await userEvent.click(screen.getByRole("button", { name: "Off" }));
    await waitFor(() => expect(setSpy).toHaveBeenCalledWith(false));
  });
});
