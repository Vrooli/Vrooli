// Converts JSON into a MUI form
import { useTheme } from '@mui/material';
import { GeneratedGrid } from 'components/inputs/generated';
import { useFormik } from 'formik';
import { generateDefaultProps, generateYupSchema } from 'forms/generators';
import { FieldData } from 'forms/types';
import { useCallback, useMemo, useState } from 'react';
import { BaseGeneratedFormProps } from '../types';

/**
 * Form component that is generated from a JSON schema
 */
export const BaseGeneratedForm = ({
    schema,
    onSubmit,
    zIndex,
}: BaseGeneratedFormProps) => {
    const theme = useTheme();

    // Add non-specified props to each input field
    const fieldInputs = useMemo<FieldData[]>(() => generateDefaultProps(schema?.fields), [schema?.fields]);

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

    // Stores uploaded files, where the key is the field name, and the value is an array of file urls
    const [uploadedFiles, setUploadedFiles] = useState<{ [x: string]: string[] }>({});
    /**
     * Callback for holding uploaded files, if any.
     * @param fieldName The name of the field that the file is being uploaded to
     * @param files Array of object URLs of the uploaded files. By URL we mean 
     * a base64 encoded string. The files has not been uploaded anywhere yet.
     */
    const onUpload = useCallback((fieldName: string, files: string[]) => {
        setUploadedFiles((prev) => {
            const newFiles = { ...prev, [fieldName]: files };
            return newFiles;
        });
    }, []);

    /**
     * Controls updates and validation of form
     */
    const formik = useFormik({
        initialValues,
        validationSchema,
        onSubmit: (values) => onSubmit(values),
    });

    return (
        <form onSubmit={formik.handleSubmit}>
            {schema && <GeneratedGrid
                childContainers={schema.containers}
                fields={schema.fields}
                layout={schema.formLayout}
                onUpload={onUpload}
                theme={theme}
                zIndex={zIndex}
            />}
        </form>
    );
}