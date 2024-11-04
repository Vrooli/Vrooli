import { FormTipType } from "@local/shared";
import { Box, Button, IconButton, Link, List, ListItem, ListItemIcon, ListItemText, Palette, Popover, TextField, Typography, styled, useTheme } from "@mui/material";
import { RichInputBase } from "components/inputs/RichInput/RichInput";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { useEditableLabel } from "hooks/useEditableLabel";
import { usePopover } from "hooks/usePopover";
import { DeleteIcon, ErrorIcon, InfoIcon, PlayIcon, WarningIcon } from "icons";
import { useCallback, useMemo } from "react";
import { PubSub } from "utils/pubsub";
import { FormSettingsButtonRow, propButtonStyle } from "../styles";
import { FormTipProps } from "../types";

const TIP_ICON_OPTIONS = [
    { icon: <ErrorIcon />, label: "Error", value: "Error" },
    { icon: <InfoIcon />, label: "Info", value: "Info" },
    { icon: <WarningIcon />, label: "Warning", value: "Warning" },
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
    icon: React.ReactNode,
    color: string,
} {
    let icon: React.ReactNode | null = null;
    let color: string | null = null;

    // Handle custom icons
    if (element.icon) {
        const iconOption = TIP_ICON_OPTIONS.find((option) => option.value === element.icon);
        if (iconOption) {
            icon = iconOption.icon;
        }
    }
    // Handle link-derived icons
    if (!icon && element.link) {
        if (isYouTubeLink(element.link)) {
            icon = <PlayIcon />;
            color = palette.mode === "light" ? "#001cd3" : "#dd86db";
        }
        // Add more URL-based icon logic if needed
    }
    // Default to info icon
    if (!icon) {
        icon = <InfoIcon />;
    }

    if (element.icon === "Warning") {
        color = palette.warning.main;
    } else if (element.icon === "Info") {
        color = palette.background.textSecondary;
    } else if (element.icon === "Error") {
        color = palette.error.main;
    }
    if (!color) {
        color = palette.primary.main;
    }

    return { icon, color };
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
    const { palette } = useTheme();

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

    const iconInfo = getIconForTip(element, palette);

    const TipContent = useMemo(() => {
        const markdownStyle = {
            color: "inherit",
        } as const;
        if (element.isMarkdown) {
            return <MarkdownDisplay content={element.label} variant="body2" sx={markdownStyle} />;
        }
        return (
            <Typography variant="body2" component="span">
                {element.label}
            </Typography>
        );
    }, [element.isMarkdown, element.label]);

    const [iconAnchorEl, openIconPopover, closeIconPopover, isIconPopoverOpen] = usePopover();

    if (!isEditing) {
        const content = (
            <Box display="flex" alignItems="center" color={iconInfo.color}>
                <Box mr={1}>
                    {iconInfo.icon}
                </Box>
                {element.link ? (
                    <StyledLink
                        href={element.link}
                        target="_blank"
                        rel="noopener noreferrer"
                        variant="body2"
                        onClick={(e) => handleLinkClick(element.link!, e)}
                    >
                        {TipContent}
                    </StyledLink>
                ) : (
                    TipContent
                )}
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
        <div>
            <Popover
                open={isIconPopoverOpen}
                anchorEl={iconAnchorEl}
                onClose={closeIconPopover}
                anchorOrigin={popoverAnchorOrigin}
            >
                <List>
                    {TIP_ICON_OPTIONS.map((option) => {
                        function handleClick() {
                            updateIcon(option.value);
                            closeIconPopover();
                        }

                        return (
                            <ListItem button key={option.value} onClick={handleClick}>
                                <ListItemIcon>{option.icon}</ListItemIcon>
                                <ListItemText primary={option.label} />
                            </ListItem>
                        );
                    })}
                </List>
            </Popover>
            <Box display="flex" alignItems="center" color={iconInfo.color}>
                <IconButton onClick={onDelete}>
                    <DeleteIcon fill={palette.error.main} width="24px" height="24px" />
                </IconButton>
                <Box mr={1}>
                    {iconInfo.icon}
                </Box>
                {element.isMarkdown ? (
                    <RichInputBase
                        disableAssistant={true}
                        name="tipText"
                        onBlur={submitLabelChange}
                        onChange={handleMarkdownChange}
                        placeholder="Enter markdown..."
                        value={editedLabel}
                        sxs={richInputSxs}
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
            />
            <FormSettingsButtonRow>
                <Button variant="text" sx={propButtonStyle} onClick={openIconPopover}>
                    Icon:{" "}
                    {
                        TIP_ICON_OPTIONS.find((option) => option.value === element.icon)
                            ?.label || "Default"
                    }
                </Button>
                <Button variant="text" sx={propButtonStyle} onClick={toggleIsMarkdown}>
                    {element.isMarkdown ? "Text" : "Markdown"}
                </Button>
            </FormSettingsButtonRow>
        </div>
    );
}
