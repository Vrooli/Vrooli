import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { AddIcon, DeleteIcon } from "@local/icons";
import { Checkbox, FormControlLabel, IconButton, Stack, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback } from "react";
import { ColorIconButton } from "../../../buttons/ColorIconButton/ColorIconButton";
export const emptyCheckboxOption = (index) => ({ label: `Enter option ${index + 1}` });
const CheckboxOption = ({ index, isEditing, label, value, onChange, onDelete, }) => {
    const { palette } = useTheme();
    const handleDelete = useCallback(() => {
        if (!isEditing)
            return;
        onDelete();
    }, [isEditing, onDelete]);
    const handleLabelChange = useCallback((e) => {
        if (!isEditing)
            return;
        onChange(index, e.target.value, value);
    }, [isEditing, index, value, onChange]);
    const handleCheckboxChange = useCallback((e) => {
        if (!isEditing)
            return;
        onChange(index, label, e.target.checked);
    }, [isEditing, index, label, onChange]);
    return (_jsxs(Stack, { direction: "row", sx: { paddingBottom: 2 }, children: [_jsx(IconButton, { onClick: handleDelete, children: _jsx(DeleteIcon, { fill: palette.error.light }) }), isEditing ? (_jsx(TextField, { label: "Label", value: label, fullWidth: true, onChange: handleLabelChange })) : (_jsx(Typography, { children: label })), _jsx(Tooltip, { placement: "top", title: 'Should this option be checked by default?', children: _jsx(FormControlLabel, { disabled: !isEditing, label: "Default", labelPlacement: 'start', control: _jsx(Checkbox, { checked: value, onChange: handleCheckboxChange }), sx: {
                        ".MuiFormControlLabel-label": {
                            display: { xs: "none", sm: "block" },
                        },
                    } }) })] }));
};
export const CheckboxStandardInput = ({ isEditing, }) => {
    const [defaultValueField, , defaultValueHelpers] = useField("defaultValue");
    const [optionsField, , optionsHelpers] = useField("options");
    const handleOptionAdd = useCallback(() => {
        optionsHelpers.setValue([...(optionsField.value ?? [emptyCheckboxOption(0)]), emptyCheckboxOption(optionsField.value.length)]);
        defaultValueHelpers.setValue([...(defaultValueField.value ?? [false]), false]);
    }, [defaultValueField.value, defaultValueHelpers, optionsField.value, optionsHelpers]);
    const handleOptionRemove = useCallback((index) => {
        const filtered = (optionsField.value ?? [emptyCheckboxOption(0)]).filter((_, i) => i !== index);
        const defaults = defaultValueField.value?.filter((_, i) => i !== index) ?? [];
        if (filtered.length === 0) {
            filtered.push(emptyCheckboxOption(0));
            defaults.push(false);
        }
        optionsHelpers.setValue(filtered);
        defaultValueHelpers.setValue(defaults);
    }, [defaultValueField.value, defaultValueHelpers, optionsField.value, optionsHelpers]);
    const handleOptionChange = useCallback((index, label, dValue) => {
        const options = [...(optionsField.value ?? [emptyCheckboxOption(0)])];
        options[index] = { label };
        optionsHelpers.setValue(options);
        const defaultValue = [...defaultValueField.value ?? [false]];
        defaultValue[index] = dValue;
        defaultValueHelpers.setValue(defaultValue);
    }, [defaultValueField.value, defaultValueHelpers, optionsField.value, optionsHelpers]);
    return (_jsxs(Stack, { direction: "column", children: [(optionsField.value ?? [emptyCheckboxOption(0)]).map((option, index) => (_jsx(CheckboxOption, { index: index, isEditing: isEditing, label: option.label, value: Array.isArray(defaultValueField.value) ? defaultValueField.value[index] : false, onChange: handleOptionChange, onDelete: () => handleOptionRemove(index) }, index))), isEditing && (_jsx(Tooltip, { placement: "top", title: "Add option", children: _jsx(ColorIconButton, { color: "inherit", onClick: handleOptionAdd, "aria-label": "Add", background: "#6daf72", sx: {
                        zIndex: 1,
                        width: "fit-content",
                        margin: "5px auto !important",
                        padding: "0",
                        marginBottom: "16px !important",
                        display: "flex",
                        alignItems: "center",
                        borderRadius: "100%",
                    }, children: _jsx(AddIcon, { fill: 'white' }) }) }))] }));
};
//# sourceMappingURL=CheckboxStandardInput.js.map