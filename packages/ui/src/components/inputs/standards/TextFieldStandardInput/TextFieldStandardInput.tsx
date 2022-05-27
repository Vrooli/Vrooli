/**
 * Input for entering (and viewing format of) TextField data that 
 * must match a certain schema.
 */
import { TextFieldStandardInputProps } from '../types';
import { InputType, textFieldStandardInputForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { Grid, TextField } from '@mui/material';
import { QuantityBox } from 'components/inputs/QuantityBox/QuantityBox';
import { useEffect } from 'react';

export const TextFieldStandardInput = ({
    isEditing,
    schema,
    onChange,
}: TextFieldStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: schema.props.defaultValue ?? '',
            autoComplete: schema.props.autoComplete ?? '',
            maxRows: schema.props.maxRows ?? 1,
            // yup: [],
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: () => { },
    });

    useEffect(() => {
        onChange({
            type: InputType.TextField,
            props: formik.values,
            fieldName: '',
            label: '',
            yup: {
                checks: [],
            },
        });
    }, [formik.values, onChange]);

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
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    disabled={!isEditing}
                    id="autoComplete"
                    name="autoComplete"
                    label="autoComplete"
                    value={formik.values.autoComplete}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={formik.touched.autoComplete && Boolean(formik.errors.autoComplete)}
                    helperText={formik.touched.autoComplete && formik.errors.autoComplete}
                />
            </Grid>
            <Grid item xs={12}>
                <QuantityBox
                    id="maxRows"
                    disabled={!isEditing}
                    label="Max Rows"
                    min={1}
                    max={100}
                    tooltip="The maximum number of rows to display"
                    value={formik.values.maxRows}
                    onBlur={formik.handleBlur}
                    handleChange={(value: number) => formik.setFieldValue('maxRows', value)}
                    error={formik.touched.maxRows && Boolean(formik.errors.maxRows)}
                    helperText={formik.touched.maxRows ? formik.errors.maxRows : null}
                />
            </Grid>
        </Grid>
    );
}