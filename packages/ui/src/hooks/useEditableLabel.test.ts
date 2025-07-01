// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18
import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { calculateEstimatedIndex, handleChangeLabel, handleKeyDownLabel, submitChangeLabel, useEditableLabel } from "./useEditableLabel.js";

describe("calculateEstimatedIndex", () => {
    afterEach(() => {
        vi.clearAllMocks();
    });

    it("places cursor at beginning when user clicks at start of text", () => {
        // GIVEN: User clicks at the beginning of text
        const clickX = 0;
        const textWidth = 200;
        const textLength = 10;

        // WHEN: Calculating cursor position
        const index = calculateEstimatedIndex(clickX, textWidth, textLength);

        // THEN: Cursor is placed at the beginning
        expect(index).toBe(0);
    });

    it("places cursor at end when user clicks at end of text", () => {
        // GIVEN: User clicks at the end of text
        const clickX = 200;
        const textWidth = 200;
        const textLength = 10;

        // WHEN: Calculating cursor position
        const index = calculateEstimatedIndex(clickX, textWidth, textLength);

        // THEN: Cursor is placed at the end
        expect(index).toBe(10);
    });

    it("places cursor proportionally when user clicks in middle of text", () => {
        // GIVEN: User clicks in the middle of text
        const clickX = 100;
        const textWidth = 200;
        const textLength = 10;

        // WHEN: Calculating cursor position
        const index = calculateEstimatedIndex(clickX, textWidth, textLength);

        // THEN: Cursor is placed proportionally (halfway = index 5)
        expect(index).toBeCloseTo(5, 0);
    });
});

describe("handleChangeLabel", () => {
    afterEach(() => {
        vi.clearAllMocks();
    });

    it("updates label text as user types", () => {
        // GIVEN: User is editing a label
        const setEditedLabelMock = vi.fn();
        const event = { target: { value: "new value" } };

        // WHEN: User types new text
        handleChangeLabel(event, setEditedLabelMock);

        // THEN: The label updates with the typed text
        expect(setEditedLabelMock).toHaveBeenCalledWith("new value");
    });
});

describe("submitChangeLabel", () => {
    afterEach(() => {
        vi.clearAllMocks();
    });

    it("prevents label changes in read-only mode", () => {
        // GIVEN: Label is in read-only mode
        const onUpdateMock = vi.fn();
        const setIsEditingLabelMock = vi.fn();
        const isEditable = false;

        // WHEN: Attempting to submit changes
        submitChangeLabel(isEditable, "edited", "original", onUpdateMock, setIsEditingLabelMock);

        // THEN: No update occurs
        expect(onUpdateMock).not.toHaveBeenCalled();
    });

    it("saves label changes when user finishes editing", () => {
        // GIVEN: User has edited a label
        const onUpdateMock = vi.fn();
        const setIsEditingLabelMock = vi.fn();
        const isEditable = true;
        const editedText = "New Title";
        const originalText = "Old Title";

        // WHEN: User submits the change
        submitChangeLabel(isEditable, editedText, originalText, onUpdateMock, setIsEditingLabelMock);

        // THEN: The new label is saved and edit mode ends
        expect(onUpdateMock).toHaveBeenCalledWith(editedText);
        expect(setIsEditingLabelMock).toHaveBeenCalledWith(false);
    });

    it("ignores submission when label text unchanged", () => {
        // GIVEN: User opened edit mode but didn't change text
        const onUpdateMock = vi.fn();
        const setIsEditingLabelMock = vi.fn();
        const isEditable = true;
        const unchangedText = "Same Title";

        // WHEN: User submits without changes
        submitChangeLabel(isEditable, unchangedText, unchangedText, onUpdateMock, setIsEditingLabelMock);

        // THEN: No update occurs but edit mode ends
        expect(onUpdateMock).not.toHaveBeenCalled();
        expect(setIsEditingLabelMock).toHaveBeenCalledWith(false);
    });
});

describe("handleKeyDownLabel", () => {
    afterEach(() => {
        vi.clearAllMocks();
    });

    it("saves label when user presses Enter", () => {
        // GIVEN: User is editing a label
        const submitLabelChangeMock = vi.fn();
        const setEditedLabelMock = vi.fn();
        const setIsEditingLabelMock = vi.fn();
        const event = { key: "Enter", preventDefault: vi.fn(), shiftKey: false };

        // WHEN: User presses Enter key
        handleKeyDownLabel(event as any, true, false, submitLabelChangeMock, setEditedLabelMock, setIsEditingLabelMock, "label");

        // THEN: Changes are submitted
        expect(event.preventDefault).toHaveBeenCalled();
        expect(submitLabelChangeMock).toHaveBeenCalled();
    });

    it("cancels editing when user presses Escape", () => {
        // GIVEN: User is editing a label
        const submitLabelChangeMock = vi.fn();
        const setEditedLabelMock = vi.fn();
        const setIsEditingLabelMock = vi.fn();
        const originalLabel = "Original Text";
        const event = { key: "Escape", preventDefault: vi.fn() };

        // WHEN: User presses Escape key
        handleKeyDownLabel(event as any, true, false, submitLabelChangeMock, setEditedLabelMock, setIsEditingLabelMock, originalLabel);

        // THEN: Edit is cancelled and original text restored
        expect(event.preventDefault).toHaveBeenCalled();
        expect(setEditedLabelMock).toHaveBeenCalledWith(originalLabel);
        expect(setIsEditingLabelMock).toHaveBeenCalledWith(false);
        expect(submitLabelChangeMock).not.toHaveBeenCalled();
    });

    it("ignores keyboard shortcuts in read-only mode", () => {
        // GIVEN: Label is in read-only mode
        const submitLabelChangeMock = vi.fn();
        const setEditedLabelMock = vi.fn();
        const setIsEditingLabelMock = vi.fn();
        const event = { key: "Enter", preventDefault: vi.fn(), shiftKey: false };

        // WHEN: User presses Enter in read-only mode
        handleKeyDownLabel(event as any, false, false, submitLabelChangeMock, setEditedLabelMock, setIsEditingLabelMock, "label");

        // THEN: Nothing happens
        expect(submitLabelChangeMock).not.toHaveBeenCalled();
        expect(setEditedLabelMock).not.toHaveBeenCalled();
        expect(setIsEditingLabelMock).not.toHaveBeenCalled();
    });
});

describe("useEditableLabel", () => {
    afterEach(() => {
        vi.clearAllMocks();
    });

    it("displays label in view mode by default", () => {
        // GIVEN: A label component is rendered
        // WHEN: Component initializes
        const { result } = renderHook(() => useEditableLabel({
            isEditable: true,
            isMultiline: false,
            label: "Product Title",
            onUpdate: vi.fn(),
        }));

        // THEN: Label is displayed in view mode
        expect(result.current.editedLabel).toBe("Product Title");
        expect(result.current.isEditingLabel).toBe(false);
    });

    it("updates display when label data changes externally", () => {
        // GIVEN: A label displaying dynamic content
        const onUpdate = vi.fn();
        const { result, rerender } = renderHook(
            ({ label }) => useEditableLabel({
                isEditable: true,
                isMultiline: false,
                label,
                onUpdate,
            }),
            {
                initialProps: { label: "Loading..." },
            },
        );

        // WHEN: External data loads and updates the label
        rerender({ label: "Product Name" });

        // THEN: Display updates to show new content
        expect(result.current.editedLabel).toBe("Product Name");
    });

    it("enters edit mode when user clicks on label", () => {
        // GIVEN: An editable label in view mode
        const { result } = renderHook(() => useEditableLabel({
            isEditable: true,
            isMultiline: false,
            label: "Click to edit",
            onUpdate: vi.fn(),
        }));

        // WHEN: User clicks on the label
        act(() => result.current.startEditingLabel({
            clientX: 10,
            currentTarget: { offsetWidth: 100, getBoundingClientRect: () => ({ left: 0 }) } as any,
        } as React.MouseEvent<HTMLElement>));

        // THEN: Label enters edit mode
        expect(result.current.isEditingLabel).toBe(true);
    });

    it("updates text as user types in edit mode", () => {
        // GIVEN: User is editing a label
        const { result } = renderHook(() => useEditableLabel({
            isEditable: true,
            isMultiline: false,
            label: "Original",
            onUpdate: vi.fn(),
        }));

        // WHEN: User types new text
        act(() => result.current.handleLabelChange({ target: { value: "Updated Text" } }));

        // THEN: Display shows the typed text
        expect(result.current.editedLabel).toBe("Updated Text");
    });

    it("saves changes when user confirms edit", () => {
        // GIVEN: User clicks to edit a product title
        const onUpdateMock = vi.fn();
        const { result } = renderHook(() => useEditableLabel({
            isEditable: true,
            isMultiline: false,
            label: "Old Product Name",
            onUpdate: onUpdateMock,
        }));

        // WHEN: User clicks to edit
        act(() => {
            result.current.startEditingLabel({
                clientX: 10,
                currentTarget: { offsetWidth: 100, getBoundingClientRect: () => ({ left: 0 }) } as any,
            } as React.MouseEvent<HTMLElement>);
        });

        // Verify we're in edit mode
        expect(result.current.isEditingLabel).toBe(true);

        // WHEN: User types new name
        act(() => {
            result.current.handleLabelChange({ target: { value: "New Product Name" } });
        });

        // WHEN: User confirms the edit
        act(() => {
            result.current.submitLabelChange();
        });

        // THEN: New name is saved and edit mode ends
        expect(onUpdateMock).toHaveBeenCalledWith("New Product Name");
        expect(result.current.isEditingLabel).toBe(false);
    });

    it("skips save when user submits without changes", () => {
        // GIVEN: Label with original text
        const onUpdateMock = vi.fn();
        const { result } = renderHook(() => useEditableLabel({
            isEditable: true,
            isMultiline: false,
            label: "Same Text",
            onUpdate: onUpdateMock,
        }));

        // WHEN: User submits without making changes
        act(() => result.current.submitLabelChange());

        // THEN: No update is triggered
        expect(onUpdateMock).not.toHaveBeenCalled();
    });
});
