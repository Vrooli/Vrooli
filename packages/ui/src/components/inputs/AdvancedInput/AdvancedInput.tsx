/* eslint-disable no-magic-numbers */
import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";
import Collapse from "@mui/material/Collapse";
import Typography from "@mui/material/Typography";
import type { Theme } from "@mui/material/styles";
import { styled, useTheme } from "@mui/material/styles";
import { getDotNotationValue, setDotNotationValue } from "@vrooli/shared";
import { useField } from "formik";
import React, { useCallback, useEffect, useMemo, useRef, useState, type MouseEvent } from "react";
import { useDropzone } from "react-dropzone";
import { useDimensions } from "../../../hooks/useDimensions.js";
import { useHotkeys } from "../../../hooks/useHotkeys.js";
import { useUndoRedo } from "../../../hooks/useUndoRedo.js";
import { IconCommon } from "../../../icons/Icons.js";
import { getTranslationData, handleTranslationChange } from "../../../utils/display/translationTools.js";
import { getCookie, setCookie } from "../../../utils/localStorage.js";
import { PubSub } from "../../../utils/pubsub.js";
import { Divider } from "../../layout/Divider.js";
import { Tooltip } from "../../Tooltip/Tooltip.js";
import { IconButton as CustomIconButton } from "../../buttons/IconButton.js";
import { MicrophoneButton } from "../../buttons/MicrophoneButton.js";
import { type MicrophoneButtonProps } from "../../buttons/types.js";
import { FindObjectDialog } from "../../dialogs/FindObjectDialog/FindObjectDialog.js";
import { SnackSeverity } from "../../snacks/BasicSnack/BasicSnack.js";
import { AdvancedInputMarkdown } from "./AdvancedInputMarkdown.js";
import { AdvancedInputToolbar, defaultActiveStates } from "./AdvancedInputToolbar.js";
import { ContextDropdown, type ListObject } from "./ContextDropdown.js";
import { AdvancedInputLexical } from "./lexical/AdvancedInputLexical.js";
import { AdvancedInputAction, DEFAULT_FEATURES, advancedInputTextareaClassName, type AdvancedInputActiveStates, type AdvancedInputBaseProps, type AdvancedInputProps, type ContextItem, type TranslatedAdvancedInputProps } from "./utils.js";
// Import extracted components
import { ContextItemDisplay, previewImageStyle } from "./ContextItems/index.js";
import { InfoMenu, PlusMenu } from "./Menus/index.js";
import { TaskChip, useTaskActions } from "./TaskManager/index.js";
import {
    MAX_ROWS_COLLAPSED,
    MAX_ROWS_EXPANDED,
    MIN_ROWS_COLLAPSED,
    MIN_ROWS_EXPANDED,
    NON_FOCUSABLE_SELECTOR,
    TRIGGER_CHARS,
    bottomRowStyles,
    findRoutineLimitTo,
    getFilesFromEvent,
    iconHeight,
    iconWidth,
    preventInputLossOnToolbarClick,
    toolbarIconButtonClassName,
    toolbarRowStyles,
    verticalMiddleStyle,
    type FocusableElement,
} from "./utils/index.js";


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



const Outer = styled("div")(({ theme }) => ({
    backgroundColor: theme.palette.background.paper,
    borderRadius: theme.spacing(3),
    padding: theme.spacing(1),
    position: "relative",
    // Ensure it can receive clicks even if children cover parts of it
    // (though padding usually handles this)
    cursor: "text", // Optional: hint that the area is for text input
}));

// List of selectors for interactive elements that shouldn't trigger input focus
const NON_FOCUSABLE_SELECTORS = [
    "button",
    "a",
    "input",
    "textarea",
    "[contenteditable=\"true\"]",
    "[class*='tw-icon-button']", // Custom IconButton classes
    ".MuiChip-root",
    ".MuiButtonBase-root",
    "input[role='switch']", // Custom Switch component
    ".MuiSlider-root",
    ".MuiMenuItem-root",
    NON_FOCUSABLE_SELECTOR, // Import the extended selector
].join(",");

// Class name added to the AdvancedInput textarea/contentEditable for identification
const ADVANCED_INPUT_CONTENT_CLASS = advancedInputTextareaClassName;












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


// Prevents click events from bubbling up and causing input focus
const stopPropagationHandler = (e: React.MouseEvent) => {
    e.stopPropagation();
};

// Wraps MicrophoneButton to stop propagation on click and maintain a stable component identity
const MicrophoneButtonWithStopPropagation: React.FC<MicrophoneButtonProps> = React.memo(({ onTranscriptChange, ...props }) => {
    const handleTranscriptChangeWithStopPropagation = useCallback((transcript: string) => {
        onTranscriptChange(transcript);
    }, [onTranscriptChange]);

    return (
        <Box onClick={stopPropagationHandler} sx={props.sx}>
            <MicrophoneButton onTranscriptChange={handleTranscriptChangeWithStopPropagation} {...props} />
        </Box>
    );
});

/** TextInput for entering rich text. Supports markdown and WYSIWYG */
export function AdvancedInputBase({
    tasks = [],
    contextData = [],
    value,
    disabled = false,
    error = false,
    features = DEFAULT_FEATURES,
    helperText,
    isRequired = false,
    name,
    onBlur,
    onChange,
    onFocus,
    onTasksChange,
    onContextDataChange,
    onSubmit,
    placeholder,
    sxs,
    tabIndex,
    title,
    "data-testid": dataTestId,
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

    // Determine min/max rows based on features or defaults
    const minRows = useMemo(() => {
        if (isExpanded) {
            return mergedFeatures.minRowsExpanded ?? MIN_ROWS_EXPANDED;
        }
        return mergedFeatures.minRowsCollapsed ?? MIN_ROWS_COLLAPSED;
    }, [isExpanded, mergedFeatures.minRowsExpanded, mergedFeatures.minRowsCollapsed]);

    const maxRows = useMemo(() => {
        if (isExpanded) {
            return mergedFeatures.maxRowsExpanded ?? MAX_ROWS_EXPANDED;
        }
        return mergedFeatures.maxRowsCollapsed ?? MAX_ROWS_COLLAPSED;
    }, [isExpanded, mergedFeatures.maxRowsExpanded, mergedFeatures.maxRowsCollapsed]);

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
        // When formatting is disabled, always use markdown
        if (!mergedFeatures.allowFormatting) {
            return true;
        }
        // When formatting is enabled and toolbar is visible, prefer lexical editor
        if (showToolbar) {
            return false;
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
                    id: nanoid(),
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
    const { dimensions: tasksContainerDimensions, ref: tasksContainerRef } = useDimensions();
    const showAllToolsButtonRef = useRef(false);
    const [showEllipsisBeforeLastTool, setShowEllipsisBeforeLastTool] = useState(false);
    const showEllipsisBeforeLastToolRef = useRef(false);
    const [tasksInFirstRow, setToolsInFirstRow] = useState(0);
    const tasksInFirstRowRef = useRef(0);
    const checkOverflow = useCallback(function checkIfToolsOverflowCallback() {
        if (tasksContainerRef.current) {
            // Get an array of children elements
            const childrenArray = Array.from(tasksContainerRef.current.children);
            if (childrenArray.length === 0) return;

            // Determine the top offset of the first row
            const firstRowTop = (childrenArray[0] as HTMLElement).offsetTop;
            // Count all children that share the same offsetTop as the first child
            const firstRowItems = childrenArray.filter(child => (child as HTMLElement).offsetTop === firstRowTop);
            const firstRowToolItems = firstRowItems.filter(child => (child as HTMLElement).getAttribute("data-type") === "task");

            // Check how much space if left in the container
            const containerRect = tasksContainerRef.current.getBoundingClientRect();
            let maxRight = 0;
            firstRowItems.forEach((child, index) => {
                // Make sure we're not adding the ellipsis before the last item.
                if (child.getAttribute("data-id") === "show-all-tasks-button" && index !== firstRowItems.length - 1) {
                    maxRight = Number.MAX_SAFE_INTEGER;
                } else if (child.getAttribute("data-type") === "task") {
                    const childRect = child.getBoundingClientRect();
                    maxRight = Math.max(maxRight, childRect.right);
                }
            });
            const remWidth = containerRect.right - maxRight;

            // Determine if the ellipsis should be added before or after the last item in the first row
            const shouldAddEllipsisBefore = Math.abs(remWidth) < 40;
            // console.log({ maxRight, remWidth, shouldAddEllipsisBefore }); // Keep for debugging if needed

            if (firstRowToolItems.length !== tasksInFirstRowRef.current) {
                setToolsInFirstRow(firstRowToolItems.length);
                tasksInFirstRowRef.current = firstRowToolItems.length;
            }
            if (shouldAddEllipsisBefore !== showEllipsisBeforeLastToolRef.current) {
                setShowEllipsisBeforeLastTool(shouldAddEllipsisBefore);
                showEllipsisBeforeLastToolRef.current = shouldAddEllipsisBefore;
            }
            if (firstRowToolItems.length < tasks.length !== showAllToolsButtonRef.current) {
                showAllToolsButtonRef.current = firstRowToolItems.length < tasks.length;
            }
        }
    }, [tasks.length, tasksContainerRef]);
    useEffect(function checkOverflowOnResizeEffect() {
        checkOverflow();
    }, [checkOverflow, tasks, tasksContainerDimensions]);

    const { handleToggleTask, handleToggleTaskExclusive, handleRemoveTask } = useTaskActions(tasks, onTasksChange);

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

    const handleOpenInfoMenu = useCallback((event: MouseEvent<HTMLElement>) => {
        setAnchorSettings(event.currentTarget);
    }, []);

    const handleCloseInfoMenu = useCallback(() => {
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
        console.log("Add new task...");
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
        // Append recognized text to the current value using functional update for stability
        changeInternalValue((prev) => {
            const trimmed = prev.trim();
            return trimmed ? `${prev} ${recognizedText}` : recognizedText;
        });
    }, [changeInternalValue]);

    const handleSubmit = useCallback(() => {
        onSubmit?.(internalValue);
    }, [internalValue, onSubmit]);

    const imgStyle = useMemo(() => previewImageStyle(theme), [theme]);
    const tasksContainerStyles = useMemo(() => ({
        display: "flex",
        flexWrap: "wrap" as const,
        alignItems: "center",
        maxHeight: isToolsExpanded ? "none" : theme.spacing(4), // Approximately one row height
        overflow: "hidden",
        transition: "max-height 0.3s ease",
        gap: theme.spacing(1),
    }), [isToolsExpanded, theme]);

    const charsProgress = mergedFeatures.maxChars ? Math.min(100, Math.ceil(((internalValue ?? "").length / mergedFeatures.maxChars) * 100)) : 0;
    const charsOverLimit = mergedFeatures.maxChars ? Math.max(0, ((internalValue ?? "").length) - mergedFeatures.maxChars) : 0;

    const progressStyle = useMemo(() => {
        let style = { color: theme.palette.success.main };
        if (charsOverLimit > 0) {
            style = { color: theme.palette.error.main };
        } else if (charsProgress >= 80) {
            style = { color: theme.palette.warning.main };
        }
        return style;
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
            console.error("AdvancedInputBase: No child handleAction function found");
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
        maxRows,
        minRows,
        name,
        onActiveStatesChange: handleActiveStatesChange,
        onBlur,
        onFocus,
        onSubmit,
        placeholder,
        setHandleAction: setChildHandleAction,
        tabIndex,
        toggleMarkdown,
    }), [
        disabled, enterWillSubmit, error, maxRows, minRows, name, // Add maxRows and minRows to dependency array
        handleActiveStatesChange, onBlur, onFocus, onSubmit, placeholder, setChildHandleAction,
        tabIndex, toggleMarkdown,
    ]);

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
    const handleOpenInfoMenuWithStopPropagation = useCallback((event: MouseEvent<HTMLElement>) => {
        // Stop propagation to prevent focusing the input
        event.stopPropagation();
        handleOpenInfoMenu(event);
    }, [handleOpenInfoMenu]);

    // Add this handler for the expand toggle
    const handleToggleExpandWithStopPropagation = useCallback((event: MouseEvent<HTMLElement>) => {
        // Stop propagation to prevent focusing the input
        event.stopPropagation();
        handleToggleExpand();
    }, [handleToggleExpand]);

    // Add this handler for the tasks expand toggle
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

    return (
        // Add the click handler to the Outer component
        <Outer
            className="advanced-input"
            data-testid={dataTestId || "advanced-input"}
            data-disabled={disabled}
            data-error={error}
            {...(mergedFeatures.allowFileAttachments ? getRootProps() : {})}
            onClick={handleOuterClick}
            ref={containerRef} // Add ref to the Outer component for hotkeys
            sx={sxs?.root}
        >
            {mergedFeatures.allowFileAttachments && <input {...getInputProps()} />}
            {mergedFeatures.allowFileAttachments && isDragActive && (
                <Box sx={{ ...dragOverlayStyles, ...(sxs?.dragOverlay ?? {}) }}>
                    <Typography variant="body1" color="text.secondary">
                        Drop files here...
                    </Typography>
                </Box>
            )}
            <Collapse in={mergedFeatures.allowTasks && tasks.some((task) => task.state === AITaskDisplayState.Exclusive)} unmountOnExit>
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
            <Box onMouseDown={preventInputLossOnToolbarClick} sx={{ ...toolbarRowStyles, ...(sxs?.toolbar ?? {}) }}>
                {mergedFeatures.allowSettingsCustomization && (
                    <CustomIconButton
                        onClick={handleOpenInfoMenuWithStopPropagation}
                        className={toolbarIconButtonClassName}
                        size={28}
                        data-testid="settings-button"
                        aria-label="Settings"
                    >
                        <IconCommon
                            decorative
                            fill="background.textSecondary"
                            name="Settings"
                        />
                    </CustomIconButton>
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
                {(!mergedFeatures.allowFormatting || !showToolbar) && <Box sx={{ ...contextRowStyles, ...(sxs?.contextRow ?? {}) }}>
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
                        <CustomIconButton
                            onClick={handleToggleExpandWithStopPropagation}
                            className={toolbarIconButtonClassName}
                            size={28}
                        >
                            <IconCommon
                                decorative
                                fill="background.textSecondary"
                                name={isExpanded ? "ExpandLess" : "ExpandMore"}
                            />
                        </CustomIconButton>
                    </Tooltip>
                )}
            </Box>
            {mergedFeatures.allowFormatting && showToolbar && <Box sx={{ ...contextRowStyles, ...(sxs?.contextRow ?? {}) }}>
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
                <Box sx={{ ...titleStyles, ...(sxs?.title ?? {}) }}>
                    <Typography variant="subtitle1" fontWeight="medium" color="text.secondary">
                        {title}
                        {isRequired && <Typography component="span" variant="subtitle1" color="error" paddingLeft="4px">*</Typography>}                    </Typography>
                </Box>
            )}
            {/* Input Area */}
            <Box sx={{ ...inputRowStyles, ...(sxs?.inputContainer ?? {}) }}>
                <Box
                    ref={inputWrapperRef}
                    maxHeight={isExpanded ? "calc(100vh - 150px)" : "unset"}
                    overflow="auto"
                    sx={sxs?.inputWrapper}
                >
                    {/* Conditionally render Markdown or Lexical, passing the inputRef */}
                    {useMarkdownEditor ? MarkdownComponent : LexicalComponent}
                </Box>
            </Box>

            {/* Bottom Section */}
            <Box sx={{ ...bottomRowStyles, ...(sxs?.bottomRow ?? {}) }}>
                {mergedFeatures.allowTasks && (
                    <Box ref={tasksContainerRef} sx={{ ...tasksContainerStyles, ...(sxs?.tasksContainer ?? {}) }}>
                        {(mergedFeatures.allowFileAttachments ||
                            mergedFeatures.allowImageAttachments ||
                            mergedFeatures.allowTextAttachments) && (
                                <CustomIconButton
                                    variant="transparent"
                                    size={iconWidth}
                                    disabled={false}
                                    onClick={handleOpenPlusMenu}
                                    className="tw-p-1"
                                >
                                    <IconCommon
                                        decorative
                                        fill="background.textSecondary"
                                        name="Add"
                                    />
                                </CustomIconButton>
                            )}
                        {tasks.map((task, index) => {
                            function onToggleTool() {
                                handleToggleTask(index);
                            }
                            function onRemoveTool() {
                                handleRemoveTask(index);
                            }
                            function onToggleToolExclusive() {
                                handleToggleTaskExclusive(index);
                            }

                            const canAddShowButton = tasksInFirstRow < tasks.length && !isToolsExpanded && ((index === tasksInFirstRow && !showEllipsisBeforeLastTool) || (index === tasksInFirstRow - 1 && showEllipsisBeforeLastTool));
                            const canAddHideButton = tasksInFirstRow < tasks.length && isToolsExpanded && index === tasks.length - 1;

                            return (
                                <React.Fragment key={`${task.displayName || task.label}-${index}`}>
                                    {canAddShowButton && <CustomIconButton
                                        data-id="show-all-tasks-button"
                                        variant="transparent"
                                        size={iconWidth}
                                        disabled={false}
                                        onClick={toggleToolsExpandedWithStopPropagation}
                                        className="tw-p-1"
                                    >
                                        <IconCommon
                                            decorative
                                            fill="background.textSecondary"
                                            name="Ellipsis"
                                        />
                                    </CustomIconButton>}
                                    <TaskChip
                                        {...task}
                                        index={index}
                                        onRemoveTool={onRemoveTool}
                                        onToggleToolExclusive={onToggleToolExclusive}
                                        onToggleTool={onToggleTool}
                                    />
                                    {canAddHideButton && <CustomIconButton
                                        data-id="hide-all-tasks-button"
                                        variant="transparent"
                                        size={iconWidth}
                                        disabled={false}
                                        onClick={toggleToolsExpandedWithStopPropagation}
                                        className="tw-p-1"
                                    >
                                        <IconCommon
                                            decorative
                                            fill="background.textSecondary"
                                            name="Invisible"
                                        />
                                    </CustomIconButton>}
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
                            sx={sxs?.voiceButton}
                        />
                    )}
                    {(mergedFeatures.allowCharacterLimit || mergedFeatures.allowSubmit) && (
                        <>
                            {/* Character limit and submit button combined UI */}
                            {mergedFeatures.allowCharacterLimit && (
                                <Box
                                    display="inline-flex"
                                    position="relative"
                                    sx={{ ...verticalMiddleStyle, ...(sxs?.characterCount ?? {}) }}
                                >
                                    <CircularProgress
                                        aria-label={
                                            mergedFeatures.maxChars
                                                ? `Character count progress: ${(internalValue ?? "").length} of ${mergedFeatures.maxChars} characters used`
                                                : "Character count progress"
                                        }
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
                                        {mergedFeatures.allowSubmit && charsOverLimit <= 0 && <CustomIconButton
                                            variant="transparent"
                                            size={24}
                                            disabled={!(internalValue ?? "").trim() || charsOverLimit > 0}
                                            onClick={handleSubmitWithStopPropagation}
                                            className="tw-p-0"
                                            data-testid="submit-button-inline"
                                            aria-label="Send"
                                            sx={sxs?.submitButton}
                                        >
                                            <IconCommon
                                                decorative
                                                fill={!(internalValue ?? "").trim()
                                                    ? "background.textSecondary"
                                                    : "background.textPrimary"
                                                }
                                                name="Send"
                                            />
                                        </CustomIconButton>}
                                    </Box>
                                </Box>
                            )}

                            {/* Submit button only (when character limit is disabled) */}
                            {!mergedFeatures.allowCharacterLimit && mergedFeatures.allowSubmit && (
                                <CustomIconButton
                                    variant="solid"
                                    size={iconWidth}
                                    disabled={!(internalValue ?? "").trim()}
                                    onClick={handleSubmitWithStopPropagation}
                                    className="tw-align-middle"
                                    data-testid="submit-button"
                                    aria-label="Send"
                                    sx={sxs?.submitButton}
                                >
                                    <IconCommon
                                        decorative
                                        fill={!(internalValue ?? "").trim()
                                            ? "background.textSecondary"
                                            : theme.palette.mode === "dark"
                                                ? "background.default"
                                                : "primary.contrastText"
                                        }
                                        name="Send"
                                    />
                                </CustomIconButton>
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
                        onConnectExternalApp={mergedFeatures.allowTasks ? handleConnectExternalApp : undefined}
                        onTakePhoto={mergedFeatures.allowImageAttachments ? handleTakePhoto : undefined}
                        onAddRoutine={mergedFeatures.allowTasks ? handleOpenFindRoutineDialog : undefined}
                    />
                )}
            {mergedFeatures.allowSettingsCustomization && (
                <InfoMenu
                    anchorEl={anchorSettings}
                    onClose={handleCloseInfoMenu}
                    enterWillSubmit={enterWillSubmit}
                    mergedFeatures={mergedFeatures}
                    onToggleEnterWillSubmit={handleToggleEnterWillSubmit}
                    showToolbar={showToolbar}
                    onToggleToolbar={handleToggleToolbar}
                    onToggleSpellcheck={handleToggleSpellcheck}
                    spellcheck={spellcheck}
                />
            )}
            {mergedFeatures.allowTasks && (
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
    isRequired,
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
            isRequired={isRequired}
        />
    );
}

export function TranslatedAdvancedInput({
    language,
    name,
    features,
    isRequired,
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
            isRequired={isRequired}
            data-testid={props["data-testid"] || `input-${name}`}
        />
    );
}

