import { InputType } from '@shared/consts';
import { checkboxStandardInputForm, jsonStandardInputForm, markdownStandardInputForm, quantityBoxStandardInputForm, radioStandardInputForm, switchStandardInputForm, textFieldStandardInputForm } from '@shared/validation';
import { Formik, useField } from 'formik';
import { createDefaultFieldData } from 'forms/generators';
import { FieldData } from 'forms/types';
import { useEffect, useMemo } from 'react';
import { CheckboxStandardInput, DropzoneStandardInput, emptyCheckboxOption, emptyRadioOption, IntegerStandardInput, JsonStandardInput, MarkdownStandardInput, RadioStandardInput, SwitchStandardInput, TextFieldStandardInput } from '../';
import { BaseStandardInputProps } from '../types';

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
}: BaseStandardInputProps) => {
    const [field, , helpers] = useField(fieldName ?? '');

    /**
     * Store schema in local storage when updated
     */
    useEffect(() => {
        if (!fieldName || !field.value || !isEditing) return;
        const typeKey = `${storageKey}-${inputType}`;
        localStorage.setItem(typeKey, JSON.stringify(field.value));
    }, [field.value, fieldName, inputType, isEditing, storageKey]);

    /**
     * Change schema when input type changes
     */
    useEffect(() => {
        if (!isEditing) return;
        let newSchema: FieldData | null = null;
        const typeKey = `${storageKey}-${inputType}`;
        // Find schema in  local storage
        if (localStorage.getItem(storageKey)) {
            try {
                const storedData = JSON.parse(localStorage.getItem(typeKey) ?? '{}');
                if (typeof storedData === 'object' && storedData.standardType === inputType) {
                    newSchema = storedData as FieldData;
                }
            } catch (error) {
                console.error('Failed to parse stored standard input from local storage', error);
            }
        }
        // If no schema found in local storage, create default schema
        if (!newSchema) {
            newSchema = createDefaultFieldData({
                fieldName,
                type: inputType,
                yup: { required: true, checks: [] },
            });
        }
        // Update state
        localStorage.setItem(typeKey, JSON.stringify(newSchema));
        helpers.setValue(newSchema);
    }, [inputType, storageKey, fieldName, isEditing, helpers]);

    // Generate input component for type-specific fields
    const { defaultValues, SchemaInput, validationSchema } = useMemo(() => {
        if (!fieldName || !field.value || !isEditing) return { SchemaInput: null, validationSchema: null };
        switch (field.value.type) {
            case InputType.TextField:
                return {
                    defaultValues: {
                        defaultValue: '',
                        autoComplete: '',
                        maxRows: 1,
                    },
                    SchemaInput: TextFieldStandardInput,
                    validationSchema: textFieldStandardInputForm
                }
            case InputType.JSON:
                return {
                    defaultValues: {
                        format: JSON.stringify({}),
                        defaultValue: '',
                        variables: {},
                    },
                    SchemaInput: JsonStandardInput,
                    validationSchema: jsonStandardInputForm
                }
            case InputType.IntegerInput:
                return {
                    defaultValues: {
                        defaultValue: 0,
                        min: 0,
                        max: 1000000,
                        step: 1,
                    },
                    SchemaInput: IntegerStandardInput,
                    validationSchema: quantityBoxStandardInputForm
                }
            case InputType.Radio:
                return {
                    defaultValues: {
                        defaultValue: '',
                        options: [emptyRadioOption(0)],
                        row: false,
                    },
                    SchemaInput: RadioStandardInput,
                    validationSchema: radioStandardInputForm
                }
            case InputType.Checkbox:
                return {
                    defaultValues: {
                        defaultValue: [false],
                        options: [emptyCheckboxOption(0)],
                        row: false,
                    },
                    SchemaInput: CheckboxStandardInput,
                    validationSchema: checkboxStandardInputForm
                }
            case InputType.Switch:
                return {
                    defaultValues: {
                        defaultValue: false,
                    },
                    SchemaInput: SwitchStandardInput,
                    validationSchema: switchStandardInputForm
                }
            case InputType.Dropzone:
                return {
                    defaultValues: {},
                    SchemaInput: DropzoneStandardInput,
                    validationSchema: null
                }
            case InputType.Markdown:
                return {
                    defaultValues: {
                        defaultValue: '',
                    },
                    SchemaInput: MarkdownStandardInput,
                    validationSchema: markdownStandardInputForm
                }
            default:
                return {
                    defaultValues: {},
                    SchemaInput: null,
                    validationSchema: null
                };
        }
    }, [field.value, fieldName, isEditing]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={{
                ...defaultValues,
                ...field.value.props
            }}
            onSubmit={(values) => {
                if (!fieldName || !field.value || !isEditing) return;
                const changedSchema = {
                    ...field.value,
                    props: {
                        ...field.value.props,
                        ...values,
                    } as any,
                }
                if (JSON.stringify(changedSchema) !== JSON.stringify(field.value)) {
                    helpers.setValue(changedSchema);
                }
            }}
            validationSchema={validationSchema}
        >
            {SchemaInput && <SchemaInput
                isEditing={isEditing}
                fieldName={fieldName}
            // yup={field.value.yup}
            />}
        </Formik>
    )
}