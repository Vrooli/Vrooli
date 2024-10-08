import { renderHook } from "@testing-library/react";
import { act } from "react";
import { useStepMetrics } from "./RunView";

// Mock the Date.now() function
const mockDateNow = jest.spyOn(Date, "now");

// Helper function to simulate passage of time
function advanceTime(ms: number) {
    mockDateNow.mockReturnValue(Date.now() + ms);
    jest.advanceTimersByTime(ms);
}

describe("useStepMetrics", () => {
    beforeEach(() => {
        jest.useFakeTimers();
        mockDateNow.mockReturnValue(0);
        Object.defineProperty(document, "hidden", { value: false, writable: true });
        jest.spyOn(document, "hasFocus").mockReturnValue(true);
    });

    afterEach(() => {
        jest.useRealTimers();
        jest.restoreAllMocks();
    });

    test("initializes with zero elapsed time and context switches", () => {
        const { result } = renderHook(() => useStepMetrics([], null));
        expect(result.current.getElapsedTime()).toBe(0);
        expect(result.current.getContextSwitches()).toBe(0);
    });

    test("initializes with provided step data", () => {
        const initialData = { contextSwitches: 5, elapsedTime: 1000 };
        const { result } = renderHook(() => useStepMetrics([1], initialData));
        expect(result.current.getElapsedTime()).toBe(1000);
        expect(result.current.getContextSwitches()).toBe(5);
    });

    test("updates elapsed time while tab is focused", () => {
        const { result } = renderHook(() => useStepMetrics([1, 4], null));
        advanceTime(5000);
        expect(result.current.getElapsedTime()).toBe(5000);
    });

    test("does not update elapsed time while tab is not focused", () => {
        const { result } = renderHook(() => useStepMetrics([], null));
        act(() => {
            Object.defineProperty(document, "hidden", { value: true });
            document.dispatchEvent(new Event("visibilitychange"));
        });
        advanceTime(5000);
        expect(result.current.getElapsedTime()).toBe(0);
    });

    test("resumes updating elapsed time when tab regains focus", () => {
        const { result } = renderHook(() => useStepMetrics([4, 3], null));
        act(() => {
            Object.defineProperty(document, "hidden", { value: true });
            document.dispatchEvent(new Event("visibilitychange"));
        });
        advanceTime(5000);
        act(() => {
            Object.defineProperty(document, "hidden", { value: false });
            document.dispatchEvent(new Event("visibilitychange"));
        });
        advanceTime(3000);
        expect(result.current.getElapsedTime()).toBe(3000);
    });

    test("increments context switches when tab gains focus", () => {
        const { result } = renderHook(() => useStepMetrics([2], {}));
        act(() => {
            Object.defineProperty(document, "hidden", { value: true });
            document.dispatchEvent(new Event("visibilitychange"));
        });
        act(() => {
            Object.defineProperty(document, "hidden", { value: false });
            document.dispatchEvent(new Event("visibilitychange"));
        });
        expect(result.current.getContextSwitches()).toBe(1);
    });

    test("handles multiple focus changes correctly", () => {
        const { result } = renderHook(() => useStepMetrics([], null));
        advanceTime(2000);
        act(() => {
            Object.defineProperty(document, "hidden", { value: true });
            document.dispatchEvent(new Event("visibilitychange"));
        });
        advanceTime(3000);
        act(() => {
            Object.defineProperty(document, "hidden", { value: false });
            document.dispatchEvent(new Event("visibilitychange"));
        });
        advanceTime(4000);
        expect(result.current.getElapsedTime()).toBe(6000);
        expect(result.current.getContextSwitches()).toBe(1);
    });

    test("updates elapsed time before unload", () => {
        const { result } = renderHook(() => useStepMetrics([7, 7, 7, 7], null));
        advanceTime(5000);
        act(() => {
            window.dispatchEvent(new Event("beforeunload"));
        });
        expect(result.current.getElapsedTime()).toBe(5000);
    });

    test("does not update context switches when tab loses focus", () => {
        const { result } = renderHook(() => useStepMetrics([], {}));
        act(() => {
            Object.defineProperty(document, "hidden", { value: true });
            document.dispatchEvent(new Event("visibilitychange"));
        });
        expect(result.current.getContextSwitches()).toBe(0);
    });

    test("handles rapid focus changes correctly", () => {
        const { result } = renderHook(() => useStepMetrics([], null));
        for (let i = 0; i < 5; i++) {
            act(() => {
                Object.defineProperty(document, "hidden", { value: true });
                document.dispatchEvent(new Event("visibilitychange"));
            });
            advanceTime(100);
            act(() => {
                Object.defineProperty(document, "hidden", { value: false });
                document.dispatchEvent(new Event("visibilitychange"));
            });
            advanceTime(100);
        }
        expect(result.current.getElapsedTime()).toBe(500);
        expect(result.current.getContextSwitches()).toBe(5);
    });

    test("restarts metrics when the location array changes", () => {
        const { result, rerender } = renderHook(
            ({ steps }) => useStepMetrics(steps, null),
            { initialProps: { steps: [1, 2, 3] } },
        );
        advanceTime(2000);
        rerender({ steps: [4, 5, 6] });
        expect(result.current.getElapsedTime()).toBe(0);
        expect(result.current.getContextSwitches()).toBe(0);
    });
});
