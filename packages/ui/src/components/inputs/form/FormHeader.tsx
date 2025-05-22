import { type FormHeaderType, FormStructureType, type HeaderTag } from "@local/shared";
import { Box, Button, IconButton, List, ListItem, ListItemIcon, ListItemText, type Palette, Popover, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import React, { forwardRef, memo, useCallback, useImperativeHandle, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useEditableLabel } from "../../../hooks/useEditableLabel.js";
import { usePopover } from "../../../hooks/usePopover.js";
import { Icon, IconCommon } from "../../../icons/Icons.js";
import { HelpButton } from "../../buttons/HelpButton.js";
import { MarkdownDisplay } from "../../text/MarkdownDisplay.js";
import { RichInputBase } from "../RichInput/RichInput.js";
import { FormSettingsButtonRow, propButtonStyle } from "./styles.js";
import { type FormHeaderProps } from "./types.js";

/** Total number of header options in HTML (i.e. h1, h2, ..., h6) */
const NUM_HEADERS = 6;
/** Offset for rendering headers */
const HEADER_OFFSET = 2;
/** Minimum rows allows for paragraphs */
const ROWS_MIN_PARAGRAPH = 1;
/** Maximum rows allowed for paragraphs */
const ROWS_MAX_PARAGRAPH = 10;

export const FORM_HEADER_SIZE_OPTIONS = [
    { type: FormStructureType.Header, tag: "h1", iconInfo: { name: "Header1", type: "Text" }, label: "Title (Largest)" },
    { type: FormStructureType.Header, tag: "h2", iconInfo: { name: "Header2", type: "Text" }, label: "Subtitle (Large)" },
    { type: FormStructureType.Header, tag: "h3", iconInfo: { name: "Header3", type: "Text" }, label: "Header (Normal)" },
    { type: FormStructureType.Header, tag: "h4", iconInfo: { name: "Header4", type: "Text" }, label: "Subheader (Small)" },
    { type: FormStructureType.Header, tag: "body1", iconInfo: null, label: "Paragraph" },
    { type: FormStructureType.Header, tag: "body2", iconInfo: null, label: "Caption" },
] as const;

const COLOR_OPTIONS = [
    { label: "Default", value: "default" },
    { label: "Primary", value: "primary" },
    { label: "Secondary", value: "secondary" },
] as const;

function getDisplayTag(tag: HeaderTag) {
    if (tag === "body1") return "body1" as const;
    if (tag === "body2") return "body2" as const;
    const tagSize = Math.min(NUM_HEADERS, parseInt(tag[1]) + HEADER_OFFSET);
    return `h${tagSize}` as Exclude<HeaderTag, "body1" | "body2">;
}

/**
 * Validates whether a string is a valid CSS color
 * Supports: hex colors, rgb/rgba, hsl/hsla, named colors, and theme colors
 */
function isValidColor(color: string): boolean {
    // Handle theme colors
    if (["primary", "secondary", "default"].includes(color)) {
        return true;
    }

    // Test if it's a valid CSS color using DOM
    const element = document.createElement("div");
    element.style.color = color;

    // If the color was invalid, the style will not be set
    return element.style.color !== "";
}

/**
 * Normalizes hex colors to standard 6-digit format
 * e.g., #fff -> #ffffff, #f00 -> #ff0000
 */
function normalizeHexColor(color: string): string {
    if (!color.startsWith("#")) return color;

    // Convert 3-digit hex to 6-digit
    if (color.length === "#fff".length) {
        const r = color[1];
        const g = color[2];
        const b = color[3];
        return `#${r}${r}${g}${g}${b}${b}`;
    }

    return color;
}

function getColorStyle(color: string | null | undefined, palette: Palette) {
    if (!color || typeof color !== "string" || color === "default" || color === "primary") return palette.background.textPrimary;
    if (color === "secondary") return palette.background.textSecondary;
    if (isValidColor(color)) return color;
    return palette.background.textPrimary;
}

const popoverAnchorOrigin = { vertical: "bottom", horizontal: "center" } as const;

interface SizeListItemProps {
    icon?: React.ReactNode;
    label: string;
    tag: FormHeaderType["tag"];
    onSetTag: (tag: FormHeaderType["tag"]) => unknown;
}
const SizeListItem = memo(function SizeListItemMemo({
    icon,
    label,
    onSetTag,
    tag,
}: SizeListItemProps) {
    const handleClick = useCallback(() => {
        onSetTag(tag);
    }, [onSetTag, tag]);

    return (
        <ListItem
            button
            onClick={handleClick}
        >
            {Boolean(icon) && <ListItemIcon>{icon}</ListItemIcon>}
            <ListItemText primary={label} />
        </ListItem>
    );
});

interface ColorListItemProps {
    label: string;
    value: string;
    onSetColor: (color: string) => unknown;
}
const ColorListItem = memo(function ColorListItemMemo({
    label,
    onSetColor,
    value,
}: ColorListItemProps) {
    const { palette } = useTheme();

    const handleClick = useCallback(() => {
        onSetColor(value);
    }, [onSetColor, value]);

    const style = useMemo(function styleMemo() {
        return {
            color: getColorStyle(value, palette),
        } as const;
    }, [palette, value]);

    return (
        <ListItem
            button
            onClick={handleClick}
        >
            <ListItemText
                primary={label}
                sx={style}
            />
        </ListItem>
    );
});

interface ColorPickerProps {
    color: string;
}
const colorPickerInputStyle = { width: "50px", height: "50px" } as const;
const ColorPicker = memo(forwardRef(function ColorPickerMemo({ color }: ColorPickerProps, ref) {
    const [tempColor, setTempColor] = useState(color);

    useImperativeHandle(ref, () => ({
        getTempColor: () => tempColor,
    }), [tempColor]);

    const handleInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setTempColor(e.target.value);
    }, []);

    const handleTextFieldChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        setTempColor(e.target.value);
    }, []);

    return (
        <Box p={2}>
            <Box mb={2} display="flex" alignItems="center" gap={2}>
                <input
                    type="color"
                    value={tempColor}
                    onChange={handleInputChange}
                    style={colorPickerInputStyle}
                />
                <TextField
                    size="small"
                    value={tempColor}
                    onChange={handleTextFieldChange}
                    placeholder="#000000"
                />
            </Box>
        </Box>
    );
}));

// const CollapseOuter = styled(Box)(() => ({
//     display: "flex",
//     flexDirection: "row",
//     alignItems: "center",
// }));

//TODO collapse stuff should be for whole form section (which we have no way of editing yet), not just header
export function FormHeader({
    element,
    isEditing,
    onUpdate,
    onDelete,
}: FormHeaderProps) {
    const { t } = useTranslation();
    const { palette, typography } = useTheme();

    const getHeaderStyle = useCallback((tag: HeaderTag, color?: string) => {
        return {
            ...((typography[getDisplayTag(tag)]) as object || {}),
            color: getColorStyle(color, palette),
            whiteSpace: "pre-wrap", // Support multiline text
        };
    }, [palette, typography]);

    const [tagAnchorEl, openTagPopover, closeTagPopover, isTagPopoverOpen] = usePopover();
    const [colorAnchorEl, openColorPopover, closeColorPopover, isColorPopoverOpen] = usePopover();

    // const [isCollapsed, setIsCollapsed] = useState(false);
    // const handleToggleCollapsed = useCallback((event: React.MouseEvent) => {
    //     event.stopPropagation();
    //     setIsCollapsed(isCollapsed => !isCollapsed);
    // }, []);

    const updateProp = useCallback(function updatePropCallback(data: Partial<FormHeaderType>) {
        if (!isEditing) {
            return;
        }
        onUpdate(data);
        closeTagPopover();
        closeColorPopover();
    }, [closeColorPopover, closeTagPopover, isEditing, onUpdate]);
    const updateLabel = useCallback(function updateLabelCallback(label: string) {
        updateProp({ label });
    }, [updateProp]);
    const updateDescription = useCallback(function updateDescriptionCallback(description: string) {
        updateProp({ description });
    }, [updateProp]);
    const updateTag = useCallback(function updateTagCallback(tag: FormHeaderType["tag"]) {
        updateProp({ tag });
    }, [updateProp]);
    const updateColor = useCallback(function updateColorCallback(color: string) {
        if (!color) return;
        const normalizedColor = normalizeHexColor(color.toLowerCase().trim());
        if (!isValidColor(normalizedColor)) {
            console.warn(`Invalid color value: ${color}`);
            return;
        }
        updateProp({ color: normalizedColor });
    }, [updateProp]);
    const toggleIsMarkdown = useCallback(function toggleIsMarkdownCallback() {
        updateProp({ isMarkdown: !element.isMarkdown });
    }, [element.isMarkdown, updateProp]);
    // const toggleIsCollapsible = useCallback(function toggleIsCollapsibleCallback() {
    //     updateProp({ isCollapsible: !element.isCollapsible });
    // }, [element.isCollapsible, updateProp]);

    const handleCloseColorPopover = useCallback(() => {
        if (colorPickerRef.current) {
            updateColor(colorPickerRef.current.getTempColor());
        }
        closeColorPopover();
    }, [closeColorPopover, updateColor]);

    const {
        editedLabel,
        handleLabelChange,
        handleLabelKeyDown,
        labelEditRef,
        submitLabelChange,
    } = useEditableLabel({
        isEditable: isEditing,
        isMultiline: element.tag.startsWith("body"),
        label: element.label,
        onUpdate: updateLabel,
    });

    const currentColor = element.color || "default";
    const colorPickerRef = useRef<{ getTempColor: () => string }>(null);
    const labelStyle = useMemo(function labelStyleMemo() {
        return getHeaderStyle(element.tag, element.color);
    }, [element.tag, element.color, getHeaderStyle]);

    const HeaderContent = useMemo(function headerContentMemo() {
        if (!isEditing) {
            if (element.isMarkdown) {
                return (
                    <MarkdownDisplay content={element.label} sx={labelStyle} />
                );
            }
            return (
                <Typography variant={getDisplayTag(element.tag)} sx={labelStyle}>
                    {element.label}
                </Typography>
            );
        }

        const maxRows = element.tag.startsWith("body") ? ROWS_MAX_PARAGRAPH : 1;
        const minRows = element.tag.startsWith("body") ? ROWS_MIN_PARAGRAPH : 1;
        const placeholder = element.isMarkdown ? "Enter markdown..." : "Enter text...";
        function handleMarkdownChange(markdown: string) {
            handleLabelChange({ target: { value: markdown } });
        }
        if (element.isMarkdown) {
            const richInputSxs = {
                root: { flexGrow: 1 },
                textArea: labelStyle,
            } as const;
            return (
                <RichInputBase
                    disableAssistant={true}
                    maxRows={maxRows}
                    minRows={minRows}
                    name="headerText"
                    onBlur={submitLabelChange}
                    onChange={handleMarkdownChange}
                    placeholder={placeholder}
                    value={editedLabel}
                    sxs={richInputSxs}
                />
            );
        }
        const inputProps = {
            disableUnderline: true,
            style: labelStyle,
        } as const;
        return (
            <TextField
                ref={labelEditRef}
                fullWidth
                InputProps={inputProps}
                maxRows={maxRows}
                minRows={minRows}
                multiline={maxRows > 1}
                onBlur={submitLabelChange}
                onChange={handleLabelChange}
                onKeyDown={handleLabelKeyDown}
                placeholder={placeholder}
                value={editedLabel}
                variant="standard"
            />
        );
    }, [editedLabel, element.isMarkdown, element.label, element.tag, handleLabelChange, handleLabelKeyDown, isEditing, labelEditRef, labelStyle, submitLabelChange]);

    const HeaderElement = useMemo(function headerElementMemo() {
        if (!isEditing) {
            // if (element.isCollapsible) {
            //     return (
            //         <CollapseOuter onClick={handleToggleCollapsed}>
            //             <Collapse in={!isCollapsed}>
            //                 {HeaderContent}
            //             </Collapse>
            //             <Tooltip title={t(isCollapsed ? "Expand" : "Shrink")}>
            //                 <IconButton edge="end" color="inherit" aria-label={t(isCollapsed ? "Expand" : "Shrink")}>
            //                     {
            //                         isCollapsed ?
            //                             <ExpandMoreIcon fill={getColorStyle(currentColor, palette)} /> :
            //                             <ExpandLessIcon fill={getColorStyle(currentColor, palette)} />
            //                     }
            //                 </IconButton>
            //             </Tooltip>
            //         </CollapseOuter>
            //     );
            // }
            return (
                <Box display="flex" alignItems="center">
                    {HeaderContent}
                    {Boolean(element.description) && element.description!.trim().length > 0 && <HelpButton
                        markdown={element.description ?? ""}
                    />}
                </Box>
            );
        }

        return (
            <Box display="flex" alignItems="center">
                <Tooltip title={t("Delete")}>
                    <IconButton
                        aria-label={t("Delete")}
                        onClick={onDelete}
                    >
                        <IconCommon
                            decorative
                            fill={palette.error.main}
                            name="Delete"
                            size={24}
                        />
                    </IconButton>
                </Tooltip>
                {HeaderContent}
                <HelpButton
                    onMarkdownChange={updateDescription}
                    markdown={element.description ?? ""}
                />
            </Box>
        );
    }, [HeaderContent, element.description, isEditing, onDelete, palette.error.main, updateDescription]);

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
                    {FORM_HEADER_SIZE_OPTIONS.map((item) => (
                        <SizeListItem
                            key={item.tag}
                            icon={item.iconInfo && <Icon
                                decorative
                                info={item.iconInfo}
                            />}
                            label={item.label}
                            tag={item.tag}
                            onSetTag={updateTag}
                        />
                    ))}
                </List>
            </Popover>
            <Popover
                open={isColorPopoverOpen}
                anchorEl={colorAnchorEl}
                onClose={handleCloseColorPopover}
                anchorOrigin={popoverAnchorOrigin}
            >
                <List>
                    {COLOR_OPTIONS.map((option) => (
                        <ColorListItem
                            key={option.value}
                            label={option.label}
                            onSetColor={updateColor}
                            value={option.value}
                        />
                    ))}
                    <Box borderTop={1} borderColor="divider">
                        {/* <ColorPicker
                            color={currentColor === "default" || COLOR_OPTIONS.some(opt => opt.value === currentColor)
                                ? "#000000"
                                : currentColor}
                            onChange={updateColor}
                        /> */}
                        <ColorPicker
                            ref={colorPickerRef}
                            color={currentColor}
                        />
                    </Box>
                </List>
            </Popover>
            {HeaderElement}
            <FormSettingsButtonRow>
                <Button variant="text" sx={propButtonStyle} onClick={openTagPopover}>
                    Size: {FORM_HEADER_SIZE_OPTIONS.find((item) => item.tag === element.tag)?.label}
                </Button>
                <Button variant="text" sx={propButtonStyle} onClick={openColorPopover}>
                    Color: {getColorStyle(currentColor, palette)}
                </Button>
                <Button variant="text" sx={propButtonStyle} onClick={toggleIsMarkdown}>
                    {element.isMarkdown ? "Text" : "Markdown"}
                </Button>
                {/* <Button variant="text" sx={propButtonStyle} onClick={toggleIsCollapsible}>
                    {element.isCollapsible ? "Not Collapsible" : "Collapsible"}
                </Button> */}
            </FormSettingsButtonRow>
        </div>
    );
}
