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

    const startEditingLabel = useCallback(() => {
        if (!isEditable) return;
        setIsEditingLabel(true);
        // Focus the input field
        setTimeout(() => {
            if (!labelEditRef.current) {
                console.error("labelEditRef.current is null - unable to focus");
                return;
            }
            labelEditRef.current.focus();
        });
    }, [isEditable]);

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
