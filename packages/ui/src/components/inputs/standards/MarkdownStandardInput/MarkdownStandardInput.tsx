/**
 * Input for entering (and viewing format of) Markdown data that 
 * must match a certain schema.
 */
import { MarkdownStandardInputProps } from '../types';
import { InputType, markdownStandardInputForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { useEffect } from 'react';

export const MarkdownStandardInput = ({
    isEditing,
    schema,
    onChange,
}: MarkdownStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: schema.props.defaultValue ?? '',
            // yup: [],
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: () => { },
    });

    useEffect(() => {
        onChange({
            type: InputType.Markdown,
            props: formik.values,
            fieldName: '',
            label: '',
            yup: {
                checks: [],
            },
        });
    }, [formik.values, onChange]);

    return (
        <></>
    );
}