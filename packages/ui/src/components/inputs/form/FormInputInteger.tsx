import { IntegerFormInput, IntegerFormInputProps, getFormikFieldName } from "@local/shared";
import { Button, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { IntegerInputBase } from "../IntegerInput/IntegerInput.js";
import { TextInput } from "../TextInput/TextInput.js";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle, propButtonWithSectionStyle } from "./styles.js";
import { FormInputProps } from "./types.js";

export function FormInputInteger({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}: FormInputProps<IntegerFormInput>) {
    const { t } = useTranslation();
    const { palette } = useTheme();

    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [field, meta, helpers] = useField(getFormikFieldName(fieldData.fieldName, fieldNamePrefix));
    const handleChange = useCallback((value: number) => {
        if (isEditing) {
            const newProps = { ...props, defaultValue: value };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
        helpers.setValue(value);
    }, [isEditing, helpers, props, onConfigUpdate, fieldData]);

    const [showMore, setShowMore] = useState(false);

    function updateProp(updatedProps: Partial<IntegerFormInputProps>) {
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

    function updateFieldData(updatedFieldData: Partial<IntegerFormInput>) {
        if (!isEditing) {
            return;
        }
        onConfigUpdate({ ...fieldData, ...updatedFieldData });
    }

    const InputElement = useMemo(() => (
        <IntegerInputBase
            disabled={disabled}
            error={meta.touched && !!meta.error}
            helperText={meta.touched && meta.error}
            name={fieldData.fieldName}
            onChange={handleChange}
            value={isEditing ? props.defaultValue ?? 0 : field.value}
            min={props.min}
            max={props.max}
            step={props.step}
            allowDecimal={props.allowDecimal}
            zeroText={props.zeroText}
            fullWidth
        />
    ), [disabled, meta.touched, meta.error, fieldData.fieldName, handleChange, isEditing, props.defaultValue, props.min, props.max, props.step, props.allowDecimal, props.zeroText, field.value]);

    const moreButtonStyle = useMemo(function moreButtonStyleMemo() {
        return propButtonWithSectionStyle(showMore);
    }, [showMore]);

    if (!isEditing) {
        return InputElement;
    }
    return (
        <div>
            {InputElement}
            {isEditing && props.defaultValue !== 0 && <Typography
                variant="caption"
                style={{
                    marginBottom: "8px",
                    marginTop: "8px",
                    marginLeft: "8px",
                    display: "block",
                    fontStyle: "italic",
                    color: palette.background.textSecondary,
                }}>
                {props.defaultValue} is the default value
            </Typography>}
            <FormSettingsButtonRow>
                <Button variant="text" sx={propButtonStyle} onClick={() => updateFieldData({ isRequired: !fieldData.isRequired })}>
                    {fieldData.isRequired ? "Optional" : "Required"}
                </Button>
                <Button variant="text" sx={propButtonStyle} onClick={() => updateProp({ allowDecimal: !props.allowDecimal })}>
                    {props.allowDecimal ? "Integer" : "Decimal"}
                </Button>
                <Button variant="text" sx={moreButtonStyle} onClick={() => setShowMore(!showMore)}>
                    More
                </Button>
            </FormSettingsButtonRow>
            {showMore && (
                <FormSettingsSection>
                    <IntegerInputBase
                        fullWidth
                        label={"Min value"}
                        name="min"
                        onChange={(value) => updateProp({ min: value })}
                        value={props.min ?? 0}
                        zeroText="No minimum"
                    />
                    <IntegerInputBase
                        fullWidth
                        label={"Max value"}
                        name="max"
                        onChange={(value) => updateProp({ max: value })}
                        value={props.max ?? 0}
                        zeroText="No maximum"
                    />
                    <IntegerInputBase
                        fullWidth
                        label={t("Step")}
                        name="step"
                        onChange={(value) => updateProp({ step: value })}
                        value={props.step ?? 1}
                        min={1}
                    />
                    <IntegerInputBase
                        fullWidth
                        label={"Default value"}
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
                    <TextInput
                        fullWidth
                        label="Zero text"
                        onChange={(event) => { updateProp({ zeroText: event.target.value }); }}
                        value={props.zeroText ?? ""}
                    />
                </FormSettingsSection>
            )}
        </div>
    );
}
