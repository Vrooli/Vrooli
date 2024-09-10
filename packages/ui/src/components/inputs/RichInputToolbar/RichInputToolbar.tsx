import { noop } from "@local/shared";
import { Box, BoxProps, Button, IconButton, List, ListItem, ListItemIcon, ListItemText, Palette, Popover, Stack, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { SessionContext } from "contexts";
import { useIsLeftHanded, useRichInputToolbarViewSize } from "hooks/subscriptions";
import { usePopover } from "hooks/usePopover";
import { TFunction } from "i18next";
import { BoldIcon, CaseSensitiveIcon, Header1Icon, Header2Icon, Header3Icon, Header4Icon, Header5Icon, Header6Icon, HeaderIcon, ItalicIcon, LinkIcon, ListBulletIcon, ListCheckIcon, ListIcon, ListNumberIcon, MagicIcon, QuoteIcon, RedoIcon, StrikethroughIcon, TableIcon, TerminalIcon, UnderlineIcon, UndoIcon, WarningIcon } from "icons";
import { forwardRef, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { SxType } from "types";
import { getCurrentUser } from "utils/authentication/session";
import { keyComboToString } from "utils/display/device";
import { RichInputAction, RichInputActiveStates } from "../types";

type PopoverActionItem = {
    action: RichInputAction | `${RichInputAction}`,
    icon: React.ReactNode,
    label: string
};

/** Determines how many options should be displayed directly */
type ViewSize = "minimal" | "partial" | "full";

type ActionPopoverProps = {
    activeStates: Omit<RichInputActiveStates, "SetValue">;
    anchorEl: HTMLElement | null;
    handleAction: (action: string, data?: unknown) => unknown;
    idPrefix: string;
    isOpen: boolean;
    items: PopoverActionItem[];
    onClose: () => unknown;
    palette: any;//Theme["palette"];
}

type TablePopoverProps = {
    anchorEl: HTMLElement | null;
    handleTableInsert: (rows: number, cols: number) => unknown;
    isOpen: boolean;
    onClose: () => unknown;
    palette: any;//Theme["palette"];
    t: TFunction<"common", undefined, "common">;
}

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
    id,
    isActive,
    label,
    onClick,
    palette,
}: {
    disabled?: boolean,
    icon: React.ReactNode,
    id?: string,
    isActive?: boolean,
    label: string,
    onClick: (event: React.MouseEvent<HTMLElement>) => unknown,
    palette: Palette,
}, ref: React.Ref<HTMLButtonElement>) => {

    const handleClick = useCallback(function handleClickCallback(event: React.MouseEvent<HTMLElement>) {
        // Stop propagation so text field doesn't lose focus
        event.preventDefault();
        event.stopPropagation();
        onClick(event);
    }, [onClick]);

    const buttonStyle = useMemo(function buttonStyleMemo() {
        return {
            background: isActive ? palette.secondary.main : palette.primary.main,
            color: palette.primary.contrastText,
            borderRadius: 2,
        };
    }, [isActive, palette.primary.contrastText, palette.primary.main, palette.secondary.main]);

    return (
        <Tooltip title={label}>
            <IconButton
                id={id}
                ref={ref}
                disabled={disabled}
                size="small"
                onClick={handleClick}
                sx={buttonStyle}
            >
                {icon}
            </IconButton>
        </Tooltip>
    );
});
ToolButton.displayName = "ToolButton";

const popoverAnchorOrigin = { vertical: "bottom", horizontal: "center" } as const;

function ActionPopover({
    activeStates,
    anchorEl,
    handleAction,
    idPrefix,
    isOpen,
    items,
    onClose,
    palette,
}: ActionPopoverProps) {
    return (
        <Popover
            id={`markdown-input-${idPrefix}-popover`}
            open={isOpen}
            anchorEl={anchorEl}
            onClose={onClose}
            anchorOrigin={popoverAnchorOrigin}
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
}

function TablePopover({
    anchorEl,
    handleTableInsert,
    isOpen,
    onClose,
    palette,
    t,
}: TablePopoverProps) {
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

    const handleKeyDown = useCallback((event) => {
        let newHoveredRow = hoveredRow;
        let newHoveredCol = hoveredCol;

        switch (event.key) {
            case "ArrowRight":
                newHoveredCol = hoveredCol < maxCols ? hoveredCol + 1 : hoveredCol;
                break;
            case "ArrowLeft":
                newHoveredCol = hoveredCol > 1 ? hoveredCol - 1 : hoveredCol;
                break;
            case "ArrowUp":
                newHoveredRow = hoveredRow > 1 ? hoveredRow - 1 : hoveredRow;
                break;
            case "ArrowDown":
                newHoveredRow = hoveredRow < maxRows ? hoveredRow + 1 : hoveredRow;
                break;
            case "Enter":
                handleTableInsert(hoveredRow, hoveredCol);
                onClose();
                break;
            default:
                return;
        }

        setHoveredRow(newHoveredRow);
        setHoveredCol(newHoveredCol);
        event.preventDefault();
    }, [hoveredRow, hoveredCol, maxRows, maxCols, handleTableInsert, onClose]);

    useEffect(() => {
        if (isOpen) {
            document.addEventListener("keydown", handleKeyDown);
        } else {
            document.removeEventListener("keydown", handleKeyDown);
        }

        return () => {
            document.removeEventListener("keydown", handleKeyDown);
        };
    }, [isOpen, handleKeyDown]);

    return (
        <Popover
            open={isOpen}
            anchorEl={anchorEl}
            onClose={onClose}
            anchorOrigin={popoverAnchorOrigin}
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
}

interface LeftSectionProps extends BoxProps {
    disabled: boolean;
    isLeftHanded: boolean;
    viewSize: ViewSize
}

const LeftSection = styled(Box, {
    shouldForwardProp: (prop) => !["disabled", "isLeftHanded", "viewSize"].includes(prop as string),
})<LeftSectionProps>(({ disabled, isLeftHanded, viewSize }) => ({
    ...((isLeftHanded || viewSize === "full") ? { marginRight: "auto" } : { marginLeft: "auto" }),
    visibility: disabled ? "hidden" : "visible",
    display: "flex",
    flexDirection: "row",
    alignItems: "center",
    gap: 0,
}));

const RightSection = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    alignItems: "center",
    gap: 0,
    [theme.breakpoints.up("sm")]: {
        gap: 0.5,
    },
    [theme.breakpoints.up("md")]: {
        gap: 1,
    },
}));

const ModeSelectorLabel = styled(Typography)(({ theme }) => ({
    cursor: "pointer",
    margin: "auto",
    padding: 1,
    background: theme.palette.primary.main,
    borderRadius: 2,
}));

export function RichInputToolbar({
    activeStates,
    canRedo,
    canUndo,
    disableAssistant = false,
    disabled = false,
    handleAction,
    handleActiveStatesChange,
    id,
    isMarkdownOn,
    sx,
}: {
    activeStates: Omit<RichInputActiveStates, "SetValue">;
    canRedo: boolean;
    canUndo: boolean;
    disableAssistant?: boolean;
    disabled?: boolean;
    handleAction: (action: RichInputAction, data?: unknown) => unknown;
    handleActiveStatesChange: (activeStates: Omit<RichInputActiveStates, "SetValue">) => unknown;
    id: string;
    isMarkdownOn: boolean;
    name: string,
    sx?: SxType;
}) {
    const { breakpoints, palette } = useTheme();
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { credits } = useMemo(() => getCurrentUser(session), [session]);
    const { dimensions, fromDims, ref: dimRef } = useDimensions();
    const viewSize = useMemo<ViewSize>(() => {
        if (dimensions.width <= 375) return "minimal";
        if (dimensions.width <= breakpoints.values.sm) return "partial";
        return "full";
    }, [breakpoints, dimensions]);
    const isLeftHanded = useIsLeftHanded();

    const handleToggleAction = useCallback(function handleToggleActionCallback(action: string, data?: unknown) {
        // Update action's active state, if we're not using markdown mode
        if (!isMarkdownOn && action in activeStates) {
            handleActiveStatesChange({
                ...activeStates,
                [action]: !activeStates[action],
            });
        }
        // Trigger handleAction
        handleAction(action as RichInputAction, data);
    }, [activeStates, handleAction, handleActiveStatesChange, isMarkdownOn]);
    useEffect(() => {
        if (isMarkdownOn) {
            handleActiveStatesChange(defaultActiveStates);
        }
    }, [handleActiveStatesChange, isMarkdownOn]);

    const handleToggleAssistant = useCallback(function handleToggleAssistantCallback() {
        handleToggleAction("Assistant");
    }, [handleToggleAction]);
    const handleToggleLink = useCallback(function handleToggleLinkCallback() {
        handleToggleAction("Link");
    }, [handleToggleAction]);
    const handleToggleUndo = useCallback(function handleToggleUndoCallback() {
        handleToggleAction("Undo");
    }, [handleToggleAction]);
    const handleToggleRedo = useCallback(function handleToggleRedoCallback() {
        handleToggleAction("Redo");
    }, [handleToggleAction]);
    const handleToggleMode = useCallback(function handleToggleModeCallback() {
        handleToggleAction("Mode");
    }, [handleToggleAction]);

    const [headerAnchorEl, openHeaderSelect, closeHeader, headerSelectOpen] = usePopover();
    const [formatAnchorEl, openFormatSelect, closeFormat, formatSelectOpen] = usePopover();
    const [listAnchorEl, openListSelect, closeList, listSelectOpen] = usePopover();
    const [tableAnchorEl, openTableSelect, closeTable, tableSelectOpen] = usePopover();
    function handleTableInsert(rows: number, cols: number) {
        handleToggleAction("Table", { rows, cols });
        closeTable();
    }
    const combinedButtonRef = useRef(null);
    const [combinedAnchorEl, openCombinedSelect, closeCombined, combinedSelectOpen] = usePopover();
    function handleCombinedAction(action: string, data?: unknown) {
        if (action === "Table") {
            closeCombined();
            const syntheticEvent = { currentTarget: combinedButtonRef.current };
            openTableSelect(syntheticEvent as any, true);
        } else {
            handleToggleAction(action, data);
        }
    }

    const headerItems = useMemo<PopoverActionItem[]>(function headerItemsMemo() {
        return [
            { action: "Header1", icon: <Header1Icon />, label: `${t("Header1")} (${keyComboToString("Alt", "1")})` },
            { action: "Header2", icon: <Header2Icon />, label: `${t("Header2")} (${keyComboToString("Alt", "2")})` },
            { action: "Header3", icon: <Header3Icon />, label: `${t("Header3")} (${keyComboToString("Alt", "3")})` },
            { action: "Header4", icon: <Header4Icon />, label: `${t("Header4")} (${keyComboToString("Alt", "4")})` },
            { action: "Header5", icon: <Header5Icon />, label: `${t("Header5")} (${keyComboToString("Alt", "5")})` },
            { action: "Header6", icon: <Header6Icon />, label: `${t("Header6")} (${keyComboToString("Alt", "6")})` },
        ];
    }, [t]);
    const formatItems = useMemo<PopoverActionItem[]>(function formatItemsMemo() {
        return [
            { action: "Bold", icon: <BoldIcon />, label: `${t("Bold")} (${keyComboToString("Ctrl", "B")})` },
            { action: "Italic", icon: <ItalicIcon />, label: `${t("Italic")} (${keyComboToString("Ctrl", "I")})` },
            { action: "Underline", icon: <UnderlineIcon />, label: `${t("Underline")} (${keyComboToString("Ctrl", "U")})` },
            { action: "Strikethrough", icon: <StrikethroughIcon />, label: `${t("Strikethrough")} (${keyComboToString("Ctrl", "Shift", "S")})` },
            { action: "Spoiler", icon: <WarningIcon />, label: `${t("Spoiler")} (${keyComboToString("Ctrl", "L")})` },
            { action: "Quote", icon: <QuoteIcon />, label: `${t("Quote")} (${keyComboToString("Ctrl", "Shift", "Q")})` },
            { action: "Code", icon: <TerminalIcon />, label: `${t("Code")} (${keyComboToString("Ctrl", "E")})` },
        ];
    }, [t]);
    const listItems = useMemo<PopoverActionItem[]>(function listItemsMemo() {
        return [
            { action: "ListBullet", icon: <ListBulletIcon />, label: `${t("ListBulleted")} (${keyComboToString("Alt", "7")})` },
            { action: "ListNumber", icon: <ListNumberIcon />, label: `${t("ListNumbered")} (${keyComboToString("Alt", "8")})` },
            { action: "ListCheckbox", icon: <ListCheckIcon />, label: `${t("ListCheckbox")} (${keyComboToString("Alt", "9")})` },
        ];
    }, [t]);
    // Combine format, link, list, and table actions for minimal view
    const combinedItems = useMemo<PopoverActionItem[]>(function combinedItemsMemo() {
        return [
            ...formatItems,
            { action: "Link", icon: <LinkIcon />, label: `${t("Link", { count: 1 })} (${keyComboToString("Ctrl", "K")})` },
            ...listItems,
            { action: "Table", icon: <TableIcon />, label: t("TableInsert") },
        ];
    }, [formatItems, listItems, t]);

    const outerBoxStyle = useMemo(function outerBoxStyleMemo() {
        return {
            display: "flex",
            flexDirection: (isLeftHanded || viewSize === "full") ? "row" : "row-reverse",
            width: "100%",
            padding: "2px",
            background: palette.primary.main,
            color: palette.primary.contrastText,
            borderRadius: "0.5rem 0.5rem 0 0",
            ...sx,
        } as const;
    }, [isLeftHanded, sx, palette.primary.contrastText, palette.primary.main, viewSize]);

    return (
        <Box
            aria-label="Rich text editor toolbar"
            component="section"
            id={id}
            ref={dimRef}
            sx={outerBoxStyle}
        >
            {/* Group of main editor controls including AI assistant and formatting tools */}
            <LeftSection
                disabled={disabled}
                isLeftHanded={isLeftHanded}
                viewSize={viewSize}
            >
                {credits && BigInt(credits) > 0 && !disableAssistant && <ToolButton
                    id={`${id}-assistant`}
                    icon={<MagicIcon fill={palette.primary.contrastText} />}
                    label={t("AIAssistant")}
                    onClick={handleToggleAssistant}
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
                            label={`${t("Link", { count: 1 })} (${keyComboToString("Ctrl", "k")})`}
                            onClick={handleToggleLink}
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
            </LeftSection>
            {/* Group for undo, redo, and switching between Markdown as WYSIWYG */}
            <RightSection>
                <div>
                    {(canUndo || canRedo) && <ToolButton
                        disabled={disabled || !canUndo}
                        icon={<UndoIcon fill={palette.primary.contrastText} />}
                        label={`${t("Undo")} (${keyComboToString("Ctrl", "z")})`}
                        onClick={handleToggleUndo}
                        palette={palette}
                    />}
                    {(canUndo || canRedo) && <ToolButton
                        disabled={disabled || !canRedo}
                        icon={<RedoIcon fill={palette.primary.contrastText} />}
                        label={`${t("Redo")} (${keyComboToString("Ctrl", "y")})`}
                        onClick={handleToggleRedo}
                        palette={palette}
                    />}
                </div>
                <Tooltip title={!isMarkdownOn ? `${t("PressToMarkdown")} (${keyComboToString("Alt", "0")})` : `${t("PressToPreview")} (${keyComboToString("Alt", "0")})`} placement="top">
                    <ModeSelectorLabel variant="body2" onClick={handleToggleMode}>
                        {!isMarkdownOn ? t("MarkdownTo") : t("PreviewTo")}
                    </ModeSelectorLabel>
                </Tooltip>
            </RightSection>
        </Box>
    );
}
