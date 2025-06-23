import type { ColorPickerFormInput } from "@vrooli/shared";
import { useField } from "formik";
import { useMemo } from "react";
import { type FormInputProps } from "./types.js";

/**
 * Color picker form input component
 */
export function FormInputColorPicker({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}: FormInputProps<ColorPickerFormInput>) {
    const props = useMemo(() => fieldData.props, [fieldData.props]);
    const fieldName = fieldNamePrefix ? `${fieldNamePrefix}-${fieldData.fieldName}` : fieldData.fieldName;
    const [field, meta, helpers] = useField(fieldName);
    
    const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
        helpers.setValue(event.target.value);
    };

    return (
        <div style={{ display: "flex", flexDirection: "column", gap: "8px" }}>
            <input
                id={fieldData.fieldName}
                type="color"
                value={field.value || props.defaultValue || "#000000"}
                onChange={handleChange}
                disabled={disabled}
                style={{
                    width: "64px",
                    height: "32px",
                    border: "1px solid #ccc",
                    borderRadius: "4px",
                    cursor: disabled ? "not-allowed" : "pointer",
                    opacity: disabled ? 0.5 : 1,
                }}
            />
        </div>
    );
}