import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { InputType } from "@local/consts";
import { CompleteIcon, RefreshIcon } from "@local/icons";
import { checkboxStandardInputForm, jsonStandardInputForm, markdownStandardInputForm, quantityBoxStandardInputForm, radioStandardInputForm, switchStandardInputForm, textFieldStandardInputForm } from "@local/validation";
import { Box, Button, Grid } from "@mui/material";
import { Formik, useField } from "formik";
import { t } from "i18next";
import { useMemo } from "react";
import { CheckboxStandardInput, DropzoneStandardInput, emptyCheckboxOption, emptyRadioOption, IntegerStandardInput, JsonStandardInput, MarkdownStandardInput, RadioStandardInput, SwitchStandardInput, TextFieldStandardInput } from "..";
export const BaseStandardInput = ({ fieldName, inputType, isEditing, label, storageKey, }) => {
    const [field, , helpers] = useField(fieldName ?? "");
    const { defaultValues, SchemaInput, validationSchema } = useMemo(() => {
        let defaultValues = {};
        let SchemaInput = null;
        let validationSchema = null;
        if (!fieldName || !isEditing)
            return { defaultValues, SchemaInput, validationSchema };
        const typeKey = `${storageKey}-${inputType}`;
        let storedProps = null;
        if (localStorage.getItem(typeKey)) {
            try {
                console.log("checking key", localStorage.getItem(typeKey));
                const parsedProps = JSON.parse(localStorage.getItem(typeKey) ?? "{}");
                if (typeof storedProps === "object") {
                    storedProps = parsedProps;
                }
            }
            catch (error) {
                console.error("Failed to parse stored standard input from local storage", error);
            }
        }
        console.log("storedProps", typeKey, storedProps);
        switch (inputType) {
            case InputType.TextField:
                return {
                    defaultValues: {
                        defaultValue: "",
                        autoComplete: "",
                        maxRows: 1,
                        ...storedProps,
                    },
                    SchemaInput: TextFieldStandardInput,
                    validationSchema: textFieldStandardInputForm,
                };
            case InputType.JSON:
                return {
                    defaultValues: {
                        format: JSON.stringify({}),
                        defaultValue: "",
                        variables: {},
                        ...storedProps,
                    },
                    SchemaInput: JsonStandardInput,
                    validationSchema: jsonStandardInputForm,
                };
            case InputType.IntegerInput:
                defaultValues = {
                    defaultValue: 0,
                    min: 0,
                    max: 1000000,
                    step: 1,
                    ...storedProps,
                };
                SchemaInput = IntegerStandardInput;
                validationSchema = quantityBoxStandardInputForm;
                break;
            case InputType.Radio:
                return {
                    defaultValues: {
                        defaultValue: "",
                        options: [emptyRadioOption(0)],
                        row: false,
                        ...storedProps,
                    },
                    SchemaInput: RadioStandardInput,
                    validationSchema: radioStandardInputForm,
                };
            case InputType.Checkbox:
                return {
                    defaultValues: {
                        defaultValue: [false],
                        options: [emptyCheckboxOption(0)],
                        row: false,
                        ...storedProps,
                    },
                    SchemaInput: CheckboxStandardInput,
                    validationSchema: checkboxStandardInputForm,
                };
            case InputType.Switch:
                return {
                    defaultValues: {
                        defaultValue: false,
                        ...storedProps,
                    },
                    SchemaInput: SwitchStandardInput,
                    validationSchema: switchStandardInputForm,
                };
            case InputType.Dropzone:
                return {
                    defaultValues: {
                        ...storedProps,
                    },
                    SchemaInput: DropzoneStandardInput,
                    validationSchema: null,
                };
            case InputType.Markdown:
                return {
                    defaultValues: {
                        defaultValue: "",
                        ...storedProps,
                    },
                    SchemaInput: MarkdownStandardInput,
                    validationSchema: markdownStandardInputForm,
                };
            default:
                return {
                    defaultValues: {},
                    SchemaInput: null,
                    validationSchema: null,
                };
        }
        return { defaultValues, SchemaInput, validationSchema };
    }, [fieldName, inputType, isEditing, storageKey]);
    console.log("basestandardinpug", fieldName, inputType, defaultValues, field.value?.props);
    return (_jsx(Formik, { enableReinitialize: true, initialValues: {
            ...defaultValues,
            ...(field.value?.props ?? {}),
        }, onSubmit: (values) => {
            console.log("onsubmit 1", fieldName, field.value, values);
            if (!fieldName || !isEditing)
                return;
            const updatedField = { ...field.value, props: values };
            console.log("should update schema?", updatedField, field.value, JSON.stringify(updatedField) === JSON.stringify(field.value));
            if (JSON.stringify(updatedField) === JSON.stringify(field.value))
                return;
            const typeKey = `${storageKey}-${inputType}`;
            localStorage.setItem(typeKey, JSON.stringify(values));
            console.log("going to update schema", typeKey, values);
            helpers.setValue(updatedField);
        }, validationSchema: validationSchema, children: (formik) => SchemaInput && _jsxs(_Fragment, { children: [_jsx(SchemaInput, { ...formik, isEditing: isEditing, fieldName: fieldName }), _jsxs(Grid, { container: true, spacing: 2, mt: 2, sx: {
                        padding: 2,
                        paddingTop: 2,
                        marginLeft: "auto",
                        marginRight: "auto",
                        maxWidth: "min(700px, 100%)",
                        left: 0,
                    }, children: [_jsx(Grid, { item: true, xs: 6, children: _jsx(Box, { onClick: () => { formik.handleSubmit(); }, children: _jsx(Button, { disabled: !formik.isValid || !formik.dirty, fullWidth: true, startIcon: _jsx(CompleteIcon, {}), children: t("Confirm") }) }) }), _jsx(Grid, { item: true, xs: 6, children: _jsx(Button, { disabled: !formik.dirty, fullWidth: true, onClick: () => {
                                    formik.resetForm();
                                    localStorage.removeItem(`${storageKey}-${inputType}`);
                                }, startIcon: _jsx(RefreshIcon, {}), children: t("Reset") }) })] })] }) }));
};
//# sourceMappingURL=BaseStandardInput.js.map