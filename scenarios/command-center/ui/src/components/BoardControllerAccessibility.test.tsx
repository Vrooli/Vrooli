import { act, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { describe, expect, it, vi } from "vitest";
import { BoardController } from "./BoardController";
import { renderWithProviders } from "../test-utils/renderWithProviders";

const { fetchBoardMock, fetchBoardSettingsMock } = vi.hoisted(() => ({
  fetchBoardMock: vi.fn(),
  fetchBoardSettingsMock: vi.fn(),
}));
vi.mock("../lib/api", () => ({ fetchBoard: fetchBoardMock, fetchBoardSettings: fetchBoardSettingsMock }));

describe("BoardController keyboard surface", () => {
  it("keeps operator pages out of the board gamepad tab order", async () => {
    fetchBoardMock.mockResolvedValue({ rooms: [{ id: "forge", title: "The Forge" }] });
    fetchBoardSettingsMock.mockResolvedValue({ cycleSeconds: 90, transition: "cut", rooms: [] });
    const spatial = initSpatialNav({ getGamepads: () => [], isVisible: () => true });
    const view = renderWithProviders(
      <MemoryRouter initialEntries={["/settings?destination=board"]}>
        <BoardController><main aria-label="Board settings">Settings content</main></BoardController>
      </MemoryRouter>,
    );

    try {
      await screen.findByRole("main", { name: "Board settings" });
      const root = view.container.querySelector("[data-board-root]");
      expect(root).toHaveAttribute("tabindex", "-1");
      expect(root).not.toHaveAttribute("aria-label", "Board controls");
      expect(document.activeElement).not.toBe(root);
    } finally {
      view.unmount();
      act(() => spatial.dispose());
    }
  });

  it("retains a focusable board control surface for room routes", async () => {
    fetchBoardMock.mockResolvedValue({ rooms: [{ id: "forge", title: "The Forge" }] });
    fetchBoardSettingsMock.mockResolvedValue({ cycleSeconds: 90, transition: "cut", rooms: [] });
    const spatial = initSpatialNav({ getGamepads: () => [], isVisible: () => true });
    const view = renderWithProviders(
      <MemoryRouter initialEntries={["/forge"]}>
        <BoardController><main>Room content</main></BoardController>
      </MemoryRouter>,
    );

    try {
      const root = await screen.findByLabelText("Board controls");
      expect(root).toHaveAttribute("tabindex", "0");
      expect(document.activeElement).toBe(root);
    } finally {
      view.unmount();
      act(() => spatial.dispose());
    }
  });
});
