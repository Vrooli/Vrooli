import { Box, IconButton, List, ListItem, ListItemIcon, ListItemText, Palette, Popover, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { SessionContext } from "contexts/SessionContext";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useWindowSize } from "hooks/useWindowSize";
import { BoldIcon, CaseSensitiveIcon, Header1Icon, Header2Icon, Header3Icon, HeaderIcon, ItalicIcon, LinkIcon, ListBulletIcon, ListCheckIcon, ListIcon, ListNumberIcon, MagicIcon, RedoIcon, StrikethroughIcon, UnderlineIcon, UndoIcon, WarningIcon } from "icons";
import { useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SxType } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { keyComboToString } from "utils/display/device";

export type RichInputAction =
    "Assistant" |
    "Bold" |
    "Header1" |
    "Header2" |
    "Header3" |
    "Italic" |
    "Link" |
    "ListBullet" |
    "ListCheckbox" |
    "ListNumber" |
    "Mode" |
    "Redo" |
    "Spoiler" |
    "Strikethrough" |
    "Underline" |
    "Undo";

type PopoverActionItem = { action: RichInputAction, icon: React.ReactNode, label: string };

const ToolButton = ({
    disabled,
    icon,
    label,
    onClick,
    palette,
}: {
    disabled?: boolean,
    icon: React.ReactNode,
    label: string,
    onClick: (event: React.MouseEvent<HTMLElement>) => void,
    palette: Palette,
}) => (
    <Tooltip title={label}>
        <IconButton
            disabled={disabled}
            size="small"
            onClick={onClick}
            sx={{
                background: palette.primary.main,
                color: palette.primary.contrastText,
                borderRadius: 2,
            }}
        >
            {icon}
        </IconButton>
    </Tooltip>
);

const ActionPopover = ({
    idPrefix,
    isOpen,
    anchorEl,
    onClose,
    items,
    handleAction,
    palette,
}) => (
    <Popover
        id={`markdown-input-${idPrefix}-popover`}
        open={isOpen}
        anchorEl={anchorEl}
        onClose={onClose}
        anchorOrigin={{
            vertical: "bottom",
            horizontal: "center",
        }}
    >
        <List sx={{ background: palette.background.paper, color: palette.background.contrastText }}>
            {items.map(({ action, icon, label }) => (
                <ListItem button key={action} onClick={() => { handleAction(action); }}>
                    <ListItemIcon>
                        {icon}
                    </ListItemIcon>
                    <ListItemText primary={label} />
                </ListItem>
            ))}
        </List>
    </Popover>
);

const usePopover = (initialState = null) => {
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(initialState);

    const openPopover = (event: React.MouseEvent<HTMLElement>, condition = true) => {
        if (!condition) return;
        setAnchorEl(event.currentTarget);
    };

    const closePopover = () => { setAnchorEl(null); };

    const isPopoverOpen = Boolean(anchorEl);

    return [anchorEl, openPopover, closePopover, isPopoverOpen] as const;
};

export const RichInputToolbar = ({
    canRedo,
    canUndo,
    disableAssistant = false,
    disabled = false,
    handleAction,
    isMarkdownOn,
    name,
    sx,
}: {
    canRedo: boolean;
    canUndo: boolean;
    disableAssistant?: boolean;
    disabled?: boolean;
    handleAction: (action: RichInputAction) => unknown;
    isMarkdownOn: boolean;
    name: string,
    sx?: SxType;
}) => {
    const { breakpoints, palette } = useTheme();
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { hasPremium } = useMemo(() => getCurrentUser(session), [session]);
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const isLeftHanded = useIsLeftHanded();

    const [headerAnchorEl, openHeaderSelect, closeHeader, headerSelectOpen] = usePopover();
    const [formatAnchorEl, openFormatSelect, closeFormat, formatSelectOpen] = usePopover();
    const [listAnchorEl, openListSelect, closeList, listSelectOpen] = usePopover();

    const headerItems: PopoverActionItem[] = [
        { action: "Header1", icon: <Header1Icon />, label: `${t("Header1")} (${keyComboToString("Alt", "1")})` },
        { action: "Header2", icon: <Header2Icon />, label: `${t("Header2")} (${keyComboToString("Alt", "2")})` },
        { action: "Header3", icon: <Header3Icon />, label: `${t("Header3")} (${keyComboToString("Alt", "3")})` },
    ];
    const formatItems: PopoverActionItem[] = [
        { action: "Bold", icon: <BoldIcon />, label: `${t("Bold")} (${keyComboToString("Ctrl", "b")})` },
        { action: "Italic", icon: <ItalicIcon />, label: `${t("Italic")} (${keyComboToString("Ctrl", "i")})` },
        { action: "Underline", icon: <UnderlineIcon />, label: `${t("Underline")} (${keyComboToString("Ctrl", "u")})` },
        { action: "Strikethrough", icon: <StrikethroughIcon />, label: `${t("Strikethrough")} (${keyComboToString("Ctrl", "Shift", "s")})` },
        { action: "Spoiler", icon: <WarningIcon />, label: `${t("Spoiler")} (${keyComboToString("Ctrl", "l")})` },
    ];
    const listItems: PopoverActionItem[] = [
        { action: "ListBullet", icon: <ListBulletIcon />, label: `${t("ListBulleted")} (${keyComboToString("Alt", "4")})` },
        { action: "ListNumber", icon: <ListNumberIcon />, label: `${t("ListNumbered")} (${keyComboToString("Alt", "5")})` },
        { action: "ListCheckbox", icon: <ListCheckIcon />, label: `${t("ListCheckbox")} (${keyComboToString("Alt", "6")})` },
    ];

    return (
        <Box sx={{
            display: "flex",
            flexDirection: (isLeftHanded || !isMobile) ? "row" : "row-reverse",
            width: "100%",
            padding: "0.5rem",
            background: palette.primary.main,
            color: palette.primary.contrastText,
            borderRadius: "0.5rem 0.5rem 0 0",
            ...sx,
        }}>
            {/* Group of main editor controls including AI assistant and formatting tools */}
            <Stack
                direction="row"
                spacing={{ xs: 0, sm: 0.5, md: 1 }}
                sx={{
                    ...((isLeftHanded || !isMobile) ? { marginRight: "auto" } : { marginLeft: "auto" }),
                    visibility: disabled ? "hidden" : "visible",
                }}
            >
                {hasPremium && !disableAssistant && <ToolButton
                    icon={<MagicIcon fill={palette.primary.contrastText} />}
                    label={t("AIAssistant")}
                    onClick={() => handleAction("Assistant")}
                    palette={palette}
                />}
                <ToolButton
                    disabled={disabled}
                    icon={<HeaderIcon fill={palette.primary.contrastText} />}
                    label={t("HeaderInsert")}
                    onClick={openHeaderSelect}
                    palette={palette}
                />
                <ActionPopover
                    idPrefix="header"
                    isOpen={headerSelectOpen}
                    anchorEl={headerAnchorEl}
                    onClose={closeHeader}
                    items={headerItems}
                    handleAction={handleAction}
                    palette={palette}
                />
                {/* Display format as popover on mobile, or display all options on desktop */}
                {isMobile && (
                    <>
                        <ToolButton
                            disabled={disabled}
                            icon={<CaseSensitiveIcon fill={palette.primary.contrastText} />}
                            label={t("TextFormat")}
                            onClick={openFormatSelect}
                            palette={palette}
                        />
                        <ActionPopover
                            idPrefix="format"
                            isOpen={formatSelectOpen}
                            anchorEl={formatAnchorEl}
                            onClose={closeFormat}
                            items={formatItems}
                            handleAction={handleAction}
                            palette={palette}
                        />
                    </>
                )}
                {!isMobile && formatItems.map(({ action, icon, label }) => (
                    <ToolButton
                        key={action}
                        disabled={disabled}
                        icon={icon}
                        label={label}
                        onClick={() => { handleAction(action); }}
                        palette={palette}
                    />
                ))}
                <ToolButton
                    disabled={disabled}
                    icon={<ListIcon fill={palette.primary.contrastText} />}
                    label={t("ListInsert")}
                    onClick={openListSelect}
                    palette={palette}
                />
                <ActionPopover
                    idPrefix="list"
                    isOpen={listSelectOpen}
                    anchorEl={listAnchorEl}
                    onClose={closeList}
                    items={listItems}
                    handleAction={handleAction}
                    palette={palette}
                />
                <ToolButton
                    disabled={disabled}
                    icon={<LinkIcon fill={palette.primary.contrastText} />}
                    label={`${t("Link")} (${keyComboToString("Ctrl", "k")})`}
                    onClick={() => { handleAction("Link"); }}
                    palette={palette}
                />
            </Stack>
            {/* Group for undo, redo, and switching between Markdown as WYSIWYG */}
            <Box sx={{
                display: "flex",
                flexDirection: "row",
                alignItems: "center",
                gap: { xs: 0, sm: 0.5, md: 1 },
            }}>
                {(canUndo || canRedo) && <ToolButton
                    disabled={disabled || !canUndo}
                    icon={<UndoIcon fill={palette.primary.contrastText} />}
                    label={`${t("Undo")} (${keyComboToString("Ctrl", "z")})`}
                    onClick={() => { handleAction("Undo"); }}
                    palette={palette}
                />}
                {(canUndo || canRedo) && <ToolButton
                    disabled={disabled || !canRedo}
                    icon={<RedoIcon fill={palette.primary.contrastText} />}
                    label={`${t("Redo")} (${keyComboToString("Ctrl", "y")})`}
                    onClick={() => { handleAction("Redo"); }}
                    palette={palette}
                />}
                <Tooltip title={!isMarkdownOn ? `${t("PressToMarkdown")} (${keyComboToString("Alt", "7")})` : `${t("PressToPreview")} (${keyComboToString("Alt", "7")})`} placement="top">
                    <Typography variant="body2" onClick={() => { handleAction("Mode"); }} sx={{
                        cursor: "pointer",
                        margin: "auto",
                        padding: 1,
                        background: palette.primary.main,
                        borderRadius: 2,
                    }}>
                        {!isMarkdownOn ? t("MarkdownTo") : t("PreviewTo")}
                    </Typography>
                </Tooltip>
            </Box>
        </Box>
    );
};
