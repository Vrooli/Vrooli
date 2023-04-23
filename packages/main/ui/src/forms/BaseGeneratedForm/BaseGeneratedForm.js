import { jsx as _jsx } from "react/jsx-runtime";
import { useTheme } from "@mui/material";
import { useFormik } from "formik";
import { useCallback, useMemo, useState } from "react";
import { GeneratedGrid } from "../../components/inputs/generated";
import { generateDefaultProps, generateYupSchema } from "../generators";
export const BaseGeneratedForm = ({ schema, onSubmit, zIndex, }) => {
    const theme = useTheme();
    const fieldInputs = useMemo(() => generateDefaultProps(schema?.fields), [schema?.fields]);
    const initialValues = useMemo(() => {
        const values = {};
        fieldInputs.forEach((field) => {
            values[field.fieldName] = field.props.defaultValue;
        });
        return values;
    }, [fieldInputs]);
    const validationSchema = useMemo(() => generateYupSchema(schema), [schema]);
    const [uploadedFiles, setUploadedFiles] = useState({});
    const onUpload = useCallback((fieldName, files) => {
        setUploadedFiles((prev) => {
            const newFiles = { ...prev, [fieldName]: files };
            return newFiles;
        });
    }, []);
    const formik = useFormik({
        initialValues,
        validationSchema,
        onSubmit: (values) => onSubmit(values),
    });
    return (_jsx("form", { onSubmit: formik.handleSubmit, children: schema && _jsx(GeneratedGrid, { childContainers: schema.containers, fields: schema.fields, layout: schema.formLayout, onUpload: onUpload, theme: theme, zIndex: zIndex }) }));
};
//# sourceMappingURL=BaseGeneratedForm.js.map