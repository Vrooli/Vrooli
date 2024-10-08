import { TaskContextInfo, getDotNotationValue, noop, setDotNotationValue } from "@local/shared";
import { Box, Chip, IconButton, Tooltip, Typography, styled, useTheme } from "@mui/material";
import { CharLimitIndicator } from "components/CharLimitIndicator/CharLimitIndicator";
import { ActiveChatContext, SessionContext } from "contexts";
import { useField } from "formik";
import useDraggableScroll from "hooks/gestures";
import { useIsLeftHanded } from "hooks/subscriptions";
import { generateContextLabel } from "hooks/tasks";
import { useUndoRedo } from "hooks/useUndoRedo";
import { CloseIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { DEFAULT_MIN_ROWS } from "utils/consts";
import { getDeviceInfo, keyComboToString } from "utils/display/device";
import { generateContext } from "utils/display/stringTools";
import { getTranslationData, handleTranslationChange } from "utils/display/translationTools";
import { getCookie, setCookie } from "utils/localStorage";
import { CHAT_SIDE_MENU_ID, PubSub } from "utils/pubsub";
import { RichInputLexical } from "../RichInputLexical/RichInputLexical";
import { RichInputMarkdown } from "../RichInputMarkdown/RichInputMarkdown";
import { RichInputToolbar, defaultActiveStates } from "../RichInputToolbar/RichInputToolbar";
import { RichInputAction, RichInputActiveStates, RichInputBaseProps, RichInputProps, TranslatedRichInputProps } from "../types";

export const LINE_HEIGHT_MULTIPLIER = 1.5;
const SHOW_CHAR_LIMIT_AT_REMAINING = 500;

const ContextsRowOuter = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    padding: theme.spacing(0.5),
    gap: theme.spacing(1),
    background: theme.palette.background.paper,
    overflowX: "auto",
    "&::-webkit-scrollbar": {
        display: "none",
    },
}));

const ContextChip = styled(Chip)(({ theme }) => ({
    fontSize: "0.75rem",
    padding: theme.spacing(0.5),
    borderRadius: theme.shape.borderRadius * 2,
}));

interface ContextsRowProps {
    activeContexts: TaskContextInfo[];
    chatId: string | null | undefined;
}

function ContextsRow({
    activeContexts,
    chatId,
}: ContextsRowProps) {
    const { palette } = useTheme();
    const ref = useRef<HTMLDivElement>(null);
    const { onMouseDown } = useDraggableScroll({ ref, options: { direction: "horizontal" } });

    function handleRemove(contextId: string) {
        if (!chatId) return;
        PubSub.get().publish("chatTask", {
            chatId,
            contexts: {
                remove: [{
                    __type: "contextId",
                    data: contextId,
                }],
            },
        });
    }

    return (
        <ContextsRowOuter
            onMouseDown={onMouseDown}
            ref={ref}
        >
            {activeContexts.map(context => {
                function onDelete() {
                    handleRemove(context.id);
                }
                return (
                    <ContextChip
                        key={context.id}
                        label={context.label}
                        onDelete={onDelete}
                        deleteIcon={<CloseIcon width={15} height={15} fill={palette.primary.contrastText} />}
                    />
                );
            })}
        </ContextsRowOuter>
    );
}

const ActionButton = styled(IconButton)(({ theme }) => ({
    background: theme.palette.primary.dark,
    color: theme.palette.primary.contrastText,
    borderRadius: theme.spacing(2),
}));

const helperTextStyle = { color: "red" } as const;
const newLineTextStyle = { fontSize: "0.5em" } as const;

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
    minRows = DEFAULT_MIN_ROWS,
    name,
    onBlur,
    onFocus,
    onChange,
    onSubmit,
    placeholder = "",
    tabIndex,
    taskInfo,
    value,
    sxs,
}: RichInputBaseProps) {
    const { palette } = useTheme();
    const session = useContext(SessionContext);
    const isLeftHanded = useIsLeftHanded();
    const { chat } = useContext(ActiveChatContext);

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

    const openAssistantDialog = useCallback((selected: string, fullText: string) => {
        if (disabled) return;
        const userId = getCurrentUser(session)?.id;
        if (!userId) return;
        const chatId = chat?.id;
        if (!chatId) return;

        const contextValue = generateContext(selected, fullText);

        // Open the side chat and provide it context
        PubSub.get().publish("sideMenu", { id: CHAT_SIDE_MENU_ID, isOpen: true, data: { tab: "Chat" } });
        const context = {
            id: `rich-${name}`,
            label: generateContextLabel(contextValue),
            data: contextValue,
            template: "Text:\n\"\"\"\n<DATA>\n\"\"\"",
            templateVariables: { data: "<DATA>" },
        };
        PubSub.get().publish("chatTask", {
            chatId,
            contexts: {
                add: {
                    behavior: "replace",
                    connect: {
                        __type: "contextId",
                        data: context.id,
                    },
                    value: [context],
                },
            },
        });
    }, [chat?.id, disabled, name, session]);

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

    const rootStyle = useMemo(function rootStyleMemo() {
        return {
            display: "flex",
            flexDirection: "column",
            gap: 0,
            ...sxs?.root,
        } as const;
    }, [sxs?.root]);

    const bottomBarStyle = useMemo(function bottomBarStyleMemo() {
        return {
            padding: "2px",
            display: "flex",
            flexDirection: isLeftHanded ? "row-reverse" : "row",
            gap: 1,
            justifyContent: "space-between",
            alignItems: "center",
            ...sxs?.bottomBar,
        } as const;
    }, [isLeftHanded, sxs?.bottomBar]);

    const actionsBoxStyle = useMemo(function actionsBoxStyleMemo() {
        return {
            display: "flex",
            gap: 2,
            ...(isLeftHanded ?
                { marginRight: "auto", flexDirection: "row-reverse" } :
                { marginLeft: "auto", flexDirection: "row" }),
        } as const;
    }, [isLeftHanded]);

    return (
        <>
            <Box
                id={`markdown-input-base-${name}`}
                onMouseDown={handleMouseDown}
                sx={rootStyle}
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
                {taskInfo !== null && taskInfo !== undefined && Boolean(taskInfo.contexts[taskInfo.activeTask.taskId]) && taskInfo.contexts[taskInfo.activeTask.taskId].length > 0 && <ContextsRow
                    activeContexts={taskInfo.contexts[taskInfo.activeTask.taskId]}
                    chatId={chat?.id}
                />}
                {isMarkdownOn ? <RichInputMarkdown {...viewProps} /> : <RichInputLexical {...viewProps} />}
                {/* Help text, characters remaining indicator, and action buttons */}
                {
                    (helperText || maxChars || (Array.isArray(actionButtons) && actionButtons.length > 0)) && <Box sx={bottomBarStyle}>
                        {helperText && <Typography variant="body1" mt="auto" mb="auto" sx={helperTextStyle}>
                            {typeof helperText === "string" ? helperText : JSON.stringify(helperText)}
                        </Typography>}
                        <Box sx={actionsBoxStyle}>
                            {/* On desktop, allow users to set the behavior of the "Enter" key. 
                                On mobile, the virtual keyboard will (hopefully) display a "Return" 
                                button - so this isn't needed */}
                            {
                                typeof enterWillSubmit === "boolean" && !getDeviceInfo().isMobile &&
                                <ActionButton
                                    size="medium"
                                    onClick={toggleEnterWillSubmit}
                                >
                                    <Typography variant="body2" mt="auto" mb="auto" sx={newLineTextStyle}>
                                        &apos;{keyComboToString(...(enterWillSubmit ? ["Shift", "Enter"] as const : ["Enter"] as const))}&apos; for new line
                                    </Typography>
                                </ActionButton>
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
                                        <ActionButton
                                            disabled={disabled || buttonDisabled}
                                            size="medium"
                                            onClick={onClick}
                                        >
                                            <Icon fill={palette.primary.contrastText} />
                                        </ActionButton>
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
