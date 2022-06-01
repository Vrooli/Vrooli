/**
 * Input for entering (and viewing format of) QuantityBox data that 
 * must match a certain schema.
 */
import { QuantityBoxStandardInputProps } from '../types';
import { InputType, quantityBoxStandardInputForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { useEffect } from 'react';
import { Grid } from '@mui/material';
import { QuantityBox } from 'components/inputs/QuantityBox/QuantityBox';

export const QuantityBoxStandardInput = ({
    isEditing,
    schema,
    onChange,
}: QuantityBoxStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: schema.props.defaultValue ?? 0,
            min: schema.props.min ?? 0,
            max: schema.props.max ?? 1000000,
            step: schema.props.step ?? 1,
            // yup: [],
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: () => { },
    });

    useEffect(() => {
        onChange({
            type: InputType.QuantityBox,
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
                <QuantityBox
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
                <QuantityBox
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
                <QuantityBox
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
                <QuantityBox
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