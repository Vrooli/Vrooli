import { Box, Button, IconButton, List, ListItem, ListItemIcon, ListItemText, Popover, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { FormHeaderType, HeaderTag } from "forms/types";
import { useEditableLabel } from "hooks/useEditableLabel";
import { usePopover } from "hooks/usePopover";
import { DeleteIcon, Header1Icon, Header2Icon, Header3Icon, Header4Icon } from "icons";
import React, { memo, useCallback, useMemo } from "react";
import { FormSettingsButtonRow, propButtonStyle } from "../styles";
import { FormHeaderProps } from "../types";

const NUM_HEADERS = 6;
const HEADER_OFFSET = 2;

function getDisplayTag(tag: HeaderTag) {
    const tagSize = Math.min(NUM_HEADERS, parseInt(tag[1]) + HEADER_OFFSET);
    return `h${tagSize}` as HeaderTag;
}

const popoverAnchorOrigin = { vertical: "bottom", horizontal: "center" } as const;

interface PopoverListItemProps {
    icon: React.ReactNode;
    label: string;
    tag: FormHeaderType["tag"];
    onSetTag: (tag: FormHeaderType["tag"]) => unknown;
}

const PopoverListItem = memo(function PopoverListItemMemo({
    icon,
    label,
    onSetTag,
    tag,
}: PopoverListItemProps) {
    const handleClick = useCallback(() => {
        onSetTag(tag);
    }, [onSetTag, tag]);

    return (
        <ListItem
            button
            onClick={handleClick}
        >
            <ListItemIcon>{icon}</ListItemIcon>
            <ListItemText primary={label} />
        </ListItem>
    );
});

export function FormHeader({
    element,
    isEditing,
    onUpdate,
    onDelete,
}: FormHeaderProps) {
    const { palette, spacing, typography } = useTheme();

    const getHeaderStyle = useCallback((tag: HeaderTag) => {
        return {
            paddingLeft: spacing(1),
            paddingRight: spacing(1),
            paddingBottom: spacing(2),
            ...((typography[getDisplayTag(tag)]) as object || {}),
        };
    }, [spacing, typography]);

    const [tagAnchorEl, openTagPopover, closeTagPopover, isTagPopoverOpen] = usePopover();

    const headerItems = useMemo(function headerItemsMemo() {
        return ([
            { type: "Header", tag: "h1", icon: <Header1Icon />, label: "Title (Largest)" },
            { type: "Header", tag: "h2", icon: <Header2Icon />, label: "Subtitle (Large)" },
            { type: "Header", tag: "h3", icon: <Header3Icon />, label: "Header (Medium)" },
            { type: "Header", tag: "h4", icon: <Header4Icon />, label: "Subheader (Small)" },
        ] as const);
    }, []);

    const updateLabel = useCallback((updatedLabel: string) => {
        if (!isEditing) {
            return;
        }
        onUpdate({ label: updatedLabel });
    }, [isEditing, onUpdate]);
    const updateTag = useCallback((updatedTag: HeaderTag) => {
        if (!isEditing) {
            return;
        }
        onUpdate({ tag: updatedTag });
        closeTagPopover();
    }, [closeTagPopover, isEditing, onUpdate]);

    const isCollapsible = useMemo(function isCollapsibleMemo() {
        return element.isCollapsible ?? true;
    }, [element.isCollapsible]);
    const toggleCollapsible = useCallback(function toggleCollapsibleCallback() {
        if (!isEditing) {
            return;
        }
        onUpdate({ isCollapsible: !isCollapsible });
    }, [isCollapsible, isEditing, onUpdate]);

    const {
        editedLabel,
        handleLabelChange,
        handleLabelKeyDown,
        labelEditRef,
        submitLabelChange,
    } = useEditableLabel({
        isEditable: isEditing,
        label: element.label,
        onUpdate: updateLabel,
    });

    const style = useMemo(() => getHeaderStyle(element.tag), [element.tag, getHeaderStyle]);
    const inputProps = useMemo(() => ({ style }), [style]);

    const HeaderElement = useMemo(function headerElementMemo() {
        return (
            isEditing ? (
                <Box display="flex" alignItems="center">
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
                </Box>
            ) : (
                // React.createElement(element.tag, style, element.label)
                <Typography variant={getDisplayTag(element.tag)} sx={style}>
                    {element.label}
                </Typography>
            )
        );
    }, [element.tag, style, element.label, isEditing, onDelete, palette.error.main, labelEditRef, inputProps, submitLabelChange, handleLabelChange, handleLabelKeyDown, editedLabel]);

    // If we're not editing, just display the input element
    if (!isEditing) {
        return HeaderElement;
    }
    return (
        <div>
            <Popover
                open={isTagPopoverOpen}
                anchorEl={tagAnchorEl}
                onClose={closeTagPopover}
                anchorOrigin={popoverAnchorOrigin}
            >
                <List>
                    {headerItems.map((item) => (
                        <PopoverListItem
                            key={item.tag}
                            icon={item.icon}
                            label={item.label}
                            tag={item.tag}
                            onSetTag={updateTag}
                        />
                    ))}
                </List>
            </Popover>
            {HeaderElement}
            <FormSettingsButtonRow>
                <Button variant="text" sx={propButtonStyle} onClick={openTagPopover}>
                    Size: {headerItems.find((item) => item.tag === element.tag)?.label}
                </Button>
                <Button variant="text" sx={propButtonStyle} onClick={toggleCollapsible}>
                    {isCollapsible ? "Not Collapsible" : "Collapsible"}
                </Button>
            </FormSettingsButtonRow>
        </div>
    );
}
