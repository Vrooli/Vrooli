/* eslint-disable no-magic-numbers */
import {
    Avatar,
    Box,
    Chip,
    Collapse,
    Divider,
    IconButton,
    IconButtonProps,
    InputBase,
    ListItemIcon,
    ListItemText,
    Menu,
    MenuItem,
    Popover,
    Switch,
    Typography,
    styled,
    useTheme,
} from "@mui/material";
import type { SxProps, Theme } from "@mui/material/styles";
import { CSSProperties } from "@mui/styles";
import { MicrophoneButton } from "components/buttons/MicrophoneButton/MicrophoneButton.js";
import { FindObjectDialog } from "components/dialogs/FindObjectDialog/FindObjectDialog.js";
import { MarkdownDisplay } from "components/text/MarkdownDisplay.js";
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
import {
    AddIcon,
    CameraOpenIcon,
    CloseIcon,
    CompleteIcon,
    EllipsisIcon,
    FileIcon,
    ImageIcon,
    InfoIcon,
    InvisibleIcon,
    LinkIcon,
    PauseIcon,
    PlayIcon,
    RoutineIcon,
    SendIcon,
} from "../../../icons/common.js";


interface ExternalApp {
    id: string;
    name: string;
    icon: React.ReactNode;
    connected: boolean;
}

// Add supported external apps here
const externalApps: ExternalApp[] = [
];
const findRoutineLimitTo = ["RoutineMultiStep", "RoutineSingleStep"] as const;

const iconHeight = 32;
const iconWidth = 32;

export enum ToolState {
    /** Tool not provided to LLM */
    Disabled = "disabled",
    /** Tool provided to LLM with other enabled tools */
    Enabled = "enabled",
    /** LLM instructed to use this tool only */
    Exclusive = "exclusive",
}
export interface Tool {
    displayName: string;
    icon: React.ReactNode;
    type: string;
    name: string;
    state: ToolState;
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
    onToolsChange?: (updatedTools: Tool[]) => unknown;
    onContextDataChange?: (updatedContext: ContextItem[]) => unknown;
    onSubmit?: (message: string) => unknown;
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
const StyledIconButton = styled(IconButton)<StyledIconButtonProps>(({ disabled, bgColor }) => ({
    width: iconWidth,
    height: iconHeight,
    opacity: disabled ? 0.5 : 1,
    padding: "4px",
    backgroundColor: bgColor || "transparent",
}));
const ShowHideIconButton = styled(StyledIconButton)(() => ({
    border: "none",
}));


type ToolChipProps = Tool & {
    index: number;
    onRemoveTool: () => unknown;
    onToggleTool: () => unknown;
    onToggleToolExclusive: () => unknown;
}

function ToolChip({
    displayName,
    icon,
    index,
    name,
    onRemoveTool,
    onToggleTool,
    onToggleToolExclusive,
    state,
}: ToolChipProps) {
    const { palette } = useTheme();

    const [isHovered, setIsHovered] = useState(false);
    function handleMouseEnter() {
        setIsHovered(true);
    }
    function handleMouseLeave() {
        setIsHovered(false);
    }

    const handlePlayClick = useCallback(function handlePlayClickCallback(event: React.MouseEvent) {
        // Prevent triggering chip's onClick
        event.stopPropagation();
        onToggleToolExclusive();
    }, [onToggleToolExclusive]);

    const chipVariant = state === ToolState.Disabled ? "outlined" : "filled";
    const chipStyle = useMemo(() => {
        let backgroundColor: string;
        let color: string;
        let border: string;

        switch (state) {
            case ToolState.Disabled: {
                backgroundColor = "transparent";
                color = palette.background.textSecondary;
                border = `1px solid ${palette.divider}`;
                break;
            }
            case ToolState.Enabled: {
                backgroundColor = palette.secondary.light,
                    color = palette.secondary.contrastText,
                    border = "none";
                break;
            }
            case ToolState.Exclusive: {
                backgroundColor = palette.secondary.main;
                color = palette.secondary.contrastText;
                border = "none";
                break;
            }
        }
        return {
            backgroundColor,
            color,
            border,
            cursor: "pointer",
            "&:hover": {
                backgroundColor,
                color,
                filter: "brightness(1.05)",
            },
        } as const;
    }, [state, palette]);

    const chipIcon = useMemo(() => {
        if (isHovered) {
            if (state !== ToolState.Exclusive) {
                return (
                    <IconButton
                        size="small"
                        onClick={handlePlayClick}
                        sx={{ padding: 0, paddingRight: 0.5 }}
                    >
                        <PlayIcon width={20} height={20} />
                    </IconButton>
                );
            }
            return (
                <IconButton
                    size="small"
                    onClick={handlePlayClick}
                    sx={{ padding: 0, paddingRight: 0.5 }}
                >
                    <PauseIcon width={20} height={20} />
                </IconButton>
            );
        }
        return icon as React.ReactElement;
    }, [isHovered, icon, state, handlePlayClick]);

    return (
        <Chip
            data-type="tool"
            key={`${name}-${index}`}
            icon={chipIcon}
            label={displayName}
            onDelete={onRemoveTool}
            onClick={onToggleTool}
            onMouseEnter={handleMouseEnter}
            onMouseLeave={handleMouseLeave}
            sx={chipStyle}
            variant={chipVariant}
        />
    );
}

export function useToolActions(tools: Tool[], onToolsChange?: (updatedTools: Tool[]) => unknown) {
    const handleToggleTool = useCallback(
        (index: number) => {
            if (!onToolsChange) return;
            const updated = [...tools];
            const currentState = updated[index].state;
            let newState: ToolState;
            if (currentState === ToolState.Disabled) {
                newState = ToolState.Enabled;
            } else if (currentState === ToolState.Enabled) {
                newState = ToolState.Disabled;
            } else {
                // Exclusive -> Enabled
                newState = ToolState.Enabled;
            }
            updated[index] = { ...updated[index], state: newState };
            onToolsChange(updated);
        },
        [tools, onToolsChange],
    );

    const handleToggleToolExclusive = useCallback(
        (index: number) => {
            if (!onToolsChange) return;
            let updated = [...tools];
            // If the tool is already exclusive, set it to enabled
            const currentState = tools[index].state;
            if (currentState === ToolState.Exclusive) {
                updated = tools.map((tool, i) => {
                    if (i === index) {
                        return { ...tool, state: ToolState.Enabled };
                    } else {
                        return tool;
                    }
                });
            }
            // Otherwise, set the tool to exclusive and any existing exclusives to enabled
            else {
                updated = tools.map((tool, i) => {
                    if (i === index) {
                        return { ...tool, state: ToolState.Exclusive };
                    } else if (tool.state === ToolState.Exclusive) {
                        return { ...tool, state: ToolState.Enabled };
                    } else {
                        return tool;
                    }
                });
            }
            onToolsChange(updated);
        },
        [tools, onToolsChange],
    );

    const handleRemoveTool = useCallback(
        (index: number) => {
            if (!onToolsChange) return;
            const updated = [...tools];
            updated.splice(index, 1);
            onToolsChange(updated);
        },
        [tools, onToolsChange],
    );

    return { handleToggleTool, handleToggleToolExclusive, handleRemoveTool };
}

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
const ContextItemChip = styled(Chip)(({ theme }) => ({
    marginRight: theme.spacing(1),
    marginBottom: theme.spacing(1),
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
        <ContextItemChip
            label={item.label}
            onDelete={handleRemove}
        />
    );
}

/** 
 * PlusMenu Component - renders the popover for additional actions.
 */
interface PlusMenuProps {
    anchorEl: HTMLElement | null;
    onClose: () => unknown;
    onAttachFile: () => unknown;
    onConnectExternalApp: () => unknown;
    onTakePhoto: () => unknown;
    onAddRoutine: () => unknown;
}

const PlusMenu: React.FC<PlusMenuProps> = React.memo(
    ({
        anchorEl,
        onClose,
        onAttachFile,
        onConnectExternalApp,
        onTakePhoto,
        onAddRoutine,
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
                        <MenuItem onClick={onAddRoutine}>
                            <ListItemIcon>
                                <RoutineIcon />
                            </ListItemIcon>
                            <ListItemText primary="Add Routine" secondary="Allow the AI to perform actions" />
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
 * InfoMemo Component - renders a menu showing information about the input and customization settings.
 */
interface InfoMemoProps {
    anchorEl: HTMLElement | null;
    onClose: () => void;
    localEnterWillSubmit: boolean;
    onToggleEnterWillSubmit: () => void;
    isWysiwyg: boolean;
    onToggleWysiwyg: () => void;
    showToolbar: boolean;
    onToggleToolbar: () => void;
}

const infoMenuAnchorOrigin = { vertical: "bottom", horizontal: "left" } as const;
const infoMenuPaperProps = {
    style: {
        maxWidth: "500px",
        maxHeight: "500px",
        overflow: "auto",
    },
} as const;
const InfoMemo: React.FC<InfoMemoProps> = React.memo(
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
                anchorOrigin={infoMenuAnchorOrigin}
                PaperProps={infoMenuPaperProps}
            >
                <Typography m={2} mb={1} variant="h6" sx={{ color: "text.secondary" }}>
                    Settings
                </Typography>
                <Divider />
                <MenuItem onClick={onToggleEnterWillSubmit}>
                    <ListItemText
                        primary="Enter key sends message"
                        secondary={"When enabled, use 'Shift + Enter' to create a new line."}
                        secondaryTypographyProps={{ style: { whiteSpace: "pre-wrap" } }}
                    />
                    <Switch
                        edge="end"
                        checked={localEnterWillSubmit}
                        onChange={(event) => {
                            event.stopPropagation();
                            onToggleEnterWillSubmit();
                        }}
                    />
                </MenuItem>
                <Divider />
                <MenuItem onClick={onToggleWysiwyg}>
                    <ListItemText
                        primary="WYSIWYG editor mode"
                        secondary="Toggle rich text editing mode."
                        secondaryTypographyProps={{ style: { whiteSpace: "pre-wrap" } }}
                    />
                    <Switch
                        edge="end"
                        checked={isWysiwyg}
                        onChange={(event) => {
                            event.stopPropagation();
                            onToggleWysiwyg();
                        }}
                    />
                </MenuItem>
                <Divider />
                <MenuItem onClick={onToggleToolbar}>
                    <ListItemText
                        primary="Show formatting toolbar"
                        secondary="Display text formatting options above the input area."
                        secondaryTypographyProps={{ style: { whiteSpace: "pre-wrap" } }}
                    />
                    <Switch
                        edge="end"
                        checked={showToolbar}
                        onChange={(event) => {
                            event.stopPropagation();
                            onToggleToolbar();
                        }}
                    />
                </MenuItem>
                <Divider />
                {/* TODO: Add this back in */}
                {/* <MenuItem onClick={onToggleToolbar}>
                    <ListItemText
                        primary="Disable default tools"
                        secondary="Will not provide the AI with default tools, such as searching the site, creating a new routine, etc."
                        secondaryTypographyProps={{ style: { whiteSpace: "pre-wrap" } }}
                    />
                    <Switch
                        edge="end"
                        checked={showToolbar}
                        onChange={(event) => {
                            event.stopPropagation();
                            onToggleDisableDefaultTools();
                        }}
                    />
                </MenuItem> */}
                <Divider />
                <Typography variant="h6" m={2} mb={1} sx={{ color: "text.secondary" }}>
                    Info
                </Typography>
                <Box p={2}>
                    <MarkdownDisplay
                        content={`This input component allows you to:

- Attach files,
- Connect to external services,
- Add tools,
- and possibly more in the future.

Some features may be unavailable depending on the AI models you're chatting with and the device this is running on. 

Tools are displayed at the bottom and can be active or inactive. Active tools are provided to the AI, which can choose to use or not use them. 

To run a tool directly, first enable it and then press the 'Run' button. This will display any input fields required by the tool, which can be filled in if desired. If you don't fill these in, the AI will choose what to enter.
                    `}
                    />
                </Box>
            </Menu >
        );
    },
);
InfoMemo.displayName = "InfoMemo";

export function AdvancedInput({
    enterWillSubmit,
    tools,
    contextData,
    onToolsChange,
    onContextDataChange,
    onSubmit,
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
                showAllToolsButtonRef.current = firstRowToolItems.length < tools.length;
            }
        }
    }, [tools.length, toolsContainerRef]);
    useEffect(function checkOverflowOnResizeEffect() {
        checkOverflow();
    }, [checkOverflow, tools, toolsContainerDimensions]);

    const { handleToggleTool, handleToggleToolExclusive, handleRemoveTool } = useToolActions(tools, onToolsChange);

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

    const handleOpenInfoMemo = useCallback((event: MouseEvent<HTMLElement>) => {
        setAnchorSettings(event.currentTarget);
    }, []);

    const handleCloseInfoMemo = useCallback(() => {
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

    const [isFindRoutineDialogOpen, setIsFindRoutineDialogOpen] = useState(false);
    const handleOpenFindRoutineDialog = useCallback(() => {
        setIsFindRoutineDialogOpen(true);
        handleClosePlusMenu();
    }, [handleClosePlusMenu]);
    const handleCloseFindRoutineDialog = useCallback(() => {
        setIsFindRoutineDialogOpen(false);
    }, []);
    const handleAddRoutine = useCallback(() => {
        console.log("Add new tool...");
        handleCloseFindRoutineDialog();
    }, [handleCloseFindRoutineDialog]);

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
            <Collapse in={tools.some((tool) => tool.state === ToolState.Exclusive)} unmountOnExit>
                <Box height={400} >
                    {/* <Formik
                        initialValues={formikRef.current?.initialValues ?? EMPTY_OBJECT}
                        innerRef={formikRef}
                        onSubmit={noopSubmit} // Form submission is handled elsewhere
                        validationSchema={inputValidationSchema}
                    >
                        {() => (
                            <FormSection>
                                {routineTypeComponents}
                            </FormSection>
                        )}
                    </Formik> */}
                </Box>
                <Divider />
            </Collapse>
            {/* Top Section */}
            {showToolbar ? (
                <>
                    <Box sx={toolbarRowStyles}>
                        <IconButton onClick={handleOpenInfoMemo} sx={{ padding: "4px", opacity: 0.5 }}>
                            <InfoIcon fill={theme.palette.background.textSecondary} />
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
                    <IconButton onClick={handleOpenInfoMemo} sx={{ padding: "4px", opacity: 0.5 }}>
                        <InfoIcon fill={theme.palette.background.textSecondary} />
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
                        <AddIcon fill={theme.palette.background.textSecondary} />
                    </StyledIconButton>
                    {tools.map((tool, index) => {
                        function onToggleTool() {
                            handleToggleTool(index);
                        }
                        function onRemoveTool() {
                            handleRemoveTool(index);
                        }
                        function onToggleToolExclusive() {
                            handleToggleToolExclusive(index);
                        }

                        const canAddShowButton = toolsInFirstRow < tools.length && !isToolsExpanded && ((index === toolsInFirstRow && !showEllipsisBeforeLastTool) || (index === toolsInFirstRow - 1 && showEllipsisBeforeLastTool));
                        const canAddHideButton = toolsInFirstRow < tools.length && isToolsExpanded && index === tools.length - 1;

                        return (
                            <>
                                {canAddShowButton && <ShowHideIconButton data-id="show-all-tools-button" disabled={false} onClick={toggleToolsExpanded}>
                                    <EllipsisIcon />
                                </ShowHideIconButton>}
                                <ToolChip
                                    {...tool}
                                    data-type="tool"
                                    index={index}
                                    key={`${tool.name}-${index}`}
                                    onRemoveTool={onRemoveTool}
                                    onToggleToolExclusive={onToggleToolExclusive}
                                    onToggleTool={onToggleTool}
                                />
                                {canAddHideButton && <ShowHideIconButton data-id="hide-all-tools-button" disabled={false} onClick={toggleToolsExpanded}>
                                    <InvisibleIcon />
                                </ShowHideIconButton>}
                            </>
                        );
                    })}
                </Box>
                <Box flex={1} />
                <Box display="flex" alignItems="center" gap={1} ml={1}>
                    <MicrophoneButton
                        fill={theme.palette.background.textSecondary}
                        onTranscriptChange={handleTranscriptChange}
                        disabled={false}
                        height={iconHeight}
                        width={iconWidth}
                    />
                    <StyledIconButton
                        bgColor={theme.palette.mode === "dark" ? theme.palette.background.textPrimary : theme.palette.primary.main}
                        disabled={!textValue.trim()}
                        onClick={handleSend}
                    >
                        <SendIcon
                            fill={!textValue.trim() ?
                                theme.palette.background.textSecondary :
                                theme.palette.mode === "dark" ?
                                    theme.palette.background.default :
                                    theme.palette.primary.contrastText
                            }
                        />
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
                onAddRoutine={handleOpenFindRoutineDialog}
            />
            <InfoMemo
                anchorEl={anchorSettings}
                onClose={handleCloseInfoMemo}
                localEnterWillSubmit={localEnterWillSubmit}
                onToggleEnterWillSubmit={handleToggleEnterWillSubmit}
                isWysiwyg={isWysiwyg}
                onToggleWysiwyg={handleToggleWysiwyg}
                showToolbar={showToolbar}
                onToggleToolbar={handleToggleToolbar}
            />
            <FindObjectDialog
                find="List"
                isOpen={isFindRoutineDialogOpen}
                handleCancel={handleCloseFindRoutineDialog}
                handleComplete={handleAddRoutine}
                limitTo={findRoutineLimitTo}
            />
        </Outer>
    );
}

