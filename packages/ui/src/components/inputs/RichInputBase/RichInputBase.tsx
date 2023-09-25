import { Box, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { CharLimitIndicator } from "components/CharLimitIndicator/CharLimitIndicator";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useUndoRedo } from "hooks/useUndoRedo";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getCookieShowMarkdown, setCookieShowMarkdown } from "utils/cookies";
import { noop } from "utils/objects";
import { assistantChatInfo, ChatCrud } from "views/objects/chat/ChatCrud/ChatCrud";
import { ChatCrudProps } from "views/objects/chat/types";
import { RichInputLexical } from "../RichInputLexical/RichInputLexical";
import { RichInputMarkdown } from "../RichInputMarkdown/RichInputMarkdown";
import { defaultActiveStates, RichInputToolbar } from "../RichInputToolbar/RichInputToolbar";
import { RichInputAction, RichInputActiveStates, RichInputBaseProps } from "../types";

export const LINE_HEIGHT_MULTIPLIER = 1.5;


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

    const { internalValue, changeInternalValue, resetInternalValue, undo, redo, canUndo, canRedo } = useUndoRedo({
        initialValue: value,
        onChange,
        forceAddToStack: (updated, resetAddToStack) => {
            // Force add if a delimiter (e.g. space, newline) is typed
            const lastChar = updated[updated.length - 1];
            if (lastChar === " " || lastChar === "\n") {
                return true;
            }
            // Otherwise, cancel the add to stack to the stack is only updated on inactivity
            resetAddToStack();
            return false;
        },
    });
    useEffect(() => {
        resetInternalValue(value);
    }, [value, resetInternalValue]);

    const [isMarkdownOn, setIsMarkdownOn] = useState(getCookieShowMarkdown() ?? false);
    const toggleMarkdown = useCallback(() => {
        setIsMarkdownOn(!isMarkdownOn);
        setCookieShowMarkdown(!isMarkdownOn);
    }, [isMarkdownOn]);

    const [assistantDialogProps, setAssistantDialogProps] = useState<ChatCrudProps>({
        context: undefined,
        isCreate: true,
        isOpen: false,
        overrideObject: assistantChatInfo,
        task: "note",
        onCompleted: () => { setAssistantDialogProps(props => ({ ...props, isOpen: false })); },
    });
    const openAssistantDialog = useCallback((highlighted: string) => {
        if (disabled) return;
        // We want to provide the assistant with the most relevant context
        const maxContextLength = 1500;
        let context = highlighted.trim();
        if (context.length > maxContextLength) context = context.substring(0, maxContextLength);
        // If there's not highlighted text, provide the full text if it's not too long
        else if (internalValue.length <= maxContextLength) context = internalValue;
        // Otherwise, provide the last 1500 characters
        else context = internalValue.substring(internalValue.length - maxContextLength, internalValue.length);
        // Open the assistant dialog
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
            e.preventDefault();
            e.stopPropagation();
        }
    }, [id]);

    // Actions which store and active state for the Toolbar. 
    // This is currently ignored when markdown mode is on, since it's 
    // a bitch to keep track of
    const [activeStates, setActiveStates] = useState<Omit<RichInputActiveStates, "SetValue">>(defaultActiveStates);
    const handleActiveStatesChange = (newActiveStates) => {
        setActiveStates(newActiveStates);
    };

    const currentHandleActionRef = useRef<((action: RichInputAction, data?: unknown) => unknown) | null>(null);
    const setChildHandleAction = useCallback((handleAction: (action: RichInputAction, data?: unknown) => unknown) => {
        currentHandleActionRef.current = handleAction;
    }, []);
    const handleAction = useCallback((action: RichInputAction, data?: unknown) => {
        if (action === "Mode") {
            toggleMarkdown();
        } else if (currentHandleActionRef.current) {
            currentHandleActionRef.current(action, data);
        } else {
            console.error("RichInputBase: No child handleAction function found");
        }
        return noop;
    }, [toggleMarkdown]);
    const viewProps = useMemo(() => ({
        autoFocus,
        disabled,
        error,
        getTaggableItems,
        id,
        maxRows,
        minRows,
        name,
        onActiveStatesChange: handleActiveStatesChange,
        onBlur,
        onChange: changeInternalValue,
        openAssistantDialog,
        placeholder,
        redo,
        setHandleAction: setChildHandleAction,
        tabIndex,
        toggleMarkdown,
        undo,
        value: internalValue,
        sx: sxs?.textArea,
    }), [autoFocus, changeInternalValue, disabled, error, getTaggableItems, id, internalValue, maxRows, minRows, name, onBlur, openAssistantDialog, placeholder, redo, setChildHandleAction, sxs?.textArea, tabIndex, toggleMarkdown, undo]);

    return (
        <>
            {/* Assistant dialog for generating text */}
            {!disableAssistant && <ChatCrud {...assistantDialogProps} />}
            <Stack
                id={`markdown-input-base-${name}`}
                direction="column"
                spacing={0}
                onMouseDown={handleMouseDown}
                sx={{ ...(sxs?.root ?? {}) }}
            >
                <RichInputToolbar
                    activeStates={activeStates}
                    canRedo={canRedo}
                    canUndo={canUndo}
                    disableAssistant={disableAssistant}
                    disabled={disabled}
                    handleAction={handleAction}
                    handleActiveStatesChange={handleActiveStatesChange}
                    isMarkdownOn={isMarkdownOn}
                    name={name}
                    sx={sxs?.bar}
                />
                {isMarkdownOn ? <RichInputMarkdown {...viewProps} /> : <RichInputLexical {...viewProps} />}
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
