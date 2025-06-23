import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import { IconButton } from "../../buttons/IconButton.js";
import Link from "@mui/material/Link";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import Popover from "@mui/material/Popover";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { styled } from "@mui/material/styles";
import { useTheme } from "@mui/material/styles";
import type { Palette } from "@mui/material/styles";
import { type FormTipType } from "@vrooli/shared";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useEditableLabel } from "../../../hooks/useEditableLabel.js";
import { usePopover } from "../../../hooks/usePopover.js";
import { Icon, IconCommon, type IconInfo } from "../../../icons/Icons.js";
import { PubSub } from "../../../utils/pubsub.js";
import { MarkdownDisplay } from "../../text/MarkdownDisplay.js";
import { AdvancedInputBase } from "../AdvancedInput/AdvancedInput.js";
import { FormSettingsButtonRow, propButtonStyle } from "./styles.js";
import { type FormTipProps } from "./types.js";

const TIP_ICON_OPTIONS = [
    { iconInfo: { name: "Error", type: "Common" }, label: "Error", value: "Error" },
    { iconInfo: { name: "Info", type: "Common" }, label: "Info", value: "Info" },
    { iconInfo: { name: "Warning", type: "Common" }, label: "Warning", value: "Warning" },
] as const;

function isYouTubeLink(link: string): boolean {
    return link.includes("youtube.com") || link.includes("youtu.be");
}

function handleLinkClick(link: string, event: React.MouseEvent) {
    if (isYouTubeLink(link)) {
        event.preventDefault();
        PubSub.get().publish("popupVideo", { src: link });
    }
}

function getIconForTip(element: FormTipType, palette: Palette): {
    iconInfo: IconInfo,
    color: string,
    borderColor: string,
} {
    let iconInfo: IconInfo | null = null;
    let color: string | null = null;
    let borderColor: string | null = null;

    // Handle custom icons
    if (element.icon) {
        const iconOption = TIP_ICON_OPTIONS.find((option) => option.value === element.icon);
        if (iconOption) {
            iconInfo = iconOption.iconInfo;
        }
    }
    // Handle link-derived icons
    if (!iconInfo && element.link) {
        if (isYouTubeLink(element.link)) {
            iconInfo = { name: "Play", type: "Common" } as const;
            color = palette.mode === "light" ? "#001cd3" : "#dd86db";
            borderColor = palette.divider;
        }
        // Add more URL-based icon logic if needed
    }
    // Default to info icon
    if (!iconInfo) {
        iconInfo = { name: "Info", type: "Common" } as const;
    }

    if (element.icon === "Warning") {
        color = palette.warning.main;
        borderColor = palette.warning.light;
    } else if (element.icon === "Info") {
        color = palette.info.main;
        borderColor = palette.info.light;
    } else if (element.icon === "Error") {
        color = palette.error.main;
        borderColor = palette.error.light;
    }
    if (!color) {
        color = palette.primary.main;
    }
    if (!borderColor) {
        borderColor = palette.divider;
    }

    return { iconInfo, color, borderColor };
}

const StyledLink = styled(Link)(({ theme }) => ({
    color: theme.palette.mode === "light" ? "#001cd3" : "#dd86db",
}));

const popoverAnchorOrigin = { vertical: "bottom", horizontal: "center" } as const;

export function FormTip({
    element,
    isEditing,
    onDelete,
    onUpdate,
}: FormTipProps) {
    const { t } = useTranslation();
    const { palette, spacing, shape } = useTheme();

    function openElementLink(event: React.MouseEvent) {
        if (!element.link) {
            console.error("No link to open");
            return;
        }
        handleLinkClick(element.link, event);
    }

    const updateProp = useCallback(
        function updatePropCallback(data: Partial<FormTipType>) {
            if (!isEditing) {
                return;
            }
            onUpdate(data);
        },
        [isEditing, onUpdate],
    );

    const updateLabel = useCallback(
        function updateLabelCallback(label: string) {
            updateProp({ label });
        },
        [updateProp],
    );

    const updateLink = useCallback(function updateLinkCallback(event: React.ChangeEvent<HTMLInputElement>) {
        const link = event.target.value;
        updateProp({ link });
    }, [updateProp]);

    const updateIcon = useCallback(
        function updateIconCallback(icon: FormTipType["icon"]) {
            updateProp({ icon });
        },
        [updateProp],
    );

    const toggleIsMarkdown = useCallback(
        function toggleIsMarkdownCallback() {
            updateProp({ isMarkdown: !element.isMarkdown });
        },
        [element.isMarkdown, updateProp],
    );

    const {
        editedLabel,
        handleLabelChange,
        handleLabelKeyDown,
        labelEditRef,
        submitLabelChange,
    } = useEditableLabel({
        isEditable: isEditing,
        isMultiline: false,
        label: element.label,
        onUpdate: updateLabel,
    });

    const icon = getIconForTip(element, palette);

    const TipContent = useMemo(() => {
        if (element.isMarkdown) {
            return (
                <MarkdownDisplay 
                    content={element.label} 
                    sx={{ 
                        "& > *:first-of-type": { 
                            marginTop: 0, 
                        },
                        "& > *:last-child": { 
                            marginBottom: 0, 
                        },
                        "& p": {
                            fontSize: "0.75rem",
                            lineHeight: 1.5,
                            letterSpacing: "0.03333em",
                        },
                        "& li": {
                            fontSize: "0.75rem",
                            lineHeight: 1.5,
                        },
                    }} 
                />
            );
        }
        return (
            <Typography variant="caption" component="span" sx={{ fontSize: "0.75rem" }}>
                {element.label}
            </Typography>
        );
    }, [element.isMarkdown, element.label]);

    const [iconAnchorEl, openIconPopover, closeIconPopover, isIconPopoverOpen] = usePopover();

    if (!isEditing) {
        const tipStyle = {
            display: "flex",
            alignItems: "center",
            padding: spacing(1.5, 2),
            borderRadius: shape.borderRadius,
            border: `1px solid ${icon.borderColor}`,
            backgroundColor: 
                element.icon === "Error" ? `${palette.error.main}08` : 
                element.icon === "Warning" ? `${palette.warning.main}08` : 
                element.icon === "Info" ? `${palette.info.main}08` : 
                palette.action.hover,
        };
        
        const content = (
            <Box 
                sx={tipStyle}
                data-testid="form-tip"
                data-editing="false"
                data-icon={element.icon || "Info"}
                data-has-link={element.link ? "true" : "false"}
                data-is-markdown={element.isMarkdown ? "true" : "false"}
                role="note"
                aria-label={`Tip: ${element.label}`}
            >
                <Box 
                    sx={{ 
                        mr: 1.5, 
                        display: "flex",
                        alignItems: "center",
                        color: icon.color,
                    }}
                    data-testid="tip-icon"
                >
                    <Icon
                        decorative
                        info={icon.iconInfo}
                        size={20}
                    />
                </Box>
                <Box sx={{ 
                    flex: 1,
                    display: "flex",
                    flexDirection: "column",
                    justifyContent: "center",
                }}>
                    {element.link ? (
                        <StyledLink
                            href={element.link}
                            target="_blank"
                            rel="noopener noreferrer"
                            variant="body2"
                            onClick={openElementLink}
                            sx={{ 
                                display: "block",
                                textDecoration: "none",
                                "&:hover": {
                                    textDecoration: "underline",
                                },
                            }}
                            data-testid="tip-link"
                            aria-label={`Tip link: ${element.label}`}
                        >
                            {TipContent}
                        </StyledLink>
                    ) : (
                        TipContent
                    )}
                </Box>
            </Box>
        );
        return content;
    }

    // Editing state
    function handleMarkdownChange(markdown: string) {
        handleLabelChange({ target: { value: markdown } });
    }
    const richInputSxs = {
        root: { flexGrow: 1 },
    } as const;
    return (
        <div 
            data-testid="form-tip"
            data-editing="true"
            data-icon={element.icon || "Info"}
            data-has-link={element.link ? "true" : "false"}
            data-is-markdown={element.isMarkdown ? "true" : "false"}
            role="group"
            aria-label="Edit tip"
        >
            <Popover
                open={isIconPopoverOpen}
                anchorEl={iconAnchorEl}
                onClose={closeIconPopover}
                anchorOrigin={popoverAnchorOrigin}
                data-testid="icon-popover"
            >
                <List>
                    {TIP_ICON_OPTIONS.map((option) => {
                        function handleClick() {
                            updateIcon(option.value);
                            closeIconPopover();
                        }

                        return (
                            <ListItem 
                                button 
                                key={option.value} 
                                onClick={handleClick}
                                data-testid={`icon-option-${option.value}`}
                                aria-label={`Select ${option.label} icon`}
                            >
                                <ListItemIcon>
                                    <Icon
                                        decorative
                                        info={option.iconInfo}
                                    />
                                </ListItemIcon>
                                <ListItemText primary={option.label} />
                            </ListItem>
                        );
                    })}
                </List>
            </Popover>
            <Box display="flex" alignItems="center" color={icon.color}>
                <IconButton
                    aria-label={t("Delete")}
                    onClick={onDelete}
                    data-testid="delete-button"
                >
                    <IconCommon
                        decorative
                        fill={palette.error.main}
                        name="Delete"
                        size={24}
                    />
                </IconButton>
                <Box mr={1} data-testid="tip-icon-edit">
                    <Icon
                        decorative
                        info={icon.iconInfo}
                        size={24}
                    />
                </Box>
                {element.isMarkdown ? (
                    <AdvancedInputBase
                        disableAssistant={true}
                        name="tipText"
                        onBlur={submitLabelChange}
                        onChange={handleMarkdownChange}
                        placeholder="Enter markdown..."
                        value={editedLabel}
                        sxs={richInputSxs}
                        data-testid="markdown-input"
                    />
                ) : (
                    <TextField
                        inputRef={labelEditRef}
                        value={editedLabel}
                        onChange={handleLabelChange}
                        onBlur={submitLabelChange}
                        onKeyDown={handleLabelKeyDown}
                        placeholder="Enter tip text..."
                        variant="standard"
                        fullWidth
                        data-testid="text-input"
                        aria-label="Tip text"
                    />
                )}
            </Box>
            <TextField
                value={element.link || ""}
                onChange={updateLink}
                placeholder="Enter URL (optional)"
                variant="standard"
                fullWidth
                margin="normal"
                data-testid="link-input"
                aria-label="Tip link URL"
            />
            <FormSettingsButtonRow>
                <Button 
                    variant="text" 
                    sx={propButtonStyle} 
                    onClick={openIconPopover}
                    data-testid="icon-button"
                    aria-label={`Icon type: ${TIP_ICON_OPTIONS.find((option) => option.value === element.icon)?.label || "Default"}`}
                >
                    Icon:{" "}
                    {
                        TIP_ICON_OPTIONS.find((option) => option.value === element.icon)
                            ?.label || "Default"
                    }
                </Button>
                <Button 
                    variant="text" 
                    sx={propButtonStyle} 
                    onClick={toggleIsMarkdown}
                    data-testid="toggle-markdown-button"
                    aria-label={`Switch to ${element.isMarkdown ? "text" : "markdown"} mode`}
                >
                    {element.isMarkdown ? "Text" : "Markdown"}
                </Button>
            </FormSettingsButtonRow>
        </div>
    );
}
