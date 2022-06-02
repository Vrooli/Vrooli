/**
 * Input for entering (and viewing format of) Checkbox data that 
 * must match a certain schema.
 */
import { InputType } from '@local/shared';
import { createDefaultFieldData } from 'forms/generators';
import { FieldData } from 'forms/types';
import { useCallback, useEffect, useMemo } from 'react';
import { CheckboxStandardInput, DropzoneStandardInput, JsonStandardInput, MarkdownStandardInput, QuantityBoxStandardInput, RadioStandardInput, SwitchStandardInput, TextFieldStandardInput } from '../';
import { BaseStandardInputProps } from '../types';

export const BaseStandardInput = ({
    fieldName,
    inputType,
    isEditing,
    label,
    onChange,
    schema,
    storageKey,
}: BaseStandardInputProps) => {
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

    const onPropsChange = useCallback((newProps: FieldData['props']) => {
        if (!schema || !isEditing) return;
        console.log('ON PROPS CHANGE', newProps, schema);
        const changedSchema = {
            ...schema,
            props: {
                ...schema.props,
                ...newProps,
            } as any,
        }
        console.log('CHANGED SCHEMA', changedSchema)
        if (JSON.stringify(changedSchema) !== JSON.stringify(schema)) {
            onChange(changedSchema);
        }
    }, [isEditing, onChange, schema]);

    // Generate input component for type-specific fields
    const SchemaInput = useMemo(() => {
        console.log('schema input switch', schema)
        if (!schema) return null;
        const props = { 
            isEditing,
            fieldName,
            label: schema.label,
            onPropsChange,
            yup: schema.yup,
        }
        switch (schema.type) {
            case InputType.TextField:
                return <TextFieldStandardInput {...props} {...schema.props} />;
            case InputType.JSON:
                return <JsonStandardInput {...props} {...schema.props} />;
            case InputType.QuantityBox:
                return <QuantityBoxStandardInput {...props} {...schema.props} />;
            case InputType.Radio:
                return <RadioStandardInput {...props} {...schema.props} />;
            case InputType.Checkbox:
                return <CheckboxStandardInput {...props} {...schema.props} />;
            case InputType.Switch:
                return <SwitchStandardInput {...props} {...schema.props} />;
            case InputType.Dropzone:
                return <DropzoneStandardInput {...props} {...schema.props} />;
            case InputType.Markdown:
                return <MarkdownStandardInput {...props} {...schema.props} />;
            default:
                return null;
        }
    }, [fieldName, isEditing, onPropsChange, schema]);

    return SchemaInput;
}