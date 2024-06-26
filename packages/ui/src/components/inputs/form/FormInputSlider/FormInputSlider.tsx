import { Button, Slider, Typography, useTheme } from "@mui/material";
import { IntegerInputBase } from "components/inputs/IntegerInput/IntegerInput";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { useField } from "formik";
import { SliderFormInput, SliderFormInputProps } from "forms/types";
import { useCallback, useMemo, useState } from "react";
import { FormInputProps } from "../types";

const propButtonStyle = { textDecoration: "underline", textTransform: "none" } as const;

export function FormInputSlider({
    disabled,
    fieldData,
    onConfigUpdate,
}: FormInputProps<SliderFormInput>) {
    const { palette } = useTheme();

    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [field, meta, helpers] = useField(fieldData.fieldName);
    const handleChange = useCallback((_, value: number | number[]) => {
        const newValue = Array.isArray(value) ? value[0] : value;
        if (typeof onConfigUpdate === "function") {
            const newProps = { ...props, defaultValue: newValue };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
        helpers.setValue(newValue);
    }, [onConfigUpdate, props, fieldData, helpers]);

    const [showMore, setShowMore] = useState(false);

    function updateProp(updatedProps: Partial<SliderFormInputProps>) {
        if (typeof onConfigUpdate !== "function") {
            return;
        }

        const newProps = { ...props };
        const updateKeys = Object.keys(updatedProps);

        // Ensure that min, max, and default values are logically consistent
        updateKeys.forEach(key => {
            switch (key) {
                case "min":
                    newProps.min = updatedProps.min;
                    if (newProps.max !== undefined && newProps.min! > newProps.max) {
                        newProps.max = newProps.min;
                    }
                    if (newProps.defaultValue !== undefined && newProps.defaultValue < newProps.min!) {
                        newProps.defaultValue = newProps.min;
                    }
                    break;
                case "max":
                    newProps.max = updatedProps.max;
                    if (newProps.min !== undefined && newProps.max! < newProps.min) {
                        newProps.min = newProps.max;
                    }
                    if (newProps.defaultValue !== undefined && newProps.defaultValue > newProps.max!) {
                        newProps.defaultValue = newProps.max;
                    }
                    break;
                case "defaultValue":
                    newProps.defaultValue = updatedProps.defaultValue;
                    if (newProps.min !== undefined && newProps.defaultValue! < newProps.min) {
                        newProps.min = newProps.defaultValue;
                    }
                    if (newProps.max !== undefined && newProps.defaultValue! > newProps.max) {
                        newProps.max = newProps.defaultValue;
                    }
                    break;
                default:
                    newProps[key] = updatedProps[key];
            }
        });

        onConfigUpdate({ ...fieldData, props: newProps });
    }

    function updateFieldData(updatedFieldData: Partial<SliderFormInput>) {
        if (typeof onConfigUpdate !== "function") {
            return;
        }
        onConfigUpdate({ ...fieldData, ...updatedFieldData });
    }

    const SliderElement = useMemo(() => (
        <Slider
            disabled={disabled}
            key={`field-${fieldData.id}`}
            min={props.min}
            max={props.max}
            name={fieldData.fieldName}
            step={props.step}
            valueLabelDisplay={props.valueLabelDisplay || "auto"}
            value={typeof onConfigUpdate === "function" ? props.defaultValue ?? props.min : field.value}
            onBlur={field.onBlur}
            onChange={handleChange}
        />
    ), [disabled, fieldData.id, fieldData.fieldName, props, field.onBlur, field.value, onConfigUpdate, handleChange]);

    if (typeof onConfigUpdate !== "function") {
        return SliderElement;
    }

    return (
        <div>
            {SliderElement}
            {typeof onConfigUpdate === "function" && props.defaultValue !== undefined && <Typography
                variant="caption"
                style={{
                    marginBottom: "8px",
                    marginLeft: "8px",
                    display: "block",
                    fontStyle: "italic",
                    color: palette.background.textSecondary,
                }}>
                Default value: {props.defaultValue}
            </Typography>}
            <div style={{ display: "flex", flexDirection: "row" }}>
                <Button variant="text" sx={propButtonStyle} onClick={() => updateFieldData({ isRequired: !fieldData.isRequired })}>
                    {fieldData.isRequired ? "Optional" : "Required"}
                </Button>
                <Button variant="text" sx={{ ...propButtonStyle, color: showMore ? "primary.main" : undefined }} onClick={() => setShowMore(!showMore)}>
                    More
                </Button>
            </div>
            {showMore && (
                <div style={{ marginTop: "10px", padding: "8px", display: "flex", flexDirection: "column", gap: "8px" }}>
                    <IntegerInputBase
                        fullWidth
                        label="Min value"
                        name="min"
                        onChange={(value) => updateProp({ min: value })}
                        value={props.min ?? 0}
                        zeroText="No minimum"
                    />
                    <IntegerInputBase
                        fullWidth
                        label="Max value"
                        name="max"
                        onChange={(value) => updateProp({ max: value })}
                        value={props.max ?? 0}
                        zeroText="No maximum"
                    />
                    <IntegerInputBase
                        fullWidth
                        label="Step"
                        name="step"
                        onChange={(value) => updateProp({ step: value })}
                        value={props.step ?? 1}
                        min={1}
                    />
                    <IntegerInputBase
                        fullWidth
                        label="Default value"
                        name="defaultValue"
                        onChange={(value) => updateProp({ defaultValue: value })}
                        value={props.defaultValue ?? 0}
                        min={props.min}
                        max={props.max}
                    />
                    <TextInput
                        fullWidth
                        label="Field name"
                        onChange={(event) => { updateFieldData({ fieldName: event.target.value }); }}
                        value={fieldData.fieldName ?? ""}
                    />
                    <SelectorBase
                        fullWidth
                        getOptionLabel={(option) => option}
                        inputAriaLabel="Value label display"
                        label="Value label display"
                        name="valueLabelDisplay"
                        onChange={(value) => { updateProp({ valueLabelDisplay: value }); }}
                        options={["auto", "on", "off"]}
                        value={props.valueLabelDisplay ?? "off"}
                    />
                </div>
            )}
        </div>
    );
}
