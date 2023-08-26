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
    const { palette } = useTheme();
    const isLeftHanded = useIsLeftHanded();
    console.log("richinputbase render", typeof getTaggableItems);

    // Stores previous states for undo/redo (since we can't use the browser's undo/redo due to programmatic changes)
    const changeStack = useRef<string[]>([value]);
    const [changeStackIndex, setChangeStackIndex] = useState<number>(0);

    // Internal value (since value passed back is debounced)
    const [internalValue, setInternalValue] = useState<string>(value ?? "");
    useEffect(() => {
        // If new value is one of the recent items in the stack 
        // (i.e. debounce is firing while user is still typing),
        // then don't update the internal value
        const recentItems = changeStack.current.slice(Math.max(changeStack.current.length - 5, 0));
        if (value === "" || !recentItems.includes(value)) {
            setInternalValue(value);
        }
    }, [value]);
    // Debounce text change
    const onChangeDebounced = useDebounce(onChange, 200);

    const [isMarkdownOn, setIsMarkdownOn] = useState(getCookieShowMarkdown() ?? false);
    const toggleMarkdown = useCallback(() => {
        setIsMarkdownOn(!isMarkdownOn);
        setCookieShowMarkdown(!isMarkdownOn);
    }, [isMarkdownOn]);

    /** Moves back one in the change stack */
    const undo = useCallback(() => {
        if (disabled || changeStackIndex <= 0) return;
        setChangeStackIndex(changeStackIndex - 1);
        setInternalValue(changeStack.current[changeStackIndex - 1]);
        onChangeDebounced(changeStack.current[changeStackIndex - 1]);
    }, [changeStackIndex, disabled, onChangeDebounced]);
    const canUndo = useMemo(() => changeStackIndex > 0 && changeStack.current.length > 0, [changeStackIndex]);
    /** Moves forward one in the change stack */
    const redo = useCallback(() => {
        if (disabled || changeStackIndex >= changeStack.current.length - 1) return;
        setChangeStackIndex(changeStackIndex + 1);
        setInternalValue(changeStack.current[changeStackIndex + 1]);
        onChangeDebounced(changeStack.current[changeStackIndex + 1]);
    }, [changeStackIndex, disabled, onChangeDebounced]);
    const canRedo = useMemo(() => changeStackIndex < changeStack.current.length - 1 && changeStack.current.length > 0, [changeStackIndex]);
    /**
     * Adds, to change stack, and removes anything from the change stack after the current index
     */
    const handleChange = useCallback((updatedText: string) => {
        console.log("handleChange", updatedText, changeStackIndex, changeStack.current);
        const newChangeStack = [...changeStack.current];
        newChangeStack.splice(changeStackIndex + 1, newChangeStack.length - changeStackIndex - 1);
        newChangeStack.push(updatedText);
        changeStack.current = newChangeStack;
        setChangeStackIndex(newChangeStack.length - 1);
        setInternalValue(updatedText);
        onChangeDebounced(updatedText);
    }, [changeStackIndex, onChangeDebounced]);

    // Get current view 
    const CurrentViewComponent = useMemo(() => isMarkdownOn ? RichInputMarkdown : RichInputLexical, [isMarkdownOn]);
    // Map view-specific functions to this component
    const handleAction: RichInputChildView["handleAction"] = useCallback((action: RichInputAction) => {
        if (CurrentViewComponent && (CurrentViewComponent as unknown as RichInputChildView).handleAction) {
            (CurrentViewComponent as unknown as RichInputChildView).handleAction(action);
        }
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        return () => { };
    }, [CurrentViewComponent]);

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
        setAssistantDialogProps(props => ({ ...props, isOpen: true, context: context ? `\`\`\`\n${context}\n\`\`\`\n\n` : undefined }));
    }, [disabled, internalValue]);

    return (
        <>
            {/* Assistant dialog for generating text */}
            {!disableAssistant && <ChatView {...assistantDialogProps} />}
            <Stack
                id={`markdown-input-base-${name}`}
                direction="column"
                spacing={0}
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
