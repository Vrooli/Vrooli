import { Grid } from '@mui/material';
import { quantityBoxStandardInputForm as validationSchema } from '@shared/validation';
import { IntegerInput } from 'components/inputs/IntegerInput/IntegerInput';
import { useFormik } from 'formik';
import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { IntegerStandardInputProps } from '../types';

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
    const { t } = useTranslation();

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
                    disabled={!isEditing}
                    label={t('DefaultValue')}
                    name="defaultValue"
                    tooltip="The default value of the input"
                />
            </Grid>
            <Grid item xs={12}>
                <IntegerInput
                    disabled={!isEditing}
                    label={t('Min')}
                    name="min"
                    tooltip="The minimum value of the integer"
                />
            </Grid>
            <Grid item xs={12}>
                <IntegerInput
                    disabled={!isEditing}
                    label={t('Max')}
                    name="max"
                    tooltip="The maximum value of the integer"
                />
            </Grid>
            <Grid item xs={12}>
                <IntegerInput
                    disabled={!isEditing}
                    label={t('Step')}
                    name="step"
                    tooltip="How much to increment/decrement by"
                />
            </Grid>
        </Grid>
    );
}