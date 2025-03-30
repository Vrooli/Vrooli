import { TranslationFuncCommon, TranslationKeyCommon } from "@local/shared";
import { Box, BoxProps, Button, IconButton, IconButtonProps, List, ListItem, ListItemIcon, ListItemProps, ListItemText, Menu, MenuItem, Palette, Popover, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { forwardRef, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useIsLeftHanded, useRichInputToolbarViewSize } from "../../../hooks/subscriptions.js";
import { usePopover } from "../../../hooks/usePopover.js";
import { Icon, IconCommon, IconInfo, IconText } from "../../../icons/Icons.js";
import { keyComboToString } from "../../../utils/display/device.js";
import { RichInputToolbarViewSize } from "../../../utils/pubsub.js";
import { RichInputAction, RichInputActiveStates } from "../types.js";

type PrePopoverActionItem = {
    action: RichInputAction | `${RichInputAction}`,
    iconInfo: IconInfo,
    labelKey: TranslationKeyCommon,
    keyCombo?: string,
};

type PopoverActionItem = {
    action: RichInputAction | `${RichInputAction}`,
    icon: React.ReactNode,
    label: string
};

type ActionPopoverProps = {
    activeStates: Omit<RichInputActiveStates, "SetValue">;
    anchorEl: Element | null;
    handleAction: (action: string, data?: unknown) => unknown;
    isOpen: boolean;
    items: PopoverActionItem[];
    onClose: () => unknown;
}

type TablePopoverProps = {
    anchorEl: Element | null;
    handleTableInsert: (rows: number, cols: number) => unknown;
    isOpen: boolean;
    onClose: () => unknown;
    palette: any;//Theme["palette"];
    t: TranslationFuncCommon;
}

export const TOOLBAR_CLASS_NAME = "advanced-input-toolbar";

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

const preHeaderItems: PrePopoverActionItem[] = [
    { action: "Header1", iconInfo: { name: "Header1", type: "Text" }, labelKey: "Header1", keyCombo: keyComboToString("Alt", "1") },
    { action: "Header2", iconInfo: { name: "Header2", type: "Text" }, labelKey: "Header2", keyCombo: keyComboToString("Alt", "2") },
    { action: "Header3", iconInfo: { name: "Header3", type: "Text" }, labelKey: "Header3", keyCombo: keyComboToString("Alt", "3") },
    { action: "Header4", iconInfo: { name: "Header4", type: "Text" }, labelKey: "Header4", keyCombo: keyComboToString("Alt", "4") },
    { action: "Header5", iconInfo: { name: "Header5", type: "Text" }, labelKey: "Header5", keyCombo: keyComboToString("Alt", "5") },
    { action: "Header6", iconInfo: { name: "Header6", type: "Text" }, labelKey: "Header6", keyCombo: keyComboToString("Alt", "6") },
];
const preFormatItems: PrePopoverActionItem[] = [
    { action: "Bold", iconInfo: { name: "Bold", type: "Text" }, labelKey: "Bold", keyCombo: keyComboToString("Ctrl", "B") },
    { action: "Italic", iconInfo: { name: "Italic", type: "Text" }, labelKey: "Italic", keyCombo: keyComboToString("Ctrl", "I") },
    { action: "Underline", iconInfo: { name: "Underline", type: "Text" }, labelKey: "Underline", keyCombo: keyComboToString("Ctrl", "U") },
    { action: "Strikethrough", iconInfo: { name: "Strikethrough", type: "Text" }, labelKey: "Strikethrough", keyCombo: keyComboToString("Ctrl", "Shift", "S") },
    { action: "Spoiler", iconInfo: { name: "Warning", type: "Common" }, labelKey: "Spoiler", keyCombo: keyComboToString("Ctrl", "L") },
    { action: "Quote", iconInfo: { name: "Quote", type: "Text" }, labelKey: "Quote", keyCombo: keyComboToString("Ctrl", "Shift", "Q") },
    { action: "Code", iconInfo: { name: "Terminal", type: "Common" }, labelKey: "Code", keyCombo: keyComboToString("Ctrl", "E") },
];
const preListItems: PrePopoverActionItem[] = [
    { action: "ListBullet", iconInfo: { name: "ListBullet", type: "Text" }, labelKey: "ListBulleted", keyCombo: keyComboToString("Alt", "7") },
    { action: "ListNumber", iconInfo: { name: "ListNumber", type: "Text" }, labelKey: "ListNumbered", keyCombo: keyComboToString("Alt", "8") },
    { action: "ListCheckbox", iconInfo: { name: "ListCheck", type: "Text" }, labelKey: "ListCheckbox", keyCombo: keyComboToString("Alt", "9") },
];
// Combine format, link, list, and table actions for minimal view
const preCombinedItems: PrePopoverActionItem[] = [
    ...preFormatItems,
    { action: "Link", iconInfo: { name: "Link", type: "Common" }, labelKey: "Link", keyCombo: keyComboToString("Ctrl", "K") },
    ...preListItems,
    { action: "Table", iconInfo: { name: "Table", type: "Common" }, labelKey: "TableInsert" },
];

interface StyledIconButtonProps extends IconButtonProps {
    isActive?: boolean;
}
const StyledIconButton = styled(IconButton, {
    shouldForwardProp: (prop) => prop !== "isActive",
})<StyledIconButtonProps>(({ theme, isActive }) => ({
    background: isActive ? theme.palette.secondary.main : "transparent",
    borderRadius: theme.spacing(1),
}));

const ToolButton = forwardRef(({
    disabled,
    icon,
    isActive,
    label,
    onClick,
}: {
    disabled?: boolean,
    icon: React.ReactNode,
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

    return (
        <Tooltip title={label}>
            <StyledIconButton
                ref={ref}
                disabled={disabled}
                isActive={isActive}
                size="small"
                onClick={handleClick}
            >
                {icon}
            </StyledIconButton>
        </Tooltip>
    );
});
ToolButton.displayName = "ToolButton";

const ActionPopoverList = styled(List)(({ theme }) => ({
    background: theme.palette.background.paper,
    color: theme.palette.background.textPrimary,
}));

interface ActionPopoverListItemProps extends ListItemProps {
    action: PopoverActionItem["action"];
    activeStates: Omit<RichInputActiveStates, "SetValue">;
}
const ActionPopoverListItem = styled(ListItem, {
    shouldForwardProp: (prop) => prop !== "action" && prop !== "activeStates",
})<ActionPopoverListItemProps>(({ action, activeStates, theme }) => ({
    background: activeStates[action] ? theme.palette.secondary.light : "transparent",
}));


const popoverAnchorOrigin = { vertical: "bottom", horizontal: "center" } as const;

function ActionPopover({
    activeStates,
    anchorEl,
    handleAction,
    isOpen,
    items,
    onClose,
}: ActionPopoverProps) {
    return (
        <Popover
            open={isOpen}
            anchorEl={anchorEl}
            onClose={onClose}
            anchorOrigin={popoverAnchorOrigin}
        >
            <ActionPopoverList>
                {items.map(({ action, icon, label }) => {
                    function onAction() {
                        handleAction(action);
                    }
                    return (
                        <ActionPopoverListItem
                            action={action}
                            activeStates={activeStates}
                            // eslint-disable-next-line
                            // @ts-ignore
                            button
                            key={action}
                            onClick={onAction}
                        >
                            <ListItemIcon>
                                {icon}
                            </ListItemIcon>
                            <ListItemText primary={label} />
                        </ActionPopoverListItem>
                    );
                })}
            </ActionPopoverList>
        </Popover>
    );
}

const TableStack = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "column",
    justifyContent: "center",
    alignItems: "center",
    gap: theme.spacing(1),
    padding: theme.spacing(2),
    background: theme.palette.background.paper,
    color: theme.palette.background.textPrimary,
}));

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

    function insertAtHovered() {
        handleTableInsert(hoveredRow, hoveredCol);
        onClose();
    }

    return (
        <Popover
            open={isOpen}
            anchorEl={anchorEl}
            onClose={onClose}
            anchorOrigin={popoverAnchorOrigin}
        >
            <TableStack>
                <Typography align="center" variant="h6" pb={1}>
                    {t("TableSelectSize")}
                </Typography>
                <Box>
                    {[...Array(maxRows)].map((_, rowIndex) => (
                        <Box key={rowIndex} display="flex">
                            {[...Array(maxCols)].map((_, colIndex) => {
                                function handleMouseEnter() {
                                    if (!canHover) {
                                        return;
                                    }
                                    setHoveredRow(rowIndex + 1);
                                    setHoveredCol(colIndex + 1);
                                }
                                function handleClick() {
                                    setHoveredRow(rowIndex + 1);
                                    setHoveredCol(colIndex + 1);
                                    // Submit immediately if the user can hover
                                    if (canHover) {
                                        handleTableInsert(rowIndex + 1, colIndex + 1);
                                    }
                                }

                                return (
                                    <Box
                                        key={colIndex}
                                        component="button"
                                        onMouseEnter={handleMouseEnter}
                                        onClick={handleClick}
                                        width={25}
                                        height={25}
                                        border={`1px solid ${palette.divider}`}
                                        bgcolor={(rowIndex < hoveredRow && colIndex < hoveredCol)
                                            ? palette.secondary.light
                                            : "transparent"
                                        }
                                    />
                                );
                            })}
                        </Box>
                    ))}
                </Box>
                <Typography align="center" variant="body2" marginTop={1}>
                    {hoveredRow} x {hoveredCol}
                </Typography>
                {!canHover && (
                    <Button variant="contained" onClick={insertAtHovered} sx={{ marginTop: 2 }}>
                        {t("Submit")}
                    </Button>
                )}
            </TableStack>
        </Popover>
    );
}

interface LeftSectionProps extends BoxProps {
    disabled: boolean;
    isLeftHanded: boolean;
}

const LeftSection = styled(Box, {
    shouldForwardProp: (prop) => !["disabled", "isLeftHanded", "viewSize"].includes(prop as string),
})<LeftSectionProps>(({ disabled, isLeftHanded }) => ({
    ...(isLeftHanded ? { marginLeft: "auto" } : { marginRight: "auto" }),
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

const ModeSelectorLabel = styled(Typography)(() => ({
    cursor: "pointer",
    margin: "auto",
    padding: 1,
    borderRadius: 2,
}));

interface OuterBoxProps extends BoxProps {
    isLeftHanded: boolean;
}
const OuterBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isLeftHanded",
})<OuterBoxProps>(({ isLeftHanded, theme }) => ({
    display: "flex",
    flexDirection: isLeftHanded ? "row-reverse" : "row",
    width: "100%",
    padding: "2px",
    color: theme.palette.background.textSecondary,
    borderRadius: "0.5rem 0.5rem 0 0",
}));

const contextItemStyle = { display: "flex", alignItems: "center" } as const;

export function RichInputToolbar({
    activeStates,
    canRedo,
    canUndo,
    disabled = false,
    handleAction,
    handleActiveStatesChange,
    isMarkdownOn,
}: {
    activeStates: Omit<RichInputActiveStates, "SetValue">;
    canRedo: boolean;
    canUndo: boolean;
    disabled?: boolean;
    handleAction: (action: RichInputAction, data?: unknown) => unknown;
    handleActiveStatesChange: (activeStates: Omit<RichInputActiveStates, "SetValue">) => unknown;
    isMarkdownOn: boolean;
}) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { dimRef, handleUpdateViewSize, viewSize } = useRichInputToolbarViewSize();
    const onUpdateViewSize = useCallback(function onUpdateViewSizeCallback(viewSize: RichInputToolbarViewSize) {
        handleUpdateViewSize(viewSize);
        setContextMenu(null);
    }, [handleUpdateViewSize]);
    function toMinimalView() {
        onUpdateViewSize("minimal");
    }
    function toPartialView() {
        onUpdateViewSize("partial");
    }
    function toFullView() {
        onUpdateViewSize("full");
    }

    const [contextMenu, setContextMenu] = useState<{ mouseX: number; mouseY: number } | null>(null);
    const handleContextMenu = useCallback(function handleContextMenuCallback(event: React.MouseEvent) {
        event.preventDefault();
        setContextMenu(
            contextMenu === null
                ? { mouseX: event.clientX - 2, mouseY: event.clientY - 4 }
                : null,
        );
    }, [contextMenu]);
    const handleContextMenuClose = useCallback(function handleContextMenuClose() {
        setContextMenu(null);
    }, []);

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

    const headerItems = useMemo(() => {
        return preHeaderItems.map(({ action, iconInfo, labelKey, keyCombo }) => ({
            action,
            icon: <Icon fill="background.textSecondary" info={iconInfo} />,
            label: keyCombo ? `${t(labelKey)} (${keyCombo})` : t(labelKey),
        }));
    }, [t]);

    const formatItems = useMemo(() => {
        return preFormatItems.map(({ action, iconInfo, labelKey, keyCombo }) => ({
            action,
            icon: <Icon fill="background.textSecondary" info={iconInfo} />,
            label: keyCombo ? `${t(labelKey)} (${keyCombo})` : t(labelKey),
        }));
    }, [t]);

    const listItems = useMemo(() => {
        return preListItems.map(({ action, iconInfo, labelKey, keyCombo }) => ({
            action,
            icon: <Icon fill="background.textSecondary" info={iconInfo} />,
            label: keyCombo ? `${t(labelKey)} (${keyCombo})` : t(labelKey),
        }));
    }, [t]);

    const combinedItems = useMemo(() => {
        return preCombinedItems.map(({ action, iconInfo, labelKey, keyCombo }) => ({
            action,
            icon: <Icon fill="background.textSecondary" info={iconInfo} />,
            label: keyCombo ? `${t(labelKey)} (${keyCombo})` : t(labelKey),
        }));
    }, [t]);

    return (
        <OuterBox
            aria-label="Rich text editor toolbar"
            className={TOOLBAR_CLASS_NAME}
            component="section"
            isLeftHanded={isLeftHanded}
            onContextMenu={handleContextMenu}
            ref={dimRef}
        >
            {/* Right-click context menu */}
            <Menu
                open={contextMenu !== null}
                onClose={handleContextMenuClose}
                anchorReference="anchorPosition"
                anchorPosition={
                    contextMenu !== null
                        ? { top: contextMenu.mouseY, left: contextMenu.mouseX }
                        : undefined
                }
            >
                <MenuItem onClick={toMinimalView}
                    sx={contextItemStyle}>
                    <ListItemText primary="Minimal View" secondary="Show fewer toolbar options for a cleaner interface." />
                </MenuItem>
                <MenuItem onClick={toPartialView}
                    sx={contextItemStyle}>
                    <ListItemText primary="Partial View" secondary="Display commonly used tools for convenient access." />
                </MenuItem>
                <MenuItem onClick={toFullView}
                    sx={contextItemStyle}>
                    <ListItemText primary="Full View" secondary="Show all available tools for maximum functionality." />
                </MenuItem>
            </Menu>
            {/* Group of main editor controls */}
            <LeftSection
                disabled={disabled}
                isLeftHanded={isLeftHanded}
            >
                <ToolButton
                    aria-label={t("HeaderInsert")}
                    disabled={disabled}
                    icon={<IconText
                        decorative
                        fill="background.textSecondary"
                        name="Header"
                    />}
                    isActive={activeStates.Header1 || activeStates.Header2 || activeStates.Header3}
                    label={t("HeaderInsert")}
                    onClick={openHeaderSelect}
                    palette={palette}
                />
                <ActionPopover
                    activeStates={activeStates}
                    isOpen={headerSelectOpen}
                    anchorEl={headerAnchorEl}
                    onClose={closeHeader}
                    items={headerItems}
                    handleAction={handleToggleAction}
                />
                {viewSize === "minimal" ? (
                    <>
                        <ToolButton
                            aria-label={t("TextFormat")}
                            ref={combinedButtonRef}
                            disabled={disabled}
                            icon={<IconText
                                decorative
                                fill="background.textSecondary"
                                name="CaseSensitive"
                            />}
                            label={t("TextFormat")}
                            onClick={openCombinedSelect}
                            palette={palette}
                        />
                        <ActionPopover
                            activeStates={activeStates}
                            isOpen={combinedSelectOpen}
                            anchorEl={combinedAnchorEl}
                            onClose={closeCombined}
                            items={combinedItems}
                            handleAction={handleCombinedAction}
                        />
                    </>
                ) : (
                    <>
                        {/* Display format as popover on mobile, or display all options on desktop */}
                        {viewSize !== "full" && (
                            <>
                                <ToolButton
                                    aria-label={t("TextFormat")}
                                    disabled={disabled}
                                    icon={<IconText
                                        decorative
                                        fill="background.textSecondary"
                                        name="CaseSensitive"
                                    />}
                                    isActive={activeStates.Bold || activeStates.Italic || activeStates.Underline || activeStates.Strikethrough || activeStates.Spoiler}
                                    label={t("TextFormat")}
                                    onClick={openFormatSelect}
                                    palette={palette}
                                />
                                <ActionPopover
                                    activeStates={activeStates}
                                    isOpen={formatSelectOpen}
                                    anchorEl={formatAnchorEl}
                                    onClose={closeFormat}
                                    items={formatItems}
                                    handleAction={handleToggleAction}
                                />
                            </>
                        )}
                        {viewSize === "full" && formatItems.map(({ action, icon, label }) => {
                            function handleClick() {
                                handleToggleAction(action);
                            }

                            return (
                                <ToolButton
                                    key={action}
                                    disabled={disabled}
                                    icon={icon}
                                    isActive={activeStates[action]}
                                    label={label}
                                    onClick={handleClick}
                                    palette={palette}
                                />
                            );
                        })}
                        <ToolButton
                            aria-label={t("Link", { count: 1 })}
                            disabled={disabled}
                            icon={<IconCommon
                                decorative
                                fill="background.textSecondary"
                                name="Link"
                            />}
                            label={`${t("Link", { count: 1 })} (${keyComboToString("Ctrl", "k")})`}
                            onClick={handleToggleLink}
                            palette={palette}
                        />
                        <ToolButton
                            aria-label={t("ListInsert")}
                            disabled={disabled}
                            icon={<IconText
                                decorative
                                fill="background.textSecondary"
                                name="List"
                            />}
                            isActive={activeStates.ListBullet || activeStates.ListNumber || activeStates.ListCheckbox}
                            label={t("ListInsert")}
                            onClick={openListSelect}
                            palette={palette}
                        />
                        <ActionPopover
                            activeStates={activeStates}
                            isOpen={listSelectOpen}
                            anchorEl={listAnchorEl}
                            onClose={closeList}
                            items={listItems}
                            handleAction={handleToggleAction}
                        />
                        <ToolButton
                            aria-label={t("TableInsert")}
                            disabled={disabled}
                            icon={<IconCommon
                                decorative
                                fill="background.textSecondary"
                                name="Table"
                            />}
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
                        aria-label={t("Undo")}
                        disabled={disabled || !canUndo}
                        icon={<IconCommon
                            decorative
                            fill="background.textSecondary"
                            name="Undo"
                        />}
                        label={`${t("Undo")} (${keyComboToString("Ctrl", "z")})`}
                        onClick={handleToggleUndo}
                        palette={palette}
                    />}
                    {(canUndo || canRedo) && <ToolButton
                        aria-label={t("Redo")}
                        disabled={disabled || !canRedo}
                        icon={<IconCommon
                            decorative
                            fill="background.textSecondary"
                            name="Redo"
                        />}
                        label={`${t("Redo")} (${keyComboToString("Ctrl", "y")})`}
                        onClick={handleToggleRedo}
                        palette={palette}
                    />}
                </div>
                <Tooltip title={!isMarkdownOn ? `${t("PressToMarkdown")} (${keyComboToString("Alt", "0")})` : `${t("PressToPreview")} (${keyComboToString("Alt", "0")})`} placement="top">
                    <ModeSelectorLabel variant="caption" onClick={handleToggleMode}>
                        {!isMarkdownOn ? t("MarkdownTo") : t("PreviewTo")}
                    </ModeSelectorLabel>
                </Tooltip>
            </RightSection>
        </OuterBox>
    );
}
