import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Calculates the estimated index in the text based on the click position.
 *
 * @param clickX The X-coordinate of the click event relative to the element.
 * @param textWidth The width of the text element in pixels.
 * @param textLength The length of the text in characters.
 * @returns The estimated character index in the text where the click occurred.
 */
export function calculateEstimatedIndex(
    clickX: number,
    textWidth: number,
    textLength: number,
): number {
    return Math.round((clickX / textWidth) * textLength);
}

/**
 * Handles the label change by updating the `editedLabel` state with the new input value.
 *
 * @param event The input event containing the updated label value.
 * @param setEditedLabel Function to update the `editedLabel` state.
 */
export function handleChangeLabel(
    event: { target: { value: string } },
    setEditedLabel: (value: string) => void,
) {
    setEditedLabel(event.target.value);
}

/**
 * Submits the label change if editing is enabled and the label has been modified.
 * If `editedLabel` differs from `label`, `onUpdate` is called with `editedLabel`.
 * Finally, sets editing state to false.
 *
 * @param isEditable Flag indicating if the label can be edited.
 * @param editedLabel The modified label text.
 * @param  label The original label text.
 * @param onUpdate Callback to handle the updated label submission.
 * @param setIsEditingLabel Function to toggle the `isEditingLabel` state.
 */
export function submitChangeLabel(
    isEditable: boolean,
    editedLabel: string,
    label: string,
    onUpdate: (updatedLabel: string) => unknown,
    setIsEditingLabel: (value: boolean) => void,
) {
    if (!isEditable) return;
    if (editedLabel !== label) {
        onUpdate(editedLabel);
    }
    setIsEditingLabel(false);
}

/**
 * Handles keyboard events for editing the label.
 * Submits the label change on "Enter" key, resets on "Escape".
 *
 * @param e The keyboard event.
 * @param isEditable Flag indicating if the label can be edited.
 * @param isMultiline Flag indicating if the label supports multiline editing.
 * @param submitLabelChange Callback function to submit the label change.
 * @param setEditedLabel Function to update the `editedLabel` state.
 * @param setIsEditingLabel Function to toggle the `isEditingLabel` state.
 * @param label The original label text to revert to if editing is canceled.
 */
export function handleKeyDownLabel(
    e: React.KeyboardEvent,
    isEditable: boolean,
    isMultiline: boolean,
    submitLabelChange: () => void,
    setEditedLabel: (value: string) => void,
    setIsEditingLabel: (value: boolean) => void,
    label: string,
) {
    if (!isEditable) return;
    if (e.key === "Enter" && (!isMultiline || (isMultiline && e.shiftKey))) {
        e.preventDefault();
        submitLabelChange();
    } else if (e.key === "Escape") {
        e.preventDefault();
        setEditedLabel(label);
        setIsEditingLabel(false);
    }
}

export function useEditableLabel({
    isEditable,
    isMultiline,
    label,
    onUpdate,
}: {
    isEditable: boolean;
    isMultiline: boolean;
    label: string;
    onUpdate: (updatedLabel: string) => unknown;
}) {
    const labelEditRef = useRef<HTMLInputElement>(null);
    const [isEditingLabel, setIsEditingLabel] = useState(false);
    const [editedLabel, setEditedLabel] = useState(label);

    useEffect(function setEditedLabelEffect() {
        setEditedLabel(label);
    }, [label]);

    const submitLabelChange = useCallback(
        () => submitChangeLabel(isEditable, editedLabel, label, onUpdate, setIsEditingLabel),
        [isEditable, editedLabel, label, onUpdate],
    );

    const handleLabelChange = useCallback(
        (e: { target: { value: string } }) => handleChangeLabel(e, setEditedLabel),
        [],
    );

    const handleLabelKeyDown = useCallback(
        (e: React.KeyboardEvent) => handleKeyDownLabel(e, isEditable, isMultiline, submitLabelChange, setEditedLabel, setIsEditingLabel, label),
        [isEditable, isMultiline, label, submitLabelChange],
    );

    const startEditingLabel = useCallback((event: React.MouseEvent<HTMLElement>) => {
        if (!isEditable) return;
        setIsEditingLabel(true);

        // Calculate click position relative to the label
        const labelElement = event.currentTarget;
        const clickX = event.clientX - labelElement.getBoundingClientRect().left;
        const textWidth = labelElement.offsetWidth;
        const textLength = label.length;
        const estimatedIndex = calculateEstimatedIndex(clickX, textWidth, textLength);

        // Focus the input field and set cursor position
        setTimeout(() => {
            const inputElement = labelEditRef.current?.querySelector("input");
            if (!inputElement) {
                console.error("inputElement is null - unable to focus");
                return;
            }
            inputElement.focus();
            inputElement.setSelectionRange(estimatedIndex, estimatedIndex);
        });
    }, [isEditable, label]);

    return {
        editedLabel,
        handleLabelChange,
        handleLabelKeyDown,
        isEditingLabel,
        labelEditRef,
        startEditingLabel,
        submitLabelChange,
    };
}
