import { renderHook } from "@testing-library/react";
import { expect } from "chai";
import { useHotkeys } from "./useHotkeys.js";

describe("useHotkeys", () => {
    let mockCallback;

    beforeEach(() => {
        mockCallback = jest.fn();
    });

    afterEach(() => {
        jest.clearAllMocks();
    });

    const scenarios = [
        // Single Modifiers with 'a'
        { keys: ["a"], ctrlKey: true, shiftKey: false, altKey: false },
        { keys: ["a"], ctrlKey: false, shiftKey: true, altKey: false },
        { keys: ["a"], ctrlKey: false, shiftKey: false, altKey: true },
        // Double Modifiers with 'a'
        { keys: ["a"], ctrlKey: true, shiftKey: true, altKey: false },
        { keys: ["a"], ctrlKey: true, shiftKey: false, altKey: true },
        { keys: ["a"], ctrlKey: false, shiftKey: true, altKey: true },
        // Triple Modifiers with 'a'
        { keys: ["a"], ctrlKey: true, shiftKey: true, altKey: true },
        // Single Modifiers with 'b'
        { keys: ["b"], ctrlKey: true, shiftKey: false, altKey: false },
        { keys: ["b"], ctrlKey: false, shiftKey: true, altKey: false },
        { keys: ["b"], ctrlKey: false, shiftKey: false, altKey: true },
        // Double Modifiers with 'b'
        { keys: ["b"], ctrlKey: true, shiftKey: true, altKey: false },
        { keys: ["b"], ctrlKey: true, shiftKey: false, altKey: true },
        { keys: ["b"], ctrlKey: false, shiftKey: true, altKey: true },
        // Triple Modifiers with 'b'
        { keys: ["b"], ctrlKey: true, shiftKey: true, altKey: true },
        // Different keys without modifiers
        { keys: ["Enter"], ctrlKey: false, shiftKey: false, altKey: false },
        { keys: ["Escape"], ctrlKey: false, shiftKey: false, altKey: false },
        // Different keys with single modifier
        { keys: ["Enter"], ctrlKey: true, shiftKey: false, altKey: false },
        { keys: ["Escape"], ctrlKey: false, shiftKey: true, altKey: false },
        // Different keys with multiple modifiers
        { keys: ["Enter"], ctrlKey: true, shiftKey: true, altKey: false },
        { keys: ["Escape"], ctrlKey: false, shiftKey: true, altKey: true },
    ];

    scenarios.forEach(({ keys, ctrlKey, shiftKey, altKey }) => {
        it(`calls callback for key "${keys[0]}" with ctrlKey: ${ctrlKey}, shiftKey: ${shiftKey}, and altKey: ${altKey}`, () => {
            const hotkeys = [{ keys, ctrlKey, shiftKey, altKey, callback: mockCallback }];

            renderHook(() => useHotkeys(hotkeys, true));

            document.dispatchEvent(new KeyboardEvent("keydown", { key: keys[0], ctrlKey, shiftKey, altKey }));

            expect(mockCallback).toHaveBeenCalledTimes(1);
        });

        it(`does not call callback for incorrect key with ctrlKey: ${ctrlKey}, shiftKey: ${shiftKey}, and altKey: ${altKey}`, () => {
            const hotkeys = [{ keys, ctrlKey, shiftKey, altKey, callback: mockCallback }];

            renderHook(() => useHotkeys(hotkeys, true));

            // Dispatch with a different key
            document.dispatchEvent(new KeyboardEvent("keydown", { key: "wrongKey", ctrlKey, shiftKey, altKey }));

            expect(mockCallback).not.toHaveBeenCalled();
        });
    });

    it("does not add event listener when condition is false", () => {
        const hotkeys = [{ keys: ["a"], ctrlKey: true, callback: mockCallback }];

        // Render the hook with the condition set to false
        renderHook(() => useHotkeys(hotkeys, false));

        // Attempt to trigger the hotkey
        document.dispatchEvent(new KeyboardEvent("keydown", { key: "a", ctrlKey: true }));

        // The callback should not have been called since the event listener should not have been added
        expect(mockCallback).not.toHaveBeenCalled();
    });

    it("attaches and detaches event listener from the specified targetRef", () => {
        // Create a mock element with mocked addEventListener and removeEventListener
        const mockElement = {
            addEventListener: jest.fn(),
            removeEventListener: jest.fn(),
        };
        const targetRef = { current: mockElement };
        const hotkeys = [{ keys: ["a"], callback: mockCallback }];

        const { unmount } = renderHook(() => useHotkeys(hotkeys, true, targetRef as any));

        expect(mockElement.addEventListener).toHaveBeenCalledWith("keydown", expect.any(Function));

        unmount();

        expect(mockElement.removeEventListener).toHaveBeenCalledWith("keydown", expect.any(Function));
    });
});
