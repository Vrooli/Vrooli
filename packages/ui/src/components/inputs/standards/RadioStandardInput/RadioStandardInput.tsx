/**
 * Input for entering (and viewing format of) Radio data that 
 * must match a certain schema.
 */
import { RadioStandardInputProps } from '../types';
import { InputType, radioStandardInputForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { useEffect } from 'react';

export const RadioStandardInput = ({
    isEditing,
    schema,
    onChange,
}: RadioStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: schema.props.defaultValue ?? '',
            options: schema.props.options ?? [],
            row: schema.props.row ?? false,
            // yup: [],
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: () => { },
    });

    useEffect(() => {
        onChange({
            type: InputType.Radio,
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