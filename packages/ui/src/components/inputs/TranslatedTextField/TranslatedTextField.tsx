import { TextField } from '@mui/material';
import { useField } from 'formik';
import { getTranslationData, handleTranslationChange } from 'utils/display/translationTools';
import { TranslatedTextFieldProps } from '../types';

export const TranslatedTextField = ({
    language,
    name,
    ...props
}: TranslatedTextFieldProps) => {
    const [field, meta, helpers] = useField('translations');
    const { value, error, touched } = getTranslationData(field, meta, language);
    console.log('got translation data', value, error, touched);

    const handleBlur = (event) => {
        field.onBlur(event);
    };

    const handleChange = (event) => {
        handleTranslationChange(field, meta, helpers, event, language);
    };

    return (
        <TextField
            {...props}
            id={name}
            name={name}
            value={value?.[name] || ''}
            error={touched?.[name] && Boolean(error?.[name])}
            helperText={touched?.[name] && error?.[name]}
            onBlur={handleBlur}
            onChange={handleChange}
        />
    );
};