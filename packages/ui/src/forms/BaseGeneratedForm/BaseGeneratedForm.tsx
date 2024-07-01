// Converts JSON into a MUI form
import { useTheme } from "@mui/material";
import { GeneratedGrid } from "components/inputs/form";
import { useFormik } from "formik";
import { generateDefaultProps, generateYupSchema } from "forms/generators";
import { FormInputType, FormSchema } from "forms/types";
import { useMemo } from "react";

export interface BaseGeneratedFormProps {
    schema: FormSchema;
    onSubmit: (values: any) => unknown;
}

//TODO probably remove this component
/**
 * Form component that is generated from a JSON schema
 */
export function BaseGeneratedForm({
    schema,
    onSubmit,
}: BaseGeneratedFormProps) {
    const theme = useTheme();

    // Add non-specified props to each input field
    const fieldInputs = useMemo<FormInputType[]>(() => generateDefaultProps(schema?.fields), [schema?.fields]);

    // Parse default values from fieldInputs, to use in formik
    const initialValues = useMemo(() => {
        const values: { [x: string]: any } = {};
        fieldInputs.forEach((field) => {
            values[field.fieldName] = field.props.defaultValue;
        });
        return values;
    }, [fieldInputs]);

    // Generate yup schema from overall schema
    const validationSchema = useMemo(() => generateYupSchema(schema), [schema]);

    /**
     * Controls updates and validation of form
     */
    const formik = useFormik({
        initialValues,
        validationSchema,
        onSubmit: (values) => { onSubmit(values); },
    });

    return (
        <form onSubmit={formik.handleSubmit}>
            {schema && <GeneratedGrid
                childContainers={schema.containers}
                fields={schema.fields}
                layout={schema.formLayout}
            />}
        </form>
    );
}
