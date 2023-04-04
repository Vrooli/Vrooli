import { Grid } from '@mui/material';
import { IntegerInput } from 'components/inputs/IntegerInput/IntegerInput';
import { useTranslation } from 'react-i18next';
import { IntegerStandardInputProps } from '../types';

/**
 * Input for entering (and viewing format of) IntegerInput data that 
 * must match a certain schema.
 */
export const IntegerStandardInput = ({
    isEditing,
}: IntegerStandardInputProps) => {
    const { t } = useTranslation();

    return (
        <Grid container spacing={2} alignItems="center" justifyContent="center">
            <Grid item xs={12} sm={6}>
                <IntegerInput
                    disabled={!isEditing}
                    label={t('DefaultValue')}
                    name="defaultValue"
                    tooltip="The default value of the input"
                />
            </Grid>
            <Grid item xs={12} sm={6}>
                <IntegerInput
                    disabled={!isEditing}
                    label={t('Min')}
                    name="min"
                    tooltip="The minimum value of the integer"
                />
            </Grid>
            <Grid item xs={12} sm={6}>
                <IntegerInput
                    disabled={!isEditing}
                    label={t('Max')}
                    name="max"
                    tooltip="The maximum value of the integer"
                />
            </Grid>
            <Grid item xs={12} sm={6}>
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