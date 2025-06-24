import type { ColorPickerFormInput } from "@vrooli/shared";
import { useField } from "formik";
import { memo, useCallback, useMemo } from "react";
import { type FormInputProps } from "./types.js";

// AI_CHECK: REACT_PERF=1 | LAST: 2025-06-24

const containerStyle = { display: "flex", flexDirection: "column", gap: "8px" } as const;

const getInputStyle = (disabled: boolean) => ({
    width: "64px",
    height: "32px",
    border: "1px solid #ccc",
    borderRadius: "4px",
    cursor: disabled ? "not-allowed" : "pointer",
    opacity: disabled ? 0.5 : 1,
});

/**
 * Color picker form input component
 */
export const FormInputColorPicker = memo<FormInputProps<ColorPickerFormInput>>(function FormInputColorPicker({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}) {
    const props = useMemo(() => fieldData.props, [fieldData.props]);
    const fieldName = useMemo(() =>
        fieldNamePrefix ? `${fieldNamePrefix}-${fieldData.fieldName}` : fieldData.fieldName,
        [fieldNamePrefix, fieldData.fieldName],
    );
    const [field, meta, helpers] = useField(fieldName);

    const handleChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        helpers.setValue(event.target.value);
    }, [helpers]);

    const inputStyle = useMemo(() => getInputStyle(disabled), [disabled]);
    const inputValue = useMemo(() =>
        field.value || props.defaultValue || "#000000",
        [field.value, props.defaultValue],
    );

    return (
        <div style={containerStyle}>
            <input
                id={fieldData.fieldName}
                type="color"
                value={inputValue}
                onChange={handleChange}
                disabled={disabled}
                style={inputStyle}
            />
        </div>
    );
});
