import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useLongPress } from "../hooks/useLongPress";
describe("useLongPress", () => {
    const onPress = vi.fn();
    const onLongPress = vi.fn();
    beforeEach(() => {
        vi.clearAllMocks();
        vi.useFakeTimers();
    });
    afterEach(() => {
        vi.useRealTimers();
    });
    const makePointerEvent = (overrides) => ({
        pointerType: "mouse",
        button: 0,
        preventDefault: vi.fn(),
        ...overrides,
    });
    it("fires onPress on short click (pointerDown then pointerUp quickly)", () => {
        const { result } = renderHook(() => useLongPress({ onPress, onLongPress }));
        act(() => result.current.onPointerDown(makePointerEvent()));
        act(() => {
            vi.advanceTimersByTime(100);
        });
        act(() => result.current.onPointerUp());
        expect(onPress).toHaveBeenCalledOnce();
        expect(onLongPress).not.toHaveBeenCalled();
    });
    it("fires onLongPress after 500ms hold", () => {
        const { result } = renderHook(() => useLongPress({ onPress, onLongPress }));
        act(() => result.current.onPointerDown(makePointerEvent()));
        act(() => {
            vi.advanceTimersByTime(500);
        });
        expect(onLongPress).toHaveBeenCalledOnce();
        expect(onPress).not.toHaveBeenCalled();
        // pointerUp after long press should not fire onPress
        act(() => result.current.onPointerUp());
        expect(onPress).not.toHaveBeenCalled();
    });
    it("fires onLongPress on right-click via contextMenu", () => {
        const { result } = renderHook(() => useLongPress({ onPress, onLongPress }));
        const event = { preventDefault: vi.fn() };
        act(() => result.current.onContextMenu(event));
        expect(onLongPress).toHaveBeenCalledOnce();
        expect(event.preventDefault).toHaveBeenCalled();
        expect(onPress).not.toHaveBeenCalled();
    });
    it("fires neither on pointerCancel", () => {
        const { result } = renderHook(() => useLongPress({ onPress, onLongPress }));
        act(() => result.current.onPointerDown(makePointerEvent()));
        act(() => {
            vi.advanceTimersByTime(100);
        });
        act(() => result.current.onPointerCancel());
        expect(onPress).not.toHaveBeenCalled();
        expect(onLongPress).not.toHaveBeenCalled();
        // Advancing past threshold should also not fire since we cancelled
        act(() => {
            vi.advanceTimersByTime(500);
        });
        expect(onLongPress).not.toHaveBeenCalled();
    });
    it("ignores non-primary mouse buttons on pointerDown", () => {
        const { result } = renderHook(() => useLongPress({ onPress, onLongPress }));
        act(() => result.current.onPointerDown(makePointerEvent({ button: 2 })));
        act(() => result.current.onPointerUp());
        // Neither should fire since button 2 was ignored
        expect(onPress).not.toHaveBeenCalled();
        expect(onLongPress).not.toHaveBeenCalled();
    });
});
