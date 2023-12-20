import { ChatInviteStatus, DUMMY_ID, noop, uuid } from "@local/shared";
import { Box, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { CharLimitIndicator } from "components/CharLimitIndicator/CharLimitIndicator";
import { SessionContext } from "contexts/SessionContext";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useUndoRedo } from "hooks/useUndoRedo";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { getCookieMatchingChat, getCookieShowMarkdown, setCookieShowMarkdown } from "utils/cookies";
import { getDeviceInfo, keyComboToString } from "utils/display/device";
import { PubSub } from "utils/pubsub";
import { ChatShape } from "utils/shape/models/chat";
import { ChatCrud, VALYXA_INFO } from "views/objects/chat/ChatCrud/ChatCrud";
import { ChatCrudProps } from "views/objects/chat/types";
import { RichInputLexical } from "../RichInputLexical/RichInputLexical";
import { RichInputMarkdown } from "../RichInputMarkdown/RichInputMarkdown";
import { RichInputToolbar, defaultActiveStates } from "../RichInputToolbar/RichInputToolbar";
import { RichInputAction, RichInputActiveStates, RichInputBaseProps } from "../types";

export const LINE_HEIGHT_MULTIPLIER = 1.5;


/** TextInput for entering rich text. Supports markdown and WYSIWYG */
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
    onFocus,
    onChange,
    onSubmit,
    placeholder = "",
    tabIndex,
    value,
    sxs,
}: RichInputBaseProps) => {
    const { palette, typography } = useTheme();
    const session = useContext(SessionContext);
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

    const closeAssistantDialog = useCallback(() => {
        setAssistantDialogProps(props => ({ ...props, context: undefined, isOpen: false, overrideObject: undefined } as ChatCrudProps));
        PubSub.get().publish("sideMenu", { id: "chat-side-menu", idPrefix: "note", isOpen: false });
    }, []);
    const [assistantDialogProps, setAssistantDialogProps] = useState<ChatCrudProps>({
        context: undefined,
        display: "dialog",
        isCreate: true,
        isOpen: false,
        onCancel: closeAssistantDialog,
        onClose: closeAssistantDialog,
        onCompleted: closeAssistantDialog,
        onDeleted: closeAssistantDialog,
        task: "note",
    });
    const openAssistantDialog = useCallback((highlighted: string) => {
        if (disabled) return;
        const userId = getCurrentUser(session)?.id;
        if (!userId) return;

        // We want to provide the assistant with the most relevant context
        const maxContextLength = 1500;
        let context = highlighted.trim();
        if (context.length > maxContextLength) context = context.substring(0, maxContextLength);
        // If there's not highlighted text, provide the full text if it's not too long
        else if (internalValue.length <= maxContextLength) context = internalValue;
        // Otherwise, provide the last 1500 characters
        else context = internalValue.substring(internalValue.length - maxContextLength, internalValue.length);
        // Put quote block around context
        if (context) context = `"""\n${context}\n"""\n\n`;

        // Now we'll try to find an existing chat with Valyxa for this task
        const existingChatId = getCookieMatchingChat([userId, VALYXA_INFO.id], "note");
        const overrideObject = {
            __typename: "Chat" as const,
            id: existingChatId ?? DUMMY_ID,
            openToAnyoneWithInvite: false,
            invites: [{
                __typename: "ChatInvite" as const,
                id: uuid(),
                status: ChatInviteStatus.Pending,
                user: VALYXA_INFO,
            }],
        } as unknown as ChatShape;

        // Open the assistant dialog
        setAssistantDialogProps(props => ({ ...props, isCreate: !existingChatId, isOpen: true, context, overrideObject } as ChatCrudProps));
    }, [disabled, internalValue, session]);

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

    const [enterWillSubmit, setEnterWillSubmit] = useState<boolean | undefined>(typeof onSubmit === "function" ? true : undefined);
    useEffect(() => {
        if (enterWillSubmit === undefined && typeof onSubmit === "function") {
            setEnterWillSubmit(true);
        } else if (typeof enterWillSubmit === "boolean" && typeof onSubmit !== "function") {
            setEnterWillSubmit(undefined);
        }
    }, [enterWillSubmit, onSubmit]);
    const toggleEnterWillSubmit = useCallback(() => { setEnterWillSubmit(!enterWillSubmit); }, [enterWillSubmit]);

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
        enterWillSubmit,
        error,
        getTaggableItems,
        id,
        maxRows,
        minRows,
        name,
        onActiveStatesChange: handleActiveStatesChange,
        onBlur,
        onFocus,
        onChange: changeInternalValue,
        onSubmit,
        openAssistantDialog,
        placeholder,
        redo,
        setHandleAction: setChildHandleAction,
        tabIndex,
        toggleMarkdown,
        undo,
        value: internalValue,
        sx: sxs?.textArea,
    }), [autoFocus, changeInternalValue, disabled, enterWillSubmit, error, getTaggableItems, id, internalValue, maxRows, minRows, name, onBlur, onFocus, onSubmit, openAssistantDialog, placeholder, redo, setChildHandleAction, sxs?.textArea, tabIndex, toggleMarkdown, undo]);

    return (
        <>
            {/* Assistant dialog for generating text */}
            {!disableAssistant && <ChatCrud {...assistantDialogProps} />}
            <Box
                id={`markdown-input-base-${name}`}
                onMouseDown={handleMouseDown}
                sx={{
                    display: "flex",
                    flexDirection: "column",
                    gap: 0,
                    ...(sxs?.root ?? {}),
                }}
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
                    sx={sxs?.topBar}
                />
                {isMarkdownOn ? <RichInputMarkdown {...viewProps} /> : <RichInputLexical {...viewProps} />}
                {/* Help text, characters remaining indicator, and action buttons */}
                {
                    (helperText || maxChars || (Array.isArray(actionButtons) && actionButtons.length > 0)) && <Box
                        sx={{
                            padding: "2px",
                            display: "flex",
                            flexDirection: isLeftHanded ? "row-reverse" : "row",
                            gap: 1,
                            justifyContent: "space-between",
                            alitnItems: "center",
                            ...sxs?.bottomBar,
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
                            {/* On desktop, allow users to set the behavior of the "Enter" key. 
                                On mobile, the virtual keyboard will (hopefully) display a "Return" 
                                button - so this isn't needed */}
                            {
                                typeof enterWillSubmit === "boolean" && !getDeviceInfo().isMobile &&
                                <IconButton
                                    size="medium"
                                    onClick={toggleEnterWillSubmit}
                                    sx={{
                                        background: palette.primary.dark,
                                        color: palette.primary.contrastText,
                                        borderRadius: 2,
                                    }}
                                >
                                    <Typography variant="body2" mt="auto" mb="auto" sx={{ fontSize: "0.5em" }}>
                                        '{keyComboToString(...(enterWillSubmit ? ["Shift", "Enter"] as const : ["Enter"] as const))}' for new line
                                    </Typography>
                                </IconButton>
                            }
                            {/* Characters remaining indicator */}
                            {
                                !disabled && maxChars !== undefined &&
                                <CharLimitIndicator
                                    chars={internalValue?.length ?? 0}
                                    minCharsToShow={Math.max(0, maxChars - 500)}
                                    maxChars={maxChars}
                                />
                            }
                            {/* Action buttons */}
                            {
                                actionButtons?.map(({ disabled: buttonDisabled, Icon, onClick, tooltip }, index) => (
                                    <Tooltip key={index} title={tooltip} placement="top">
                                        <IconButton
                                            disabled={disabled || buttonDisabled}
                                            size="medium"
                                            onClick={onClick}
                                            sx={{
                                                background: palette.primary.dark,
                                                color: palette.primary.contrastText,
                                                borderRadius: 2,
                                            }}
                                        >
                                            <Icon fill={palette.primary.contrastText} />
                                        </IconButton>
                                    </Tooltip>
                                ))
                            }
                        </Box>
                    </Box>
                }
            </Box>
        </>
    );
};
