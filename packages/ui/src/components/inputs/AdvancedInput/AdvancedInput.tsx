/* eslint-disable no-magic-numbers */
import {
    Avatar,
    Box,
    Checkbox,
    Chip,
    ChipProps,
    IconButton,
    IconButtonProps,
    InputBase,
    ListItemIcon,
    ListItemText,
    Menu,
    MenuItem,
    Popover,
    Typography,
    styled,
    useTheme,
} from "@mui/material";
import type { SxProps, Theme } from "@mui/material/styles";
import { CSSProperties } from "@mui/styles";
import { MicrophoneButton } from "components/buttons/MicrophoneButton/MicrophoneButton.js";
import { useDimensions } from "hooks/useDimensions.js";
import React, {
    KeyboardEvent,
    MouseEvent,
    useCallback,
    useEffect,
    useMemo,
    useRef,
    useState,
} from "react";
import { WindowsKey, keyComboToString } from "utils/display/device.js";
import {
    AddIcon,
    CameraOpenIcon,
    CloseIcon,
    CompleteIcon,
    EllipsisIcon,
    FileIcon,
    ImageIcon,
    InvisibleIcon,
    LinkIcon,
    RoutineIcon,
    SendIcon,
    SettingsIcon,
} from "../../../icons/common.js";

const iconHeight = 32;
const iconWidth = 32;

export interface Tool {
    displayName: string;
    enabled: boolean;
    icon: React.ReactNode;
    type: string;
    name: string;
    arguments: Record<string, any>;
}

export interface ContextItem {
    id: string;
    type: "file" | "image" | "text";
    label: string;
    src?: string;
}

interface AdvancedInputProps {
    enterWillSubmit: boolean;
    tools: Tool[];
    contextData: ContextItem[];
    onToolsChange?: (updatedTools: Tool[]) => void;
    onContextDataChange?: (updatedContext: ContextItem[]) => void;
    onSubmit?: (message: string) => void;
    onAddNewTool?: () => void;
}

// Pre-defined style objects outside of the component scope.
const topRowStyles: SxProps<Theme> = {
    display: "flex",
    alignItems: "center",
    flexWrap: "wrap",
    mb: 1,
};

const toolbarRowStyles: SxProps<Theme> = {
    display: "flex",
    alignItems: "center",
    mb: 1,
};

const contextRowStyles: SxProps<Theme> = {
    display: "flex",
    alignItems: "center",
    flexWrap: "wrap",
    mb: 1,
    background: "none",
};

const inputRowStyles: SxProps<Theme> = {
    mb: 1,
    px: 2, // additional left/right padding
};

const bottomRowStyles: SxProps<Theme> = {
    display: "flex",
    alignItems: "center",
};

const popoverAnchorOrigin = { vertical: "top", horizontal: "left" } as const;
const popoverTransformOrigin = { vertical: "bottom", horizontal: "left" } as const;

const Outer = styled("div")(({ theme }) => ({
    backgroundColor: theme.palette.background.paper,
    borderRadius: theme.spacing(3),
    padding: theme.spacing(1),
}));

type StyledIconButtonProps = IconButtonProps & {
    bgColor?: string;
    disabled?: boolean;
};
const StyledIconButton = styled(IconButton)<StyledIconButtonProps>(({ theme, disabled, bgColor }) => ({
    width: iconWidth,
    height: iconHeight,
    border: `1px solid ${theme.palette.divider}`,
    opacity: disabled ? 0.5 : 1,
    padding: "4px",
    backgroundColor: bgColor || "transparent",
}));
const ShowHideIconButton = styled(StyledIconButton)(() => ({
    border: "none",
}));

type ToolChipProps = ChipProps & {
    enabled: boolean;
};
const ToolChip = styled(Chip)<ToolChipProps>(({ theme, enabled }) => ({
    backgroundColor: enabled ? theme.palette.mode === "dark" ? "#5d5d5d" : "#bfcdcd" : theme.palette.background.paper,
    color: enabled ? theme.palette.background.textPrimary : theme.palette.background.textSecondary,
    border: enabled ? "none" : `1px solid ${theme.palette.divider}`,
    cursor: "pointer",
    "&:hover": {
        filter: "brightness(1.05)",
    },
}));

const PreviewContainer = styled("div")(({ theme }) => ({
    position: "relative",
    width: theme.spacing(7),
    height: theme.spacing(7),
    borderRadius: theme.spacing(1),
    overflow: "visible",
    marginRight: theme.spacing(1),
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
}));

function previewImageStyle(theme: Theme) {
    return {
        width: "100%",
        height: "100%",
        objectFit: "cover",
        borderRadius: theme.spacing(1),
        border: `1px solid ${theme.palette.divider}`,
    } as const;
}

const PreviewImageAvatar = styled(Avatar)(({ theme }) => ({
    width: "100%",
    height: "100%",
    borderRadius: theme.spacing(1),
    border: `1px solid ${theme.palette.divider}`,
}));

const RemoveIconButton = styled(IconButton)(({ theme }) => ({
    width: theme.spacing(2),
    height: theme.spacing(2),
    padding: theme.spacing(0.5),
    position: "absolute",
    top: theme.spacing(-0.5),
    right: theme.spacing(-0.5),
    backgroundColor: theme.palette.background.textSecondary,
    boxShadow: theme.shadows[1],
    "&:hover": {
        backgroundColor: theme.palette.background.textPrimary,
    },
}));

type ContextItemDisplayProps = {
    imgStyle: CSSProperties;
    item: ContextItem;
    onRemove: (id: string) => unknown;
    theme: Theme;
};

function ContextItemDisplay({
    imgStyle,
    item,
    onRemove,
    theme,
}: ContextItemDisplayProps) {
    function handleRemove() {
        onRemove(item.id);
    }

    if (item.type === "image" || item.type === "file") {
        return (
            <PreviewContainer>
                {item.type === "image" && item.src ? (
                    <img src={item.src} alt={item.label} style={imgStyle} />
                ) : (
                    <PreviewImageAvatar variant="square">
                        <ImageIcon />
                    </PreviewImageAvatar>
                )}
                <RemoveIconButton onClick={handleRemove}>
                    <CloseIcon fill={theme.palette.background.default} />
                </RemoveIconButton>
            </PreviewContainer>
        );
    }
    return (
        <Chip
            label={item.label}
            onDelete={handleRemove}
            sx={{ mr: 1, mb: 1 }}
        />
    );
}

interface ExternalApp {
    id: string;
    name: string;
    icon: React.ReactNode;
    connected: boolean;
}

// Add supported external apps here
const externalApps: ExternalApp[] = [
];

/** 
 * PlusMenu Component - renders the popover for additional actions.
 */
interface PlusMenuProps {
    anchorEl: HTMLElement | null;
    onClose: () => void;
    onAttachFile: () => void;
    onConnectExternalApp: () => void;
    onTakePhoto: () => void;
    onAddNewTool: () => void;
}

const PlusMenu: React.FC<PlusMenuProps> = React.memo(
    ({
        anchorEl,
        onClose,
        onAttachFile,
        onConnectExternalApp,
        onTakePhoto,
        onAddNewTool,
    }) => {
        const [externalAppAnchor, setExternalAppAnchor] = useState<HTMLElement | null>(null);

        function handleOpenExternalApps(event: MouseEvent<HTMLElement>) {
            setExternalAppAnchor(event.currentTarget);
        }

        function handleCloseExternalApps() {
            setExternalAppAnchor(null);
        }

        return (
            <>
                <Popover
                    open={Boolean(anchorEl)}
                    anchorEl={anchorEl}
                    onClose={onClose}
                    anchorOrigin={popoverAnchorOrigin}
                    transformOrigin={popoverTransformOrigin}
                >
                    <Box>
                        <MenuItem onClick={onAttachFile}>
                            <ListItemIcon>
                                <FileIcon />
                            </ListItemIcon>
                            <ListItemText primary="Attach File" secondary="Attach a file from your device" />
                        </MenuItem>
                        {externalApps.length > 0 && <MenuItem onClick={handleOpenExternalApps}>
                            <ListItemIcon>
                                <LinkIcon />
                            </ListItemIcon>
                            <ListItemText primary="Connect External App" secondary="Connect an external app to your account" />
                        </MenuItem>}
                        <MenuItem onClick={onTakePhoto}>
                            <ListItemIcon>
                                <CameraOpenIcon />
                            </ListItemIcon>
                            <ListItemText primary="Take Photo" secondary="Take a photo from your device" />
                        </MenuItem>
                        <MenuItem onClick={onAddNewTool}>
                            <ListItemIcon>
                                <RoutineIcon />
                            </ListItemIcon>
                            <ListItemText primary="Add New Tool" secondary="Add a new tool to your account" />
                        </MenuItem>
                    </Box>
                </Popover>
                <Menu
                    anchorEl={externalAppAnchor}
                    open={Boolean(externalAppAnchor)}
                    onClose={handleCloseExternalApps}
                    anchorOrigin={popoverAnchorOrigin}
                    transformOrigin={popoverTransformOrigin}
                >
                    {externalApps.map((app) => (
                        <MenuItem
                            key={app.id}
                            onClick={() => {
                                // Toggle connection for this app, e.g., call a function like toggleAppConnection(app.id)
                                handleCloseExternalApps();
                            }}
                        >
                            <ListItemIcon>
                                {app.icon}
                            </ListItemIcon>
                            <ListItemText primary={app.name} secondary="Description about the app" />
                            {app.connected && <CompleteIcon />}
                        </MenuItem>
                    ))}
                </Menu>
            </>
        );
    });
PlusMenu.displayName = "PlusMenu";

/** 
 * SettingsMenu Component - renders the settings menu.
 */
interface SettingsMenuProps {
    anchorEl: HTMLElement | null;
    onClose: () => void;
    localEnterWillSubmit: boolean;
    onToggleEnterWillSubmit: () => void;
    isWysiwyg: boolean;
    onToggleWysiwyg: () => void;
    showToolbar: boolean;
    onToggleToolbar: () => void;
}

const settingsMenuAnchorOrigin = { vertical: "bottom", horizontal: "left" } as const;
const SettingsMenu: React.FC<SettingsMenuProps> = React.memo(
    ({
        anchorEl,
        onClose,
        localEnterWillSubmit,
        onToggleEnterWillSubmit,
        isWysiwyg,
        onToggleWysiwyg,
        showToolbar,
        onToggleToolbar,
    }) => {
        return (
            <Menu
                anchorEl={anchorEl}
                open={Boolean(anchorEl)}
                onClose={onClose}
                anchorOrigin={settingsMenuAnchorOrigin}
            >
                <Box display="flex" flexDirection="column">
                    <MenuItem onClick={onToggleEnterWillSubmit}>
                        <ListItemIcon>
                            <Checkbox checked={localEnterWillSubmit} color="success" />
                        </ListItemIcon>
                        <ListItemText
                            primary="Enter key sends message"
                            secondary={`When enabled, use '${keyComboToString("Shift", WindowsKey.Enter)}' to create a new line.`}
                        />
                    </MenuItem>
                    <MenuItem onClick={onToggleWysiwyg}>
                        <ListItemIcon>
                            <Checkbox checked={isWysiwyg} color="success" />
                        </ListItemIcon>
                        <ListItemText
                            primary="WYSIWYG editor mode"
                            secondary="Toggle rich text editing mode."
                        />
                    </MenuItem>
                    <MenuItem onClick={onToggleToolbar}>
                        <ListItemIcon>
                            <Checkbox checked={showToolbar} color="success" />
                        </ListItemIcon>
                        <ListItemText
                            primary="Show formatting toolbar"
                            secondary="Display text formatting options above the input area."
                        />
                    </MenuItem>
                </Box>
            </Menu>
        );
    });
SettingsMenu.displayName = "SettingsMenu";

export function AdvancedInput({
    enterWillSubmit,
    tools,
    contextData,
    onToolsChange,
    onContextDataChange,
    onSubmit,
    onAddNewTool,
}: AdvancedInputProps) {
    const theme = useTheme();

    const [textValue, setTextValue] = useState("");
    const [anchorPlus, setAnchorPlus] = useState<HTMLElement | null>(null);
    const [anchorSettings, setAnchorSettings] = useState<HTMLElement | null>(null);

    // Settings toggles
    const [localEnterWillSubmit, setLocalEnterWillSubmit] = useState(enterWillSubmit);
    const [showToolbar, setShowToolbar] = useState(false);
    const [isWysiwyg, setIsWysiwyg] = useState(false);

    // Tools overflow
    const [isToolsExpanded, setIsToolsExpanded] = useState(false);
    const toggleToolsExpanded = useCallback(() => {
        setIsToolsExpanded((prev) => !prev);
    }, []);
    const { dimensions: toolsContainerDimensions, ref: toolsContainerRef } = useDimensions();
    const [showAllToolsButton, setShowAllToolsButton] = useState(false);
    const showAllToolsButtonRef = useRef(false);
    const [showEllipsisBeforeLastTool, setShowEllipsisBeforeLastTool] = useState(false);
    const showEllipsisBeforeLastToolRef = useRef(false);
    const [toolsInFirstRow, setToolsInFirstRow] = useState(0);
    const toolsInFirstRowRef = useRef(0);
    const checkOverflow = useCallback(function checkIfToolsOverflowCallback() {
        if (toolsContainerRef.current) {
            // Get an array of children elements
            const childrenArray = Array.from(toolsContainerRef.current.children);
            if (childrenArray.length === 0) return;

            // Determine the top offset of the first row
            const firstRowTop = (childrenArray[0] as HTMLElement).offsetTop;
            // Count all children that share the same offsetTop as the first child
            const firstRowItems = childrenArray.filter(child => (child as HTMLElement).offsetTop === firstRowTop);
            const firstRowToolItems = firstRowItems.filter(child => (child as HTMLElement).getAttribute("data-type") === "tool");

            // Check how much space if left in the container
            const containerRect = toolsContainerRef.current.getBoundingClientRect();
            let maxRight = 0;
            firstRowItems.forEach((child, index) => {
                // Make sure we're not adding the ellipsis before the last item.
                if (child.getAttribute("data-id") === "show-all-tools-button" && index !== firstRowItems.length - 1) {
                    maxRight = Number.MAX_SAFE_INTEGER;
                } else if (child.getAttribute("data-type") === "tool") {
                    const childRect = child.getBoundingClientRect();
                    maxRight = Math.max(maxRight, childRect.right);
                }
            });
            const remWidth = containerRect.right - maxRight;

            // Determine if the ellipsis should be added before or after the last item in the first row
            const shouldAddEllipsisBefore = Math.abs(remWidth) < 40;
            console.log({ maxRight, remWidth, shouldAddEllipsisBefore });

            if (firstRowToolItems.length !== toolsInFirstRowRef.current) {
                setToolsInFirstRow(firstRowToolItems.length);
                toolsInFirstRowRef.current = firstRowToolItems.length;
            }
            if (shouldAddEllipsisBefore !== showEllipsisBeforeLastToolRef.current) {
                setShowEllipsisBeforeLastTool(shouldAddEllipsisBefore);
                showEllipsisBeforeLastToolRef.current = shouldAddEllipsisBefore;
            }
            if (firstRowToolItems.length < tools.length !== showAllToolsButtonRef.current) {
                setShowAllToolsButton(firstRowToolItems.length < tools.length);
                showAllToolsButtonRef.current = firstRowToolItems.length < tools.length;
            }
        }
    }, [tools.length, toolsContainerRef]);
    useEffect(function checkOverflowOnResizeEffect() {
        checkOverflow();
    }, [checkOverflow, tools, toolsContainerDimensions]);
    const handleToggleTool = useCallback((index: number) => {
        if (!onToolsChange) return;
        const updated = [...tools];
        updated[index] = { ...updated[index], enabled: !updated[index].enabled };
        onToolsChange(updated);
    }, [onToolsChange, tools]);
    const handleRemoveTool = useCallback((index: number) => {
        if (!onToolsChange) return;
        const updated = [...tools];
        updated.splice(index, 1);
        onToolsChange(updated);
    }, [onToolsChange, tools]);

    // Memoized sorted context items (files/images come before text)
    const sortedContextData = useMemo(() => {
        return [...contextData].sort((a, b) => {
            const aIsFile = a.type === "file" || a.type === "image";
            const bIsFile = b.type === "file" || b.type === "image";
            if (aIsFile && !bIsFile) return -1;
            if (!aIsFile && bIsFile) return 1;
            return 0;
        });
    }, [contextData]);

    // Handlers memoized with useCallback
    const handleOpenPlusMenu = useCallback((event: MouseEvent<HTMLElement>) => {
        setAnchorPlus(event.currentTarget);
    }, []);

    const handleClosePlusMenu = useCallback(() => {
        setAnchorPlus(null);
    }, []);

    const handleOpenSettingsMenu = useCallback((event: MouseEvent<HTMLElement>) => {
        setAnchorSettings(event.currentTarget);
    }, []);

    const handleCloseSettingsMenu = useCallback(() => {
        setAnchorSettings(null);
    }, []);

    const handleToggleEnterWillSubmit = useCallback(() => {
        setLocalEnterWillSubmit((prev) => !prev);
    }, []);

    const handleToggleToolbar = useCallback(() => {
        setShowToolbar((prev) => !prev);
    }, []);

    const handleToggleWysiwyg = useCallback(() => {
        setIsWysiwyg((prev) => !prev);
    }, []);

    const handleAttachFile = useCallback(() => {
        console.log("Attach file...");
        handleClosePlusMenu();
    }, [handleClosePlusMenu]);

    const handleConnectExternalApp = useCallback(() => {
        console.log("Connect external app...");
        handleClosePlusMenu();
    }, [handleClosePlusMenu]);

    const handleTakePhoto = useCallback(() => {
        console.log("Take photo...");
        handleClosePlusMenu();
    }, [handleClosePlusMenu]);

    const handleClickAddNewTool = useCallback(() => {
        if (onAddNewTool) onAddNewTool();
        handleClosePlusMenu();
    }, [onAddNewTool, handleClosePlusMenu]);

    const handleRemoveContextItem = useCallback(
        (id: string) => {
            if (!onContextDataChange) return;
            const updated = contextData.filter((item) => item.id !== id);
            onContextDataChange(updated);
        },
        [onContextDataChange, contextData],
    );

    const handleSend = useCallback(() => {
        if (onSubmit && textValue.trim()) {
            onSubmit(textValue.trim());
        }
        setTextValue("");
    }, [onSubmit, textValue]);

    const handleInputChange = useCallback(
        (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
            setTextValue(e.target.value);
        },
        [],
    );

    const handleKeyDown = useCallback(
        (e: KeyboardEvent<HTMLTextAreaElement | HTMLDivElement>) => {
            if (e.key === "Enter" && !e.shiftKey) {
                if (localEnterWillSubmit) {
                    e.preventDefault();
                    handleSend();
                }
            }
        },
        [handleSend, localEnterWillSubmit],
    );

    const handleTranscriptChange = useCallback((recognizedText: string) => {
        setTextValue((prev) => {
            // Append recognized text to whatever is currently typed
            if (!prev.trim()) return recognizedText;
            return `${prev} ${recognizedText}`;
        });
    }, []);

    // Additional memoized styles for the WYSIWYG box
    const wysiwygBoxStyles = useMemo(
        () => ({
            minHeight: 60,
            p: 1,
        }),
        [],
    );

    const imgStyle = useMemo(() => previewImageStyle(theme), [theme]);
    const toolsContainerStyles = useMemo(() => ({
        display: "flex",
        flexWrap: "wrap" as const,
        alignItems: "center",
        maxHeight: isToolsExpanded ? "none" : theme.spacing(4), // Approximately one row height
        overflow: "hidden",
        transition: "max-height 0.3s ease",
        gap: theme.spacing(1),
    }), [isToolsExpanded, theme]);

    return (
        <Outer>
            {/* Top Section */}
            {showToolbar ? (
                <>
                    <Box sx={toolbarRowStyles}>
                        <IconButton onClick={handleOpenSettingsMenu} sx={{ padding: "4px" }}>
                            <SettingsIcon fill={theme.palette.background.textSecondary} />
                        </IconButton>
                        <Typography variant="body2" ml={1} color="text.secondary">
                            [Toolbar placeholder: Bold, Italic, Link, etc.]
                        </Typography>
                    </Box>
                    <Box sx={contextRowStyles}>
                        {sortedContextData.map((item) => (
                            <ContextItemDisplay
                                key={item.id}
                                item={item}
                                imgStyle={imgStyle}
                                onRemove={handleRemoveContextItem}
                                theme={theme}
                            />
                        ))}
                    </Box>
                </>
            ) : (
                <Box sx={topRowStyles}>
                    <IconButton onClick={handleOpenSettingsMenu} sx={{ padding: "4px" }}>
                        <SettingsIcon fill={theme.palette.background.textSecondary} />
                    </IconButton>
                    {sortedContextData.map((item) => (
                        <ContextItemDisplay
                            key={item.id}
                            item={item}
                            imgStyle={imgStyle}
                            onRemove={handleRemoveContextItem}
                            theme={theme}
                        />
                    ))}
                </Box>
            )}

            {/* Input Area */}
            <Box sx={inputRowStyles}>
                {isWysiwyg ? (
                    <Box sx={wysiwygBoxStyles}>
                        <Typography variant="body2" color="text.secondary">
                            [WYSIWYG Editor Placeholder]
                        </Typography>
                    </Box>
                ) : (
                    <InputBase
                        multiline
                        fullWidth
                        minRows={1}
                        maxRows={6}
                        value={textValue}
                        onChange={handleInputChange}
                        onKeyDown={handleKeyDown}
                        placeholder="Type your message..."
                    />
                )}
            </Box>

            {/* Bottom Section */}
            <Box sx={bottomRowStyles}>
                <Box ref={toolsContainerRef} sx={toolsContainerStyles}>
                    <StyledIconButton disabled={false} onClick={handleOpenPlusMenu}>
                        <AddIcon />
                    </StyledIconButton>
                    {tools.map((tool, index) => {
                        function onToggleTool() {
                            handleToggleTool(index);
                        }
                        function onRemoveTool() {
                            handleRemoveTool(index);
                        }

                        const canAddShowButton = toolsInFirstRow < tools.length && !isToolsExpanded && ((index === toolsInFirstRow && !showEllipsisBeforeLastTool) || (index === toolsInFirstRow - 1 && showEllipsisBeforeLastTool));
                        const canAddHideButton = toolsInFirstRow < tools.length && isToolsExpanded && index === tools.length - 1;

                        return (
                            <>
                                {canAddShowButton && <ShowHideIconButton data-id="show-all-tools-button" disabled={false} onClick={toggleToolsExpanded}>
                                    <EllipsisIcon />
                                </ShowHideIconButton>}
                                <ToolChip
                                    data-type="tool"
                                    enabled={tool.enabled}
                                    key={`${tool.name}-${index}`}
                                    icon={tool.icon as React.ReactElement}
                                    label={tool.displayName}
                                    onDelete={onRemoveTool}
                                    onClick={onToggleTool}
                                />
                                {canAddHideButton && <ShowHideIconButton data-id="hide-all-tools-button" disabled={false} onClick={toggleToolsExpanded}>
                                    <InvisibleIcon />
                                </ShowHideIconButton>}
                            </>
                        );
                    })}
                </Box>
                {/* {toolsOverflow && <Typography variant="body2" color="text.secondary" onClick={toggleToolsExpanded}>...</Typography>} */}
                <Box flex={1} />
                <Box display="flex" alignItems="center" gap={1} ml={1}>
                    <MicrophoneButton
                        onTranscriptChange={handleTranscriptChange}
                        disabled={false}
                        height={iconHeight}
                        width={iconWidth}
                    />
                    <StyledIconButton bgColor={theme.palette.background.textPrimary} disabled={!textValue.trim()} onClick={handleSend}>
                        <SendIcon fill={!textValue.trim() ? theme.palette.background.textSecondary : theme.palette.background.default} />
                    </StyledIconButton>
                </Box>
            </Box>

            {/* Popup Menus */}
            <PlusMenu
                anchorEl={anchorPlus}
                onClose={handleClosePlusMenu}
                onAttachFile={handleAttachFile}
                onConnectExternalApp={handleConnectExternalApp}
                onTakePhoto={handleTakePhoto}
                onAddNewTool={handleClickAddNewTool}
            />

            <SettingsMenu
                anchorEl={anchorSettings}
                onClose={handleCloseSettingsMenu}
                localEnterWillSubmit={localEnterWillSubmit}
                onToggleEnterWillSubmit={handleToggleEnterWillSubmit}
                isWysiwyg={isWysiwyg}
                onToggleWysiwyg={handleToggleWysiwyg}
                showToolbar={showToolbar}
                onToggleToolbar={handleToggleToolbar}
            />
        </Outer>
    );
}

