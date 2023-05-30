import { checkboxStandardInputForm, CompleteIcon, InputType, jsonStandardInputForm, markdownStandardInputForm, quantityBoxStandardInputForm, radioStandardInputForm, RefreshIcon, switchStandardInputForm, textFieldStandardInputForm } from "@local/shared";
import { Box, Button, Grid } from "@mui/material";
import { Formik, useField } from "formik";
import { FieldData } from "forms/types";
import { t } from "i18next";
import { useMemo } from "react";
import { CheckboxStandardInput, DropzoneStandardInput, emptyCheckboxOption, emptyRadioOption, IntegerStandardInput, JsonStandardInput, MarkdownStandardInput, RadioStandardInput, SwitchStandardInput, TextFieldStandardInput } from "../";
import { BaseStandardInputProps } from "../types";

/**
 * Input for entering (and viewing format of) Checkbox data that 
 * must match a certain schema.
 */
export const BaseStandardInput = ({
    fieldName,
    inputType,
    isEditing,
    label,
    storageKey,
    zIndex,
}: BaseStandardInputProps) => {
    const [field, , helpers] = useField(fieldName ?? "");

    // Generate input component for type-specific fields
    const { defaultValues, SchemaInput, validationSchema } = useMemo(() => {
        let defaultValues: FieldData["props"] = {};
        let SchemaInput: ((props: any) => JSX.Element) | null = null;
        let validationSchema: any = null;
        if (!fieldName || !isEditing) return { defaultValues, SchemaInput, validationSchema };
        // Try to find schema in local storage
        const typeKey = `${storageKey}-${inputType}`;
        let storedProps: FieldData["props"] | null = null;
        if (localStorage.getItem(typeKey)) {
            try {
                console.log("checking key", localStorage.getItem(typeKey));
                const parsedProps = JSON.parse(localStorage.getItem(typeKey) ?? "{}");
                if (typeof storedProps === "object") {
                    storedProps = parsedProps as FieldData["props"];
                }
            } catch (error) {
                console.error("Failed to parse stored standard input from local storage", error);
            }
        }
        console.log("storedProps", typeKey, storedProps);
        console.log("getting input data", inputType);
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
        // localStorage.setItem(typeKey, JSON.stringify(newSchema));
    }, [fieldName, inputType, isEditing, storageKey]);

    console.log("basestandardinpug", fieldName, inputType, defaultValues, field.value?.props);

    // useEffect(() => {
    //     if (field.value && field.value.type !== inputType) {
    //         console.log('standardinput useeffect setting value', JSON.stringify(field.value), field.value?.type, inputType)
    //         helpers.setValue({ ...field.value, type: inputType as any, props: defaultValues });
    //     }
    // }, [inputType, field.value, helpers, defaultValues]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={{
                ...defaultValues,
                ...(field.value?.props ?? {}),
            }}
            onSubmit={(values) => {
                console.log("onsubmit 1", fieldName, field.value, values);
                if (!fieldName || !isEditing) return;
                const updatedField = { ...field.value, props: values };
                console.log("should update schema?", updatedField, field.value, JSON.stringify(updatedField) === JSON.stringify(field.value));
                if (JSON.stringify(updatedField) === JSON.stringify(field.value)) return;
                // Store schema props in local storage. 
                // This allows us to keep the schema when switching between input types, 
                // which may prevent work from being lost.
                const typeKey = `${storageKey}-${inputType}`;
                localStorage.setItem(typeKey, JSON.stringify(values));
                console.log("going to update schema", typeKey, values);
                helpers.setValue(updatedField);
            }}
            validationSchema={validationSchema}
        >
            {(formik) => SchemaInput && <>
                <SchemaInput
                    {...formik}
                    isEditing={isEditing}
                    fieldName={fieldName}
                    // yup={field.value.yup}
                    zIndex={zIndex}
                />
                <Grid container spacing={2} mt={2} sx={{
                    padding: 2,
                    paddingTop: 2,
                    marginLeft: "auto",
                    marginRight: "auto",
                    maxWidth: "min(700px, 100%)",
                    left: 0,
                }}
                >
                    <Grid item xs={6}>
                        <Box onClick={() => { formik.handleSubmit(); }}>
                            <Button
                                disabled={!formik.isValid || !formik.dirty}
                                fullWidth
                                startIcon={<CompleteIcon />}
                            >{t("Confirm")}</Button>
                        </Box>
                    </Grid>
                    {/* Cancel button */}
                    <Grid item xs={6}>
                        <Button
                            disabled={!formik.dirty}
                            fullWidth
                            onClick={() => {
                                formik.resetForm();
                                localStorage.removeItem(`${storageKey}-${inputType}`);
                            }}
                            startIcon={<RefreshIcon />}
                        >{t("Reset")}</Button>
                    </Grid>
                </Grid>
            </>}
        </Formik>
    );
};
