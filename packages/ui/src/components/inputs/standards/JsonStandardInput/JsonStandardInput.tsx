/**
 * Input for entering (and viewing format of) JSON data that 
 * must match a certain schema.
 */
import { JsonStandardInputProps } from '../types';
import { InputType, jsonStandardInputForm as validationSchema } from '@local/shared';
import { useEffect } from 'react';
import { useFormik } from 'formik';

export const JsonStandardInput = ({
    isEditing,
    schema,
    onChange,
}: JsonStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            format: schema.props.format ?? '',
            defaultValue: schema.props.defaultValue ?? '',
            variables: schema.props.variables ?? {},
            // yup: [],
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: () => { },
    });

    useEffect(() => {
        onChange({
            type: InputType.JSON,
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