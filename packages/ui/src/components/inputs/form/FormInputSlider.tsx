import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Slider from "@mui/material/Slider";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { getFormikFieldName, type SliderFormInput, type SliderFormInputProps } from "@vrooli/shared";
import { useField } from "formik";
import { useCallback, useMemo, useState } from "react";
import { IntegerInputBase } from "../IntegerInput/IntegerInput.js";
import { SelectorBase } from "../Selector/Selector.js";
import { TextInput } from "../TextInput/TextInput.js";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle, propButtonWithSectionStyle } from "./styles.js";
import { type FormInputProps } from "./types.js";

export function FormInputSlider({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}: FormInputProps<SliderFormInput>) {
    const { palette } = useTheme();

    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [field, , helpers] = useField(getFormikFieldName(fieldData.fieldName, fieldNamePrefix));
    const handleChange = useCallback((_, value: number | number[]) => {
        const newValue = Array.isArray(value) ? value[0] : value;
        if (isEditing) {
            const newProps = { ...props, defaultValue: newValue };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
        helpers.setValue(newValue);
    }, [isEditing, helpers, props, onConfigUpdate, fieldData]);

    const [showMore, setShowMore] = useState(false);

    function updateProp(updatedProps: Partial<SliderFormInputProps>) {
        if (!isEditing) {
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
        if (!isEditing) {
            return;
        }
        onConfigUpdate({ ...fieldData, ...updatedFieldData });
    }

    const SliderElement = useMemo(() => (
        <Box pl={3} pr={3} pt={4}>
            <Slider
                disabled={disabled}
                key={`field-${fieldData.id}`}
                min={props.min}
                max={props.max}
                name={fieldData.fieldName}
                step={props.step}
                valueLabelDisplay={props.valueLabelDisplay || "auto"}
                value={isEditing ? props.defaultValue ?? props.min : field.value}
                onBlur={field.onBlur}
                onChange={handleChange}
            />
        </Box>
    ), [disabled, fieldData.id, fieldData.fieldName, props.min, props.max, props.step, props.valueLabelDisplay, props.defaultValue, isEditing, field.value, field.onBlur, handleChange]);

    const moreButtonStyle = useMemo(function moreButtonStyleMemo() {
        return propButtonWithSectionStyle(showMore);
    }, [showMore]);

    if (!isEditing) {
        return SliderElement;
    }
    return (
        <div>
            {SliderElement}
            {isEditing && props.defaultValue !== undefined && <Typography
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
            <FormSettingsButtonRow>
                <Button variant="text" sx={propButtonStyle} onClick={() => updateFieldData({ isRequired: !fieldData.isRequired })}>
                    {fieldData.isRequired ? "Optional" : "Required"}
                </Button>
                <Button variant="text" sx={moreButtonStyle} onClick={() => setShowMore(!showMore)}>
                    More
                </Button>
            </FormSettingsButtonRow>
            {showMore && (
                <FormSettingsSection>
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
                </FormSettingsSection>
            )}
        </div>
    );
}
