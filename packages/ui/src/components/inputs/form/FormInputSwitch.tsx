import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import FormControlLabel from "@mui/material/FormControlLabel";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { Switch } from "../Switch/Switch.js";
import { getFormikFieldName, type SwitchFormInput, type SwitchFormInputProps } from "@vrooli/shared";
import { useField } from "formik";
import React, { useCallback, useMemo, useState } from "react";
import { useEditableLabel } from "../../../hooks/useEditableLabel.js";
import { SelectorBase } from "../Selector/Selector.js";
import { TextInput } from "../TextInput/TextInput.js";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle, propButtonWithSectionStyle } from "./styles.js";
import { type FormInputProps } from "./types.js";

export function FormInputSwitch({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}: FormInputProps<SwitchFormInput>) {
    const { palette, typography } = useTheme();

    const props = useMemo(() => {
        const rawProps = fieldData.props || {};
        // Ensure label is a string or undefined, not an object
        if (rawProps.label && typeof rawProps.label === 'object' && !React.isValidElement(rawProps.label)) {
            return { ...rawProps, label: fieldData.label || "" };
        }
        return rawProps;
    }, [fieldData.props, fieldData.label]);

    const [field, meta, helpers] = useField(getFormikFieldName(fieldData.fieldName, fieldNamePrefix));
    const handleChange = useCallback((checked: boolean) => {
        if (isEditing) {
            const newProps = { ...props, defaultValue: checked };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
        helpers.setValue(checked);
    }, [isEditing, helpers, props, onConfigUpdate, fieldData]);

    const [showMore, setShowMore] = useState(false);
    const [customColor, setCustomColor] = useState("");

    const updateProp = useCallback(function updatePropCallback(updatedProps: Partial<SwitchFormInputProps>) {
        if (isEditing) {
            const newProps = { ...props, ...updatedProps };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [isEditing, props, onConfigUpdate, fieldData]);

    function updateFieldData(updatedFieldData: Partial<SwitchFormInput>) {
        if (isEditing) {
            onConfigUpdate({ ...fieldData, ...updatedFieldData });
        }
    }

    const updateLabel = useCallback((updatedLabel: string) => { updateProp({ label: updatedLabel }); }, [updateProp]);
    const {
        editedLabel,
        handleLabelChange,
        handleLabelKeyDown,
        isEditingLabel,
        labelEditRef,
        startEditingLabel,
        submitLabelChange,
    } = useEditableLabel({
        isEditable: isEditing,
        isMultiline: false,
        label: props.label ?? fieldData.label ?? "",
        onUpdate: updateLabel,
    });

    const labelText = props.label || fieldData.label || "";
    const SwitchElement = useMemo(() => (
        <Switch
            disabled={disabled}
            checked={isEditing ? props.defaultValue ?? false : field.value}
            onChange={handleChange}
            label={isEditingLabel ?
                <TextField
                    ref={labelEditRef}
                    InputProps={{ style: typography["body1"] as React.CSSProperties }}
                    onBlur={submitLabelChange}
                    onChange={handleLabelChange}
                    onKeyDown={handleLabelKeyDown}
                    size="small"
                    value={editedLabel}
                    sx={{
                        flexGrow: 1,
                        width: "auto",
                        "& .MuiInputBase-input": {
                            padding: 0,
                        },
                    }}
                /> : labelText ? <Typography
                    onClick={startEditingLabel}
                    sx={{ cursor: isEditing ? "pointer" : "default" }}
                    variant="body1"
                >
                    {labelText}
                </Typography> : undefined}
            size={props.size === "small" ? "sm" : props.size === "medium" ? "md" : "md"}
            variant={props.color === "custom" ? "custom" : "default"}
            color={props.color === "custom" ? customColor : undefined}
        />
    ), [disabled, isEditing, props.defaultValue, props.size, props.color, labelText, field.value, handleChange, customColor, isEditingLabel, labelEditRef, typography, submitLabelChange, handleLabelChange, handleLabelKeyDown, editedLabel, startEditingLabel]);

    const moreButtonStyle = useMemo(function moreButtonStyleMemo() {
        return propButtonWithSectionStyle(showMore);
    }, [showMore]);

    if (!isEditing) {
        return SwitchElement;
    }
    return (
        <div>
            {SwitchElement}
            {isEditing && props.defaultValue !== undefined && (
                <Typography
                    variant="caption"
                    style={{
                        marginBottom: "8px",
                        marginLeft: "8px",
                        display: "block",
                        fontStyle: "italic",
                        color: palette.background.textSecondary,
                    }}
                >
                    Defaults to {props.defaultValue ? "On" : "Off"}
                </Typography>
            )}
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
                    <TextInput
                        fullWidth
                        label="Label"
                        onChange={(event) => { updateProp({ label: event.target.value }); }}
                        value={props.label ?? fieldData.label ?? ""}
                    />
                    <SelectorBase
                        fullWidth
                        getOptionLabel={(option) => option}
                        inputAriaLabel="Size"
                        label="Size"
                        name="size"
                        onChange={(value) => { updateProp({ size: value }); }}
                        options={["small", "medium"]}
                        value={props.size ?? "medium"}
                    />
                    <Box sx={{ display: "flex", alignItems: "center" }}>
                        <SelectorBase
                            fullWidth
                            getOptionLabel={(option) => option}
                            inputAriaLabel="Color"
                            label="Color"
                            name="color"
                            onChange={(value) => { updateProp({ color: value }); if (value !== "custom") setCustomColor(""); }}
                            options={["primary", "secondary", "error", "info", "success", "warning", "default", "custom"]}
                            value={props.color ?? "primary"}
                        />
                        {props.color === "custom" && (
                            <input
                                type="color"
                                onChange={(event) => setCustomColor(event.target.value)}
                                value={customColor}
                                style={{
                                    height: "64px",
                                    width: "40px",
                                    background: "transparent",
                                    marginTop: "-4px",
                                    marginBottom: "-4px",
                                    border: "none",
                                    cursor: "pointer",
                                    WebkitAppearance: "none",
                                    outline: "none",
                                    padding: "0",
                                }}
                            />
                        )}
                    </Box>
                    <TextInput
                        fullWidth
                        label="Field name"
                        onChange={(event) => { updateFieldData({ fieldName: event.target.value }); }}
                        value={fieldData.fieldName ?? ""}
                    />
                </FormSettingsSection>
            )}
        </div>
    );
}
