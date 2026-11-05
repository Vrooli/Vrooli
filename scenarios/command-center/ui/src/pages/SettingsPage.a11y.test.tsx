import { describe, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { screen } from "@testing-library/react";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import SettingsPage from "./SettingsPage";

vi.mock("../lib/api", () => ({
  fetchCatalogs: async () => ({ rooms: [], themes: [], compositions: [], signals: [], connectors: [], readouts: [] }),
  fetchBoardSettings: async () => ({ cycleSeconds: 60, transition: "crossfade", rooms: [] }),
  saveBoardSettings: async (value: unknown) => value,
  saveCatalogEntry: async (value: unknown) => value,
  deleteCatalogEntry: async () => undefined,
}));

describe("SettingsPage accessibility", () => {
  it("has no axe violations on the destination home", async () => {
    const { container } = renderWithProviders(<MemoryRouter initialEntries={["/settings"]}><SettingsPage /></MemoryRouter>);
    await screen.findByText("Shape the board");
    await expectNoA11yViolations(container);
  });
});
