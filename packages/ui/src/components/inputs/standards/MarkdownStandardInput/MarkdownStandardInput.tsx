/**
 * Input for entering (and viewing format of) Markdown data that 
 * must match a certain schema.
 */
import { MarkdownStandardInputProps } from '../types';
import { InputType, markdownStandardInputForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { useEffect } from 'react';
import { Grid, TextField } from '@mui/material';

export const MarkdownStandardInput = ({
    isEditing,
    schema,
    onChange,
}: MarkdownStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: schema.props.defaultValue ?? '',
            // yup: [],
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: () => { },
    });

    useEffect(() => {
        onChange({
            type: InputType.Markdown,
            props: formik.values,
            fieldName: schema.fieldName,
            label: schema.label,
            yup: schema.yup ?? {
                checks: [],
            }
        });
    }, [formik.values, onChange, schema.fieldName, schema.label, schema.yup]);

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