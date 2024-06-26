import { Box, IconButton, TextField, Tooltip, useTheme } from "@mui/material";
import { FormHeaderType } from "forms/types";
import { DeleteIcon } from "icons";
import React, { useCallback, useState } from "react";
import { FormHeaderProps } from "../types";

export function FormHeader({
    element,
    isSelected,
    onUpdate,
    onDelete,
}: FormHeaderProps) {
    const { palette, typography } = useTheme();
    const [isEditing, setIsEditing] = useState(false);
    const [editedLabel, setEditedLabel] = useState(element.label);

    const getHeaderStyle = useCallback((tag: FormHeaderType["tag"]) => {
        const NUM_HEADERS = 6;
        const HEADER_OFFSET = 2;
        const tagSize = Math.min(NUM_HEADERS, parseInt(tag[1]) + HEADER_OFFSET);
        return {
            paddingLeft: "8px",
            paddingRight: "8px",
            ...((typography[`h${tagSize}` as keyof typeof typography]) as object || {}),
        };
    }, [typography]);

    function handleLabelChange(event: React.ChangeEvent<HTMLInputElement>) {
        setEditedLabel(event.target.value);
    }

    const submitLabelChange = useCallback(function submitLabelChangeCallback() {
        console.log("in submit label change", editedLabel, element.label, onUpdate);
        if (editedLabel !== element.label) {
            onUpdate({ label: editedLabel });
        }
        setIsEditing(false);
    }, [editedLabel, element.label, onUpdate]);

    const handleLabelKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (e.key === "Enter") {
            e.preventDefault();
            submitLabelChange();
        } else if (e.key === "Escape") {
            e.preventDefault();
            setEditedLabel(element.label);
            setIsEditing(false);
        }
    }, [element.label, submitLabelChange]);

    return (
        <Box sx={{ display: "flex", alignItems: "center" }}>
            {isSelected ? (
                <>
                    <Tooltip title="Delete">
                        <IconButton onClick={onDelete}>
                            <DeleteIcon fill={palette.error.main} width="24px" height="24px" />
                        </IconButton>
                    </Tooltip>
                    <TextField
                        autoFocus
                        fullWidth
                        InputProps={{ style: getHeaderStyle(element.tag) }}
                        onBlur={submitLabelChange}
                        onChange={handleLabelChange}
                        onKeyDown={handleLabelKeyDown}
                        value={editedLabel}
                        variant="outlined"
                    />
                </>
            ) : (
                React.createElement(
                    element.tag,
                    { style: getHeaderStyle(element.tag) },
                    element.label,
                )
            )}
        </Box>
    );
}
