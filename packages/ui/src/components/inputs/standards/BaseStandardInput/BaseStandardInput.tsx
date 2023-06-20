import { checkboxStandardInputForm, InputType, jsonStandardInputForm, quantityBoxStandardInputForm, radioStandardInputForm, switchStandardInputForm, textStandardInputForm } from "@local/shared";
import { FormikProvider, useField, useFormik } from "formik";
import { defaultStandardPropsMap } from "forms/generators";
import { FieldData } from "forms/types";
import { useEffect, useMemo } from "react";
import { CheckboxStandardInput, CodeStandardInput, DropzoneStandardInput, IntegerStandardInput, RadioStandardInput, SwitchStandardInput, TextStandardInput } from "../";
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
    const { SchemaInput, validationSchema, ...defaultValues } = useMemo(() => {
        const SchemaInput: ((props: any) => JSX.Element) | null = null;
        const validationSchema: any = null;
        if (!fieldName || !isEditing) return { SchemaInput, validationSchema };
        // Try to find schema in local storage
        const typeKey = `${storageKey}-${inputType}`;
        let storedProps: FieldData["props"] | object = {};
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
            case InputType.Checkbox:
                return {
                    ...defaultStandardPropsMap[InputType.Checkbox](storedProps),
                    ...storedProps,
                    SchemaInput: CheckboxStandardInput,
                    validationSchema: checkboxStandardInputForm,
                };
            case InputType.Dropzone:
                return {
                    ...defaultStandardPropsMap[InputType.Dropzone](storedProps),
                    ...storedProps,
                    SchemaInput: DropzoneStandardInput,
                    validationSchema: null,
                };
            case InputType.IntegerInput:
                return {
                    ...defaultStandardPropsMap[InputType.IntegerInput](storedProps),
                    ...storedProps,
                    SchemaInput: IntegerStandardInput,
                    validationSchema: quantityBoxStandardInputForm,
                };
            case InputType.JSON:
                return {
                    ...defaultStandardPropsMap[InputType.JSON](storedProps),
                    ...storedProps,
                    SchemaInput: CodeStandardInput,
                    validationSchema: jsonStandardInputForm,
                };
            case InputType.Radio:
                return {
                    ...defaultStandardPropsMap[InputType.Radio](storedProps),
                    ...storedProps,
                    SchemaInput: RadioStandardInput,
                    validationSchema: radioStandardInputForm,
                };
            case InputType.Switch:
                return {
                    ...defaultStandardPropsMap[InputType.Switch](storedProps),
                    ...storedProps,
                    SchemaInput: SwitchStandardInput,
                    validationSchema: switchStandardInputForm,
                };
            case InputType.Text:
                return {
                    ...defaultStandardPropsMap[InputType.Text](storedProps),
                    ...storedProps,
                    SchemaInput: TextStandardInput,
                    validationSchema: textStandardInputForm,
                };
            default:
                return {
                    SchemaInput: null,
                    validationSchema: null,
                };
        }
        // localStorage.setItem(typeKey, JSON.stringify(newSchema));
    }, [fieldName, inputType, isEditing, storageKey]);

    console.log("basestandardinput render", field.value, defaultValues, inputType);

    const formik = useFormik({
        enableReinitialize: true,
        initialValues: {
            ...defaultValues,
            ...(field.value?.props ?? {}),
        },
        onSubmit: (values) => {
            if (!fieldName || !isEditing) return;
            const updatedField = { ...field.value, props: values };
            if (JSON.stringify(updatedField) === JSON.stringify(field.value)) return;
            // Store schema props in local storage. 
            // This allows us to keep the schema when switching between input types, 
            // which may prevent work from being lost.
            const typeKey = `${storageKey}-${inputType}`;
            localStorage.setItem(typeKey, JSON.stringify(values));
            helpers.setValue(updatedField);
        },
        validateOnChange: true,
        validationSchema,
    });

    useEffect(() => {
        if (formik.isValid && formik.dirty) {
            formik.submitForm();
        }
    }, [formik]);

    return (
        <FormikProvider value={formik}>
            {SchemaInput && <>
                <SchemaInput
                    {...formik.values}
                    isEditing={isEditing}
                    fieldName={fieldName}
                    // yup={field.value.yup}
                    zIndex={zIndex}
                />
                <p>{JSON.stringify(formik.values)}</p>
            </>}
        </FormikProvider>
    );
};
