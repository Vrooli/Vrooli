import React, { forwardRef, useMemo, useCallback } from "react";
import { useField } from "formik";
import { cn } from "../../../utils/tailwind-theme.js";
import { getTranslationData, handleTranslationChange } from "../../../utils/display/translationTools.js";
import { buildTextInputClasses, buildContainerClasses } from "./textInputStyles.js";
import type { TailwindTextInputBaseProps, TailwindTextInputProps, TranslatedTailwindTextInputProps } from "./types.js";

function StyledLabel({ label, isRequired, htmlFor }: { label: string; isRequired?: boolean; htmlFor?: string }) {
    return (
        <label htmlFor={htmlFor} className="tw-block tw-text-sm tw-font-medium tw-text-text-primary tw-mb-1">
            {label}
            {isRequired && (
                <span className="tw-text-danger-main tw-ml-1">*</span>
            )}
        </label>
    );
}

function HelperText({ helperText, error }: { helperText?: string | boolean | null | undefined; error?: boolean }) {
    if (!helperText) return null;
    
    const text = typeof helperText === "string" ? helperText : JSON.stringify(helperText);
    const textClass = error 
        ? "tw-text-danger-main" 
        : "tw-text-text-secondary";
    
    return (
        <p className={cn("tw-mt-1 tw-text-xs", textClass)}>
            {text}
        </p>
    );
}

export const TailwindTextInputBase = forwardRef<HTMLInputElement | HTMLTextAreaElement, TailwindTextInputBaseProps>(
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
        multiline = false,
        startAdornment,
        endAdornment,
        id,
        ...props 
    }, ref) => {
        const hasAdornments = startAdornment || endAdornment;
        const inputRef = React.useRef<HTMLInputElement | HTMLTextAreaElement>(null);
        
        // Use provided ref or internal ref
        React.useImperativeHandle(ref, () => inputRef.current as HTMLInputElement | HTMLTextAreaElement);
        
        const inputClasses = useMemo(() => {
            if (hasAdornments) {
                // When we have adornments, the input needs styling but without borders/padding
                return cn(
                    "tw-flex-1",
                    "tw-bg-transparent",
                    "tw-border-0",
                    "tw-outline-none",
                    "focus:tw-outline-none",
                    "focus:tw-ring-0",
                    // Horizontal padding - less on left if there's a start adornment
                    startAdornment ? "tw-pl-0" : (size === "sm" || size === "md" ? "tw-pl-3" : "tw-pl-4"),
                    size === "sm" && "tw-pr-3",
                    size === "md" && "tw-pr-3", 
                    size === "lg" && "tw-pr-4",
                    "tw-appearance-none", // Remove browser defaults
                    "tw-text-text-primary placeholder:tw-text-text-secondary",
                    "disabled:tw-cursor-not-allowed",
                    "tw-m-0", // Remove any default margins
                    // Text sizing with line height to match container
                    size === "sm" && "tw-text-sm tw-leading-8",
                    size === "md" && "tw-text-base tw-leading-10", 
                    size === "lg" && "tw-text-lg tw-leading-12",
                    // Apply variant-specific text color from parent focus state
                    "focus:tw-text-text-primary",
                );
            }
            // Regular input styling when no adornments
            return buildTextInputClasses({ variant, size, error, disabled, fullWidth });
        }, [variant, size, error, disabled, fullWidth, hasAdornments, startAdornment]);

        const containerClasses = useMemo(() => {
            if (!hasAdornments) return "";
            // Container gets the full input styling when adornments are present
            return cn(
                "tw-flex tw-items-center tw-cursor-text",
                buildContainerClasses({ variant, size, error, disabled, fullWidth }),
                disabled && "tw-cursor-not-allowed",
            );
        }, [variant, size, error, disabled, fullWidth, hasAdornments]);

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
                onKeyDown(event as React.KeyboardEvent<HTMLInputElement>);
            }
        }, [enterWillSubmit, onSubmit, onKeyDown]);

        const handleContainerClick = useCallback(() => {
            // Focus the input when clicking anywhere in the container
            if (inputRef.current && !disabled) {
                inputRef.current.focus();
            }
        }, [disabled]);

        const handleContainerKeyDown = useCallback((event: React.KeyboardEvent) => {
            if (event.key === "Enter") {
                handleContainerClick();
            }
        }, [handleContainerClick]);

        const InputElement = multiline ? "textarea" : "input";

        // If we have adornments and it's not multiline, wrap in container
        if (hasAdornments && !multiline) {
            return (
                <div className={fullWidth ? "tw-w-full" : ""} data-testid="tailwind-text-input-base">
                    {label && <StyledLabel label={label} isRequired={isRequired} htmlFor={id} />}
                    <div 
                        className={containerClasses} 
                        onClick={handleContainerClick}
                        onKeyDown={handleContainerKeyDown}
                        role="button"
                        tabIndex={0}
                    >
                        {startAdornment && (
                            <div className="tw-flex tw-items-center tw-justify-center tw-pl-3 tw-text-text-secondary tw-flex-shrink-0">
                                {startAdornment}
                            </div>
                        )}
                        <InputElement
                            ref={inputRef as React.RefObject<HTMLInputElement | HTMLTextAreaElement>["current"]}
                            disabled={disabled}
                            className={cn(inputClasses, className)}
                            onKeyDown={handleKeyDown}
                            id={id}
                            {...props}
                        />
                        {endAdornment && (
                            <div className="tw-flex tw-items-center tw-justify-center tw-pr-3 tw-text-text-secondary tw-flex-shrink-0">
                                {endAdornment}
                            </div>
                        )}
                    </div>
                    <HelperText helperText={helperText} error={error} />
                </div>
            );
        }

        // Standard input without adornments or multiline with adornments (adornments not supported for multiline)
        return (
            <div className={fullWidth ? "tw-w-full" : ""} data-testid="tailwind-text-input-base">
                {label && <StyledLabel label={label} isRequired={isRequired} htmlFor={id} />}
                <InputElement
                    ref={inputRef as React.RefObject<HTMLInputElement | HTMLTextAreaElement>["current"]}
                    disabled={disabled}
                    className={cn(inputClasses, className)}
                    onKeyDown={handleKeyDown}
                    id={id}
                    {...props}
                />
                <HelperText helperText={helperText} error={error} />
            </div>
        );
    },
);

TailwindTextInputBase.displayName = "TailwindTextInputBase";

export function TailwindTextInput({
    name,
    validate,
    enterWillSubmit,
    onSubmit,
    ...props
}: TailwindTextInputProps) {
    const [field, meta] = useField({ name, validate });

    const handleKeyDown = useCallback((event: React.KeyboardEvent<HTMLInputElement>) => {
        // Handle enter submit logic
        if (enterWillSubmit && typeof onSubmit === "function") {
            if (event.key === "Enter" && !event.shiftKey) {
                event.preventDefault();
                onSubmit();
            }
        }
    }, [enterWillSubmit, onSubmit]);

    return (
        <TailwindTextInputBase
            {...props}
            {...field}
            error={meta.touched && Boolean(meta.error)}
            helperText={meta.touched && meta.error}
            onKeyDown={handleKeyDown}
            data-testid="tailwind-text-input"
        />
    );
}

export function TranslatedTailwindTextInput({
    language,
    name,
    ...props
}: TranslatedTailwindTextInputProps) {
    const [field, meta, helpers] = useField("translations");
    const { value, error, touched } = getTranslationData(field, meta, language);

    const handleBlur = useCallback((event: React.FocusEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        field.onBlur(event);
    }, [field]);

    const handleChange = useCallback((event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
        handleTranslationChange(field, meta, helpers, event, language);
    }, [field, meta, helpers, language]);

    return (
        <TailwindTextInputBase
            {...props}
            id={name}
            name={name}
            value={value?.[name] || ""}
            error={touched?.[name] && Boolean(error?.[name])}
            helperText={touched?.[name] && error?.[name]}
            onBlur={handleBlur}
            onChange={handleChange}
        />
    );
}

export const TailwindTextInputFactory = {
    Outline: (props: Omit<TailwindTextInputProps, "variant">) => (
        <TailwindTextInput variant="outline" {...props} />
    ),
    Filled: (props: Omit<TailwindTextInputProps, "variant">) => (
        <TailwindTextInput variant="filled" {...props} />
    ),
    Underline: (props: Omit<TailwindTextInputProps, "variant">) => (
        <TailwindTextInput variant="underline" {...props} />
    ),
} as const;
