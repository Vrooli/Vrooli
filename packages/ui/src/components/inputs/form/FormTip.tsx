import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import IconButton from "@mui/material/IconButton";
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
import { RichInputBase } from "../RichInput/RichInput.js";
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
} {
    let iconInfo: IconInfo | null = null;
    let color: string | null = null;

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
        }
        // Add more URL-based icon logic if needed
    }
    // Default to info icon
    if (!iconInfo) {
        iconInfo = { name: "Info", type: "Common" } as const;
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

    return { iconInfo, color };
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
    const { palette } = useTheme();

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
        const markdownStyle = {
            color: "inherit",
        } as const;
        if (element.isMarkdown) {
            return <MarkdownDisplay content={element.label} variant="caption" sx={markdownStyle} />;
        }
        return (
            <Typography variant="caption" component="span">
                {element.label}
            </Typography>
        );
    }, [element.isMarkdown, element.label]);

    const [iconAnchorEl, openIconPopover, closeIconPopover, isIconPopoverOpen] = usePopover();

    if (!isEditing) {
        const content = (
            <Box display="flex" alignItems="center" color={icon.color}>
                <Box mr={1}>
                    <Icon
                        decorative
                        info={icon.iconInfo}
                    />
                </Box>
                {element.link ? (
                    <StyledLink
                        href={element.link}
                        target="_blank"
                        rel="noopener noreferrer"
                        variant="body2"
                        onClick={openElementLink}
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
                >
                    <IconCommon
                        decorative
                        fill={palette.error.main}
                        name="Delete"
                        size={24}
                    />
                </IconButton>
                <Box mr={1}>
                    <Icon
                        decorative
                        info={icon.iconInfo}
                        size={24}
                    />
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
