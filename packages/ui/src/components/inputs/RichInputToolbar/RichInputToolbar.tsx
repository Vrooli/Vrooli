import { noop } from "@local/shared";
import { Box, Button, IconButton, List, ListItem, ListItemIcon, ListItemText, Palette, Popover, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { SessionContext } from "contexts/SessionContext";
import { useDimensions } from "hooks/useDimensions";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { BoldIcon, CaseSensitiveIcon, Header1Icon, Header2Icon, Header3Icon, Header4Icon, Header5Icon, Header6Icon, HeaderIcon, ItalicIcon, LinkIcon, ListBulletIcon, ListCheckIcon, ListIcon, ListNumberIcon, MagicIcon, RedoIcon, StrikethroughIcon, TableIcon, UnderlineIcon, UndoIcon, WarningIcon } from "icons";
import { forwardRef, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { SxType } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { keyComboToString } from "utils/display/device";
import { RichInputAction, RichInputActiveStates } from "../types";

type PopoverActionItem = { action: RichInputAction, icon: React.ReactNode, label: string };

/** Determines how many options should be displayed directly */
type ViewSize = "minimal" | "partial" | "full";

export const defaultActiveStates: Omit<RichInputActiveStates, "SetValue"> = {
    Bold: false,
    Code: false,
    Header1: false,
    Header2: false,
    Header3: false,
    Header4: false,
    Header5: false,
    Header6: false,
    Italic: false,
    Link: false,
    ListBullet: false,
    ListNumber: false,
    ListCheckbox: false,
    Quote: false,
    Spoiler: false,
    Strikethrough: false,
    Table: false,
    Underline: false,
};

const ToolButton = forwardRef(({
    disabled,
    icon,
    isActive,
    label,
    onClick,
    palette,
}: {
    disabled?: boolean,
    icon: React.ReactNode,
    isActive?: boolean,
    label: string,
    onClick: (event: React.MouseEvent<HTMLElement>) => unknown,
    palette: Palette,
}, ref: React.Ref<HTMLButtonElement>) => (
    <Tooltip title={label}>
        <IconButton
            ref={ref}
            disabled={disabled}
            size="small"
            onClick={onClick}
            sx={{
                background: isActive ? palette.secondary.main : palette.primary.main,
                color: palette.primary.contrastText,
                borderRadius: 2,
            }}
        >
            {icon}
        </IconButton>
    </Tooltip>
));

const ActionPopover = ({
    activeStates,
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
                <ListItem
                    button
                    key={action}
                    onClick={() => { handleAction(action); }}
                    sx={{
                        background: activeStates[action] ? palette.secondary.light : "transparent",
                    }}
                >
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
        console.log("in open popover", event, condition);
        if (!condition) return;
        setAnchorEl(event.currentTarget);
    };

    const closePopover = () => { setAnchorEl(null); };

    const isPopoverOpen = Boolean(anchorEl);

    return [anchorEl, openPopover, closePopover, isPopoverOpen] as const;
};

const TablePopover = ({ isOpen, anchorEl, onClose, handleTableInsert, palette, t }) => {
    const [hoveredRow, setHoveredRow] = useState(1);
    const [hoveredCol, setHoveredCol] = useState(1);
    const [canHover, setCanHover] = useState(true);

    const maxRows = 10;
    const maxCols = 5;

    useEffect(() => {
        if (window.matchMedia("(hover: none)").matches) {
            setCanHover(false);
        }
    }, []);

    return (
        <Popover
            open={isOpen}
            anchorEl={anchorEl}
            onClose={onClose}
            anchorOrigin={{ vertical: "bottom", horizontal: "center" }}
        >
            <Stack
                direction="column"
                justifyContent="center"
                alignItems="center"
                spacing={1}
                p={2}
                sx={{ background: palette.background.paper, color: palette.background.contrastText }}
            >
                <Typography align="center" variant="h6" pb={1}>
                    {t("TableSelectSize")}
                </Typography>
                <Box>
                    {[...Array(maxRows)].map((_, rowIndex) => (
                        <Box key={rowIndex} display="flex">
                            {[...Array(maxCols)].map((_, colIndex) => (
                                <Box
                                    key={colIndex}
                                    component="button"
                                    onMouseEnter={canHover ? () => {
                                        setHoveredRow(rowIndex + 1);
                                        setHoveredCol(colIndex + 1);
                                    } : noop}
                                    onClick={() => {
                                        setHoveredRow(rowIndex + 1);
                                        setHoveredCol(colIndex + 1);
                                        // Submit immediately if the user can hover
                                        if (canHover) {
                                            handleTableInsert(rowIndex + 1, colIndex + 1);
                                        }
                                    }}
                                    sx={{
                                        width: 25,
                                        height: 25,
                                        border: `1px solid ${palette.divider}`,
                                        background: (rowIndex < hoveredRow && colIndex < hoveredCol) ?
                                            palette.secondary.light :
                                            "transparent",
                                        cursor: "pointer",
                                    }}
                                />
                            ))}
                        </Box>
                    ))}
                </Box>
                <Typography align="center" variant="body2" sx={{ marginTop: 1 }}>
                    {hoveredRow} x {hoveredCol}
                </Typography>
                {!canHover && (
                    <Button variant="contained" onClick={() => handleTableInsert(hoveredRow, hoveredCol)} sx={{ marginTop: 2 }}>
                        {t("Submit")}
                    </Button>
                )}
            </Stack>
        </Popover>
    );
};

export const RichInputToolbar = ({
    activeStates,
    canRedo,
    canUndo,
    disableAssistant = false,
    disabled = false,
    handleAction,
    handleActiveStatesChange,
    isMarkdownOn,
    name,
    sx,
}: {
    activeStates: Omit<RichInputActiveStates, "SetValue">;
    canRedo: boolean;
    canUndo: boolean;
    disableAssistant?: boolean;
    disabled?: boolean;
    handleAction: (action: RichInputAction, data?: unknown) => unknown;
    handleActiveStatesChange: (activeStates: Omit<RichInputActiveStates, "SetValue">) => unknown;
    isMarkdownOn: boolean;
    name: string,
    sx?: SxType;
}) => {
    const { breakpoints, palette } = useTheme();
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { hasPremium } = useMemo(() => getCurrentUser(session), [session]);
    const { dimensions, fromDims, ref: dimRef } = useDimensions();
    const viewSize = useMemo<ViewSize>(() => {
        if (dimensions.width <= 375) return "minimal";
        if (dimensions.width <= breakpoints.values.sm) return "partial";
        return "full";
    }, [breakpoints, dimensions]);
    const isLeftHanded = useIsLeftHanded();

    const handleToggleAction = (action: string, data?: unknown) => {
        // Update action's active state, if we're not using markdown mode
        if (!isMarkdownOn && action in activeStates) {
            handleActiveStatesChange({
                ...activeStates,
                [action]: !activeStates[action],
            });
        }
        // Trigger handleAction
        handleAction(action as RichInputAction, data);
    };
    useEffect(() => {
        if (isMarkdownOn) {
            handleActiveStatesChange(defaultActiveStates);
        }
    }, [handleActiveStatesChange, isMarkdownOn]);

    const [headerAnchorEl, openHeaderSelect, closeHeader, headerSelectOpen] = usePopover();
    const [formatAnchorEl, openFormatSelect, closeFormat, formatSelectOpen] = usePopover();
    const [listAnchorEl, openListSelect, closeList, listSelectOpen] = usePopover();
    const [tableAnchorEl, openTableSelect, closeTable, tableSelectOpen] = usePopover();
    const handleTableInsert = (rows: number, cols: number) => {
        handleToggleAction("Table", { rows, cols });
        closeTable();
    };
    const combinedButtonRef = useRef(null);
    const [combinedAnchorEl, openCombinedSelect, closeCombined, combinedSelectOpen] = usePopover();
    const handleCombinedAction = (action: string, data?: unknown) => {
        if (action === "Table") {
            closeCombined();
            const syntheticEvent = { currentTarget: combinedButtonRef.current };
            openTableSelect(syntheticEvent as any, true);
        } else {
            handleToggleAction(action, data);
        }
    };

    const headerItems: PopoverActionItem[] = [
        { action: "Header1", icon: <Header1Icon />, label: `${t("Header1")} (${keyComboToString("Alt", "1")})` },
        { action: "Header2", icon: <Header2Icon />, label: `${t("Header2")} (${keyComboToString("Alt", "2")})` },
        { action: "Header3", icon: <Header3Icon />, label: `${t("Header3")} (${keyComboToString("Alt", "3")})` },
        { action: "Header4", icon: <Header4Icon />, label: `${t("Header4")} (${keyComboToString("Alt", "4")})` },
        { action: "Header5", icon: <Header5Icon />, label: `${t("Header5")} (${keyComboToString("Alt", "5")})` },
        { action: "Header6", icon: <Header6Icon />, label: `${t("Header6")} (${keyComboToString("Alt", "6")})` },
    ];
    const formatItems: PopoverActionItem[] = [
        { action: "Bold", icon: <BoldIcon />, label: `${t("Bold")} (${keyComboToString("Ctrl", "b")})` },
        { action: "Italic", icon: <ItalicIcon />, label: `${t("Italic")} (${keyComboToString("Ctrl", "i")})` },
        { action: "Underline", icon: <UnderlineIcon />, label: `${t("Underline")} (${keyComboToString("Ctrl", "u")})` },
        { action: "Strikethrough", icon: <StrikethroughIcon />, label: `${t("Strikethrough")} (${keyComboToString("Ctrl", "Shift", "s")})` },
        { action: "Spoiler", icon: <WarningIcon />, label: `${t("Spoiler")} (${keyComboToString("Ctrl", "l")})` },
    ];
    const listItems: PopoverActionItem[] = [
        { action: "ListBullet", icon: <ListBulletIcon />, label: `${t("ListBulleted")} (${keyComboToString("Alt", "7")})` },
        { action: "ListNumber", icon: <ListNumberIcon />, label: `${t("ListNumbered")} (${keyComboToString("Alt", "8")})` },
        { action: "ListCheckbox", icon: <ListCheckIcon />, label: `${t("ListCheckbox")} (${keyComboToString("Alt", "9")})` },
    ];
    // Combine format, link, list, and table actions for minimal view
    const combinedItems: PopoverActionItem[] = [
        ...formatItems,
        { action: "Link", icon: <LinkIcon />, label: `${t("Link")} (${keyComboToString("Ctrl", "k")})` },
        ...listItems,
        { action: "Table", icon: <TableIcon />, label: t("TableInsert") },
    ];

    return (
        <Box ref={dimRef} sx={{
            display: "flex",
            flexDirection: (isLeftHanded || viewSize === "full") ? "row" : "row-reverse",
            width: "100%",
            padding: "2px",
            background: palette.primary.main,
            color: palette.primary.contrastText,
            borderRadius: "0.5rem 0.5rem 0 0",
            ...sx,
        }}>
            {/* Group of main editor controls including AI assistant and formatting tools */}
            <Stack
                direction="row"
                spacing={fromDims({ xs: 0, sm: 0.5, md: 1 })}
                sx={{
                    ...((isLeftHanded || viewSize === "full") ? { marginRight: "auto" } : { marginLeft: "auto" }),
                    visibility: disabled ? "hidden" : "visible",
                }}
            >
                {hasPremium && !disableAssistant && <ToolButton
                    icon={<MagicIcon fill={palette.primary.contrastText} />}
                    label={t("AIAssistant")}
                    onClick={() => handleToggleAction("Assistant")}
                    palette={palette}
                />}
                <ToolButton
                    disabled={disabled}
                    icon={<HeaderIcon fill={palette.primary.contrastText} />}
                    isActive={activeStates.Header1 || activeStates.Header2 || activeStates.Header3}
                    label={t("HeaderInsert")}
                    onClick={openHeaderSelect}
                    palette={palette}
                />
                <ActionPopover
                    activeStates={activeStates}
                    idPrefix="header"
                    isOpen={headerSelectOpen}
                    anchorEl={headerAnchorEl}
                    onClose={closeHeader}
                    items={headerItems}
                    handleAction={handleToggleAction}
                    palette={palette}
                />
                {viewSize === "minimal" ? (
                    <>
                        <ToolButton
                            ref={combinedButtonRef}
                            disabled={disabled}
                            icon={<CaseSensitiveIcon fill={palette.primary.contrastText} />}
                            label={t("TextFormat")}
                            onClick={openCombinedSelect}
                            palette={palette}
                        />
                        <ActionPopover
                            activeStates={activeStates}
                            idPrefix="combined"
                            isOpen={combinedSelectOpen}
                            anchorEl={combinedAnchorEl}
                            onClose={closeCombined}
                            items={combinedItems}
                            handleAction={handleCombinedAction}
                            palette={palette}
                        />
                    </>
                ) : (
                    <>
                        {/* Display format as popover on mobile, or display all options on desktop */}
                        {viewSize !== "full" && (
                            <>
                                <ToolButton
                                    disabled={disabled}
                                    icon={<CaseSensitiveIcon fill={palette.primary.contrastText} />}
                                    isActive={activeStates.Bold || activeStates.Italic || activeStates.Underline || activeStates.Strikethrough || activeStates.Spoiler}
                                    label={t("TextFormat")}
                                    onClick={openFormatSelect}
                                    palette={palette}
                                />
                                <ActionPopover
                                    activeStates={activeStates}
                                    idPrefix="format"
                                    isOpen={formatSelectOpen}
                                    anchorEl={formatAnchorEl}
                                    onClose={closeFormat}
                                    items={formatItems}
                                    handleAction={handleToggleAction}
                                    palette={palette}
                                />
                            </>
                        )}
                        {viewSize === "full" && formatItems.map(({ action, icon, label }) => (
                            <ToolButton
                                key={action}
                                disabled={disabled}
                                icon={icon}
                                isActive={activeStates[action]}
                                label={label}
                                onClick={() => { handleToggleAction(action); }}
                                palette={palette}
                            />
                        ))}
                        <ToolButton
                            disabled={disabled}
                            icon={<LinkIcon fill={palette.primary.contrastText} />}
                            label={`${t("Link")} (${keyComboToString("Ctrl", "k")})`}
                            onClick={() => { handleToggleAction("Link"); }}
                            palette={palette}
                        />
                        <ToolButton
                            disabled={disabled}
                            icon={<ListIcon fill={palette.primary.contrastText} />}
                            isActive={activeStates.ListBullet || activeStates.ListNumber || activeStates.ListCheckbox}
                            label={t("ListInsert")}
                            onClick={openListSelect}
                            palette={palette}
                        />
                        <ActionPopover
                            activeStates={activeStates}
                            idPrefix="list"
                            isOpen={listSelectOpen}
                            anchorEl={listAnchorEl}
                            onClose={closeList}
                            items={listItems}
                            handleAction={handleToggleAction}
                            palette={palette}
                        />
                        <ToolButton
                            disabled={disabled}
                            icon={<TableIcon fill={palette.primary.contrastText} />}
                            label={t("TableInsert")}
                            onClick={openTableSelect}
                            palette={palette}
                        />
                    </>
                )}
                <TablePopover
                    isOpen={tableSelectOpen}
                    anchorEl={tableAnchorEl}
                    onClose={closeTable}
                    handleTableInsert={handleTableInsert}
                    palette={palette}
                    t={t}
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
                    onClick={() => { handleToggleAction("Undo"); }}
                    palette={palette}
                />}
                {(canUndo || canRedo) && <ToolButton
                    disabled={disabled || !canRedo}
                    icon={<RedoIcon fill={palette.primary.contrastText} />}
                    label={`${t("Redo")} (${keyComboToString("Ctrl", "y")})`}
                    onClick={() => { handleToggleAction("Redo"); }}
                    palette={palette}
                />}
                <Tooltip title={!isMarkdownOn ? `${t("PressToMarkdown")} (${keyComboToString("Alt", "0")})` : `${t("PressToPreview")} (${keyComboToString("Alt", "0")})`} placement="top">
                    <Typography variant="body2" onClick={() => { handleToggleAction("Mode"); }} sx={{
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
