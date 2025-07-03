import { type ColorPickerFormInput } from "@vrooli/shared";
import { useField } from "formik";
import { memo, useCallback, useMemo, useState } from "react";
import { Typography } from "../../text/Typography.js";
import { cn } from "../../../utils/tailwind-theme.js";
import { type FormInputProps } from "./types.js";

// AI_CHECK: REACT_PERF=1 | LAST: 2025-06-24

/**
 * Color picker form input component with enhanced visual design
 */
export const FormInputColorPicker = memo<FormInputProps<ColorPickerFormInput>>(function FormInputColorPicker({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}) {
    const [isHovered, setIsHovered] = useState(false);
    const [isFocused, setIsFocused] = useState(false);
    
    const props = useMemo(() => fieldData.props, [fieldData.props]);
    const fieldName = useMemo(() =>
        fieldNamePrefix ? `${fieldNamePrefix}-${fieldData.fieldName}` : fieldData.fieldName,
        [fieldNamePrefix, fieldData.fieldName],
    );
    const [field, meta, helpers] = useField(fieldName);

    const handleChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        helpers.setValue(event.target.value);
    }, [helpers]);

    const inputValue = useMemo(() =>
        field.value || props.defaultValue || "#000000",
        [field.value, props.defaultValue],
    );

    const containerClasses = cn(
        "tw-flex tw-flex-col tw-gap-2",
    );

    const colorPreviewClasses = cn(
        "tw-relative tw-w-24 tw-h-12 tw-rounded-lg tw-transition-all tw-duration-200",
        "tw-border-2 tw-overflow-hidden tw-cursor-pointer",
        disabled ? [
            "tw-opacity-50 tw-cursor-not-allowed tw-border-gray-300",
        ] : [
            "tw-border-gray-300",
            isHovered && "tw-border-primary tw-shadow-md",
            isFocused && "tw-ring-2 tw-ring-primary tw-ring-opacity-50",
        ],
    );

    const colorSwatchClasses = cn(
        "tw-absolute tw-inset-0 tw-w-full tw-h-full",
        !disabled && "hover:tw-brightness-110",
    );

    const hiddenInputClasses = cn(
        "tw-absolute tw-inset-0 tw-w-full tw-h-full tw-opacity-0",
        disabled ? "tw-cursor-not-allowed" : "tw-cursor-pointer",
    );

    return (
        <div className={containerClasses}>
            {props.label && (
                <Typography variant="body2" color="secondary" className="tw-mb-1">
                    {props.label}
                </Typography>
            )}
            <div className="tw-flex tw-items-center tw-gap-3">
                <div
                    className={colorPreviewClasses}
                    onMouseEnter={() => !disabled && setIsHovered(true)}
                    onMouseLeave={() => setIsHovered(false)}
                >
                    <div 
                        className={colorSwatchClasses}
                        style={{ backgroundColor: inputValue }}
                    />
                    <input
                        id={fieldData.fieldName}
                        type="color"
                        value={inputValue}
                        onChange={handleChange}
                        onFocus={() => setIsFocused(true)}
                        onBlur={() => setIsFocused(false)}
                        disabled={disabled}
                        className={hiddenInputClasses}
                    />
                </div>
                <Typography 
                    variant="body2" 
                    className="tw-font-mono tw-text-sm tw-select-all"
                    color={disabled ? "secondary" : "text"}
                >
                    {inputValue.toUpperCase()}
                </Typography>
            </div>
        </div>
    );
});
