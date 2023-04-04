import { Grid, TextField } from '@mui/material';
import { textFieldStandardInputForm as validationSchema } from '@shared/validation';
import { IntegerInput } from 'components/inputs/IntegerInput/IntegerInput';
import { useFormik } from 'formik';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { TextFieldStandardInputProps } from '../types';

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
    const { t } = useTranslation();

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
                    disabled={!isEditing}
                    label={t('MaxRows')}
                    name="maxRows"
                    tooltip="The maximum number of rows to display"
                />
            </Grid>
        </Grid>
    );
}