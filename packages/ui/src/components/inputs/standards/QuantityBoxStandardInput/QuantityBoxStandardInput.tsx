/**
 * Input for entering (and viewing format of) QuantityBox data that 
 * must match a certain schema.
 */
import { QuantityBoxStandardInputProps } from '../types';
import { InputType, quantityBoxStandardInputForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { useEffect } from 'react';

export const QuantityBoxStandardInput = ({
    isEditing,
    schema,
    onChange,
}: QuantityBoxStandardInputProps) => {

    const formik = useFormik({
        initialValues: {
            defaultValue: schema.props.defaultValue ?? '',
            min: schema.props.min ?? 0,
            max: schema.props.max ?? 1000000,
            step: schema.props.step ?? 1,
            // yup: [],
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: () => { },
    });

    useEffect(() => {
        onChange({
            type: InputType.QuantityBox,
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