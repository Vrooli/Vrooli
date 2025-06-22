import React, { forwardRef, useMemo, useCallback } from "react";
import { useField } from "formik";
import { cn } from "../../../utils/tailwind-theme.js";
import { getTranslationData, handleTranslationChange } from "../../../utils/display/translationTools.js";
import { buildTextInputClasses } from "./textInputStyles.js";
import type { TailwindTextInputBaseProps, TailwindTextInputProps, TranslatedTailwindTextInputProps } from "./types.js";

function StyledLabel({ label, isRequired }: { label: string; isRequired?: boolean }) {
    return (
        <span className="tw-block tw-text-sm tw-font-medium tw-text-text-primary tw-mb-1">
            {label}
            {isRequired && (
                <span className="tw-text-danger-main tw-ml-1">*</span>
            )}
        </span>
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
        ...props 
    }, ref) => {
        const hasAdornments = startAdornment || endAdornment;
        
        const inputClasses = useMemo(() => 
            buildTextInputClasses({ 
                variant, 
                size, 
                error, 
                disabled, 
                fullWidth, 
                hasAdornments 
            }),
            [variant, size, error, disabled, fullWidth, hasAdornments]
        );

        const containerClasses = useMemo(() => {
            if (!hasAdornments) return "";
            return cn(
                "tw-flex tw-items-center tw-relative",
                buildTextInputClasses({ variant, size, error, disabled, fullWidth })
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

        const InputElement = multiline ? "textarea" : "input";

        // If we have adornments and it's not multiline, wrap in container
        if (hasAdornments && !multiline) {
            return (
                <div className={fullWidth ? "tw-w-full" : ""}>
                    {label && <StyledLabel label={label} isRequired={isRequired} />}
                    <div className={containerClasses}>
                        {startAdornment && (
                            <div className="tw-flex tw-items-center tw-pl-3 tw-pr-2 tw-text-text-secondary">
                                {startAdornment}
                            </div>
                        )}
                        <InputElement
                            ref={ref as any}
                            disabled={disabled}
                            className={cn(
                                "tw-flex-1 tw-border-0 tw-bg-transparent focus:tw-ring-0 tw-outline-none",
                                startAdornment && "tw-pl-0",
                                endAdornment && "tw-pr-0",
                                className
                            )}
                            onKeyDown={handleKeyDown}
                            {...props}
                        />
                        {endAdornment && (
                            <div className="tw-flex tw-items-center tw-pr-3 tw-pl-2 tw-text-text-secondary">
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
            <div className={fullWidth ? "tw-w-full" : ""}>
                {label && <StyledLabel label={label} isRequired={isRequired} />}
                <InputElement
                    ref={ref as any}
                    disabled={disabled}
                    className={cn(inputClasses, className)}
                    onKeyDown={handleKeyDown}
                    {...props}
                />
                <HelperText helperText={helperText} error={error} />
            </div>
        );
    }
);

TailwindTextInputBase.displayName = "TailwindTextInputBase";

export function TailwindTextInput({
    enterWillSubmit,
    helperText,
    label,
    isRequired,
    onSubmit,
    placeholder,
    ref,
    ...props
}: TailwindTextInputProps) {
    const inputLabelProps = useMemo(function inputLabelPropsMemo() {
        return (label && placeholder) ? { shrink: true } : {};
    }, [label, placeholder]);

    function onKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
        if (enterWillSubmit !== true || typeof onSubmit !== "function") return;
        if (event.key === "Enter" && !event.shiftKey) {
            event.preventDefault();
            onSubmit();
        }
    }

    return (
        <TailwindTextInputBase
            helperText={helperText}
            label={label}
            isRequired={isRequired}
            onKeyDown={onKeyDown}
            placeholder={placeholder}
            ref={ref as React.RefObject<HTMLInputElement>}
            enterWillSubmit={enterWillSubmit}
            onSubmit={onSubmit}
            {...props}
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

    function handleBlur(event: React.FocusEvent<HTMLInputElement | HTMLTextAreaElement>) {
        field.onBlur(event);
    }

    function handleChange(event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) {
        handleTranslationChange(field, meta, helpers, event, language);
    }

    return (
        <TailwindTextInput
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