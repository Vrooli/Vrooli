import { MemoryRouter } from "react-router-dom";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { BoardController } from "./BoardController";
import { AmbientShell } from "./AmbientShell";
import { renderWithProviders } from "../test-utils/renderWithProviders";
import { useBoardController } from "../lib/boardContext";

const api = vi.hoisted(() => ({ fetchBoard: vi.fn(), fetchBoardSettings: vi.fn() }));
vi.mock("../lib/api", () => ({ fetchBoard: api.fetchBoard, fetchBoardSettings: api.fetchBoardSettings }));

const rooms = [
  { id: "mission-control", title: "Mission Control" },
  { id: "hive", title: "The Hive" },
  { id: "forge", title: "The Forge" },
  { id: "ledger", title: "Ledger" },
  { id: "broadcast", title: "Broadcast" },
  { id: "panorama", title: "Panorama" },
];

function NavigateToHive() {
  const board = useBoardController();
  return <button type="button" onClick={() => board.goTo("/hive")}>Navigate to The Hive</button>;
}

describe("ambient room navigation", () => {
  beforeEach(() => {
    api.fetchBoard.mockResolvedValue({ rooms });
    api.fetchBoardSettings.mockResolvedValue({ cycleSeconds: 60, transition: "cut", rooms: rooms.map(({ id }) => ({ id, enabled: true })) });
  });

  it("exposes every room destination as a native button in one named group", async () => {
    const spatial = initSpatialNav({ getGamepads: () => [], isVisible: () => true });
    try {
      const view = renderWithProviders(<MemoryRouter initialEntries={["/mission-control"]}>
        <BoardController><AmbientShell theme="ground-control" title="Mission Control" position="ROOM 1 OF 6"><NavigateToHive /></AmbientShell></BoardController>
      </MemoryRouter>);
      await waitFor(() => expect(view.container.querySelectorAll(".cc-room-dot")).toHaveLength(6));
      const group = view.container.querySelector<HTMLElement>('[aria-label="Go to room"]');
      expect(group).toHaveAttribute("role", "group");
      expect(view.container.querySelector('button[aria-label="Mission Control"]')).toHaveAttribute("aria-current", "page");
      for (const room of rooms) {
        const button = view.container.querySelector<HTMLElement>(`button[aria-label="${room.title}"]`);
        expect(button).toBeInTheDocument();
        expect(button).toHaveAttribute("type", "button");
        expect(button).not.toHaveAttribute("role");
        expect(group).toContainElement(button);
      }
      expect(view.container.querySelector('nav[aria-label="Board controls"]')).toHaveAttribute("aria-hidden", "true");
      const announcement = screen.getByTestId("room-announcement");
      expect(announcement).toHaveAttribute("role", "status");
      expect(announcement).toHaveAttribute("aria-live", "polite");
      expect(announcement).toBeEmptyDOMElement();
      await userEvent.setup().click(screen.getByRole("button", { name: "Navigate to The Hive" }));
      await waitFor(() => expect(announcement).toHaveTextContent("The Hive, room 2 of 6"));
      expect(view.container.querySelector('button[aria-label="The Hive"]')).toHaveAttribute("aria-current", "page");
    } finally {
      spatial.dispose();
    }
  });
});
