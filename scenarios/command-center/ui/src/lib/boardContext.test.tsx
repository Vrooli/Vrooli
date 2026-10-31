import { MemoryRouter, useLocation } from "react-router-dom";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { BoardController } from "../components/BoardController";
import { renderWithProviders } from "../test-utils/renderWithProviders";
vi.mock("./api", () => ({ fetchBoard: async () => ({ rooms: [{ id: "hive", title: "The Hive" }] }), fetchBoardSettings: async () => ({ cycleSeconds: 60, transition: "crossfade", rooms: [] }) }));
import { renderHook, act, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { parseSamples, useBoardController, useBoardProgress } from "./boardContext";

describe("board context", () => {
  it("does not turn a settings room selection into a live-room navigation", async () => {
    function LocationProbe() {
      return <output data-testid="location">{useLocation().pathname}</output>;
    }
    const controller = initSpatialNav({ getGamepads: () => [], isVisible: () => true });
    try {
      renderWithProviders(<MemoryRouter initialEntries={["/settings?destination=rooms&room=hive"]}>
        <BoardController><LocationProbe /></BoardController>
      </MemoryRouter>);
      expect(await screen.findByTestId("location")).toHaveTextContent("/settings");
    } finally {
      controller.dispose();
    }
  });

  it("defaults the audience mode to mark, so a screenshot carries its own legend", () => {
    expect(parseSamples(null)).toBe("mark");
    expect(parseSamples("nonsense")).toBe("mark");
    expect(parseSamples("hide")).toBe("hide");
    expect(parseSamples("full")).toBe("full");
  });
  it("refuses to run outside the controller", () => {
    expect(() => renderHook(() => useBoardController())).toThrow(/inside BoardController/);
  });
});

it("shares board gamepad input while leaving focused buttons selectable", async () => {
  vi.useFakeTimers();
  let buttonIndex = -1;
  const getGamepads = vi.fn(() => [{ id: 'board-controller', index: 0, connected: true,
    timestamp: 0, mapping: 'standard', axes: [0, 0],
    vibrationActuator: { type: 'dual-rumble', effects: [], playEffect: async () => 'complete', reset: async () => 'complete' },
    buttons: Array.from({ length: 17 }, (_, index) => ({ pressed: index === buttonIndex, touched: false, value: index === buttonIndex ? 1 : 0 })),
  } as Gamepad]);
  const controller = initSpatialNav({ getGamepads, isVisible: () => true });
  const clicked = vi.fn();
  function Controls() {
    const board = useBoardController();
    return <><output>{board.helpVisible ? 'Help open' : 'Help closed'}</output><button onClick={clicked}>Inspect reading</button></>;
  }
  const view = renderWithProviders(<MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
    <BoardController><Controls /></BoardController>
  </MemoryRouter>);
  try {
    await act(async () => { await vi.advanceTimersByTimeAsync(1); });
    window.dispatchEvent(new Event('gamepadconnected'));
    getGamepads.mockClear();
    act(() => { buttonIndex = 9; vi.advanceTimersByTime(16); });
    expect(screen.getByText('Help open')).toBeInTheDocument();
    expect(getGamepads).toHaveBeenCalledTimes(1);
    screen.getByText('Inspect reading').focus();
    act(() => { buttonIndex = -1; vi.advanceTimersByTime(16); });
    act(() => { buttonIndex = 0; vi.advanceTimersByTime(16); });
    expect(clicked).toHaveBeenCalledOnce();
  } finally {
    view.unmount();
    controller.dispose();
    vi.useRealTimers();
  }
});

it("advances the cycle rail without re-rendering the room surfaces", async () => {
  vi.useFakeTimers();
  const controller = initSpatialNav({ getGamepads: () => [], isVisible: () => true });
  let surfaceRenders = 0;
  let railProgress = 0;
  function Surface() {
    useBoardController();
    surfaceRenders += 1;
    return null;
  }
  function Rail() {
    railProgress = useBoardProgress().progress;
    return null;
  }
  const view = renderWithProviders(<MemoryRouter initialEntries={["/mission-control"]} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
    <BoardController><Surface /><Rail /></BoardController>
  </MemoryRouter>);
  try {
    await act(async () => { await vi.advanceTimersByTimeAsync(1); });
    const settledRenders = surfaceRenders;
    // Three seconds of animation frames.
    await act(async () => { await vi.advanceTimersByTimeAsync(3_000); });
    expect(railProgress).toBeGreaterThan(0);
    expect(surfaceRenders).toBe(settledRenders);
  } finally {
    view.unmount();
    controller.dispose();
    vi.useRealTimers();
  }
});
