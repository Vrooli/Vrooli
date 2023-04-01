import { TextField, Typography } from '@mui/material';
import { isOfType } from '@shared/utils';
import { MarkdownInput } from 'components/inputs/MarkdownInput/MarkdownInput';
import { TranslatedMarkdownInput } from 'components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput';
import { TranslatedTextField } from 'components/inputs/TranslatedTextField/TranslatedTextField';
import { Field, useField } from 'formik';
import Markdown from 'markdown-to-jsx';
import { EditableTextProps, EditTextComponent, PropsByComponentType } from '../types';

export function EditableText<T extends EditTextComponent>({
    component,
    isEditing,
    name,
    props,
    showOnNoText,
    variant,
}: EditableTextProps<T>) {
    const [field] = useField(component.startsWith('Translated') ? `translations.${name}` : name);

    if (!isEditing && (!field.value || field.value.trim().length === 0) && !showOnNoText) return null;
    return (
        <>
            {/* Editing components */}
            {isEditing && component === 'Markdown' && <MarkdownInput name="name" {...(props as PropsByComponentType['Markdown'])} />}
            {isEditing && component === 'TranslatedMarkdown' && <TranslatedMarkdownInput name="name" {...(props as PropsByComponentType['TranslatedMarkdown'])} />}
            {isEditing && component === 'TranslatedTextField' && <TranslatedTextField name={name} {...(props as PropsByComponentType['TranslatedTextField'])} />}
            {isEditing && component === 'TextField' && <Field name={name} as={TextField} {...(props as PropsByComponentType['TextField'])} />}
            {/* Display components */}
            {!isEditing && isOfType(component, 'Markdown', 'TranslatedMarkdown') && <Markdown variant={variant}>{field.value}</Markdown>}
            {!isEditing && isOfType('TextField', 'TranslatedTextField') && <Typography variant={variant}>{field.value}</Typography>}
        </>
    )
}