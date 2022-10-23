/**
 * Input for entering (and viewing format of) Radio data that 
 * must match a certain schema.
 */
import { RadioStandardInputProps } from '../types';
import { radioStandardInputForm as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { useCallback, useEffect } from 'react';
import { Checkbox, FormControlLabel, IconButton, Stack, TextField, Tooltip, Typography } from '@mui/material';
import { AddIcon, DeleteIcon } from '@shared/icons';
import { ColorIconButton } from 'components/buttons';

/**
 * Create new option
 */
 const emptyOption = (index: number) => ({
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

export const RadioStandardInput = ({
    defaultValue,
    isEditing,
    onPropsChange,
    options,
    row,
}: RadioStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: defaultValue ?? '',
            options: options ?? [emptyOption(0)],
            row: row ?? false,
        },
        validationSchema,
        onSubmit: () => { onPropsChange(formik.values) },
    });

    const handleOptionAdd = useCallback(() => {
        const newOption = emptyOption(formik.values.options.length);
        formik.setFieldValue('options', [...formik.values.options, newOption]);
        // If default value was not set before, set it to the new option
        if (!(typeof formik.values.defaultValue === 'string') || formik.values.defaultValue.length === 0) {
            formik.setFieldValue('defaultValue', newOption.value);
        }
    }, [formik]);
    const handleOptionRemove = useCallback((index: number) => {
        const filtered = formik.values.options.filter((_, i) => i !== index);
        // If there will be no options left, add default
        if (filtered.length === 0) {
            filtered.push(emptyOption(0));
        }
        // If defaultValue is not one of the values in filtered, set it to the first value
        if (!filtered.some(o => o.value === formik.values.defaultValue)) {
            formik.setFieldValue('defaultValue', filtered[0].value);
        }
        formik.setFieldValue('options', filtered);
    }, [formik]);
    const handleOptionChange = useCallback((index: number, label: string, dValue: boolean) => {
        const options = [...formik.values.options];
        options[index] = {
            ...options[index],
            label,
        };
        formik.setFieldValue('options', options);
        if (dValue) {
            formik.setFieldValue('defaultValue', options[index].value);
        }
    }, [formik]);

    useEffect(() => {
        // Call onPropsChange callback
        // If default value is not set, try setting it to the first available option
        const firstValue = formik.values.options.length > 0 ? formik.values.options[0].value : undefined;
        onPropsChange({
            ...formik.values,
            defaultValue: formik.values.defaultValue ?? firstValue,
        });
    }, [formik, onPropsChange]);

    return (
        <Stack direction="column">
            {formik.values.options.map((option, index) => (
                <RadioOption
                    key={index}
                    index={index}
                    isEditing={isEditing}
                    label={option.label}
                    value={formik.values.defaultValue === option.value}
                    onChange={handleOptionChange}
                    onDelete={() => handleOptionRemove(index)}
                />
            ))}
            {isEditing && (
                <Tooltip placement="top" title="Add option">
                    <ColorIconButton
                        color="inherit"
                        onClick={handleOptionAdd}
                        aria-label="Add"
                        background="#6daf72"
                        sx={{
                            zIndex: 1,
                            width: 'fit-content',
                            margin: '5px auto !important',
                            padding: '0',
                            marginBottom: '16px !important',
                            display: 'flex',
                            alignItems: 'center',
                            borderRadius: '100%',
                        }}
                    >
                        <AddIcon fill="white" />
                    </ColorIconButton>
                </Tooltip>
            )}
        </Stack>
    );
}