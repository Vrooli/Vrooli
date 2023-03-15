import { IntegerStandardInputProps } from '../types';
import { quantityBoxStandardInputForm as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { useEffect } from 'react';
import { Grid } from '@mui/material';
import { IntegerInput } from 'components/inputs/IntegerInput/IntegerInput';

/**
 * Input for entering (and viewing format of) IntegerInput data that 
 * must match a certain schema.
 */
export const IntegerStandardInput = ({
    defaultValue,
    isEditing,
    max,
    min,
    step,
    onPropsChange,
}: IntegerStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: defaultValue ?? 0,
            min: min ?? 0,
            max: max ?? 1000000,
            step: step ?? 1,
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
                <IntegerInput
                    id="defaultValue"
                    disabled={!isEditing}
                    label="Default Value"
                    tooltip="The default value of the input"
                    value={formik.values.defaultValue}
                    onBlur={formik.handleBlur}
                    handleChange={(value: number) => formik.setFieldValue('defaultValue', value)}
                    error={formik.touched.defaultValue && Boolean(formik.errors.defaultValue)}
                    helperText={formik.touched.defaultValue ? formik.errors.defaultValue as string : null}
                />
            </Grid>
            <Grid item xs={12}>
                <IntegerInput
                    id="min"
                    disabled={!isEditing}
                    label="Minimum"
                    tooltip="The minimum value of the integer"
                    value={formik.values.min}
                    onBlur={formik.handleBlur}
                    handleChange={(value: number) => formik.setFieldValue('min', value)}
                    error={formik.touched.min && Boolean(formik.errors.min)}
                    helperText={formik.touched.min ? formik.errors.min : null}
                />
            </Grid>
            <Grid item xs={12}>
                <IntegerInput
                    id="max"
                    disabled={!isEditing}
                    label="Maximum"
                    tooltip="The maximum value of the integer"
                    value={formik.values.max}
                    onBlur={formik.handleBlur}
                    handleChange={(value: number) => formik.setFieldValue('max', value)}
                    error={formik.touched.max && Boolean(formik.errors.max)}
                    helperText={formik.touched.max ? formik.errors.max : null}
                />
            </Grid>
            <Grid item xs={12}>
                <IntegerInput
                    id="step"
                    disabled={!isEditing}
                    label="Step"
                    tooltip="How much to increment/decrement by"
                    value={formik.values.step}
                    onBlur={formik.handleBlur}
                    handleChange={(value: number) => formik.setFieldValue('step', value)}
                    error={formik.touched.step && Boolean(formik.errors.step)}
                    helperText={formik.touched.step ? formik.errors.step : null}
                />
            </Grid>
        </Grid>
    );
}