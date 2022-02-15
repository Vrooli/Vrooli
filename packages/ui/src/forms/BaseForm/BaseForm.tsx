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
    // Add non-specified props to each input field
    const fieldInputs = useMemo<FieldData[]>(() => generateDefaultProps(schema.fields), [schema.fields]);

    // Parse default values from fieldInputs, to use in formik
    const initialValues = useMemo(() => {
        let values: { [x: string]: any } = {};
        fieldInputs.forEach((field) => {
            values[field.fieldName] = field.props.defaultValue;
        });
        return values;
    }, [fieldInputs])

    // Generate yup schema from overall schema
    const validationSchema = useMemo(() => generateYupSchema(schema), [schema]);
    console.log('GENERATED YUP SCHEMA', validationSchema);

    /**
     * Controls updates and validation of form
     */
    const formik = useFormik({
        initialValues,
        validationSchema,
        onSubmit: (values) => onSubmit(values),
    });

    const grid = useMemo(() => generateGrid(schema.formLayout, schema.containers, schema.fields, formik), [schema, formik])

    return (
        <form onSubmit={formik.handleSubmit}>
            {grid}
        </form>
    );
}