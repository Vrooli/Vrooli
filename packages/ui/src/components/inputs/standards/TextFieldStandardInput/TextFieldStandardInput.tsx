import { TextFieldStandardInputProps } from '../types';
import { textFieldStandardInputForm as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { Grid, TextField } from '@mui/material';
import { IntegerInput } from 'components/inputs/IntegerInput/IntegerInput';
import { useEffect } from 'react';

/**
 * Input for entering (and viewing format of) TextField data that 
 * must match a certain schema.
 */
export const TextFieldStandardInput = ({
    autoComplete,
    defaultValue,
    isEditing,
    maxRows,
    onPropsChange,
}: TextFieldStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: defaultValue ?? '',
            autoComplete: autoComplete ?? '',
            maxRows: maxRows ?? 1,
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
            {/* <Grid item xs={12}>
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
            </Grid> */}
            <Grid item xs={12}>
                <IntegerInput
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