import { useField } from 'formik';
import { getTranslationData, handleTranslationChange } from 'utils/display/translationTools';
import { MarkdownInputBase } from '../MarkdownInputBase/MarkdownInputBase';
import { TranslatedMarkdownInputProps } from '../types';

export const TranslatedMarkdownInput = ({
    language,
    name,
    ...props
}: TranslatedMarkdownInputProps) => {
    const [field, meta, helpers] = useField('translations');
    const { value, error, touched } = getTranslationData(field, meta, language);

    const handleBlur = (event) => {
        field.onBlur(event);
    };

    const handleChange = (newText: string) => {
        handleTranslationChange(field, meta, helpers, { target: { name, value: newText } }, language);
    };

    return (
        <MarkdownInputBase
            {...props}
            name={name}
            value={value?.[name] || ''}
            error={touched?.[name] && Boolean(error?.[name])}
            helperText={touched?.[name] && error?.[name]}
            onBlur={handleBlur}
            onChange={handleChange}
        />
    );
};