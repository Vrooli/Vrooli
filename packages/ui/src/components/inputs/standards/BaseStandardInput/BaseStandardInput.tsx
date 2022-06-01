/**
 * Input for entering (and viewing format of) Checkbox data that 
 * must match a certain schema.
 */
import { InputType } from '@local/shared';
import { createDefaultFieldData } from 'forms/generators';
import { FieldData } from 'forms/types';
import { useEffect, useMemo } from 'react';
import { CheckboxStandardInput, DropzoneStandardInput, JsonStandardInput, MarkdownStandardInput, QuantityBoxStandardInput, RadioStandardInput, SwitchStandardInput, TextFieldStandardInput } from '../';
import { BaseStandardInputProps, CheckboxStandardInputProps, DropzoneStandardInputProps, JsonStandardInputProps, MarkdownStandardInputProps, QuantityBoxStandardInputProps, RadioStandardInputProps, SwitchStandardInputProps, TextFieldStandardInputProps } from '../types';

export const BaseStandardInput = ({
    fieldName,
    inputType,
    isEditing,
    schema,
    onChange,
    storageKey,
}: BaseStandardInputProps<FieldData>) => {

    /**
     * Store schema in local storage when updated
     */
    useEffect(() => {
        if (!schema || !isEditing) return;
        const typeKey = `${storageKey}-${schema.type}`;
        localStorage.setItem(typeKey, JSON.stringify(schema));
    }, [isEditing, schema, storageKey]);

    /**
     * Change schema when input type changes
     */
    useEffect(() => {
        if (!isEditing) return;
        console.log('input type change', inputType)
        let newSchema: FieldData | null = null;
        const typeKey = `${storageKey}-${inputType}`;
        // Find schema in  local storage
        if (localStorage.getItem(storageKey)) {
            try {
                console.log('is in local storage', localStorage.getItem(typeKey))
                const storedData = JSON.parse(localStorage.getItem(typeKey) ?? '{}');
                console.log('stored data', storedData)
                if (typeof storedData === 'object' && storedData.type === inputType) {
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
        console.log('🙃setting local storage', newSchema)
        localStorage.setItem(typeKey, JSON.stringify(newSchema));
        onChange(newSchema as FieldData)
    }, [onChange, inputType, storageKey, fieldName, isEditing]);

    // Generate input component for type-specific fields
    const SchemaInput = useMemo(() => {
        console.log('schema input switch', schema)
        if (!schema) return null;
        const props = { isEditing, schema, onChange }
        switch (schema.type) {
            case InputType.TextField:
                return <TextFieldStandardInput {...props as TextFieldStandardInputProps} />;
            case InputType.JSON:
                return <JsonStandardInput {...props as JsonStandardInputProps} />;
            case InputType.QuantityBox:
                return <QuantityBoxStandardInput {...props as QuantityBoxStandardInputProps} />;
            case InputType.Radio:
                return <RadioStandardInput {...props as RadioStandardInputProps} />;
            case InputType.Checkbox:
                return <CheckboxStandardInput {...props as CheckboxStandardInputProps} />;
            case InputType.Switch:
                return <SwitchStandardInput {...props as SwitchStandardInputProps} />;
            case InputType.Dropzone:
                return <DropzoneStandardInput {...props as DropzoneStandardInputProps} />;
            case InputType.Markdown:
                return <MarkdownStandardInput {...props as MarkdownStandardInputProps} />;
            default:
                return null;
        }
    }, [isEditing, onChange, schema]);

    return SchemaInput;
}