import { beforeEach, describe, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import { screen, waitFor } from "@testing-library/react";
import { useQueryClient } from "@tanstack/react-query";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import userEvent from "@testing-library/user-event";
import { BoardController } from "../components/BoardController";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import SettingsPage from "./SettingsPage";

const store = vi.hoisted(() => {
  const initial = { cycleSeconds: 60, transition: "crossfade", rooms: [{ id: "mission-control", enabled: true }] };
  let saved = structuredClone(initial);
  let failNextRead = false;
  return {
    reset() { saved = structuredClone(initial); failNextRead = false; },
    failNextSettingsRead() { failNextRead = true; },
    fetchSettings: vi.fn(() => {
      if (failNextRead) { failNextRead = false; return Promise.reject(new Error("settings source unavailable")); }
      return Promise.resolve(structuredClone(saved));
    }),
    saveSettings: vi.fn((next: typeof initial) => {
      saved = structuredClone(next);
      return Promise.resolve(structuredClone(saved));
    }),
  };
});

vi.mock("../lib/api", () => ({
  fetchCatalogs: () => Promise.resolve({ rooms: [{ id: "mission-control", title: "Mission Control" }] }),
  fetchBoard: () => Promise.resolve({ rooms: [{ id: "mission-control", title: "Mission Control" }] }),
  fetchBoardSettings: store.fetchSettings,
  saveBoardSettings: store.saveSettings,
  saveCatalogEntry: () => Promise.resolve({ id: "unused" }),
  deleteCatalogEntry: () => Promise.resolve(),
}));

describe("saved board settings journey", () => {
  beforeEach(() => { store.reset(); vi.clearAllMocks(); });

  it("saves a user choice, reloads it in a fresh client, and keeps it on source failure", async () => {
    function RefreshSettings() {
      const queryClient = useQueryClient();
      return <button onClick={() => void queryClient.invalidateQueries({ queryKey: ["board-settings"] })}>Refresh saved settings</button>;
    }
    const user = userEvent.setup();
    const editor = renderWithProviders(<MemoryRouter initialEntries={["/settings?destination=board"]}><SettingsPage /></MemoryRouter>);
    await screen.findByTestId("settings-board");
    await user.selectOptions(screen.getByLabelText("Transition"), "cut");
    await user.click(screen.getByRole("button", { name: "Save board settings" }));
    await screen.findByText("Board settings saved.");
    expect(store.saveSettings).toHaveBeenCalledWith(expect.objectContaining({ transition: "cut" }));
    editor.unmount();

    const spatial = initSpatialNav({ getGamepads: () => [], isVisible: () => true });
    const controller = renderWithProviders(<MemoryRouter initialEntries={["/mission-control"]}><BoardController><RefreshSettings /></BoardController></MemoryRouter>);
    try {
      const board = await screen.findByLabelText("Board controls");
      await waitFor(() => expect(board).toHaveAttribute("data-board-transition", "cut"));
      expect(board).toHaveStyle({ "--motion-room": "0ms" });
      store.failNextSettingsRead();
      await user.click(screen.getByRole("button", { name: "Refresh saved settings" }));
      await waitFor(() => expect(store.fetchSettings).toHaveBeenCalledTimes(4));
      expect(board).toHaveAttribute("data-board-transition", "cut");
      expect(board).toHaveStyle({ "--motion-room": "0ms" });
    } finally {
      controller.unmount();
      spatial.dispose();
    }
  });
});
