import { Box, IconButton, Palette, Popover, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { SessionContext } from "contexts/SessionContext";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useWindowSize } from "hooks/useWindowSize";
import { BoldIcon, Header1Icon, Header2Icon, Header3Icon, HeaderIcon, ItalicIcon, LinkIcon, ListBulletIcon, ListCheckIcon, ListIcon, ListNumberIcon, MagicIcon, RedoIcon, StrikethroughIcon, UndoIcon } from "icons";
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
    "Strikethrough" |
    "Undo";

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
            sx={{ background: palette.primary.main, borderRadius: 2 }}
        >
            {icon}
        </IconButton>
    </Tooltip>
);

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

    const [headerAnchorEl, setHeaderAnchorEl] = useState<HTMLElement | null>(null);
    const openHeaderSelect = (event: React.MouseEvent<HTMLElement>) => {
        if (disabled) return;
        setHeaderAnchorEl(event.currentTarget);
    };
    const closeHeader = () => { setHeaderAnchorEl(null); };
    const headerSelectOpen = Boolean(headerAnchorEl);

    const [listAnchorEl, setListAnchorEl] = useState<HTMLElement | null>(null);
    const openListSelect = (event: React.MouseEvent<HTMLElement>) => {
        if (disabled) return;
        setListAnchorEl(event.currentTarget);
    };
    const closeList = () => { setListAnchorEl(null); };
    const listSelectOpen = Boolean(listAnchorEl);

    const headerItems: { action: RichInputAction, icon: React.ReactNode, label: string }[] = [
        { action: "Header1", icon: <Header1Icon fill={palette.primary.contrastText} />, label: `${t("Header1")} (${keyComboToString("Alt", "1")})` },
        { action: "Header2", icon: <Header2Icon fill={palette.primary.contrastText} />, label: `${t("Header2")} (${keyComboToString("Alt", "2")})` },
        { action: "Header3", icon: <Header3Icon fill={palette.primary.contrastText} />, label: `${t("Header3")} (${keyComboToString("Alt", "3")})` },
    ];

    const listItems: { action: RichInputAction, icon: React.ReactNode, label: string }[] = [
        { action: "ListBullet", icon: <ListBulletIcon fill={palette.primary.contrastText} />, label: `${t("ListBulleted")} (${keyComboToString("Alt", "4")})` },
        { action: "ListNumber", icon: <ListNumberIcon fill={palette.primary.contrastText} />, label: `${t("ListNumbered")} (${keyComboToString("Alt", "5")})` },
        { action: "ListCheckbox", icon: <ListCheckIcon fill={palette.primary.contrastText} />, label: `${t("ListCheckbox")} (${keyComboToString("Alt", "6")})` },
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
                <Popover
                    id={`markdown-input-header-popover-${name}`}
                    open={headerSelectOpen}
                    anchorEl={headerAnchorEl}
                    onClose={closeHeader}
                    anchorOrigin={{
                        vertical: "bottom",
                        horizontal: "center",
                    }}
                >
                    <Stack direction="row" spacing={0} sx={{
                        background: palette.primary.main,
                        color: palette.primary.contrastText,
                    }}>
                        {headerItems.map(({ action, icon, label }) => (
                            <ToolButton
                                key={action}
                                icon={icon}
                                label={label}
                                onClick={() => { handleAction(action); }}
                                palette={palette}
                            />
                        ))}
                    </Stack>
                </Popover>
                <ToolButton
                    disabled={disabled}
                    icon={<BoldIcon fill={palette.primary.contrastText} />}
                    label={`${t("Bold")} (${keyComboToString("Ctrl", "b")})`}
                    onClick={() => { handleAction("Bold"); }}
                    palette={palette}
                />
                <ToolButton
                    disabled={disabled}
                    icon={<ItalicIcon fill={palette.primary.contrastText} />}
                    label={`${t("Italic")} (${keyComboToString("Ctrl", "i")})`}
                    onClick={() => { handleAction("Italic"); }}
                    palette={palette}
                />
                <ToolButton
                    disabled={disabled}
                    icon={<StrikethroughIcon fill={palette.primary.contrastText} />}
                    label={`${t("Strikethrough")} (${keyComboToString("Ctrl", "Shift", "s")})`}
                    onClick={() => { handleAction("Strikethrough"); }}
                    palette={palette}
                />
                <ToolButton
                    disabled={disabled}
                    icon={<ListIcon fill={palette.primary.contrastText} />}
                    label={t("ListInsert")}
                    onClick={openListSelect}
                    palette={palette}
                />
                <Popover
                    id={`markdown-input-list-popover-${name}`}
                    open={listSelectOpen}
                    anchorEl={listAnchorEl}
                    onClose={closeList}
                    anchorOrigin={{
                        vertical: "bottom",
                        horizontal: "center",
                    }}
                >
                    <Stack direction="row" spacing={0} sx={{
                        background: palette.primary.main,
                        color: palette.primary.contrastText,
                    }}>
                        {listItems.map(({ action, icon, label }) => (
                            <ToolButton
                                key={action}
                                icon={icon}
                                label={label}
                                onClick={() => { handleAction(action); }}
                                palette={palette}
                            />
                        ))}
                    </Stack>
                </Popover>
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
