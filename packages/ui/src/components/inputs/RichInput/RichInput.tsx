import { ChatInviteStatus, DUMMY_ID, getDotNotationValue, noop, setDotNotationValue, uuid } from "@local/shared";
import { Box, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { CharLimitIndicator } from "components/CharLimitIndicator/CharLimitIndicator";
import { SessionContext } from "contexts/SessionContext";
import { useField } from "formik";
import { useIsLeftHanded } from "hooks/useIsLeftHanded";
import { useUndoRedo } from "hooks/useUndoRedo";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { getCookie, getCookieMatchingChat, setCookie } from "utils/cookies";
import { getDeviceInfo, keyComboToString } from "utils/display/device";
import { generateContext } from "utils/display/stringTools";
import { getTranslationData, handleTranslationChange } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { ChatShape } from "utils/shape/models/chat";
import { ChatCrud, VALYXA_INFO } from "views/objects/chat/ChatCrud/ChatCrud";
import { ChatCrudProps } from "views/objects/chat/types";
import { RichInputLexical } from "../RichInputLexical/RichInputLexical";
import { RichInputMarkdown } from "../RichInputMarkdown/RichInputMarkdown";
import { RichInputToolbar, defaultActiveStates } from "../RichInputToolbar/RichInputToolbar";
import { RichInputAction, RichInputActiveStates, RichInputBaseProps, RichInputProps, TranslatedRichInputProps } from "../types";

export const LINE_HEIGHT_MULTIPLIER = 1.5;
const SHOW_CHAR_LIMIT_AT_REMAINING = 500;

/** TextInput for entering rich text. Supports markdown and WYSIWYG */
export function RichInputBase({
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
}: RichInputBaseProps) {
    const { palette } = useTheme();
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
    useEffect(function resetValueEffect() {
        resetInternalValue(value);
    }, [value, resetInternalValue]);

    const [isMarkdownOn, setIsMarkdownOn] = useState(getCookie("ShowMarkdown"));
    const toggleMarkdown = useCallback(() => {
        setIsMarkdownOn(!isMarkdownOn);
        setCookie("ShowMarkdown", !isMarkdownOn);
    }, [isMarkdownOn]);

    const id = useMemo(() => `input-container-${name}`, [name]);

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
    const openAssistantDialog = useCallback((selected: string, fullText: string) => {
        console.log("in openAssistantDialog", id);
        if (disabled) return;
        const userId = getCurrentUser(session)?.id;
        if (!userId) return;

        const context = generateContext(selected, fullText);
        console.log("got context", context);

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
    }, [disabled, id, session]);

    /** Prevents input from losing focus when the toolbar is pressed */
    const handleMouseDown = useCallback((event: React.MouseEvent) => {
        // Check for the toolbar ID at each parent element
        let parent = event.target as HTMLElement | null | undefined;
        do {
            // If the toolbar is clicked, prevent default
            if (parent?.id === `${id}-toolbar`) {
                event.preventDefault();
                event.stopPropagation();
                return;
            }
            parent = parent?.parentElement;
        } while (parent);
    }, [id]);

    // Actions which store and active state for the Toolbar. 
    // This is currently ignored when markdown mode is on, since it's 
    // a bitch to keep track of
    const [activeStates, setActiveStates] = useState<Omit<RichInputActiveStates, "SetValue">>(defaultActiveStates);
    function handleActiveStatesChange(newActiveStates) {
        setActiveStates(newActiveStates);
    }

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
        sxs: {
            inputRoot: sxs?.inputRoot,
            textArea: sxs?.textArea,
        },
    }), [autoFocus, changeInternalValue, disabled, enterWillSubmit, error, getTaggableItems, id, internalValue, maxRows, minRows, name, onBlur, onFocus, onSubmit, openAssistantDialog, placeholder, redo, setChildHandleAction, sxs?.inputRoot, sxs?.textArea, tabIndex, toggleMarkdown, undo]);

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
                    id={`${id}-toolbar`}
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
                            alignItems: "center",
                            ...sxs?.bottomBar,
                        }}
                    >
                        {helperText && <Typography variant="body1" mt="auto" mb="auto" sx={{ color: "red" }}>
                            {typeof helperText === "string" ? helperText : JSON.stringify(helperText)}
                        </Typography>}
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
                                        &apos;{keyComboToString(...(enterWillSubmit ? ["Shift", "Enter"] as const : ["Enter"] as const))}&apos; for new line
                                    </Typography>
                                </IconButton>
                            }
                            {/* Characters remaining indicator */}
                            {
                                !disabled && maxChars !== undefined &&
                                <CharLimitIndicator
                                    chars={internalValue?.length ?? 0}
                                    minCharsToShow={Math.max(0, maxChars - SHOW_CHAR_LIMIT_AT_REMAINING)}
                                    maxChars={maxChars}
                                />
                            }
                            {/* Action buttons */}
                            {
                                actionButtons?.map(({ disabled: buttonDisabled, Icon, onClick, tooltip }) => (
                                    <Tooltip key={tooltip} title={tooltip} placement="top">
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
}

export function RichInput({
    name,
    ...props
}: RichInputProps) {
    const [field, meta, helpers] = useField(name);

    function handleChange(value) {
        helpers.setValue(value);
    }

    return (
        <RichInputBase
            {...props}
            name={name}
            value={field.value}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            onBlur={field.onBlur}
            onChange={handleChange}
        />
    );
}

export function TranslatedRichInput({
    language,
    name,
    ...props
}: TranslatedRichInputProps) {
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
        <RichInputBase
            {...props}
            name={name}
            value={fieldValue || ""}
            error={fieldTouched && Boolean(fieldError)}
            helperText={fieldTouched && fieldError}
            onBlur={handleBlur}
            onChange={handleChange}
        />
    );
}
