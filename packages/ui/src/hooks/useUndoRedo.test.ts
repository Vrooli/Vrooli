// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18
import { act, renderHook } from "@testing-library/react";
import { describe, it, expect, afterAll, beforeEach, afterEach, vi } from "vitest";
import { useUndoRedo } from "./useUndoRedo.js";

describe("useUndoRedo", () => {
    const initialValue = "initial";
    let onChange: ReturnType<typeof vi.fn>;

    beforeEach(() => {
        onChange = vi.fn();
        vi.useFakeTimers();
    });

    afterEach(() => {
        vi.useRealTimers();
        vi.clearAllMocks();
    });

    afterAll(() => {
        vi.useRealTimers();
    });

    it("should initialize with the initial value", () => {
        // GIVEN: A hook initialized with an initial value
        // WHEN: The hook is rendered
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        // THEN: The value should match the initial value with no undo/redo available
        expect(result.current.internalValue).toBe(initialValue);
        expect(result.current.canUndo).toBe(false);
        expect(result.current.canRedo).toBe(false);
    });

    it("updates value immediately when user types", () => {
        // GIVEN: An editor with undo/redo functionality
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        // WHEN: User types new content
        act(() => {
            result.current.changeInternalValue("new value");
        });

        // THEN: The value updates immediately for responsive typing
        expect(result.current.internalValue).toBe("new value");
    });

    it("enables undo after user stops typing", () => {
        // GIVEN: A user is typing in an editor
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        act(() => {
            result.current.changeInternalValue("new value");
        });
        
        // Initially undo is not available (still typing)
        expect(result.current.canUndo).toBe(false);

        // WHEN: User stops typing (debounce time passes)
        act(() => {
            vi.advanceTimersByTime(200);
        });

        // THEN: Undo becomes available
        expect(result.current.canUndo).toBe(true);
        expect(result.current.canRedo).toBe(false);
    });

    it("undoes last change when user presses undo", () => {
        // GIVEN: User has made edits to a document
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
                delimiters: [], // Disable debouncing for immediate testing
            }),
        );

        act(() => {
            result.current.changeInternalValue("first change");
            result.current.changeInternalValue("second change");
        });

        // WHEN: User presses undo (Ctrl+Z)
        act(() => {
            result.current.undo();
        });

        // THEN: Previous state is restored
        expect(result.current.internalValue).toBe("first change");
        expect(result.current.canUndo).toBe(true);
        expect(result.current.canRedo).toBe(true);
    });

    it("redoes change when user presses redo after undo", () => {
        // GIVEN: User has made changes and then undone them
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
                delimiters: [],
            }),
        );

        // Make changes and undo
        act(() => {
            result.current.changeInternalValue("first change");
        });
        act(() => {
            result.current.changeInternalValue("second change");
        });
        act(() => {
            result.current.undo();
        });

        // Verify we're at the previous state
        expect(result.current.internalValue).toBe("first change");

        // WHEN: User presses redo (Ctrl+Y)
        act(() => {
            result.current.redo();
        });

        // THEN: The undone change is restored
        expect(result.current.internalValue).toBe("second change");
        expect(result.current.canUndo).toBe(true);
        expect(result.current.canRedo).toBe(false);
    });

    it("creates undo point when user types space or enters new line", () => {
        // GIVEN: An editor configured to create undo points at word boundaries
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
                delimiters: [" ", "\n"],
            }),
        );

        // WHEN: User types a word without space
        act(() => {
            result.current.changeInternalValue("typing");
        });
        
        // THEN: No undo point yet (mid-word)
        expect(result.current.canUndo).toBe(false);

        // WHEN: User types a space (word boundary)
        act(() => {
            result.current.changeInternalValue("typing ");
        });
        
        // THEN: Undo point is created immediately
        expect(result.current.canUndo).toBe(true);
    });

    it("records every change immediately when delimiters disabled", () => {
        // GIVEN: An application that needs precise undo history (e.g., drawing app)
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
                delimiters: [], // Disable debouncing
            }),
        );

        // WHEN: User makes rapid changes
        act(() => {
            result.current.changeInternalValue("change 1");
        });
        
        // THEN: Each change is recorded immediately
        expect(result.current.canUndo).toBe(true);
    });

    it("preserves undo history when external value changes", () => {
        // GIVEN: User has made edits and built up undo history
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        act(() => {
            result.current.changeInternalValue("user edit");
            vi.advanceTimersByTime(200);
        });
        
        const canUndoBefore = result.current.canUndo;

        // WHEN: External system updates the value (e.g., real-time collaboration)
        act(() => {
            result.current.resetInternalValue("external update");
        });

        // THEN: Value updates but undo history is preserved
        expect(result.current.internalValue).toBe("external update");
        expect(result.current.canUndo).toBe(canUndoBefore);
    });

    it("clears undo history when user saves document", () => {
        // GIVEN: User has made multiple edits
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        act(() => {
            result.current.changeInternalValue("edit 1");
            vi.advanceTimersByTime(200);
            result.current.changeInternalValue("edit 2");
            vi.advanceTimersByTime(200);
        });

        // WHEN: User saves the document
        act(() => {
            result.current.resetStack();
        });

        // THEN: Undo history is cleared but current value remains
        expect(result.current.canUndo).toBe(false);
        expect(result.current.canRedo).toBe(false);
        expect(result.current.internalValue).toBe("edit 2");
    });

    it("prevents undo/redo when feature is disabled", () => {
        // GIVEN: Undo/redo is disabled (e.g., in read-only mode)
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
                disabled: true,
            }),
        );

        act(() => {
            result.current.changeInternalValue("change");
            vi.advanceTimersByTime(200);
        });

        const valueBefore = result.current.internalValue;

        // WHEN: User attempts to undo
        act(() => {
            result.current.undo();
        });

        // THEN: Nothing happens
        expect(result.current.internalValue).toBe(valueBefore);
    });

    it("maintains undo functionality with large documents", () => {
        // GIVEN: User is editing a very large document
        const largeDocument = "x".repeat(500000);
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue: largeDocument,
                onChange,
                delimiters: [],
            }),
        );

        // WHEN: User makes a large edit
        act(() => {
            result.current.changeInternalValue("y".repeat(600000));
        });

        // THEN: Undo still works despite large memory usage
        expect(result.current.canUndo).toBe(true);
        
        act(() => {
            result.current.undo();
        });
        
        expect(result.current.internalValue).toBe(largeDocument);
    });

    it("supports function updates for complex state transformations", () => {
        // GIVEN: An application needs to transform content based on current state
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        // WHEN: Update depends on previous value (e.g., text transformation)
        act(() => {
            result.current.changeInternalValue((prev) => prev + " updated");
            vi.advanceTimersByTime(200);
        });

        // THEN: Function updater works and creates undo point
        expect(result.current.internalValue).toBe("initial updated");
        expect(result.current.canUndo).toBe(true);
    });

    it("notifies parent component after user stops typing", () => {
        // GIVEN: Parent component needs to know when to save drafts
        const { result } = renderHook(() =>
            useUndoRedo({
                initialValue,
                onChange,
            }),
        );

        // WHEN: User types
        act(() => {
            result.current.changeInternalValue("draft content");
        });
        
        // THEN: onChange not called immediately (user still typing)
        expect(onChange).not.toHaveBeenCalled();

        // WHEN: User stops typing
        act(() => {
            vi.advanceTimersByTime(200);
        });
        
        // THEN: Parent is notified to save draft
        expect(onChange).toHaveBeenCalledWith("draft content");
    });
}); 
