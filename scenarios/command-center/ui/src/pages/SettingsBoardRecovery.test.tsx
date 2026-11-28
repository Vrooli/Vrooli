import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import SettingsPage from "./SettingsPage";

const store = vi.hoisted(() => {
  const initial = () => ({ cycleSeconds: 60, transition: "crossfade", rooms: [{ id: "mission-control", enabled: true }] });
  let saved = initial();
  let settingsFailures = 0;
  let settingsErrorCode = "";
  let saveFails = false;
  return {
    reset() { saved = initial(); settingsFailures = 0; settingsErrorCode = ""; saveFails = false; },
    failSettings(count: number, code = "") { settingsFailures = count; settingsErrorCode = code; },
    failSave() { saveFails = true; },
    fetchCatalogs: vi.fn(() => Promise.reject(new Error("catalogs unavailable"))),
    fetchBoardSettings: vi.fn(() => {
      if (settingsFailures > 0) {
        settingsFailures -= 1;
        return Promise.reject(Object.assign(new Error("settings unavailable"), { code: settingsErrorCode }));
      }
      const current = structuredClone(saved);
      if (!current.rooms.length) current.rooms = [{ id: "mission-control", enabled: true }];
      return Promise.resolve(current);
    }),
    saveBoardSettings: vi.fn((next: typeof saved) => {
      if (saveFails) return Promise.reject(new Error("disk is read-only"));
      saved = structuredClone(next);
      return Promise.resolve(structuredClone(saved));
    }),
  };
});

vi.mock("../lib/api", () => ({
  DEFAULT_BOARD_SETTINGS: { cycleSeconds: 60, transition: "crossfade", rooms: [] },
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
    expect(screen.queryByRole("button", { name: "Restore defaults" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Retry" }));
    await screen.findByTestId("settings-board");
    expect(screen.getByLabelText("Transition")).toHaveValue("crossfade");
    expect(store.fetchBoardSettings).toHaveBeenCalledTimes(2);
  });

  it("requires confirmation, restores default settings, and enables configured rooms", async () => {
    store.failSettings(1, "board_settings_invalid");
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
    const user = userEvent.setup();
    renderCustomizer();
    await openBoardSettings(user);
    expect(await screen.findByRole("alert")).toHaveTextContent("settings unavailable");
    await user.click(screen.getByRole("button", { name: "Restore defaults" }));
    expect(confirm).toHaveBeenCalledWith(expect.stringContaining("60-second crossfade"));
    await screen.findByTestId("settings-board");
    expect(await screen.findByRole("status")).toHaveTextContent("Defaults restored. All configured rooms are back in rotation.");
    expect(store.saveBoardSettings).toHaveBeenCalledWith({ cycleSeconds: 60, transition: "crossfade", rooms: [] });
    expect(screen.getByLabelText("mission-control")).toBeChecked();
    confirm.mockRestore();
  });

  it("does not replace corrupt settings when confirmation is declined", async () => {
    store.failSettings(1, "board_settings_invalid");
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
    const user = userEvent.setup();
    renderCustomizer();
    await openBoardSettings(user);
    await screen.findByRole("alert");
    await user.click(screen.getByRole("button", { name: "Restore defaults" }));
    expect(store.saveBoardSettings).not.toHaveBeenCalled();
    confirm.mockRestore();
  });

  it("keeps the corruption alert and reports a failed default write", async () => {
    store.failSettings(1, "board_settings_invalid");
    store.failSave();
    const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
    const user = userEvent.setup();
    renderCustomizer();
    await openBoardSettings(user);
    await screen.findByRole("alert");
    await user.click(screen.getByRole("button", { name: "Restore defaults" }));
    expect(await screen.findByText(/Saved board settings were not replaced: disk is read-only/)).toBeInTheDocument();
    expect(screen.queryByText("Defaults restored. All configured rooms are back in rotation.")).not.toBeInTheDocument();
    confirm.mockRestore();
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
