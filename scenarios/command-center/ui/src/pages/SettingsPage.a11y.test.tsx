import { describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { screen } from "@testing-library/react";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import SettingsPage from "./SettingsPage";

vi.mock("../lib/api", () => ({
  fetchCatalogs: () => Promise.resolve({ rooms: [], themes: [], compositions: [], signals: [], connectors: [], readouts: [] }),
  fetchBoardSettings: () => Promise.resolve({ cycleSeconds: 60, transition: "crossfade", rooms: [] }),
  saveBoardSettings: (value: unknown) => Promise.resolve(value),
  saveCatalogEntry: (value: unknown) => Promise.resolve(value),
  deleteCatalogEntry: () => Promise.resolve(undefined),
}));

describe("SettingsPage accessibility", () => {
  it("has no axe violations on the destination home", async () => {
    const { container } = renderWithProviders(<MemoryRouter initialEntries={["/settings"]}><SettingsPage /></MemoryRouter>);
    await screen.findByText("Shape the board");
    await expectNoA11yViolations(container);
  });

  it("keeps board settings controls named and grouped at their destination", async () => {
    const { container } = renderWithProviders(<MemoryRouter initialEntries={["/settings?destination=board"]}><SettingsPage /></MemoryRouter>);
    await screen.findByRole("slider", { name: "Default dwell" });
    expect(screen.getByRole("combobox", { name: "Transition" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save board settings" })).toBeInTheDocument();
    await expectNoA11yViolations(container);
  });
});
