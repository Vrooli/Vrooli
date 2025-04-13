/* eslint-disable no-magic-numbers */
import { FormStructureType, getDotNotationValue, noop, setDotNotationValue } from "@local/shared";
import { Avatar, Box, Chip, CircularProgress, Collapse, Divider, IconButton, IconButtonProps, ListItemIcon, ListItemText, Menu, MenuItem, Popover, Switch, Tooltip, Typography, styled, useTheme } from "@mui/material";
import type { SxProps, Theme } from "@mui/material/styles";
import { CSSProperties } from "@mui/styles";
import { useField } from "formik";
import React, { MouseEvent, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useDropzone } from "react-dropzone";
import { useDimensions } from "../../../hooks/useDimensions.js";
import { useHotkeys } from "../../../hooks/useHotkeys.js";
import { useUndoRedo } from "../../../hooks/useUndoRedo.js";
import { Icon, IconCommon, IconInfo, IconRoutine } from "../../../icons/Icons.js";
import { randomString } from "../../../utils/codes.js";
import { keyComboToString } from "../../../utils/display/device.js";
import { getTranslationData, handleTranslationChange } from "../../../utils/display/translationTools.js";
import { getCookie, setCookie } from "../../../utils/localStorage.js";
import { PubSub } from "../../../utils/pubsub.js";
import { MicrophoneButton } from "../../buttons/MicrophoneButton.js";
import { FindObjectDialog } from "../../dialogs/FindObjectDialog/FindObjectDialog.js";
import { SnackSeverity } from "../../snacks/BasicSnack/BasicSnack.js";
import { FormTip } from "../form/FormTip.js";
import { AdvancedInputMarkdown } from "./AdvancedInputMarkdown.js";
import { AdvancedInputToolbar, TOOLBAR_CLASS_NAME, defaultActiveStates } from "./AdvancedInputToolbar.js";
import { ContextDropdown, ListObject } from "./ContextDropdown.js";
import { AdvancedInputLexical } from "./lexical/AdvancedInputLexical.js";
import { AdvancedInputAction, AdvancedInputActiveStates, AdvancedInputBaseProps, AdvancedInputFeatures, AdvancedInputProps, ContextItem, DEFAULT_FEATURES, ExternalApp, Tool, ToolState, TranslatedAdvancedInputProps, advancedInputTextareaClassName } from "./utils.js";

// Add supported external apps here
const externalApps: ExternalApp[] = [
];
const findRoutineLimitTo = ["RoutineMultiStep", "RoutineSingleStep"] as const;

const iconHeight = 32;
const iconWidth = 32;
const MAX_ROWS_EXPANDED = 50;
const MIN_ROWS_EXPANDED = 5;
const MAX_ROWS_COLLAPSED = 6;
const MIN_ROWS_COLLAPSED = 1;

// Add these near the top with other constants
const TRIGGER_CHARS = {
    AT: "@",
    SLASH: "/",
} as const;

// Simple interface for elements that can be focused
interface FocusableElement {
    focus: (options?: FocusOptions) => void;
}

// Example of how to add context
// const openAssistantDialog = useCallback(() => {
//     if (disabled) return;
//     const userId = getCurrentUser(session)?.id;
//     if (!userId) return;
//     const chatId = chat?.id;
//     if (!chatId) return;

//     if (!codeMirrorRef.current || !codeMirrorRef.current.view) {
//         console.error("CodeMirror not found");
//         return;
//     }
//     const codeDoc = codeMirrorRef.current.view.state.doc;
//     const selectionRanges = codeMirrorRef.current.view.state.selection.ranges;
//     // Only use the first selection range, if it exists
//     const selection = selectionRanges.length > 0 ? codeDoc.sliceString(selectionRanges[0].from, selectionRanges[0].to) : "";
//     const fullText = codeDoc.sliceString(0, Number.MAX_SAFE_INTEGER);
//     const contextValue = generateContext(selection, fullText);

//     // Open the side chat and provide it context
//     //TODO
//     // PubSub.get().publish("menu", { id: ELEMENT_IDS.LeftDrawer, isOpen: true, data: { tab: "Chat" } });
//     const context = {
//         id: `code-${name}`,
//         data: contextValue,
//         label: generateContextLabel(contextValue),
//         template: `Code:\n\`\`\`${codeLanguage}\n<DATA>\n\`\`\``,
//         templateVariables: { data: "<DATA>" },
//     };
//     PubSub.get().publish("chatTask", {
//         chatId,
//         contexts: {
//             add: {
//                 behavior: "replace",
//                 connect: {
//                     __type: "contextId",
//                     data: context.id,
//                 },
//                 value: [context],
//             },
//         },
//     });
// }, [chat?.id, codeLanguage, disabled, name, session]);

const toolbarRowStyles: SxProps<Theme> = {
    display: "flex",
    alignItems: "center",
    mb: 1,
};

const contextRowStyles: SxProps<Theme> = {
    display: "flex",
    alignItems: "center",
    flexWrap: "wrap",
    marginRight: "auto",
    background: "none",
};

const inputRowStyles: SxProps<Theme> = {
    mb: 1,
    px: 2, // additional left/right padding
    transition: "max-height 0.3s ease",
};

const titleStyles: SxProps<Theme> = {
    px: 2,
    mb: 1,
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
    // Ensure it can receive clicks even if children cover parts of it
    // (though padding usually handles this)
    // cursor: "text", // Optional: hint that the area is for text input
}));

// List of selectors for interactive elements that shouldn't trigger input focus
const NON_FOCUSABLE_SELECTORS = [
    "button",
    "a",
    "input",
    "textarea",
    "[contenteditable=\"true\"]",
    ".MuiIconButton-root",
    ".MuiChip-root",
    ".MuiButtonBase-root",
    ".MuiSwitch-root",
    ".MuiSlider-root",
    ".MuiMenuItem-root",
    ".MuiTab-root",
    ".AdvancedInputToolbar",  // The toolbar itself
    `.${TOOLBAR_CLASS_NAME}`, // Using the toolbar class name constant
    "[aria-haspopup=\"true\"]", // Elements with popups/dropdowns
    // Context elements
    ".context-item",
    ".tool-item",
    // File dropzone elements
    ".dropzone",
    // Specific elements we know about
    "[data-toolbar-button=\"true\"]",
].join(",");

// Class name added to the AdvancedInput textarea/contentEditable for identification
const ADVANCED_INPUT_CONTENT_CLASS = advancedInputTextareaClassName;

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

    const handleToolToggle = useCallback(function handleToolToggleCallback(event: React.MouseEvent) {
        // Stop propagation to prevent focusing the input
        event.stopPropagation();
        onToggleTool();
    }, [onToggleTool]);

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
            data-type="tool" // Keep this for the selector
            key={`${name}-${index}`}
            icon={chipIcon}
            label={displayName}
            onDelete={onRemoveTool} // MuiChip handles stopPropagation internally for deleteIcon
            onClick={handleToolToggle}
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

/** Prevents input from losing focus when the toolbar is pressed */
function preventInputLossOnToolbarClick(event: React.MouseEvent) {
    // Check for the toolbar ID at each parent element
    let parent = event.target as HTMLElement | null | undefined;
    let numParentsTraversed = 0;
    const maxParentsToTraverse = 10;
    do {
        // If the toolbar is clicked, prevent default (stops focus loss) and stop propagation (prevents outer click handler)
        if (parent?.classList.contains(TOOLBAR_CLASS_NAME)) {
            event.preventDefault();
            // We don't stop propagation here anymore, let the Outer click handler decide based on selector
            // event.stopPropagation();
            return;
        }
        parent = parent?.parentElement;
        numParentsTraversed++;
    } while (parent && numParentsTraversed < maxParentsToTraverse);
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
    function handleRemove(event: React.MouseEvent) {
        // Stop propagation to prevent Outer click handler
        event.stopPropagation();
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
            <PreviewContainer data-type="context-item"> {/* Add data-type */}
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
                {/* Ensure the remove button stops propagation */}
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
            data-type="context-item" // Add data-type
            icon={<Icon decorative info={fallbackIconInfo} />}
            label={truncatedLabel}
            onDelete={handleRemove} // MuiChip handles stopPropagation for deleteIcon
            title={item.label} // Show full name on hover
        // No onClick needed here, delete is handled
        />
    );
}

// Add these styles near the top with other styles
const toolbarIconButtonStyle = { padding: "4px", opacity: 0.5 } as const;

// Move style objects to constants
const verticalMiddleStyle = { verticalAlign: "middle" } as const;
const dividerStyle = { my: 1, opacity: 0.2 } as const;

/**
 * PlusMenu Component - renders the popover for additional actions.
 */
interface PlusMenuProps {
    anchorEl: HTMLElement | null;
    onClose: () => unknown;
    onAttachFile?: () => unknown;
    onConnectExternalApp?: () => unknown;
    onTakePhoto?: () => unknown;
    onAddRoutine?: () => unknown;
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
                        {onAttachFile && <MenuItem onClick={onAttachFile}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    name="File"
                                />
                            </ListItemIcon>
                            <ListItemText primary="Attach File" secondary="Attach a file from your device" />
                        </MenuItem>}
                        {externalApps.length > 0 && onConnectExternalApp && <MenuItem onClick={handleOpenExternalApps}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    name="Link"
                                />
                            </ListItemIcon>
                            <ListItemText primary="Connect External App" secondary="Connect an external app to your account" />
                        </MenuItem>}
                        {onTakePhoto && <MenuItem onClick={onTakePhoto}>
                            <ListItemIcon>
                                <IconCommon
                                    decorative
                                    name="CameraOpen"
                                />
                            </ListItemIcon>
                            <ListItemText primary="Take Photo" secondary="Take a photo from your device" />
                        </MenuItem>}
                        {onAddRoutine && <MenuItem onClick={onAddRoutine}>
                            <ListItemIcon>
                                <IconRoutine
                                    decorative
                                    name="Routine"
                                />
                            </ListItemIcon>
                            <ListItemText primary="Add Routine" secondary="Allow the AI to perform actions" />
                        </MenuItem>}
                    </Box>
                </Popover>
                <Menu
                    anchorEl={externalAppAnchor}
                    open={Boolean(externalAppAnchor)}
                    onClose={handleCloseExternalApps}
                    anchorOrigin={popoverAnchorOrigin}
                    transformOrigin={popoverTransformOrigin}
                >
                    {externalApps.map((app) => {
                        function connectApp() {
                            handleAppConnection(app.id);
                        }

                        return (
                            <MenuItem
                                key={app.id}
                                onClick={connectApp}
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
                        );
                    })}
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
    enterWillSubmit: boolean;
    mergedFeatures: AdvancedInputFeatures;
    onClose: () => void;
    onToggleEnterWillSubmit: () => void;
    onToggleToolbar: () => void;
    onToggleSpellcheck: () => void;
    showToolbar: boolean;
    spellcheck: boolean;
}

const infoMenuAnchorOrigin = { vertical: "bottom", horizontal: "left" } as const;
const secondaryTypographyProps = {
    style: { whiteSpace: "pre-wrap", fontSize: "0.875rem", color: "#666" },
} as const;
const infoMemoTipData = {
    id: randomString(),
    icon: "Warning",
    isMarkdown: false,
    label: "Some features may be unavailable depending on the AI models you're chatting with and the device this is running on.",
    type: FormStructureType.Tip,
} as const;

const InfoMemo: React.FC<InfoMemoProps> = React.memo(({
    anchorEl,
    enterWillSubmit,
    mergedFeatures,
    onClose,
    onToggleEnterWillSubmit,
    onToggleToolbar,
    onToggleSpellcheck,
    showToolbar,
    spellcheck,
}: InfoMemoProps) => {
    const theme = useTheme();

    const infoMenuSlotProps = useMemo(() => ({
        paper: {
            style: {
                backgroundColor: theme.palette.background.default,
                borderRadius: "12px",
                maxWidth: "500px",
                maxHeight: "500px",
                overflow: "auto",
                padding: "8px 0",
            },
        },
    }), [theme]);

    return (
        <Menu
            anchorEl={anchorEl}
            open={Boolean(anchorEl)}
            onClose={onClose}
            anchorOrigin={infoMenuAnchorOrigin}
            slotProps={infoMenuSlotProps}
        >
            <Box px={2} pb={1}>
                <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                    Settings
                </Typography>
            </Box>

            <MenuItem>
                <ListItemText
                    primary="Enter key to submit"
                    secondary={`When enabled, use ${keyComboToString("Enter")} to submit, and ${keyComboToString("Shift", "Enter")} to create a new line.`}
                    secondaryTypographyProps={secondaryTypographyProps}
                />
                <Switch
                    edge="end"
                    checked={enterWillSubmit}
                    onChange={onToggleEnterWillSubmit}
                    color="secondary"
                />
            </MenuItem>

            {mergedFeatures.allowFormatting && (
                <MenuItem>
                    <ListItemText
                        primary="Show formatting toolbar"
                        secondary="Display text formatting options above the input area."
                        secondaryTypographyProps={secondaryTypographyProps}
                    />
                    <Switch
                        edge="end"
                        checked={showToolbar}
                        onChange={onToggleToolbar}
                        color="secondary"
                    />
                </MenuItem>
            )}
            {mergedFeatures.allowSpellcheck && (
                <MenuItem>
                    <ListItemText
                        primary="Enable spellcheck"
                        secondary="When enabled, the browser will check for spelling errors in the input."
                        secondaryTypographyProps={secondaryTypographyProps}
                    />
                    <Switch
                        edge="end"
                        checked={spellcheck}
                        onChange={onToggleSpellcheck}
                        color="secondary"
                    />
                </MenuItem>
            )}
            <Divider sx={dividerStyle} />
            <Box px={2}>
                <FormTip
                    element={infoMemoTipData}
                    isEditing={false}
                    onUpdate={noop}
                    onDelete={noop}
                />
            </Box>
        </Menu>
    );
});
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

/** TextInput for entering rich text. Supports markdown and WYSIWYG */
export function AdvancedInputBase({
    tools = [],
    contextData = [],
    maxChars,
    value,
    disabled = false,
    error = false,
    features = DEFAULT_FEATURES,
    helperText,
    name,
    onBlur,
    onChange,
    onFocus,
    onToolsChange,
    onContextDataChange,
    onSubmit,
    placeholder,
    tabIndex,
    title,
}: AdvancedInputBaseProps) {
    const theme = useTheme();

    // Merge with default features to ensure all properties exist
    const mergedFeatures = useMemo(() => ({
        ...DEFAULT_FEATURES,
        ...features,
    }), [features]);

    // Reference to the editor input element
    const inputRef = useRef<FocusableElement | null>(null);

    // Separate ref for the container element to handle hotkeys
    const containerRef = useRef<HTMLDivElement>(null);

    // Add dropdown state
    const [dropdownAnchor, setDropdownAnchor] = useState<HTMLElement | null>(null);
    const [initialCategory, setInitialCategory] = useState<"Routine" | null>(null);
    const [searchText, setSearchText] = useState("");
    const inputWrapperRef = useRef<HTMLDivElement>(null);
    const triggerCharRef = useRef<string | null>(null);

    // Add expanded view state
    const [isExpanded, setIsExpanded] = useState(false);
    const handleToggleExpand = useCallback(() => {
        setIsExpanded(prev => !prev);
    }, []);

    // Use the useUndoRedo hook for managing input value
    const { internalValue, changeInternalValue, resetInternalValue, undo, redo, canUndo, canRedo } = useUndoRedo({
        initialValue: value,
        onChange,
        delimiters: [" ", "\n"],
    });
    useEffect(function resetValueEffect() {
        resetInternalValue(value);
    }, [value, resetInternalValue]);

    // Settings state from localStorage with toolbar visibility handling
    const [settings, setSettings] = useState(() => getCookie("AdvancedInputSettings"));
    const [isMarkdownOn, setIsMarkdownOn] = useState(getCookie("ShowMarkdown"));

    // Derived value - showToolbar is forced true when formatting is allowed but settings customization is not
    const showToolbar = useMemo(() => {
        if (!mergedFeatures.allowSettingsCustomization && mergedFeatures.allowFormatting) {
            return true; // Force toolbar to be visible
        }
        return settings.showToolbar;
    }, [mergedFeatures.allowSettingsCustomization, mergedFeatures.allowFormatting, settings.showToolbar]);

    // Determine if we should use markdown editor
    const useMarkdownEditor = useMemo(() => {
        // When both settings customization and formatting are disabled, always use markdown
        if (!mergedFeatures.allowFormatting) {
            return true;
        }
        // Otherwise, use markdown if it's enabled or toolbar is hidden
        return isMarkdownOn || !showToolbar;
    }, [mergedFeatures.allowFormatting, isMarkdownOn, showToolbar]);

    const { enterWillSubmit, spellcheck } = settings;

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
        // Only allow toggling if we're not in the forced-visible state
        if (!mergedFeatures.allowSettingsCustomization && mergedFeatures.allowFormatting) {
            return; // Cannot toggle toolbar in this case
        }

        const newSettings = {
            ...settings,
            showToolbar: !settings.showToolbar,
        };
        setSettings(newSettings);
        setCookie("AdvancedInputSettings", newSettings);
    }, [settings, mergedFeatures.allowSettingsCustomization, mergedFeatures.allowFormatting]);

    const handleToggleSpellcheck = useCallback(() => {
        const newSettings = {
            ...settings,
            spellcheck: !settings.spellcheck,
        };
        setSettings(newSettings);
        setCookie("AdvancedInputSettings", newSettings);
    }, [settings]);

    // Add dropzone functionality
    const { getRootProps, getInputProps, isDragActive } = useDropzone({
        noClick: true, // Don't trigger file dialog on click
        noKeyboard: true, // Don't use keyboard to trigger file dialog
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
            // console.log({ maxRight, remWidth, shouldAddEllipsisBefore }); // Keep for debugging if needed

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
        // Stop propagation to prevent focusing the input
        event.stopPropagation();
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

    const handleTranscriptChange = useCallback((recognizedText: string) => {
        // Append recognized text to whatever is currently typed
        if (!internalValue.trim()) {
            changeInternalValue(recognizedText);
        } else {
            changeInternalValue(`${internalValue} ${recognizedText}`);
        }
    }, [internalValue, changeInternalValue]);

    const handleSubmit = useCallback(() => {
        onSubmit?.(internalValue);
    }, [internalValue, onSubmit]);

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

    const charsProgress = maxChars ? Math.min(100, Math.ceil(((internalValue?.length ?? 0) / maxChars) * 100)) : 0;
    const charsOverLimit = maxChars ? Math.max(0, (internalValue?.length ?? 0) - maxChars) : 0;

    const progressStyle = useMemo(() => {
        let progressStyle = { color: theme.palette.success.main };
        if (charsOverLimit > 0) {
            progressStyle = { color: theme.palette.error.main };
        } else if (charsProgress >= 80) {
            progressStyle = { color: theme.palette.warning.main };
        }
        return progressStyle;
    }, [charsOverLimit, charsProgress, theme.palette.error.main, theme.palette.success.main, theme.palette.warning.main]);

    const toggleMarkdown = useCallback(() => {
        if (mergedFeatures.allowFormatting) {
            setIsMarkdownOn(!isMarkdownOn);
            setCookie("ShowMarkdown", !isMarkdownOn);
        }
    }, [isMarkdownOn, mergedFeatures.allowFormatting]);

    const currentHandleActionRef = useRef<((action: AdvancedInputAction, data?: unknown) => unknown) | null>(null);
    const setChildHandleAction = useCallback((handleAction: (action: AdvancedInputAction, data?: unknown) => unknown) => {
        currentHandleActionRef.current = handleAction;
    }, []);
    const handleAction = useCallback((action: AdvancedInputAction, data?: unknown) => {
        if (action === "Mode") {
            if (mergedFeatures.allowFormatting) {
                toggleMarkdown();
            }
        } else if (currentHandleActionRef.current) {
            currentHandleActionRef.current(action, data);
        } else {
            console.error("RichInputBase: No child handleAction function found");
        }
        return noop;
    }, [toggleMarkdown, mergedFeatures.allowFormatting]);

    useHotkeys([
        // Markdown/Lexical hotkeys
        { keys: ["1"], altKey: true, callback: () => { handleAction(AdvancedInputAction.Header1); } },
        { keys: ["2"], altKey: true, callback: () => { handleAction(AdvancedInputAction.Header2); } },
        { keys: ["3"], altKey: true, callback: () => { handleAction(AdvancedInputAction.Header3); } },
        { keys: ["4"], altKey: true, callback: () => { handleAction(AdvancedInputAction.Header4); } },
        { keys: ["5"], altKey: true, callback: () => { handleAction(AdvancedInputAction.Header5); } },
        { keys: ["6"], altKey: true, callback: () => { handleAction(AdvancedInputAction.Header6); } },
        { keys: ["7"], altKey: true, callback: () => { handleAction(AdvancedInputAction.ListBullet); } },
        { keys: ["8"], altKey: true, callback: () => { handleAction(AdvancedInputAction.ListNumber); } },
        { keys: ["9"], altKey: true, callback: () => { handleAction(AdvancedInputAction.ListCheckbox); } },
        { keys: ["0"], altKey: true, callback: () => { toggleMarkdown(); } },
        { keys: ["b"], ctrlKey: true, callback: () => { handleAction(AdvancedInputAction.Bold); } },
        { keys: ["i"], ctrlKey: true, callback: () => { handleAction(AdvancedInputAction.Italic); } },
        { keys: ["k"], ctrlKey: true, callback: () => { handleAction(AdvancedInputAction.Link); } },
        { keys: ["e"], ctrlKey: true, callback: () => { handleAction(AdvancedInputAction.Code); } },
        { keys: ["Q"], ctrlKey: true, callback: () => { handleAction(AdvancedInputAction.Quote); } },
        { keys: ["S"], ctrlKey: true, callback: () => { handleAction(AdvancedInputAction.Strikethrough); } },
        { keys: ["l"], ctrlKey: true, callback: () => { handleAction(AdvancedInputAction.Spoiler); } },
        { keys: ["u"], ctrlKey: true, callback: () => { handleAction(AdvancedInputAction.Underline); } },
        // Undo/Redo
        { keys: ["z"], ctrlKey: true, callback: () => { handleAction(AdvancedInputAction.Undo); } },
        { keys: ["Z"], ctrlKey: true, callback: () => { handleAction(AdvancedInputAction.Redo); } },
        { keys: ["y"], ctrlKey: true, callback: () => { handleAction(AdvancedInputAction.Redo); } },
        // Context dropdown
        ...(mergedFeatures.allowContextDropdown ? [
            {
                keys: ["@"],
                requirePrecedingWhitespace: true,
                preventDefault: false, // Allow the key to be typed if user keeps pressing
                callback: () => {
                    handleContextTrigger(TRIGGER_CHARS.AT);
                },
            },
            {
                keys: ["/"],
                requirePrecedingWhitespace: true,
                preventDefault: false, // Allow the key to be typed if user keeps pressing
                callback: () => {
                    handleContextTrigger(TRIGGER_CHARS.SLASH);
                },
            },
        ] : []),
        // Enter will submit
        {
            keys: ["Enter"],
            shiftKey: false,
            preventDefault: false, // We decide if we want to prevent the default behavior
            callback: (event) => {
                if (enterWillSubmit) {
                    event.preventDefault();
                    handleSubmit();
                }
            },
        },
    ], true, containerRef); // Use containerRef for hotkeys instead of inputRef

    // Handle toolbar active states
    const [activeStates, setActiveStates] = useState<Omit<AdvancedInputActiveStates, "SetValue">>(defaultActiveStates);
    const handleActiveStatesChange = useCallback((newActiveStates) => {
        setActiveStates(newActiveStates);
    }, []);

    // Split viewProps into stable and dynamic parts
    const stableViewProps = useMemo(() => ({
        disabled,
        enterWillSubmit,
        error,
        maxRows: isExpanded ? MAX_ROWS_EXPANDED : MAX_ROWS_COLLAPSED,
        minRows: isExpanded ? MIN_ROWS_EXPANDED : MIN_ROWS_COLLAPSED,
        name,
        onActiveStatesChange: handleActiveStatesChange,
        onBlur,
        onFocus,
        onSubmit,
        placeholder,
        setHandleAction: setChildHandleAction,
        tabIndex,
        toggleMarkdown,
    }), [disabled, enterWillSubmit, error, isExpanded, name, handleActiveStatesChange, onBlur, onFocus, onSubmit, placeholder, setChildHandleAction, tabIndex, toggleMarkdown]);

    // Only the frequently changing props
    const dynamicViewProps = useMemo(() => ({
        value: internalValue,
        onChange: changeInternalValue,
        undo,
        redo,
    }), [internalValue, changeInternalValue, undo, redo]);

    // Create spellcheck-aware feature objects to pass to children
    const mergedFeaturesWithSpellcheck = useMemo(() => ({
        ...mergedFeatures,
        allowSpellcheck: mergedFeatures.allowSpellcheck && spellcheck,
    }), [mergedFeatures, spellcheck]);

    const MarkdownFeatures = useMemo(() => mergedFeaturesWithSpellcheck, [mergedFeaturesWithSpellcheck]);
    const LexicalFeatures = useMemo(() => mergedFeaturesWithSpellcheck, [mergedFeaturesWithSpellcheck]);

    const MarkdownComponent = useMemo(() => (
        <AdvancedInputMarkdown
            {...stableViewProps}
            {...dynamicViewProps}
            mergedFeatures={MarkdownFeatures}
            ref={inputRef as React.Ref<HTMLTextAreaElement>}
        />
    ), [stableViewProps, dynamicViewProps, MarkdownFeatures, inputRef]);

    const LexicalComponent = useMemo(() => (
        <AdvancedInputLexical
            {...stableViewProps}
            {...dynamicViewProps}
            mergedFeatures={LexicalFeatures}
            ref={inputRef as React.Ref<{ focus: (options?: FocusOptions) => void }>}
        />
    ), [stableViewProps, dynamicViewProps, LexicalFeatures, inputRef]);

    // Add dropdown handlers
    const handleContextTrigger = useCallback((triggerChar: string) => {
        // Check if context dropdown is allowed
        if (!mergedFeatures.allowContextDropdown) return;

        // Use the wrapper element instead of trying to get the exact cursor position
        if (!inputWrapperRef.current) return;

        // Set the anchor element and position
        setDropdownAnchor(inputWrapperRef.current);

        // Store the trigger character for later use
        triggerCharRef.current = triggerChar;
        setSearchText("");

        // Set initial category based on trigger char
        const category = triggerChar === TRIGGER_CHARS.SLASH ? "Routine" : null;
        setInitialCategory(category);

        // Add the trigger character to the input
        changeInternalValue(internalValue + triggerChar);
    }, [internalValue, changeInternalValue, mergedFeatures.allowContextDropdown]);

    const handleCloseDropdown = useCallback((shouldAddSearchText = true) => {
        // If we have search text and we're closing without selection, add it to the input
        if (shouldAddSearchText && searchText && triggerCharRef.current) {
            // Get the current value and find the last trigger character
            const lastIndex = internalValue.lastIndexOf(triggerCharRef.current);
            if (lastIndex >= 0) {
                const beforeTrigger = internalValue.slice(0, lastIndex);
                const afterTrigger = internalValue.slice(lastIndex + 1);
                const newText = `${beforeTrigger}${triggerCharRef.current}${searchText}${afterTrigger}`;
                changeInternalValue(newText);
            }
        }

        // Clear dropdown state
        setDropdownAnchor(null);
        setInitialCategory(null);
        setSearchText("");
        triggerCharRef.current = null;
    }, [searchText, changeInternalValue, internalValue]);

    const handleSelectContext = useCallback((item: ListObject) => {
        // Get the current value and find the last trigger character
        const lastIndex = internalValue.lastIndexOf(triggerCharRef.current ?? "");
        if (lastIndex >= 0) {
            const beforeTrigger = internalValue.slice(0, lastIndex);
            const afterTrigger = internalValue.slice(lastIndex + 1);
            const newText = `${beforeTrigger}[[${item.name}]]${afterTrigger}`;
            changeInternalValue(newText);
        }

        // Close the dropdown without adding search text since we're selecting an item
        handleCloseDropdown(false);
    }, [changeInternalValue, handleCloseDropdown, internalValue]);

    /**
     * Handler to focus the input when clicking on non-interactive areas
     * 
     * The approach here is to focus the input for any click on the Outer component
     * unless it's coming from a known interactive element or has already been handled.
     * This is more reliable than trying to determine what's "interactive" since the 
     * Outer component itself can have role="button" from Dropzone.
     */
    const handleOuterClick = useCallback((event: React.MouseEvent<HTMLDivElement>) => {
        // Skip if disabled
        if (disabled) return;

        // Get the target element
        const target = event.target as HTMLElement;

        // Log what was clicked for debugging
        console.log("AdvancedInput: Click detected on", target.tagName, target.className);

        // Already clicked directly on the input area
        if (target.classList?.contains(ADVANCED_INPUT_CONTENT_CLASS)) {
            console.log("AdvancedInput: Click on input area - no focus action needed");
            return;
        }

        // Check if the click target is an interactive element that we should ignore
        // We're using a more specific approach now rather than the broad role="button" 
        const isInteractive = target.closest(NON_FOCUSABLE_SELECTORS) !== null;

        // For debugging only
        if (isInteractive) {
            console.log("AdvancedInput: Click on interactive element, but focusing anyway unless stopped by propagation");
        }

        // Focus the input if available
        if (inputRef.current) {
            console.log("AdvancedInput: Focusing input");
            inputRef.current.focus();
        } else {
            console.log("AdvancedInput: Input reference not available", inputRef);
        }
    }, [disabled]);

    // Add this handler for the settings button
    const handleOpenInfoMemoWithStopPropagation = useCallback((event: MouseEvent<HTMLElement>) => {
        // Stop propagation to prevent focusing the input
        event.stopPropagation();
        handleOpenInfoMemo(event);
    }, [handleOpenInfoMemo]);

    // Add this handler for the expand toggle
    const handleToggleExpandWithStopPropagation = useCallback((event: MouseEvent<HTMLElement>) => {
        // Stop propagation to prevent focusing the input
        event.stopPropagation();
        handleToggleExpand();
    }, [handleToggleExpand]);

    // Add this handler for the tools expand toggle
    const toggleToolsExpandedWithStopPropagation = useCallback((event: MouseEvent<HTMLElement>) => {
        // Stop propagation to prevent focusing the input
        event.stopPropagation();
        toggleToolsExpanded();
    }, [toggleToolsExpanded]);

    // Add this handler for the submit button
    const handleSubmitWithStopPropagation = useCallback((event: MouseEvent<HTMLElement>) => {
        // Stop propagation to prevent focusing the input
        event.stopPropagation();
        handleSubmit();
    }, [handleSubmit]);

    // Create a wrapper for the MicrophoneButton
    function MicrophoneButtonWithStopPropagation({ onTranscriptChange, ...props }: MicrophoneButtonProps) {
        // Create a wrapper for the onTranscriptChange function
        const handleTranscriptChangeWithStopPropagation = useCallback((transcript) => {
            // Don't need to stop propagation here as this is a callback function
            onTranscriptChange(transcript);
        }, [onTranscriptChange]);

        // Create a component that stops propagation for any clicks
        return (
            <Box onClick={(e) => e.stopPropagation()}>
                <MicrophoneButton
                    onTranscriptChange={handleTranscriptChangeWithStopPropagation}
                    {...props}
                />
            </Box>
        );
    }

    return (
        // Add the click handler to the Outer component
        <Outer
            className="advanced-input"
            {...(mergedFeatures.allowFileAttachments ? getRootProps() : {})}
            onClick={handleOuterClick}
            ref={containerRef} // Add ref to the Outer component for hotkeys
        >
            {mergedFeatures.allowFileAttachments && <input {...getInputProps()} />}
            {mergedFeatures.allowFileAttachments && isDragActive && (
                <Box sx={dragOverlayStyles}>
                    <Typography variant="body1" color="text.secondary">
                        Drop files here...
                    </Typography>
                </Box>
            )}
            <Collapse in={mergedFeatures.allowTools && tools.some((tool) => tool.state === ToolState.Exclusive)} unmountOnExit>
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
            {/* Make toolbar prevent default on mouse down to avoid focus loss but allow outer click */}
            <Box onMouseDown={preventInputLossOnToolbarClick} sx={toolbarRowStyles}>
                {mergedFeatures.allowSettingsCustomization && (
                    <IconButton
                        onClick={handleOpenInfoMemoWithStopPropagation}
                        sx={toolbarIconButtonStyle}
                    >
                        <IconCommon
                            decorative
                            fill="background.textSecondary"
                            name="Settings"
                            size={20}
                        />
                    </IconButton>
                )}
                {mergedFeatures.allowFormatting && showToolbar && <AdvancedInputToolbar
                    activeStates={activeStates}
                    canRedo={canRedo}
                    canUndo={canUndo}
                    disabled={disabled}
                    handleAction={handleAction}
                    handleActiveStatesChange={handleActiveStatesChange}
                    isMarkdownOn={isMarkdownOn}
                />}
                {(!mergedFeatures.allowFormatting || !showToolbar) && <Box sx={contextRowStyles}>
                    {sortedContextData.map((item) => (
                        <ContextItemDisplay
                            key={item.id}
                            item={item}
                            imgStyle={imgStyle}
                            onRemove={handleRemoveContextItem}
                        />
                    ))}
                </Box>}
                {mergedFeatures.allowExpand && (
                    <Tooltip placement="top" title={isExpanded ? "Collapse" : "Expand"}>
                        <IconButton
                            onClick={handleToggleExpandWithStopPropagation}
                            sx={toolbarIconButtonStyle}
                        >
                            <IconCommon
                                decorative
                                fill="background.textSecondary"
                                name={isExpanded ? "ExpandLess" : "ExpandMore"}
                            />
                        </IconButton>
                    </Tooltip>
                )}
            </Box>
            {mergedFeatures.allowFormatting && showToolbar && <Box sx={contextRowStyles}>
                {sortedContextData.map((item) => (
                    <ContextItemDisplay
                        key={item.id}
                        item={item}
                        imgStyle={imgStyle}
                        onRemove={handleRemoveContextItem}
                    />
                ))}
            </Box>}
            {/* Title Section */}
            {title && (
                <Box sx={titleStyles}>
                    <Typography variant="subtitle1" fontWeight="medium" color="text.secondary">
                        {title}
                    </Typography>
                </Box>
            )}
            {/* Input Area */}
            <Box sx={inputRowStyles}>
                <Box
                    ref={inputWrapperRef}
                    maxHeight={isExpanded ? "calc(100vh - 150px)" : "unset"}
                    overflow="auto"
                >
                    {/* Conditionally render Markdown or Lexical, passing the inputRef */}
                    {useMarkdownEditor ? MarkdownComponent : LexicalComponent}
                </Box>
            </Box>

            {/* Bottom Section */}
            <Box sx={bottomRowStyles}>
                {mergedFeatures.allowTools && (
                    <Box ref={toolsContainerRef} sx={toolsContainerStyles}>
                        {(mergedFeatures.allowFileAttachments ||
                            mergedFeatures.allowImageAttachments ||
                            mergedFeatures.allowTextAttachments) && (
                                <StyledIconButton disabled={false} onClick={handleOpenPlusMenu}>
                                    <IconCommon
                                        decorative
                                        fill="background.textSecondary"
                                        name="Add"
                                    />
                                </StyledIconButton>
                            )}
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
                                <React.Fragment key={`${tool.name}-${index}`}>
                                    {canAddShowButton && <ShowHideIconButton data-id="show-all-tools-button" disabled={false} onClick={toggleToolsExpandedWithStopPropagation}>
                                        <IconCommon
                                            decorative
                                            fill="background.textSecondary"
                                            name="Ellipsis"
                                        />
                                    </ShowHideIconButton>}
                                    <ToolChip
                                        {...tool}
                                        index={index}
                                        onRemoveTool={onRemoveTool}
                                        onToggleToolExclusive={onToggleToolExclusive}
                                        onToggleTool={onToggleTool}
                                    />
                                    {canAddHideButton && <ShowHideIconButton data-id="hide-all-tools-button" disabled={false} onClick={toggleToolsExpandedWithStopPropagation}>
                                        <IconCommon
                                            decorative
                                            fill="background.textSecondary"
                                            name="Invisible"
                                        />
                                    </ShowHideIconButton>}
                                </React.Fragment>
                            );
                        })}
                    </Box>
                )}
                <Box flex={1} />
                <Box display="flex" alignItems="center" gap={1} ml={1}>
                    {mergedFeatures.allowVoiceInput && (
                        <MicrophoneButtonWithStopPropagation
                            fill={theme.palette.background.textSecondary}
                            onTranscriptChange={handleTranscriptChange}
                            disabled={false}
                            height={iconHeight}
                            width={iconWidth}
                        />
                    )}
                    {(mergedFeatures.allowCharacterLimit || mergedFeatures.allowSubmit) && (
                        <>
                            {/* Character limit and submit button combined UI */}
                            {mergedFeatures.allowCharacterLimit && (
                                <Box
                                    display="inline-flex"
                                    position="relative"
                                    sx={verticalMiddleStyle}
                                >
                                    <CircularProgress
                                        aria-label={maxChars ? `Character count progress: ${internalValue.length} of ${maxChars} characters used` : "Character count progress"}
                                        size={34}
                                        sx={progressStyle}
                                        value={charsProgress}
                                        variant="determinate"
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
                                        {mergedFeatures.allowSubmit && charsOverLimit <= 0 && <StyledIconButton
                                            bgColor="transparent"
                                            disabled={!internalValue.trim() || charsOverLimit > 0}
                                            onClick={handleSubmitWithStopPropagation}
                                        >
                                            <IconCommon
                                                decorative
                                                fill={!internalValue.trim()
                                                    ? "background.textSecondary"
                                                    : "background.textPrimary"
                                                }
                                                name="Send"
                                            />
                                        </StyledIconButton>}
                                    </Box>
                                </Box>
                            )}

                            {/* Submit button only (when character limit is disabled) */}
                            {!mergedFeatures.allowCharacterLimit && mergedFeatures.allowSubmit && (
                                <StyledIconButton
                                    bgColor={theme.palette.mode === "dark"
                                        ? theme.palette.background.textPrimary
                                        : theme.palette.primary.main
                                    }
                                    disabled={!internalValue.trim()}
                                    onClick={handleSubmitWithStopPropagation}
                                    sx={verticalMiddleStyle}
                                >
                                    <IconCommon
                                        decorative
                                        fill={!internalValue.trim()
                                            ? "background.textSecondary"
                                            : theme.palette.mode === "dark"
                                                ? "background.default"
                                                : "primary.contrastText"
                                        }
                                        name="Send"
                                    />
                                </StyledIconButton>
                            )}
                        </>
                    )}
                </Box>
            </Box>

            {/* Popup Menus */}
            {(mergedFeatures.allowFileAttachments ||
                mergedFeatures.allowImageAttachments ||
                mergedFeatures.allowTextAttachments) && (
                    <PlusMenu
                        anchorEl={anchorPlus}
                        onClose={handleClosePlusMenu}
                        onAttachFile={mergedFeatures.allowFileAttachments ? handleAttachFile : undefined}
                        onConnectExternalApp={mergedFeatures.allowTools ? handleConnectExternalApp : undefined}
                        onTakePhoto={mergedFeatures.allowImageAttachments ? handleTakePhoto : undefined}
                        onAddRoutine={mergedFeatures.allowTools ? handleOpenFindRoutineDialog : undefined}
                    />
                )}
            {mergedFeatures.allowSettingsCustomization && (
                <InfoMemo
                    anchorEl={anchorSettings}
                    onClose={handleCloseInfoMemo}
                    enterWillSubmit={enterWillSubmit}
                    mergedFeatures={mergedFeatures}
                    onToggleEnterWillSubmit={handleToggleEnterWillSubmit}
                    showToolbar={showToolbar}
                    onToggleToolbar={handleToggleToolbar}
                    onToggleSpellcheck={handleToggleSpellcheck}
                    spellcheck={spellcheck}
                />
            )}
            {mergedFeatures.allowTools && (
                <FindObjectDialog
                    find="List"
                    isOpen={isFindRoutineDialogOpen}
                    handleCancel={handleCloseFindRoutineDialog}
                    handleComplete={handleAddRoutine}
                    limitTo={findRoutineLimitTo}
                />
            )}
            {/* Add Context Dropdown */}
            {mergedFeatures.allowContextDropdown && (
                <ContextDropdown
                    anchorEl={dropdownAnchor}
                    onClose={handleCloseDropdown}
                    onSelect={handleSelectContext}
                    initialCategory={initialCategory}
                    searchText={searchText}
                    onSearchChange={setSearchText}
                />
            )}
        </Outer>
    );
}

export function AdvancedInput({
    name,
    features,
    ...props
}: AdvancedInputProps) {
    const [field, meta, helpers] = useField(name);

    function handleChange(value: string) {
        helpers.setValue(value);
    }

    return (
        <AdvancedInputBase
            {...props}
            name={name}
            features={features}
            value={field.value}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error ? String(meta.error) : undefined}
            onBlur={field.onBlur}
            onChange={handleChange}
        />
    );
}

export function TranslatedAdvancedInput({
    language,
    name,
    features,
    ...props
}: TranslatedAdvancedInputProps) {
    const [field, meta, helpers] = useField("translations");
    const translationData = getTranslationData(field, meta, language);

    const fieldValue = getDotNotationValue(translationData.value, name);
    const fieldError = getDotNotationValue(translationData.error, name);
    const fieldTouched = getDotNotationValue(translationData.touched, name);

    function handleBlur(event) {
        field.onBlur(event);
    }

    function handleChange(newText: string) {
        // Only use dot notation if the name has a dot in it
        if (name.includes(".")) {
            const updatedValue = setDotNotationValue(translationData.value ?? {}, name, newText);
            helpers.setValue(field.value.map((translation) => {
                if (translation.language === language) return updatedValue;
                return translation;
            }));
        }
        else handleTranslationChange(field, meta, helpers, { target: { name, value: newText } }, language);
    }

    return (
        <AdvancedInputBase
            {...props}
            name={name}
            features={features}
            value={typeof fieldValue === "string" ? fieldValue : ""}
            error={Boolean(fieldTouched && fieldError)}
            helperText={fieldTouched && fieldError ? String(fieldError) : undefined}
            onBlur={handleBlur}
            onChange={handleChange}
        />
    );
}

