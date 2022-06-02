/**
 * Input for entering (and viewing format of) Markdown data that 
 * must match a certain schema.
 */
import { MarkdownStandardInputProps } from '../types';
import { markdownStandardInputForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { useEffect } from 'react';
import { Grid, TextField } from '@mui/material';

export const MarkdownStandardInput = ({
    defaultValue,
    isEditing,
    onPropsChange,
}: MarkdownStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: defaultValue ?? '',
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: () => { },
    });

    useEffect(() => {
        onPropsChange({
            ...formik.values,
        });
    }, [formik.values, onPropsChange]);

    return (
        <Grid container spacing={2}>
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    disabled={!isEditing}
                    id="defaultValue"
                    name="defaultValue"
                    label="Default Value"
                    multiline
                    value={formik.values.defaultValue}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={formik.touched.defaultValue && Boolean(formik.errors.defaultValue)}
                    helperText={formik.touched.defaultValue && formik.errors.defaultValue}
                />
            </Grid>
        </Grid>
    );
}