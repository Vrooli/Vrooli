import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import SettingsPage from "./SettingsPage";

const store = vi.hoisted(() => {
  const saved = { cycleSeconds: 60, transition: "crossfade", rooms: [{ id: "mission-control", enabled: true }] };
  let settingsFailures = 0;
  let saveFails = false;
  return {
    reset() { settingsFailures = 0; saveFails = false; },
    failSettings(count: number) { settingsFailures = count; },
    failSave() { saveFails = true; },
    fetchCatalogs: vi.fn(() => Promise.reject(new Error("catalogs unavailable"))),
    fetchBoardSettings: vi.fn(() => {
      if (settingsFailures > 0) { settingsFailures -= 1; return Promise.reject(new Error("settings unavailable")); }
      return Promise.resolve(structuredClone(saved));
    }),
    saveBoardSettings: vi.fn(() => saveFails ? Promise.reject(new Error("disk is read-only")) : Promise.resolve()),
  };
});

vi.mock("../lib/api", () => ({
  fetchCatalogs: store.fetchCatalogs,
  fetchBoardSettings: store.fetchBoardSettings,
  saveBoardSettings: store.saveBoardSettings,
  saveCatalogEntry: vi.fn(),
  deleteCatalogEntry: vi.fn(),
}));

const renderCustomizer = () => renderWithProviders(<MemoryRouter initialEntries={["/settings"]}><SettingsPage /></MemoryRouter>);
const openBoardSettings = async (user: ReturnType<typeof userEvent.setup>) => {
  await user.click(await screen.findByRole("button", { name: /Board settings/ }));
};

describe("board customization recovery", () => {
  beforeEach(() => { store.reset(); vi.clearAllMocks(); });

  it("loads saved board settings without waiting for unrelated catalogs", async () => {
    const user = userEvent.setup();
    renderCustomizer();
    await openBoardSettings(user);
    await screen.findByTestId("settings-board");
    expect(screen.getByLabelText("Transition")).toHaveValue("crossfade");
    expect(screen.getByLabelText("mission-control")).toBeInTheDocument();
    expect(store.fetchCatalogs).toHaveBeenCalledTimes(1);
  });

  it("shows a retry when the saved settings read fails, then recovers", async () => {
    store.failSettings(1);
    const user = userEvent.setup();
    renderCustomizer();
    await openBoardSettings(user);
    expect(await screen.findByRole("alert")).toHaveTextContent("settings unavailable");
    await user.click(screen.getByRole("button", { name: "Retry" }));
    await screen.findByTestId("settings-board");
    expect(screen.getByLabelText("Transition")).toHaveValue("crossfade");
    expect(store.fetchBoardSettings).toHaveBeenCalledTimes(2);
  });

  it("keeps the draft and explains a failed save", async () => {
    store.failSave();
    const user = userEvent.setup();
    renderCustomizer();
    await openBoardSettings(user);
    await screen.findByTestId("settings-board");
    await user.selectOptions(screen.getByLabelText("Transition"), "cut");
    await user.click(screen.getByRole("button", { name: "Save board settings" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("disk is read-only");
    expect(screen.getByLabelText("Transition")).toHaveValue("cut");
    await waitFor(() => expect(store.saveBoardSettings).toHaveBeenCalledTimes(1));
  });
});
