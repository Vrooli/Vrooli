/**
 * Input for entering (and viewing format of) Radio data that 
 * must match a certain schema.
 */
import { AddIcon, DeleteIcon } from "@local/shared";
import { Button, IconButton, Radio, Stack, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { RadioProps } from "forms/types";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { RadioStandardInputProps } from "../types";

/**
 * Create new option
 */
export const emptyRadioOption = (index: number) => ({
    label: `Enter option ${index + 1}`,
    // Random string for value
    value: Math.random().toString(36).substring(2, 15),
});

/**
 * Radio option with delete icon, TextField for label, and Checkbox for default value.
 */
const RadioOption = ({
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
    onChange: (index: number, label: string, defaultValue: boolean) => void,
    onDelete: () => void,
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
            <Tooltip placement="top" title='Should this option be selected by default?'>
                <Radio
                    checked={value}
                    onChange={handleCheckboxChange}
                />
            </Tooltip>{isEditing ? (
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

export const RadioStandardInput = ({
    isEditing,
}: RadioStandardInputProps) => {
    const { t } = useTranslation();

    const [defaultValueField, , defaultValueHelpers] = useField<RadioProps["defaultValue"]>("defaultValue");
    const [optionsField, , optionsHelpers] = useField<RadioProps["options"]>("options");
    // const [rowField, , rowHelpers] = useField<RadioProps['row']>('row');

    const handleOptionAdd = useCallback(() => {
        const newOption = emptyRadioOption((optionsField.value ?? [emptyRadioOption(0)]).length);
        optionsHelpers.setValue([...(optionsField.value ?? [emptyRadioOption(0)]), newOption]);
        // If default value was not set before, set it to the new option
        if (!(typeof defaultValueField.value === "string") || defaultValueField.value.length === 0) {
            defaultValueHelpers.setValue(newOption.value);
        }
    }, [defaultValueField.value, defaultValueHelpers, optionsField.value, optionsHelpers]);
    const handleOptionRemove = useCallback((index: number) => {
        const filtered = (optionsField.value ?? [emptyRadioOption(0)]).filter((_, i) => i !== index);
        // If there will be no options left, add default
        if (filtered.length === 0) {
            filtered.push(emptyRadioOption(0));
        }
        // If defaultValue is not one of the values in filtered, set it to null
        if (!filtered.some(o => o.value === defaultValueField.value)) {
            defaultValueHelpers.setValue(null);
        }
        optionsHelpers.setValue(filtered);
    }, [defaultValueField.value, defaultValueHelpers, optionsField.value, optionsHelpers]);
    const handleOptionChange = useCallback((index: number, label: string, dValue: boolean) => {
        const options = [...(optionsField.value ?? [emptyRadioOption(0)])];
        options[index] = {
            ...options[index],
            label,
        };
        optionsHelpers.setValue(options);
        if (dValue) {
            defaultValueHelpers.setValue(options[index].value);
        } else if (defaultValueField.value === options[index].value) {
            defaultValueHelpers.setValue(null);
        }
    }, [defaultValueField.value, defaultValueHelpers, optionsField.value, optionsHelpers]);

    return (
        <Stack direction="column">
            {(optionsField.value ?? [emptyRadioOption(0)]).map((option, index) => (
                <RadioOption
                    key={index}
                    index={index}
                    isEditing={isEditing}
                    label={option.label}
                    value={defaultValueField.value === option.value}
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
