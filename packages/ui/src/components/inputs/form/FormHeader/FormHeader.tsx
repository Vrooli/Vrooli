import { Box, IconButton, TextField, Tooltip, useTheme } from "@mui/material";
import { HeaderTag } from "forms/types";
import { useEditableLabel } from "hooks/useEditableLabel";
import { DeleteIcon } from "icons";
import React, { useCallback, useMemo } from "react";
import { FormHeaderProps } from "../types";

const NUM_HEADERS = 6;
const HEADER_OFFSET = 2;

export function FormHeader({
    element,
    isSelected,
    onUpdate,
    onDelete,
}: FormHeaderProps) {
    const { palette, typography } = useTheme();

    const getHeaderStyle = useCallback((tag: HeaderTag) => {
        const tagSize = Math.min(NUM_HEADERS, parseInt(tag[1]) + HEADER_OFFSET);
        return {
            paddingLeft: "8px",
            paddingRight: "8px",
            ...((typography[`h${tagSize}` as keyof typeof typography]) as object || {}),
        };
    }, [typography]);

    const updateLabel = useCallback((updatedLabel: string) => { onUpdate({ label: updatedLabel }); }, [onUpdate]);
    const {
        editedLabel,
        handleLabelChange,
        handleLabelKeyDown,
        labelEditRef,
        submitLabelChange,
    } = useEditableLabel({
        isEditable: isSelected,
        label: element.label,
        onUpdate: updateLabel,
    });

    const style = useMemo(() => getHeaderStyle(element.tag), [element.tag, getHeaderStyle]);
    const inputProps = useMemo(() => ({ style }), [style]);

    return (
        <Box display="flex" alignItems="center">
            {isSelected ? (
                <>
                    <Tooltip title="Delete">
                        <IconButton onClick={onDelete}>
                            <DeleteIcon fill={palette.error.main} width="24px" height="24px" />
                        </IconButton>
                    </Tooltip>
                    <TextField
                        ref={labelEditRef}
                        autoFocus
                        fullWidth
                        InputProps={inputProps}
                        onBlur={submitLabelChange}
                        onChange={handleLabelChange}
                        onKeyDown={handleLabelKeyDown}
                        value={editedLabel}
                        variant="outlined"
                    />
                </>
            ) : (
                React.createElement(element.tag, style, element.label)
            )}
        </Box>
    );
}
