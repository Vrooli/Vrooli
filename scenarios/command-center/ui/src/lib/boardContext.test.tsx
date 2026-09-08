import { MemoryRouter } from "react-router-dom";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { BoardController } from "../components/BoardController";
import { renderWithProviders } from "../test-utils/renderWithProviders";
vi.mock("./api", () => ({ fetchBoard: async () => ({ rooms: [] }) }));
import { renderHook, act, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { parseSamples, useBoardController } from "./boardContext";

describe("board context", () => {
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
