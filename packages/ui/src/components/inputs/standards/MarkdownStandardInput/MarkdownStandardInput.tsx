/**
 * Input for entering (and viewing format of) Markdown data that 
 * must match a certain schema.
 */
import { MarkdownStandardInputProps } from '../types';
import { markdownStandardInputForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { useEffect } from 'react';
import { Grid } from '@mui/material';
import { MarkdownInput } from 'components/inputs/MarkdownInput/MarkdownInput';

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
                <MarkdownInput
                    id="defaultValue"
                    disabled={!isEditing}
                    placeholder="Default value"
                    value={formik.values.defaultValue}
                    minRows={3}
                    onChange={(newText: string) => formik.setFieldValue('defaultValue', newText)}
                    error={formik.touched.defaultValue && Boolean(formik.errors.defaultValue)}
                    helperText={formik.touched.defaultValue ? formik.errors.defaultValue as string : null}
                />
            </Grid>
        </Grid>
    );
}