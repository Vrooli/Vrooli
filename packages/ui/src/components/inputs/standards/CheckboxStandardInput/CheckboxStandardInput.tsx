/**
 * Input for entering (and viewing format of) Checkbox data that 
 * must match a certain schema.
 */
import { CheckboxStandardInputProps } from '../types';
import { checkboxStandardInputForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { useCallback, useEffect } from 'react';
import { Checkbox, FormControlLabel, IconButton, Stack, TextField, Tooltip, Typography } from '@mui/material';
import {
    Add as AddIcon,
    Delete as DeleteIcon,
} from '@mui/icons-material';

/**
 * Create new option
 */
const emptyOption = (index: number) => ({ label: `Enter option ${index + 1}` });

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
    onChange: (index: number, label: string, defaultValue: boolean) => void,
    onDelete: () => void,
}) => {

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
            <IconButton
                onClick={handleDelete}
            >
                <DeleteIcon />
            </IconButton>
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
            <Tooltip placement="top" title='Should this option be checked by default?'>
                <FormControlLabel
                    disabled={!isEditing}
                    label="Default"
                    labelPlacement='start'
                    control={
                        <Checkbox
                            checked={value}
                            onChange={handleCheckboxChange}
                        />
                    }
                />
            </Tooltip>
        </Stack>
    );
};

export const CheckboxStandardInput = ({
    color,
    defaultValue,
    isEditing,
    onPropsChange,
    options,
    row,
}: CheckboxStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: defaultValue ?? [false],
            options: options ?? [emptyOption(0)],
            row: row ?? false,
        },
        validationSchema,
        onSubmit: () => { },
    });

    const handleOptionAdd = useCallback(() => {
        formik.setFieldValue('options', [...formik.values.options, emptyOption(formik.values.options.length)]);
        formik.setFieldValue('defaultValue', [...formik.values.defaultValue, false]);
    }, [formik]);
    const handleOptionRemove = useCallback((index: number) => {
        const filtered = formik.values.options.filter((_, i) => i !== index);
        const defaults = formik.values.defaultValue.filter((_, i) => i !== index)
        // If there will be no options left, add default
        if (filtered.length === 0) {
            filtered.push(emptyOption(0));
            defaults.push(false);
        }
        formik.setFieldValue('options', filtered);
        formik.setFieldValue('defaultValue', defaults);
    }, [formik]);
    const handleOptionChange = useCallback((index: number, label: string, dValue: boolean) => {
        const options = [...formik.values.options];
        options[index] = { label };
        formik.setFieldValue('options', options);
        const defaultValue = [...formik.values.defaultValue];
        defaultValue[index] = dValue;
        formik.setFieldValue('defaultValue', defaultValue);
    }, [formik]);

    useEffect(() => {
        onPropsChange(formik.values);
    }, [formik.values, onPropsChange]);

    return (
        <Stack direction="column">
            {formik.values.options.map((option, index) => (
                <CheckboxOption
                    key={index}
                    index={index}
                    isEditing={isEditing}
                    label={option.label}
                    value={formik.values.defaultValue[index]}
                    onChange={handleOptionChange}
                    onDelete={() => handleOptionRemove(index)}
                />
            ))}
            {isEditing && (
                <Tooltip placement="top" title="Add option">
                    <IconButton
                        color="inherit"
                        onClick={handleOptionAdd}
                        aria-label="Add"
                        sx={{
                            zIndex: 1,
                            width: 'fit-content',
                            margin: '5px auto !important',
                            padding: '0',
                            marginBottom: '16px !important',
                            display: 'flex',
                            alignItems: 'center',
                            backgroundColor: '#6daf72',
                            color: 'white',
                            borderRadius: '100%',
                            '&:hover': {
                                backgroundColor: '#6daf72',
                                filter: `brightness(110%)`,
                                transition: 'filter 0.2s',
                            },
                        }}
                    >
                        <AddIcon />
                    </IconButton>
                </Tooltip>
            )}
        </Stack>
    );
}