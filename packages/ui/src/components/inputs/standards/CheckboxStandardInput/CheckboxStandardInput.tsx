/**
 * Input for entering (and viewing format of) Checkbox data that 
 * must match a certain schema.
 */
import { Button, Checkbox, IconButton, Stack, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { CheckboxProps } from "forms/types";
import { AddIcon, DeleteIcon } from "icons";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { CheckboxStandardInputProps } from "../types";

/**
 * Create new option
 */
export const emptyCheckboxOption = (index: number) => ({ label: `Enter option ${index + 1}` });

/**
 * Checkbox option with delete icon, TextField for label, and Checkbox for default value.
 */
const CheckboxOption = ({
    index,
    isEditing,
    label,
    value,
    onChange,
    onDelete,
}: {
    index: number;
    isEditing: boolean;
    label: string,
    value: any,
    onChange: (index: number, label: string, defaultValue: boolean) => unknown,
    onDelete: () => unknown,
}) => {
    const { palette } = useTheme();

    const handleDelete = useCallback(() => {
        if (!isEditing) return;
        onDelete();
    }, [isEditing, onDelete]);

    const handleLabelChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        if (!isEditing) return;
        onChange(index, e.target.value, value);
    }, [isEditing, index, value, onChange]);

    const handleCheckboxChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        if (!isEditing) return;
        onChange(index, label, e.target.checked);
    }, [isEditing, index, label, onChange]);

    return (
        <Stack direction="row" sx={{ paddingBottom: 2 }}>
            <Tooltip placement="top" title='Should this option be checked by default?'>
                <Checkbox
                    checked={value}
                    onChange={handleCheckboxChange}
                />
            </Tooltip>
            {isEditing ? (
                <TextField
                    label="Label"
                    value={label}
                    fullWidth
                    onChange={handleLabelChange}
                />
            ) : (
                <Typography>{label}</Typography>
            )}
            <IconButton
                onClick={handleDelete}
            >
                <DeleteIcon fill={palette.error.light} />
            </IconButton>
        </Stack>
    );
};

export const CheckboxStandardInput = ({
    isEditing,
}: CheckboxStandardInputProps) => {
    const { t } = useTranslation();

    const [defaultValueField, , defaultValueHelpers] = useField<CheckboxProps["defaultValue"]>("defaultValue");
    const [optionsField, , optionsHelpers] = useField<CheckboxProps["options"]>("options");

    const handleOptionAdd = useCallback(() => {
        optionsHelpers.setValue([...(optionsField.value ?? [emptyCheckboxOption(0)]), emptyCheckboxOption(optionsField.value.length)]);
        defaultValueHelpers.setValue([...(defaultValueField.value ?? [false]), false]);
    }, [defaultValueField.value, defaultValueHelpers, optionsField.value, optionsHelpers]);
    const handleOptionRemove = useCallback((index: number) => {
        const filtered = (optionsField.value ?? [emptyCheckboxOption(0)]).filter((_, i) => i !== index);
        const defaults = defaultValueField.value?.filter((_, i) => i !== index) ?? [];
        // If there will be no options left, add default
        if (filtered.length === 0) {
            filtered.push(emptyCheckboxOption(0));
            defaults.push(false);
        }
        optionsHelpers.setValue(filtered);
        defaultValueHelpers.setValue(defaults);
    }, [defaultValueField.value, defaultValueHelpers, optionsField.value, optionsHelpers]);
    const handleOptionChange = useCallback((index: number, label: string, dValue: boolean) => {
        const options = [...(optionsField.value ?? [emptyCheckboxOption(0)])];
        options[index] = { label };
        optionsHelpers.setValue(options);
        const defaultValue = [...defaultValueField.value ?? [false]];
        defaultValue[index] = dValue;
        defaultValueHelpers.setValue(defaultValue);
    }, [defaultValueField.value, defaultValueHelpers, optionsField.value, optionsHelpers]);

    return (
        <Stack direction="column">
            {(optionsField.value ?? [emptyCheckboxOption(0)]).map((option, index) => (
                <CheckboxOption
                    key={index}
                    index={index}
                    isEditing={isEditing}
                    label={option.label}
                    value={Array.isArray(defaultValueField.value) ? defaultValueField.value[index] : false}
                    onChange={handleOptionChange}
                    onDelete={() => handleOptionRemove(index)}
                />
            ))}
            {isEditing && (
                <Tooltip placement="top" title="Add option">
                    <Button
                        color="secondary"
                        onClick={handleOptionAdd}
                        variant="outlined"
                        startIcon={<AddIcon />}
                        sx={{ margin: 1 }}
                    >{t("Add")}</Button>
                </Tooltip>
            )}
        </Stack>
    );
};
