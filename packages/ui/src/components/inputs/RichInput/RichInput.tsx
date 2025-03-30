import { TaskContextInfo, getDotNotationValue, noop, setDotNotationValue } from "@local/shared";
import { Box, Chip, styled, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { ActiveChatContext } from "../../../contexts/activeChat.js";
import { SessionContext } from "../../../contexts/session.js";
import useDraggableScroll from "../../../hooks/gestures.js";
import { generateContextLabel } from "../../../hooks/tasks.js";
import { useUndoRedo } from "../../../hooks/useUndoRedo.js";
import { IconCommon } from "../../../icons/Icons.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { DEFAULT_MIN_ROWS } from "../../../utils/consts.js";
import { generateContext } from "../../../utils/display/stringTools.js";
import { getTranslationData, handleTranslationChange } from "../../../utils/display/translationTools.js";
import { getCookie, setCookie } from "../../../utils/localStorage.js";
import { PubSub } from "../../../utils/pubsub.js";
import { AdvancedInputToolbar, defaultActiveStates } from "../AdvancedInput/AdvancedInputToolbar.js";
import { RichInputLexical } from "../RichInputLexical/RichInputLexical.js";
import { RichInputMarkdown } from "../RichInputMarkdown/RichInputMarkdown.js";
import { RichInputAction, RichInputActiveStates, RichInputBaseProps, RichInputProps, TranslatedRichInputProps } from "../types.js";

export const LINE_HEIGHT_MULTIPLIER = 1.5;

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
                        deleteIcon={<IconCommon
                            decorative
                            fill={palette.primary.contrastText}
                            name="Close"
                            size={15}
                        />}
                    />
                );
            })}
        </ContextsRowOuter>
    );
}

/** TextInput for entering rich text. Supports markdown and WYSIWYG */
export function RichInputBase({
    autoFocus = false,
    disabled = false,
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
    const session = useContext(SessionContext);
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
        //TODO
        // PubSub.get().publish("menu", { id: SITE_NAVIGATOR_MENU_ID, isOpen: true, data: { tab: "Chat" } });
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

    // Actions which store and active state for the Toolbar. 
    // This is currently ignored when markdown mode is on, since it's 
    // a bitch to keep track of
    const [activeStates, setActiveStates] = useState<Omit<RichInputActiveStates, "SetValue">>(defaultActiveStates);
    function handleActiveStatesChange(newActiveStates) {
        setActiveStates(newActiveStates);
    }

    const [enterWillSubmit, setEnterWillSubmit] = useState<boolean | undefined>(typeof onSubmit === "function" ? true : undefined);

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

    return (
        <>
            <Box
                id={`markdown-input-base-${name}`}
                sx={rootStyle}
            >
                <AdvancedInputToolbar
                    activeStates={activeStates}
                    canRedo={canRedo}
                    canUndo={canUndo}
                    disabled={disabled}
                    handleAction={handleAction}
                    handleActiveStatesChange={handleActiveStatesChange}
                    isMarkdownOn={isMarkdownOn}
                />
                {taskInfo !== null && taskInfo !== undefined && Boolean(taskInfo.contexts[taskInfo.activeTask.taskId]) && taskInfo.contexts[taskInfo.activeTask.taskId].length > 0 && <ContextsRow
                    activeContexts={taskInfo.contexts[taskInfo.activeTask.taskId]}
                    chatId={chat?.id}
                />}
                {isMarkdownOn ? <RichInputMarkdown {...viewProps} /> : <RichInputLexical {...viewProps} />}
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
