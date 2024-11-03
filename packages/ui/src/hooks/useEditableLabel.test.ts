import { renderHook } from "@testing-library/react";
import { act } from "react";
import {
    calculateEstimatedIndex,
    handleChangeLabel,
    handleKeyDownLabel,
    submitChangeLabel,
    useEditableLabel,
} from "./useEditableLabel";

describe("calculateEstimatedIndex", () => {
    it("calculates the correct index based on click position", () => {
        expect(calculateEstimatedIndex(50, 200, 10)).toBeCloseTo(3, 0);
        expect(calculateEstimatedIndex(100, 200, 10)).toBeCloseTo(5, 0);
        expect(calculateEstimatedIndex(150, 200, 10)).toBeCloseTo(8, 0);
    });

    it("returns 0 if clickX is 0", () => {
        expect(calculateEstimatedIndex(0, 200, 10)).toBe(0);
    });

    it("returns textLength if clickX is equal to textWidth", () => {
        expect(calculateEstimatedIndex(200, 200, 10)).toBe(10);
    });
});

describe("handleChangeLabel", () => {
    it("updates the editedLabel state with the new value", () => {
        const setEditedLabelMock = jest.fn();
        handleChangeLabel({ target: { value: "new value" } }, setEditedLabelMock);
        expect(setEditedLabelMock).toHaveBeenCalledWith("new value");
    });
});

describe("submitChangeLabel", () => {
    it("does not call onUpdate if isEditable is false", () => {
        const onUpdateMock = jest.fn();
        const setIsEditingLabelMock = jest.fn();
        submitChangeLabel(false, "edited", "original", onUpdateMock, setIsEditingLabelMock);
        expect(onUpdateMock).not.toHaveBeenCalled();
    });

    it("calls onUpdate if isEditable is true and editedLabel differs from label", () => {
        const onUpdateMock = jest.fn();
        const setIsEditingLabelMock = jest.fn();
        submitChangeLabel(true, "edited", "original", onUpdateMock, setIsEditingLabelMock);
        expect(onUpdateMock).toHaveBeenCalledWith("edited");
        expect(setIsEditingLabelMock).toHaveBeenCalledWith(false);
    });

    it("does not call onUpdate if editedLabel is the same as label", () => {
        const onUpdateMock = jest.fn();
        const setIsEditingLabelMock = jest.fn();
        submitChangeLabel(true, "original", "original", onUpdateMock, setIsEditingLabelMock);
        expect(onUpdateMock).not.toHaveBeenCalled();
        expect(setIsEditingLabelMock).toHaveBeenCalledWith(false);
    });
});

describe("handleKeyDownLabel", () => {
    it("submits label change on Enter key", () => {
        const submitLabelChangeMock = jest.fn();
        const setEditedLabelMock = jest.fn();
        const setIsEditingLabelMock = jest.fn();

        const event = { key: "Enter", preventDefault: jest.fn(), shiftKey: false };
        handleKeyDownLabel(event as any, true, false, submitLabelChangeMock, setEditedLabelMock, setIsEditingLabelMock, "label");
        expect(event.preventDefault).toHaveBeenCalled();
        expect(submitLabelChangeMock).toHaveBeenCalled();
    });

    it("cancels label edit on Escape key", () => {
        const submitLabelChangeMock = jest.fn();
        const setEditedLabelMock = jest.fn();
        const setIsEditingLabelMock = jest.fn();

        const event = { key: "Escape", preventDefault: jest.fn() };
        handleKeyDownLabel(event as any, true, false, submitLabelChangeMock, setEditedLabelMock, setIsEditingLabelMock, "label");
        expect(event.preventDefault).toHaveBeenCalled();
        expect(setEditedLabelMock).toHaveBeenCalledWith("label");
        expect(setIsEditingLabelMock).toHaveBeenCalledWith(false);
    });

    it("does not handle keys if isEditable is false", () => {
        const submitLabelChangeMock = jest.fn();
        const setEditedLabelMock = jest.fn();
        const setIsEditingLabelMock = jest.fn();

        const event = { key: "Enter", preventDefault: jest.fn(), shiftKey: false };
        handleKeyDownLabel(event as any, false, false, submitLabelChangeMock, setEditedLabelMock, setIsEditingLabelMock, "label");
        expect(submitLabelChangeMock).not.toHaveBeenCalled();
        expect(setEditedLabelMock).not.toHaveBeenCalled();
        expect(setIsEditingLabelMock).not.toHaveBeenCalled();
    });
});

describe("useEditableLabel", () => {
    it("initializes with the provided label", () => {
        const { result } = renderHook(() => useEditableLabel({
            isEditable: true,
            isMultiline: false,
            label: "initial",
            onUpdate: jest.fn(),
        }));

        expect(result.current.editedLabel).toBe("initial");
        expect(result.current.isEditingLabel).toBe(false);
    });

    it("updates editedLabel when label prop changes", () => {
        const { result, rerender } = renderHook(() => useEditableLabel({
            isEditable: true,
            isMultiline: false,
            label: "initial",
            onUpdate: jest.fn(),
        }));

        rerender({ isEditable: true, isMultiline: false, label: "updated", onUpdate: jest.fn() });
        expect(result.current.editedLabel).toBe("updated");
    });

    it("starts editing label on startEditingLabel call", () => {
        const { result } = renderHook(() => useEditableLabel({
            isEditable: true,
            isMultiline: false,
            label: "label",
            onUpdate: jest.fn(),
        }));

        act(() => result.current.startEditingLabel({
            clientX: 10,
            currentTarget: { offsetWidth: 100, getBoundingClientRect: () => ({ left: 0 }) } as any,
        } as React.MouseEvent<HTMLElement>));

        expect(result.current.isEditingLabel).toBe(true);
    });

    it("handles label change and updates editedLabel state", () => {
        const { result } = renderHook(() => useEditableLabel({
            isEditable: true,
            isMultiline: false,
            label: "label",
            onUpdate: jest.fn(),
        }));

        act(() => result.current.handleLabelChange({ target: { value: "new label" } }));
        expect(result.current.editedLabel).toBe("new label");
    });

    it("submits label change when editedLabel is modified", () => {
        const onUpdateMock = jest.fn();
        const { result } = renderHook(() => useEditableLabel({
            isEditable: true,
            isMultiline: false,
            label: "label",
            onUpdate: onUpdateMock,
        }));

        act(() => {
            result.current.handleLabelChange({ target: { value: "new label" } });
            result.current.submitLabelChange();
        });

        expect(onUpdateMock).toHaveBeenCalledWith("new label");
    });

    it("does not submit label if editedLabel is unchanged", () => {
        const onUpdateMock = jest.fn();
        const { result } = renderHook(() => useEditableLabel({
            isEditable: true,
            isMultiline: false,
            label: "label",
            onUpdate: onUpdateMock,
        }));

        act(() => result.current.submitLabelChange());
        expect(onUpdateMock).not.toHaveBeenCalled();
    });
});
