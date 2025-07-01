import { useField } from "formik";
import React, { forwardRef, useCallback, useState } from "react";
import { getTranslationData, handleTranslationChange } from "../../../utils/display/translationTools.js";
import { cn } from "../../../utils/tailwind-theme.js";
import { InputContainer } from "../InputContainer/index.js";
import type { TextInputBaseProps, TextInputProps, TranslatedTextInputProps } from "./types.js";

const INPUT_BASE_STYLES = cn(
    "tw-w-full",
    "tw-bg-transparent",
    "tw-border-0",
    "tw-outline-none",
    "focus:tw-outline-none",
    "focus:tw-ring-0",
    "tw-appearance-none",
    "tw-text-text-primary placeholder:tw-text-text-secondary",
    "disabled:tw-cursor-not-allowed",
    "tw-m-0",
    "tw-resize-none", // For textarea
);

const INPUT_SIZE_STYLES = {
    sm: "tw-text-sm",
    md: "tw-text-base",
    lg: "tw-text-lg",
};

const TEXTAREA_LINE_HEIGHT_EM = 1.5;

export const TextInputBase = forwardRef<HTMLInputElement | HTMLTextAreaElement, TextInputBaseProps>(
    ({
        variant = "filled",
        size = "md",
        error = false,
        disabled = false,
        fullWidth = false,
        className,
        helperText,
        label,
        isRequired = false,
        enterWillSubmit = false,
        onSubmit,
        onKeyDown,
        onFocus,
        onBlur,
        multiline = false,
        startAdornment,
        endAdornment,
        id,
        minRows,
        maxRows,
        ...props
    }, ref) => {
        const [isFocused, setIsFocused] = useState(false);
        const inputRef = React.useRef<HTMLInputElement | HTMLTextAreaElement>(null);

        // Use provided ref or internal ref
        React.useImperativeHandle(ref, () => inputRef.current as HTMLInputElement | HTMLTextAreaElement);

        const handleKeyDown = useCallback((event: React.KeyboardEvent<HTMLInputElement | HTMLTextAreaElement>) => {
            // Handle enter submit logic
            if (enterWillSubmit && typeof onSubmit === "function") {
                if (event.key === "Enter" && !event.shiftKey) {
                    event.preventDefault();
                    onSubmit();
                }
            }

            // Call any additional onKeyDown handler
            if (onKeyDown) {
                onKeyDown(event);
            }
        }, [enterWillSubmit, onSubmit, onKeyDown]);

        const handleFocus = useCallback((event: React.FocusEvent<HTMLInputElement | HTMLTextAreaElement>) => {
            setIsFocused(true);
            onFocus?.(event);
        }, [onFocus]);

        const handleBlur = useCallback((event: React.FocusEvent<HTMLInputElement | HTMLTextAreaElement>) => {
            setIsFocused(false);
            onBlur?.(event);
        }, [onBlur]);

        const handleContainerClick = useCallback(() => {
            // Focus the input when clicking the container
            if (inputRef.current && !disabled) {
                inputRef.current.focus();
            }
        }, [disabled]);

        const InputElement = multiline ? "textarea" : "input";

        const inputClasses = cn(
            INPUT_BASE_STYLES,
            INPUT_SIZE_STYLES[size],
            className,
        );

        // For multiline without adornments, we don't use InputContainer since adornments aren't supported
        if (multiline && (startAdornment || endAdornment)) {
            console.warn("Adornments are not supported for multiline TextInput");
        }

        // Filter out non-DOM props before spreading to input element
        const {
            variant: _variant,
            size: _size,
            error: _error,
            fullWidth: _fullWidth,
            helperText: _helperText,
            label: _label,
            isRequired: _isRequired,
            enterWillSubmit: _enterWillSubmit,
            onSubmit: _onSubmit,
            multiline: _multiline,
            startAdornment: _startAdornment,
            endAdornment: _endAdornment,
            InputProps: _InputProps,
            codeLanguage: _codeLanguage,
            ...filteredProps
        } = props;

        // Prepare props for the input element, handling textarea-specific props
        const inputProps = multiline ? {
            ...filteredProps,
            ...(minRows !== undefined && { rows: minRows }),
            ...(maxRows !== undefined && { style: { ...filteredProps.style, maxHeight: `${maxRows * TEXTAREA_LINE_HEIGHT_EM}em` } }),
        } : filteredProps;

        return (
            <InputContainer
                variant={variant}
                size={size}
                error={error}
                disabled={disabled}
                fullWidth={fullWidth}
                focused={isFocused}
                onClick={!multiline ? handleContainerClick : undefined}
                label={label}
                isRequired={isRequired}
                helperText={helperText}
                htmlFor={id}
                startAdornment={!multiline ? startAdornment : undefined}
                endAdornment={!multiline ? endAdornment : undefined}
            >
                <InputElement
                    ref={inputRef as any}
                    disabled={disabled}
                    className={inputClasses}
                    onKeyDown={handleKeyDown}
                    onFocus={handleFocus}
                    onBlur={handleBlur}
                    id={id}
                    {...inputProps}
                />
            </InputContainer>
        );
    },
);

TextInputBase.displayName = "TextInputBase";

export const TextInput = forwardRef<HTMLInputElement | HTMLTextAreaElement, TextInputProps>(({
    name,
    validate,
    id,
    ...props
}, ref) => {
    const [field, meta] = useField({ name, validate });

    // Use provided id or fall back to the field name
    const inputId = id || name;

    return (
        <TextInputBase
            {...field}
            {...props}
            id={inputId}
            error={meta.touched && Boolean(meta.error)}
            helperText={meta.touched && meta.error}
            data-testid={props["data-testid"] || "text-input"}
            ref={ref}
        />
    );
});

TextInput.displayName = "TextInput";

export function TranslatedTextInput({
    language,
    name,
    ...props
}: TranslatedTextInputProps) {
    const [field, meta, helpers] = useField("translations");
    const { value, error, touched } = getTranslationData(field, meta, language);

    const handleBlur = useCallback((event: React.FocusEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        field.onBlur(event);
    }, [field]);

    const handleChange = useCallback((event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        handleTranslationChange(field, meta, helpers, event, language);
    }, [field, meta, helpers, language]);

    return (
        <TextInputBase
            {...props}
            id={name}
            name={name}
            value={value?.[name] || ""}
            error={touched?.[name] && Boolean(error?.[name])}
            helperText={touched?.[name] && error?.[name]}
            onBlur={handleBlur}
            onChange={handleChange}
            data-testid={props["data-testid"] || `input-${name}`}
        />
    );
}

export const TextInputFactory = {
    Outline: (props: Omit<TextInputProps, "variant">) => (
        <TextInput variant="outline" {...props} />
    ),
    Filled: (props: Omit<TextInputProps, "variant">) => (
        <TextInput variant="filled" {...props} />
    ),
    Underline: (props: Omit<TextInputProps, "variant">) => (
        <TextInput variant="underline" {...props} />
    ),
} as const;
