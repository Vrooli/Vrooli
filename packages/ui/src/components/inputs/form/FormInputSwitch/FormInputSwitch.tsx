import { SwitchFormInput, SwitchFormInputProps, getFormikFieldName } from "@local/shared";
import { Box, Button, FormControlLabel, Switch, TextField, Typography, useTheme } from "@mui/material";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { useField } from "formik";
import { useEditableLabel } from "hooks/useEditableLabel";
import { useCallback, useMemo, useState } from "react";
import { FormSettingsButtonRow, FormSettingsSection, propButtonStyle, propButtonWithSectionStyle } from "../styles";
import { FormInputProps } from "../types";

export function FormInputSwitch({
    disabled,
    fieldData,
    fieldNamePrefix,
    isEditing,
    onConfigUpdate,
}: FormInputProps<SwitchFormInput>) {
    const { palette, typography } = useTheme();

    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [field, meta, helpers] = useField(getFormikFieldName(fieldData.fieldName, fieldNamePrefix));
    const handleChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        const newValue = event.target.checked;
        if (isEditing) {
            const newProps = { ...props, defaultValue: newValue };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
        helpers.setValue(newValue);
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
        label: props.label ?? "",
        onUpdate: updateLabel,
    });

    const SwitchElement = useMemo(() => (
        <FormControlLabel
            control={
                <Switch
                    disabled={disabled}
                    checked={isEditing ? props.defaultValue ?? false : field.value}
                    onChange={handleChange}
                    name={fieldData.fieldName}
                    size={props.size}
                    color={props.color === "custom" ? undefined : props.color as "primary" | "secondary" | "error" | "info" | "success" | "warning" | "default"}
                    sx={props.color === "custom" ? {
                        "& .MuiSwitch-switchBase.Mui-checked": { color: customColor },
                        "& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track": { backgroundColor: customColor },
                    } : undefined}
                />
            }
            label={isEditingLabel ?
                <TextField
                    ref={labelEditRef}
                    InputProps={{ style: (typography["body1"] as object || {}) }}
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
                /> : <Typography
                    onClick={startEditingLabel}
                    sx={{ cursor: isEditing ? "pointer" : "default" }}
                    variant="body1"
                >
                    {props.label}
                </Typography>}
        />
    ), [disabled, isEditing, props.defaultValue, props.size, props.color, props.label, field.value, handleChange, fieldData.fieldName, customColor, isEditingLabel, labelEditRef, typography, submitLabelChange, handleLabelChange, handleLabelKeyDown, editedLabel, startEditingLabel]);

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
                        value={props.label ?? ""}
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
