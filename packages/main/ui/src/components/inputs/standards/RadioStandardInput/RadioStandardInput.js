import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { AddIcon, DeleteIcon } from "@local/icons";
import { Checkbox, FormControlLabel, IconButton, Stack, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback } from "react";
import { ColorIconButton } from "../../../buttons/ColorIconButton/ColorIconButton";
export const emptyRadioOption = (index) => ({
    label: `Enter option ${index + 1}`,
    value: Math.random().toString(36).substring(2, 15),
});
const RadioOption = ({ index, isEditing, label, value, onChange, onDelete, }) => {
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
export const RadioStandardInput = ({ isEditing, }) => {
    const [defaultValueField, , defaultValueHelpers] = useField("defaultValue");
    const [optionsField, , optionsHelpers] = useField("options");
    const handleOptionAdd = useCallback(() => {
        const newOption = emptyRadioOption((optionsField.value ?? [emptyRadioOption(0)]).length);
        optionsHelpers.setValue([...(optionsField.value ?? [emptyRadioOption(0)]), newOption]);
        if (!(typeof defaultValueField.value === "string") || defaultValueField.value.length === 0) {
            defaultValueHelpers.setValue(newOption.value);
        }
    }, [defaultValueField.value, defaultValueHelpers, optionsField.value, optionsHelpers]);
    const handleOptionRemove = useCallback((index) => {
        const filtered = (optionsField.value ?? [emptyRadioOption(0)]).filter((_, i) => i !== index);
        if (filtered.length === 0) {
            filtered.push(emptyRadioOption(0));
        }
        if (!filtered.some(o => o.value === defaultValueField.value)) {
            defaultValueHelpers.setValue(filtered[0].value);
        }
        optionsHelpers.setValue(filtered);
    }, [defaultValueField.value, defaultValueHelpers, optionsField.value, optionsHelpers]);
    const handleOptionChange = useCallback((index, label, dValue) => {
        const options = [...(optionsField.value ?? [emptyRadioOption(0)])];
        options[index] = {
            ...options[index],
            label,
        };
        optionsHelpers.setValue(options);
        if (dValue) {
            defaultValueHelpers.setValue(options[index].value);
        }
    }, [defaultValueHelpers, optionsField.value, optionsHelpers]);
    return (_jsxs(Stack, { direction: "column", children: [(optionsField.value ?? [emptyRadioOption(0)]).map((option, index) => (_jsx(RadioOption, { index: index, isEditing: isEditing, label: option.label, value: defaultValueField.value === option.value, onChange: handleOptionChange, onDelete: () => handleOptionRemove(index) }, index))), isEditing && (_jsx(Tooltip, { placement: "top", title: "Add option", children: _jsx(ColorIconButton, { color: "inherit", onClick: handleOptionAdd, "aria-label": "Add", background: "#6daf72", sx: {
                        zIndex: 1,
                        width: "fit-content",
                        margin: "5px auto !important",
                        padding: "0",
                        marginBottom: "16px !important",
                        display: "flex",
                        alignItems: "center",
                        borderRadius: "100%",
                    }, children: _jsx(AddIcon, { fill: "white" }) }) }))] }));
};
//# sourceMappingURL=RadioStandardInput.js.map