import { useCallback, useEffect, useRef, useState } from "react";

export function useEditableLabel({
    isEditable,
    label,
    onUpdate,
}: {
    isEditable: boolean;
    label: string;
    onUpdate: (updatedLabel: string) => unknown;
}) {
    const labelEditRef = useRef<HTMLInputElement>(null);

    const [isEditingLabel, setIsEditingLabel] = useState(false);

    const [editedLabel, setEditedLabel] = useState(label);
    useEffect(function setEditedLabelEffect() {
        setEditedLabel(label);
    }, [label]);

    function handleLabelChange(event: React.ChangeEvent<HTMLInputElement>) {
        setEditedLabel(event.target.value);
    }

    const submitLabelChange = useCallback(function submitLabelChangeCallback() {
        if (!isEditable) return;
        if (editedLabel !== label) {
            onUpdate(editedLabel);
        }
        setIsEditingLabel(false);
    }, [isEditable, editedLabel, label, onUpdate]);

    const handleLabelKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (!isEditable) return;
        if (e.key === "Enter") {
            e.preventDefault();
            submitLabelChange();
        } else if (e.key === "Escape") {
            e.preventDefault();
            setEditedLabel(label);
            setIsEditingLabel(false);
        }
    }, [isEditable, submitLabelChange, label]);

    const startEditingLabel = useCallback((event: React.MouseEvent<HTMLElement>) => {
        if (!isEditable) return;
        setIsEditingLabel(true);

        // Calculate click position relative to the label
        const labelElement = event.currentTarget;
        const clickX = event.clientX - labelElement.getBoundingClientRect().left;
        const textWidth = labelElement.offsetWidth;
        const textLength = label.length;
        const estimatedIndex = Math.round((clickX / textWidth) * textLength);

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
