import { getDotNotationValue, noop, setDotNotationValue } from "@local/shared";
import { Box } from "@mui/material";
import { useField } from "formik";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useUndoRedo } from "../../../hooks/useUndoRedo.js";
import { DEFAULT_MIN_ROWS } from "../../../utils/consts.js";
import { getTranslationData, handleTranslationChange } from "../../../utils/display/translationTools.js";
import { getCookie, setCookie } from "../../../utils/localStorage.js";
import { AdvancedInputToolbar, defaultActiveStates } from "../AdvancedInput/AdvancedInputToolbar.js";
import { RichInputLexical } from "../RichInputLexical/RichInputLexical.js";
import { RichInputMarkdown } from "../RichInputMarkdown/RichInputMarkdown.js";
import { RichInputAction, RichInputActiveStates, RichInputBaseProps, RichInputProps, TranslatedRichInputProps } from "../types.js";

export const LINE_HEIGHT_MULTIPLIER = 1.5;

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
}: RichInputBaseProps) {

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

    // Actions which store and active state for the Toolbar. 
    // This is currently ignored when markdown mode is on, since it's 
    // a bitch to keep track of
    const [activeStates, setActiveStates] = useState<Omit<RichInputActiveStates, "SetValue">>(defaultActiveStates);
    function handleActiveStatesChange(newActiveStates) {
        setActiveStates(newActiveStates);
    }

    const [enterWillSubmit] = useState<boolean | undefined>(typeof onSubmit === "function" ? true : undefined);

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
        placeholder,
        redo,
        setHandleAction: setChildHandleAction,
        tabIndex,
        toggleMarkdown,
        undo,
        value: internalValue,
    }), [autoFocus, changeInternalValue, disabled, enterWillSubmit, error, getTaggableItems, id, internalValue, maxRows, minRows, name, onBlur, onFocus, onSubmit, placeholder, redo, setChildHandleAction, tabIndex, toggleMarkdown, undo]);

    return (
        <>
            <Box
                id={`markdown-input-base-${name}`}
                display="flex"
                flexDirection="column"
                gap={0}
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
