/* eslint-disable no-magic-numbers */
import { Avatar, Box, Chip, CircularProgress, Collapse, Divider, IconButton, IconButtonProps, InputBase, ListItemIcon, ListItemText, Menu, MenuItem, Popover, Switch, Typography, styled, useTheme } from "@mui/material";
import type { SxProps, Theme } from "@mui/material/styles";
import { CSSProperties } from "@mui/styles";
import React, { KeyboardEvent, MouseEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useDropzone } from "react-dropzone";
import { useDebounce } from "../../../hooks/useDebounce.js";
import { useDimensions } from "../../../hooks/useDimensions.js";
import { Icon, IconCommon, IconInfo, IconRoutine } from "../../../icons/Icons.js";
import { getCookie, setCookie } from "../../../utils/localStorage.js";
import { PubSub } from "../../../utils/pubsub.js";
import { MicrophoneButton } from "../../buttons/MicrophoneButton/MicrophoneButton.js";
import { FindObjectDialog } from "../../dialogs/FindObjectDialog/FindObjectDialog.js";
import { SnackSeverity } from "../../snacks/BasicSnack/BasicSnack.js";
import { MarkdownDisplay } from "../../text/MarkdownDisplay.js";

interface ExternalApp {
    id: string;
    name: string;
    iconInfo: IconInfo;
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
    iconInfo: IconInfo;
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
    file?: File;
}

export interface AdvancedInputProps {
    tools: Tool[];
    contextData: ContextItem[];
    maxChars?: number;
    message?: string;
    onMessageChange?: (message: string) => unknown;
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
    transition: "max-height 0.3s ease",
};

const expandedInputStyles: SxProps<Theme> = {
    maxHeight: "calc(100vh - 500px)", // Leave space for other UI elements
    overflow: "auto",
};

const compactInputStyles: SxProps<Theme> = {
    maxHeight: "unset",
    overflow: "visible",
};

const expandButtonStyles: SxProps<Theme> = {
    position: "absolute",
    top: (theme) => theme.spacing(1),
    right: (theme) => theme.spacing(1),
    padding: "4px",
    opacity: 0.5,
    "&:hover": {
        opacity: 0.8,
    },
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
    position: "relative",
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
const toolChipIconButtonStyle = { padding: 0, paddingRight: 0.5 } as const;


type ToolChipProps = Tool & {
    index: number;
    onRemoveTool: () => unknown;
    onToggleTool: () => unknown;
    onToggleToolExclusive: () => unknown;
}

function ToolChip({
    displayName,
    iconInfo,
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
                        sx={toolChipIconButtonStyle}
                    >
                        <IconCommon
                            decorative
                            name="Play"
                            size={20}
                        />
                    </IconButton>
                );
            }
            return (
                <IconButton
                    size="small"
                    onClick={handlePlayClick}
                    sx={toolChipIconButtonStyle}
                >
                    <IconCommon
                        decorative
                        name="Pause"
                        size={20}
                    />
                </IconButton>
            );
        }
        return <Icon decorative info={iconInfo} />;
    }, [isHovered, iconInfo, state, handlePlayClick]);

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
};

const MAX_LABEL_LENGTH = 20;
const CONTEXT_ITEM_LIMIT = 20;
const IMAGE_FILE_REGEX = /\.(jpg|jpeg|png|gif|bmp|tiff|ico|webp|svg|heic|heif|ppt|pptx)$/i;
const TEXT_FILE_REGEX = /\.(md|txt|markdown|word|doc|docx|pdf)$/i;
const CODE_FILE_REGEX = /\.(js|jsx|ts|tsx|json|xls|xlsx|yaml|yml|xml|html|css|scss|less|py|java|c|cpp|h|hxx|cxx|hpp|hxx|rb|php|go|swift|kotlin|scala|groovy|rust|haskell|erlang|elixir|dart|typescript|kotlin|swift|ruby|php|go|rust|haskell|erlang|elixir|dart|typescript)$/i;
const VIDEO_FILE_REGEX = /\.(mp4|mov|avi|wmv|flv|mpeg|mpg|m4v|webm|mkv)$/i;
const ENV_FILE_REGEX = /\.(env|env-example|env-local|env-production|env-development|env-test)$/i;
const EXECUTABLE_FILE_REGEX = /\.(exe|bat|sh|bash|cmd|ps1|ps2|ps3|ps4|ps5|ps6|ps7|ps8|ps9|ps10)$/i;

function truncateLabel(label: string, maxLength: number): string {
    if (label.length <= maxLength) return label;
    const extension = label.split(".").pop();
    const nameWithoutExt = label.slice(0, label.lastIndexOf("."));
    if (!extension || nameWithoutExt.length <= maxLength - 4) return label;
    return `${nameWithoutExt.slice(0, maxLength - 4)}...${extension}`;
}

function ContextItemDisplay({
    imgStyle,
    item,
    onRemove,
}: ContextItemDisplayProps) {
    function handleRemove() {
        onRemove(item.id);
    }

    const fallbackIconInfo = useMemo<IconInfo>(function fallbackInfoBasedOnTypeMemo() {
        if (item.type === "image") return { name: "Image", type: "Common" } as const;
        if (item.type === "text") return { name: "Article", type: "Common" } as const;
        // For files, check if it's a text-based file
        if (item.type === "file" && item.file?.type) {
            // Image files
            if (item.file.type.startsWith("image/") || IMAGE_FILE_REGEX.test(item.file.name)) {
                return { name: "Image", type: "Common" } as const;
            }
            // Text/document files
            if (item.file.type.startsWith("text/") || TEXT_FILE_REGEX.test(item.file.name)) {
                return { name: "Article", type: "Common" } as const;
            }
            // Code files
            if (item.file.type.includes("javascript") ||
                item.file.type.includes("json") ||
                CODE_FILE_REGEX.test(item.file.name)) {
                return { name: "Object", type: "Common" } as const;
            }
            // Video files
            if (item.file.type.startsWith("video/") || VIDEO_FILE_REGEX.test(item.file.name)) {
                return { name: "SocialVideo", type: "Common" } as const;
            }
            // Environment files
            if (ENV_FILE_REGEX.test(item.file.name)) {
                return { name: "Invisible", type: "Common" } as const;
            }
            // Executable files
            if (EXECUTABLE_FILE_REGEX.test(item.file.name)) {
                return { name: "Terminal", type: "Common" } as const;
            }
        }
        return { name: "File", type: "Common" } as const;
    }, [item.type, item.file?.type, item.file?.name]);

    // Check if this is an image that should be displayed as a preview
    const shouldShowPreview = useMemo(() => {
        if (item.type === "image") return true;
        if (item.type === "file" && item.file?.type?.startsWith("image/")) return true;
        if (item.type === "file" && IMAGE_FILE_REGEX.test(item.file?.name ?? "")) return true;
        return false;
    }, [item.type, item.file?.type, item.file?.name]);

    const truncatedLabel = useMemo(() => truncateLabel(item.label, MAX_LABEL_LENGTH), [item.label]);

    if (shouldShowPreview) {
        return (
            <PreviewContainer>
                {item.src ? (
                    <img src={item.src} alt={item.label} style={imgStyle} />
                ) : (
                    <PreviewImageAvatar variant="square">
                        <Icon
                            decorative
                            info={fallbackIconInfo}
                        />
                    </PreviewImageAvatar>
                )}
                <RemoveIconButton onClick={handleRemove}>
                    <IconCommon
                        decorative
                        fill="background.default"
                        name="Close"
                    />
                </RemoveIconButton>
            </PreviewContainer>
        );
    }

    return (
        <ContextItemChip
            icon={<Icon decorative info={fallbackIconInfo} />}
            label={truncatedLabel}
            onDelete={handleRemove}
            title={item.label} // Show full name on hover
        />
    );
}

// Add these styles near the top with other styles
const toolbarIconButtonStyle = { padding: "4px", opacity: 0.5 } as const;

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

        const handleAppConnection = useCallback((appId: string) => {
            // Toggle connection for this app, e.g., call a function like toggleAppConnection(app.id)
            handleCloseExternalApps();
        }, []);

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
                                <IconCommon
                                    decorative
                                    name="File"
                                />
                            </ListItemIcon>
                            <ListItemText primary="Attach File" secondary="Attach a file from your device" />
                        </MenuItem>
                        {externalApps.length > 0 && <MenuItem onClick={handleOpenExternalApps}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    name="Link"
                                />
                            </ListItemIcon>
                            <ListItemText primary="Connect External App" secondary="Connect an external app to your account" />
                        </MenuItem>}
                        <MenuItem onClick={onTakePhoto}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    name="CameraOpen"
                                />
                            </ListItemIcon>
                            <ListItemText primary="Take Photo" secondary="Take a photo from your device" />
                        </MenuItem>
                        <MenuItem onClick={onAddRoutine}>
                            <ListItemIcon>
                                <IconRoutine
                                    decorative
                                    name="Routine"
                                />
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
                            onClick={() => handleAppConnection(app.id)}
                        >
                            <ListItemIcon>
                                <Icon decorative info={app.iconInfo} />
                            </ListItemIcon>
                            <ListItemText primary={app.name} secondary="Description about the app" />
                            {app.connected && <IconCommon
                                decorative
                                name="Complete"
                            />}
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

const infoButtonStyle = { padding: "4px", opacity: 0.5 } as const;
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
        const handleEnterWillSubmitChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
            event.stopPropagation();
            onToggleEnterWillSubmit();
        }, [onToggleEnterWillSubmit]);

        const handleWysiwygChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
            event.stopPropagation();
            onToggleWysiwyg();
        }, [onToggleWysiwyg]);

        const handleToolbarChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
            event.stopPropagation();
            onToggleToolbar();
        }, [onToggleToolbar]);

        const secondaryTypographyProps = useMemo(() => ({
            style: { whiteSpace: "pre-wrap" } as const,
        }), []);

        return (
            <Menu
                anchorEl={anchorEl}
                open={Boolean(anchorEl)}
                onClose={onClose}
                anchorOrigin={infoMenuAnchorOrigin}
                PaperProps={infoMenuPaperProps}
            >
                <Typography m={2} mb={1} variant="h6" color="text.secondary">
                    Settings
                </Typography>
                <Divider />
                <MenuItem onClick={onToggleEnterWillSubmit}>
                    <ListItemText
                        primary="Enter key sends message"
                        secondary="When enabled, use 'Shift + Enter' to create a new line."
                        secondaryTypographyProps={secondaryTypographyProps}
                    />
                    <Switch
                        edge="end"
                        checked={localEnterWillSubmit}
                        onChange={handleEnterWillSubmitChange}
                    />
                </MenuItem>
                <Divider />
                <MenuItem onClick={onToggleWysiwyg}>
                    <ListItemText
                        primary="WYSIWYG editor mode"
                        secondary="Toggle rich text editing mode."
                        secondaryTypographyProps={secondaryTypographyProps}
                    />
                    <Switch
                        edge="end"
                        checked={isWysiwyg}
                        onChange={handleWysiwygChange}
                    />
                </MenuItem>
                <Divider />
                <MenuItem onClick={onToggleToolbar}>
                    <ListItemText
                        primary="Show formatting toolbar"
                        secondary="Display text formatting options above the input area."
                        secondaryTypographyProps={secondaryTypographyProps}
                    />
                    <Switch
                        edge="end"
                        checked={showToolbar}
                        onChange={handleToolbarChange}
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
                {/* <Divider /> */}
                <Typography variant="h6" m={2} mb={1} color="text.secondary">
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

const dragOverlayStyles = {
    position: "absolute",
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: "rgba(0, 0, 0, 0.1)",
    borderRadius: (theme: Theme) => theme.spacing(3),
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    zIndex: theme => theme.zIndex.modal - 1,
    pointerEvents: "none",
} as const;

// Add this function before the AdvancedInput component
async function getFilesFromEvent(event: any): Promise<(File | DataTransferItem)[]> {
    const items = event.dataTransfer ? [...event.dataTransfer.items] : [];
    const files = event.dataTransfer ? [...event.dataTransfer.files] : [];

    // Handle text drops
    const textItems = items.filter(item => item.kind === "string" && item.type === "text/plain");
    const textPromises = textItems.map(item => new Promise<File>((resolve) => {
        item.getAsString((text: string) => {
            // Create a file-like object for the text
            const file = new File([text], "dropped-text.txt", { type: "text/plain" });
            resolve(file);
        });
    }));

    // Wait for all text items to be processed
    const textFiles = await Promise.all(textPromises);

    // Return both regular files and text files
    return [...files, ...textFiles];
}

const inputBaseInputProps = {
    className: "advanced-input-field",
} as const;

export function AdvancedInput({
    tools,
    contextData,
    maxChars,
    message,
    onMessageChange,
    onToolsChange,
    onContextDataChange,
    onSubmit,
}: AdvancedInputProps) {
    const theme = useTheme();

    // Add expanded view state
    const [isExpanded, setIsExpanded] = useState(false);
    const handleToggleExpand = useCallback(() => {
        setIsExpanded(prev => !prev);
    }, []);

    // Local state for the input value
    const [localMessage, setLocalMessage] = useState(message ?? "");
    const latestMessageRef = useRef(localMessage);
    const [debouncedMessageChange] = useDebounce((value: string) => {
        // Only notify parent if the value has actually changed
        if (value !== latestMessageRef.current) {
            onMessageChange?.(value);
        }
    }, 300);

    // Update local message when parent message changes, but only if it's different from what we have
    useEffect(function updateLocalMessageEffect() {
        if (message !== undefined && message !== latestMessageRef.current) {
            setLocalMessage(message);
            latestMessageRef.current = message;
        }
    }, [message]);

    // Settings state from localStorage
    const [settings, setSettings] = useState(() => getCookie("AdvancedInputSettings"));
    const { enterWillSubmit: localEnterWillSubmit, showToolbar, isWysiwyg } = settings;

    // Settings toggles
    const handleToggleEnterWillSubmit = useCallback(() => {
        const newSettings = {
            ...settings,
            enterWillSubmit: !settings.enterWillSubmit,
        };
        setSettings(newSettings);
        setCookie("AdvancedInputSettings", newSettings);
    }, [settings]);

    const handleToggleToolbar = useCallback(() => {
        const newSettings = {
            ...settings,
            showToolbar: !settings.showToolbar,
        };
        setSettings(newSettings);
        setCookie("AdvancedInputSettings", newSettings);
    }, [settings]);

    const handleToggleWysiwyg = useCallback(() => {
        const newSettings = {
            ...settings,
            isWysiwyg: !settings.isWysiwyg,
        };
        setSettings(newSettings);
        setCookie("AdvancedInputSettings", newSettings);
    }, [settings]);

    // Add dropzone functionality
    const { getRootProps, getInputProps, isDragActive } = useDropzone({
        noClick: true,
        noKeyboard: true,
        getFilesFromEvent,
        onDrop: (acceptedFiles) => {
            if (!onContextDataChange || acceptedFiles.length === 0) return;

            // Calculate how many new items we can add
            const remainingSlots = CONTEXT_ITEM_LIMIT - contextData.length;
            if (remainingSlots <= 0) {
                PubSub.get().publish("snack", {
                    message: `Cannot add more items. Maximum of ${CONTEXT_ITEM_LIMIT} items allowed.`,
                    severity: SnackSeverity.Error,
                });
                return;
            }

            // Only take as many files as we have slots for
            const filesToAdd = acceptedFiles.slice(0, remainingSlots);

            const newContextItems: ContextItem[] = filesToAdd.map(file => {
                // Check if this is a dropped text file
                const isDroppedText = file.name === "dropped-text.txt" && file.type === "text/plain";

                return {
                    id: crypto.randomUUID(),
                    type: isDroppedText ? "text" :
                        file.type.startsWith("image/") ? "image" : "file",
                    label: isDroppedText ? "Dropped Text" : file.name,
                    src: file.type.startsWith("image/") ? URL.createObjectURL(file) : undefined,
                    file,
                };
            });

            onContextDataChange([...contextData, ...newContextItems]);

            // If we couldn't add all files, show a warning
            if (acceptedFiles.length > remainingSlots) {
                PubSub.get().publish("snack", {
                    message: `Only added ${remainingSlots} out of ${acceptedFiles.length} files due to the ${CONTEXT_ITEM_LIMIT} item limit.`,
                    severity: SnackSeverity.Error,
                });
            }
        },
    });

    // Clean up object URLs when component unmounts
    useEffect(() => () => {
        contextData.forEach(item => {
            if (item.src?.startsWith("blob:")) {
                URL.revokeObjectURL(item.src);
            }
        });
    }, [contextData]);

    const [anchorPlus, setAnchorPlus] = useState<HTMLElement | null>(null);
    const [anchorSettings, setAnchorSettings] = useState<HTMLElement | null>(null);

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

    function handleMessageChange(event: React.ChangeEvent<HTMLInputElement>) {
        const newValue = event.target.value;
        setLocalMessage(newValue);
        latestMessageRef.current = newValue;
        debouncedMessageChange(newValue);
    }

    const handleSubmit = useCallback(() => {
        if (onSubmit) {
            onSubmit(localMessage);
            setLocalMessage("");
        }
    }, [localMessage, onSubmit]);
    const handleKeyDown = useCallback(
        (e: KeyboardEvent<HTMLTextAreaElement | HTMLDivElement>) => {
            if (e.key === "Enter" && !e.shiftKey) {
                if (localEnterWillSubmit) {
                    e.preventDefault();
                    handleSubmit();
                }
            }
        },
        [handleSubmit, localEnterWillSubmit],
    );

    const handleTranscriptChange = useCallback((recognizedText: string) => {
        setLocalMessage((prev) => {
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

    const charsProgress = maxChars ? Math.min(100, Math.ceil((localMessage.length / maxChars) * 100)) : 0;
    const charsOverLimit = maxChars ? Math.max(0, localMessage.length - maxChars) : 0;

    const progressStyle = useMemo(() => {
        let progressStyle = { color: theme.palette.success.main };
        if (charsOverLimit > 0) {
            progressStyle = { color: theme.palette.error.main };
        } else if (charsProgress >= 80) {
            progressStyle = { color: theme.palette.warning.main };
        }
        return progressStyle;
    }, [charsOverLimit, charsProgress, theme.palette.error.main, theme.palette.success.main, theme.palette.warning.main]);

    return (
        <Outer {...getRootProps()}>
            <input {...getInputProps()} />
            {isDragActive && (
                <Box sx={dragOverlayStyles}>
                    <Typography variant="body1" color="text.secondary">
                        Drop files here...
                    </Typography>
                </Box>
            )}
            <IconButton
                onClick={handleToggleExpand}
                sx={expandButtonStyles}
                title={isExpanded ? "Collapse" : "Expand"}
            >
                <IconCommon
                    decorative
                    fill="background.textSecondary"
                    name={isExpanded ? "ExpandLess" : "ExpandMore"}
                />
            </IconButton>
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
                        <IconButton onClick={handleOpenInfoMemo} sx={toolbarIconButtonStyle}>
                            <IconCommon
                                decorative
                                fill="background.textSecondary"
                                name="Info"
                            />
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
                            />
                        ))}
                    </Box>
                </>
            ) : (
                <Box sx={topRowStyles}>
                    <IconButton onClick={handleOpenInfoMemo} sx={infoButtonStyle}>
                        <IconCommon
                            decorative
                            fill="background.textSecondary"
                            name="Info"
                        />
                    </IconButton>
                    {sortedContextData.map((item) => (
                        <ContextItemDisplay
                            key={item.id}
                            item={item}
                            imgStyle={imgStyle}
                            onRemove={handleRemoveContextItem}
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
                    <Box
                        maxHeight={isExpanded ? "calc(100vh - 150px)" : "unset"}
                        overflow="auto"
                    >
                        <InputBase
                            inputProps={inputBaseInputProps}
                            multiline
                            fullWidth
                            minRows={isExpanded ? 5 : 1}
                            maxRows={isExpanded ? 50 : 6}
                            value={localMessage}
                            onChange={handleMessageChange}
                            onKeyDown={handleKeyDown}
                            placeholder="Type your message..."
                        />
                    </Box>
                )}
            </Box>

            {/* Bottom Section */}
            <Box sx={bottomRowStyles}>
                <Box ref={toolsContainerRef} sx={toolsContainerStyles}>
                    <StyledIconButton disabled={false} onClick={handleOpenPlusMenu}>
                        <IconCommon
                            decorative
                            fill="background.textSecondary"
                            name="Add"
                        />
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
                                    <IconCommon
                                        decorative
                                        name="Ellipsis"
                                    />
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
                                    <IconCommon
                                        decorative
                                        name="Invisible"
                                    />
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
                    <Box position="relative" display="inline-flex" verticalAlign="middle">
                        <CircularProgress
                            variant="determinate"
                            size={34}
                            value={charsProgress}
                            sx={progressStyle}
                        />
                        <Box
                            top={0}
                            left={0}
                            bottom={0}
                            right={0}
                            position="absolute"
                            display="flex"
                            alignItems="center"
                            justifyContent="center"
                        >
                            {charsOverLimit > 0 && <Typography variant="caption" component="div">
                                {charsOverLimit >= 1000 ? "99+" : charsOverLimit}
                            </Typography>}
                            {charsOverLimit <= 0 && <StyledIconButton
                                bgColor={maxChars ?
                                    "transparent" :
                                    theme.palette.mode === "dark"
                                        ? theme.palette.background.textPrimary
                                        : theme.palette.primary.main
                                }
                                disabled={!localMessage.trim() || charsOverLimit > 0}
                                onClick={handleSubmit}
                            >
                                <IconCommon
                                    decorative
                                    fill={!localMessage.trim()
                                        ? "background.textSecondary"
                                        : maxChars
                                            ? "background.textPrimary"
                                            : theme.palette.mode === "dark"
                                                ? "background.default"
                                                : "primary.contrastText"
                                    }
                                    name="Send"
                                />
                            </StyledIconButton>}
                        </Box>
                    </Box>
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

