// Converts JSON into a MUI form
import { useMemo } from 'react';
import { BaseFormProps } from '../types';
import { useFormik } from 'formik';
import { FieldData } from 'forms/types';
import { generateDefaultProps, generateGrid, generateYupSchema } from 'forms/generators';

/**
 * Form component that is generated from a JSON schema
 */
export const BaseForm = ({
    schema,
    onSubmit,
}: BaseFormProps) => {
    console.log('in baseform render', schema, onSubmit);
    // Add non-specified props to each input field
    const fieldInputs = useMemo<FieldData[]>(() => generateDefaultProps(schema?.fields), [schema?.fields]);
    console.log('fieldInputs', fieldInputs);
    // Parse default values from fieldInputs, to use in formik
    const initialValues = useMemo(() => {
        let values: { [x: string]: any } = {};
        fieldInputs.forEach((field) => {
            values[field.fieldName] = field.props.defaultValue;
        });
        return values;
    }, [fieldInputs])
    console.log('initialValues', initialValues);
    // Generate yup schema from overall schema
    const validationSchema = useMemo(() => generateYupSchema(schema), [schema]);
    console.log('validationSchema', validationSchema);
    /**
     * Controls updates and validation of form
     */
    const formik = useFormik({
        initialValues,
        validationSchema,
        onSubmit: (values) => onSubmit(values),
    });
    console.log('got formik', formik);
    const grid = useMemo(() => {
        if (!schema) return null;
        console.log('going to generate grid', schema);
        return generateGrid(schema.formLayout, schema.containers, schema.fields, formik)
    }, [schema, formik])

    return (
        <form onSubmit={formik.handleSubmit}>
            {grid}
        </form>
    );
}