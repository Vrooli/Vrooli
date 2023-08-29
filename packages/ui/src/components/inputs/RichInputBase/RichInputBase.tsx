import { Box, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { CharLimitIndicator } from "components/CharLimitIndicator/CharLimitIndicator";
import { useDebounce } from "hooks/useDebounce";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getCookieShowMarkdown, setCookieShowMarkdown } from "utils/cookies";
import { assistantChatInfo, ChatView } from "views/ChatView/ChatView";
import { ChatViewProps } from "views/types";
import { RichInputLexical } from "../RichInputLexical/RichInputLexical";
import { RichInputMarkdown } from "../RichInputMarkdown/RichInputMarkdown";
import { RichInputAction, RichInputToolbar } from "../RichInputToolbar/RichInputToolbar";
import { RichInputBaseProps, RichInputChildView } from "../types";

export const LINE_HEIGHT_MULTIPLIER = 1.5;
const MAX_STACK_SIZE = 1000000; // Total characters stored in the change stack. 1 million characters will be about 1-1.5 MB

/** TextField for entering rich text. Supports markdown and WYSIWYG */
export const RichInputBase = ({
    actionButtons,
    autoFocus = false,
    disabled = false,
    disableAssistant = false,
    error = false,
    getTaggableItems,
    helperText,
    maxChars,
    maxRows,
    minRows = 4,
    name,
    onBlur,
    onChange,
    placeholder = "",
    tabIndex,
    value,
    sxs,
}: RichInputBaseProps) => {
    const { palette, typography } = useTheme();
    const isLeftHanded = useIsLeftHanded();
    console.log("richinputbase render", typeof getTaggableItems);

    // Stores previous states for undo/redo (since we can't use the browser's undo/redo due to programmatic changes)
    const stack = useRef<string[]>([value]);
    const [stackIndex, setStackIndex] = useState<number>(0);
    const stackSize = useRef<number>(value?.length ?? 0);

    // Internal value (since value passed back is debounced)
    const [internalValue, setInternalValue] = useState<string>(value ?? "");
    useEffect(() => {
        // If new value is one of the recent items in the stack 
        // (i.e. debounce is firing while user is still typing),
        // then don't update the internal value
        const recentItems = stack.current.slice(Math.max(stack.current.length - 5, 0));
        if (value === "" || !recentItems.includes(value)) {
            setInternalValue(value);
        }
    }, [value]);
    // Debounce text change
    const onChangeDebounced = useDebounce(onChange, 200);

    /** Moves back one in the change stack */
    const undo = useCallback(() => {
        if (disabled || stackIndex <= 0) return;
        setStackIndex(stackIndex - 1);
        setInternalValue(stack.current[stackIndex - 1]);
        onChangeDebounced(stack.current[stackIndex - 1]);
    }, [stackIndex, disabled, onChangeDebounced]);
    const canUndo = useMemo(() => stackIndex > 0 && stack.current.length > 0, [stackIndex]);
    /** Moves forward one in the change stack */
    const redo = useCallback(() => {
        if (disabled || stackIndex >= stack.current.length - 1) return;
        setStackIndex(stackIndex + 1);
        setInternalValue(stack.current[stackIndex + 1]);
        onChangeDebounced(stack.current[stackIndex + 1]);
    }, [stackIndex, disabled, onChangeDebounced]);
    const canRedo = useMemo(() => stackIndex < stack.current.length - 1 && stack.current.length > 0, [stackIndex]);
    /**
     * Adds, to change stack, and removes anything from the change stack after the current index
     */
    const handleChange = useCallback((updatedText: string) => {
        console.log("handleChange", updatedText, stackIndex, stack.current);
        // Add to change stack
        const newstack = [...stack.current];
        newstack.splice(stackIndex + 1, newstack.length - stackIndex - 1);
        newstack.push(updatedText);
        stackSize.current += updatedText.length;
        // Remove oldest item(s) if stack is too long
        if (stackSize.current > MAX_STACK_SIZE) {
            console.info("RichInputBase: change stack is too long, removing oldest items");
            // Remove as many items as needed to get back to the max stack size
            while (stackSize.current > MAX_STACK_SIZE) {
                stackSize.current -= newstack[0].length;
                newstack.shift();
            }
        }
        stack.current = newstack;
        setStackIndex(newstack.length - 1);
        setInternalValue(updatedText);
        onChangeDebounced(updatedText);
    }, [stackIndex, onChangeDebounced]);

    const [isMarkdownOn, setIsMarkdownOn] = useState(getCookieShowMarkdown() ?? true); //TODO default to false once lexical view is better
    const toggleMarkdown = useCallback(() => {
        setIsMarkdownOn(!isMarkdownOn);
        setCookieShowMarkdown(!isMarkdownOn);
    }, [isMarkdownOn]);

    // Get current view 
    const CurrentViewComponent = useMemo(() => isMarkdownOn ? RichInputMarkdown : RichInputLexical, [isMarkdownOn]);
    // Map view-specific functions to this component
    const handleAction = useCallback((action: RichInputAction, data?: unknown) => {
        console.log("in RichInputBase handleAction", action, CurrentViewComponent, (CurrentViewComponent as unknown as RichInputChildView)?.handleAction);
        // We can handle Mode without passing to the view
        if (action === "Mode") {
            toggleMarkdown();
        }
        // Otherwise, pass to the view
        else if (CurrentViewComponent && (CurrentViewComponent as unknown as RichInputChildView).handleAction) {
            (CurrentViewComponent as unknown as RichInputChildView).handleAction(action, data);
        } else {
            console.error("RichInputBase: CurrentViewComponent does not have a handleAction function");
        }
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        return () => { };
    }, [CurrentViewComponent, toggleMarkdown]);

    const [assistantDialogProps, setAssistantDialogProps] = useState<ChatViewProps>({
        chatInfo: assistantChatInfo,
        context: undefined,
        isOpen: false,
        task: "note",
        onClose: () => { setAssistantDialogProps(props => ({ ...props, isOpen: false })); },
        // handleComplete: (data) => { console.log("completed", data); setAssistantDialogProps(props => ({ ...props, isOpen: false })); },
    });
    const openAssistantDialog = useCallback((highlighted: string) => {
        if (disabled) return;
        // We want to provide the assistant with the most relevant context
        const maxContextLength = 1500;
        let context = highlighted.trim();
        if (context.length > maxContextLength) context = context.substring(0, maxContextLength);
        if (context.length > 0) context = context;
        // If there's not highlighted text, provide the full text if it's not too long
        else if (internalValue.length <= maxContextLength) context = internalValue;
        // Otherwise, provide the last 1500 characters
        else context = internalValue.substring(internalValue.length - maxContextLength, internalValue.length);
        // Open the assistant dialog
        console.log("context here", context);
        setAssistantDialogProps(props => ({ ...props, isOpen: true, context: context ? `"""\n${context}\n"""\n` : undefined }));
    }, [disabled, internalValue]);

    // Resize input area to fit content
    const id = useMemo(() => `input-container-${name}`, [name]);
    const resize = useCallback(() => {
        const container = document.getElementById(id);
        if (!container || typeof internalValue !== "string" || sxs?.textArea?.height) return;
        const lines = (internalValue.match(/\n/g)?.length || 0) + 1;
        const lineHeight = Math.round(typography.fontSize * LINE_HEIGHT_MULTIPLIER);
        const minRowsNum = minRows ? Number.parseInt(minRows + "") : 2;
        const maxRowsNum = maxRows ? Number.parseInt(maxRows + "") : lines;
        const linesShown = Math.max(minRowsNum, Math.min(lines, maxRowsNum));
        const padding = 34;
        container.style.height = `${linesShown * lineHeight + padding}px`;
    }, [id, minRows, maxRows, typography, internalValue, sxs?.textArea?.height]);
    useEffect(() => {
        resize();
    }, [resize, isMarkdownOn]);

    /** Prevents input from losing focus when the toolbar is pressed */
    const handleMouseDown = useCallback((e) => {
        // Find the first parent element with an id
        let parent = e.target;
        while (parent && !parent.id) {
            parent = parent.parentElement;
        }
        // If the target is not the textArea, then prevent default
        if (parent && parent.id !== id) {
            console.log("preventing default", parent.id);
            e.preventDefault();
            e.stopPropagation();
        }
    }, [id]);

    return (
        <>
            {/* Assistant dialog for generating text */}
            {!disableAssistant && <ChatView {...assistantDialogProps} />}
            <Stack
                id={`markdown-input-base-${name}`}
                direction="column"
                spacing={0}
                onMouseDown={handleMouseDown}
                sx={{ ...(sxs?.root ?? {}) }}
            >
                <RichInputToolbar
                    canRedo={canRedo}
                    canUndo={canUndo}
                    disableAssistant={disableAssistant}
                    disabled={disabled}
                    handleAction={handleAction}
                    isMarkdownOn={isMarkdownOn}
                    name={name}
                    sx={sxs?.bar}
                />
                {/* TextField for entering Markdown or WYSISWG */}
                <CurrentViewComponent
                    autoFocus={autoFocus}
                    disabled={disabled}
                    error={error}
                    getTaggableItems={getTaggableItems}
                    id={id}
                    maxRows={maxRows}
                    minRows={minRows}
                    name={name}
                    onBlur={onBlur}
                    onChange={handleChange}
                    openAssistantDialog={openAssistantDialog}
                    placeholder={placeholder}
                    redo={redo}
                    tabIndex={tabIndex}
                    toggleMarkdown={toggleMarkdown}
                    undo={undo}
                    value={internalValue}
                    sx={sxs?.textArea}
                />
                {/* Help text, characters remaining indicator, and action buttons */}
                {
                    (helperText || maxChars || (Array.isArray(actionButtons) && actionButtons.length > 0)) && <Box
                        sx={{
                            padding: 1,
                            display: "flex",
                            flexDirection: isLeftHanded ? "row-reverse" : "row",
                            gap: 1,
                            justifyContent: "space-between",
                            alitnItems: "center",
                        }}
                    >
                        <Typography variant="body1" mt="auto" mb="auto" sx={{ color: "red" }}>
                            {helperText}
                        </Typography>
                        <Box sx={{
                            display: "flex",
                            gap: 2,
                            ...(isLeftHanded ?
                                { marginRight: "auto", flexDirection: "row-reverse" } :
                                { marginLeft: "auto", flexDirection: "row" }),
                        }}>
                            {/* Characters remaining indicator */}
                            {
                                !disabled && maxChars !== undefined &&
                                <CharLimitIndicator
                                    chars={internalValue?.length ?? 0}
                                    maxChars={maxChars}
                                />
                            }
                            {/* Action buttons */}
                            {
                                actionButtons?.map(({ disabled: buttonDisabled, Icon, onClick, tooltip }, index) => (
                                    <Tooltip key={index} title={tooltip} placement="top">
                                        <IconButton
                                            disabled={disabled || buttonDisabled}
                                            size="small"
                                            onClick={onClick}
                                            sx={{ background: palette.primary.dark }}
                                        >
                                            <Icon fill={palette.primary.contrastText} />
                                        </IconButton>
                                    </Tooltip>
                                ))
                            }
                        </Box>
                    </Box>
                }
            </Stack >
        </>
    );
};
