import { Box, Button, FormControlLabel, Switch, TextField, Typography, useTheme } from "@mui/material";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { useField } from "formik";
import { SwitchFormInput, SwitchFormInputProps } from "forms/types";
import { useEditableLabel } from "hooks/useEditableLabel";
import { useCallback, useMemo, useState } from "react";
import { FormInputProps } from "../types";

const propButtonStyle = { textDecoration: "underline", textTransform: "none" } as const;

export function FormInputSwitch({
    disabled,
    fieldData,
    onConfigUpdate,
}: FormInputProps<SwitchFormInput>) {
    const { palette, typography } = useTheme();

    const props = useMemo(() => fieldData.props, [fieldData.props]);

    const [field, , helpers] = useField(fieldData.fieldName);
    const handleChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        const newValue = event.target.checked;
        if (typeof onConfigUpdate === "function") {
            const newProps = { ...props, defaultValue: newValue };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
        helpers.setValue(newValue);
    }, [onConfigUpdate, props, fieldData, helpers]);

    const [showMore, setShowMore] = useState(false);
    const [customColor, setCustomColor] = useState("");

    const updateProp = useCallback(function updatePropCallback(updatedProps: Partial<SwitchFormInputProps>) {
        if (typeof onConfigUpdate === "function") {
            const newProps = { ...props, ...updatedProps };
            onConfigUpdate({ ...fieldData, props: newProps });
        }
    }, [onConfigUpdate, props, fieldData]);

    function updateFieldData(updatedFieldData: Partial<SwitchFormInput>) {
        if (typeof onConfigUpdate === "function") {
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
        isEditable: typeof onConfigUpdate === "function",
        label: props.label ?? "",
        onUpdate: updateLabel,
    });

    const SwitchElement = useMemo(() => (
        <FormControlLabel
            control={
                <Switch
                    disabled={disabled}
                    checked={typeof onConfigUpdate === "function" ? props.defaultValue ?? false : field.value}
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
                    sx={{ cursor: typeof onConfigUpdate === "function" ? "pointer" : "default" }}
                    variant="body1"
                >
                    {props.label}
                </Typography>}
        />
    ), [disabled, onConfigUpdate, props.defaultValue, props.size, props.color, props.label, field.value, handleChange, fieldData.fieldName, customColor, isEditingLabel, typography, submitLabelChange, handleLabelChange, handleLabelKeyDown, editedLabel, startEditingLabel]);

    if (typeof onConfigUpdate !== "function") {
        return SwitchElement;
    }

    return (
        <div>
            {SwitchElement}
            {typeof onConfigUpdate === "function" && props.defaultValue !== undefined && (
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
                </div>
            )}
        </div>
    );
}
