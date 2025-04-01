import { act, renderHook } from "@testing-library/react";
import { expect } from "chai";
import sinon from "sinon";
import { useUndoRedo } from "./useUndoRedo.js";

describe("useUndoRedo", () => {
    const initialValue = "initial";
    let onChange: sinon.SinonStub;
    let clock: sinon.SinonFakeTimers;

    beforeEach(() => {
        onChange = sinon.stub();
        // Use fake timers but don't interfere with native timers used by the test framework
        clock = sinon.useFakeTimers({
            shouldAdvanceTime: true,
            now: Date.now(),
        });
    });

    afterEach(() => {
        // Restore all fakes and stubs
        onChange.reset();
        if (clock) {
            // Restore the clock first
            clock.restore();
        }
        // Then restore all other sinon fakes/stubs
        sinon.restore();
    });

    // Add a describe-level after hook to ensure cleanup
    after(() => {
        // Ensure all timers are cleaned up
        if (clock) {
            clock.restore();
        }
        sinon.restore();
    });

    it("should initialize with the initial value", () => {
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        expect(result.current.internalValue).to.equal(initialValue);
        expect(result.current.canUndo).to.equal(false);
        expect(result.current.canRedo).to.equal(false);
    });

    it("should update value and add to stack after debounce", async () => {
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        // Change the value
        act(() => {
            result.current.changeInternalValue("new value");
        });

        // Value should update immediately
        expect(result.current.internalValue).to.equal("new value");
        expect(result.current.canUndo).to.equal(false); // Not yet added to stack

        // Fast forward past debounce time
        act(() => {
            clock.tick(200);
        });

        // Now should be in stack
        expect(result.current.canUndo).to.equal(true);
        expect(result.current.canRedo).to.equal(false);
    });

    it("should handle undo/redo operations", async () => {
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
                delimiters: [], // Disable debouncing for this test
            }),
        );

        // Make some changes
        act(() => {
            result.current.changeInternalValue("first change");
        });

        act(() => {
            result.current.changeInternalValue("second change");
        });

        // Test undo
        act(() => {
            result.current.undo();
        });
        expect(result.current.internalValue).to.equal("first change");
        expect(result.current.canUndo).to.equal(true);
        expect(result.current.canRedo).to.equal(true);

        // Test redo
        act(() => {
            result.current.redo();
        });
        expect(result.current.internalValue).to.equal("second change");
        expect(result.current.canUndo).to.equal(true);
        expect(result.current.canRedo).to.equal(false);
    });

    it("should handle immediate stack addition with delimiters", () => {
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
                delimiters: [" ", "\n"],
            }),
        );

        // Type without delimiter
        act(() => {
            result.current.changeInternalValue("typing");
        });
        expect(result.current.canUndo).to.equal(false); // Not yet in stack

        // Type with space delimiter
        act(() => {
            result.current.changeInternalValue("typing ");
        });
        expect(result.current.canUndo).to.equal(true); // Immediately added to stack

        // Type with newline delimiter
        act(() => {
            result.current.changeInternalValue("typing \nmore");
        });
        expect(result.current.canUndo).to.equal(true);
    });

    it("should disable debouncing with empty delimiters array", () => {
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
                delimiters: [],
            }),
        );

        // Every change should be added immediately
        act(() => {
            result.current.changeInternalValue("change 1");
        });
        expect(result.current.canUndo).to.equal(true);

        act(() => {
            result.current.changeInternalValue("change 2");
        });
        expect(result.current.canUndo).to.equal(true);
    });

    it("should handle resetInternalValue without adding to stack", () => {
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        // Make a change that will be added to stack
        act(() => {
            result.current.changeInternalValue("change");
            clock.tick(200);
        });
        expect(result.current.canUndo).to.equal(true);

        // Reset value
        act(() => {
            result.current.resetInternalValue("reset value");
        });
        expect(result.current.internalValue).to.equal("reset value");
        // Stack state should not change
        expect(result.current.canUndo).to.equal(true);
    });

    it("should handle resetStack", () => {
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        // Make some changes
        act(() => {
            result.current.changeInternalValue("change 1");
            clock.tick(200);
            result.current.changeInternalValue("change 2");
            clock.tick(200);
        });
        expect(result.current.canUndo).to.equal(true);

        // Reset stack
        act(() => {
            result.current.resetStack();
        });
        expect(result.current.canUndo).to.equal(false);
        expect(result.current.canRedo).to.equal(false);
        expect(result.current.internalValue).to.equal("change 2");
    });

    it("should respect disabled flag", () => {
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
                disabled: true,
            }),
        );

        // Make changes
        act(() => {
            result.current.changeInternalValue("change");
            clock.tick(200);
        });

        // Try undo/redo
        act(() => {
            result.current.undo();
        });
        expect(result.current.internalValue).to.equal("change"); // Should not change

        act(() => {
            result.current.redo();
        });
        expect(result.current.internalValue).to.equal("change"); // Should not change
    });

    it("should handle stack size limit", () => {
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue: "x".repeat(500000), // Half of max size
                onChange,
                delimiters: [], // Disable debouncing for this test
            }),
        );

        // Add another large value that would exceed the limit
        act(() => {
            result.current.changeInternalValue("y".repeat(600000));
        });

        // Should still be able to undo
        expect(result.current.canUndo).to.equal(true);

        act(() => {
            result.current.undo();
        });

        // Should still have the initial value
        expect(result.current.internalValue).to.equal("x".repeat(500000));
        expect(result.current.canUndo).to.equal(false);
        expect(result.current.canRedo).to.equal(true);
    });

    it("should handle function updater pattern", () => {
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        // Use function updater
        act(() => {
            result.current.changeInternalValue((prev) => prev + " updated");
            clock.tick(200);
        });

        expect(result.current.internalValue).to.equal("initial updated");
        expect(result.current.canUndo).to.equal(true);
    });

    it("should call onChange with debounce", () => {
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        // Make a change
        act(() => {
            result.current.changeInternalValue("new value");
        });
        sinon.assert.notCalled(onChange); // Not called immediately

        // Advance time
        act(() => {
            clock.tick(200);
        });
        sinon.assert.calledWith(onChange, "new value");
    });
}); 
