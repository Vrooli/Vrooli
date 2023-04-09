import { Grid, TextField } from '@mui/material';
import { IntegerInput } from 'components/inputs/IntegerInput/IntegerInput';
import { useTranslation } from 'react-i18next';
import { TextFieldStandardInputProps } from '../types';

/**
 * Input for entering (and viewing format of) TextField data that 
 * must match a certain schema.
 */
export const TextFieldStandardInput = ({
    isEditing,
}: TextFieldStandardInputProps) => {
    const { t } = useTranslation();

    // defaultValue?: TextFieldProps['defaultValue'];
    // autoComplete?: TextFieldProps['autoComplete'];
    // maxRows?: TextFieldProps['maxRows'];

    return (
        <Grid container spacing={2}>
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    disabled={!isEditing}
                    name="defaultValue"
                    label="Default Value"
                    multiline
                />
            </Grid>
            {/* <Grid item xs={12}>
                <TextField
                    fullWidth
                    disabled={!isEditing}
                    name="autoComplete"
                    label="autoComplete"
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