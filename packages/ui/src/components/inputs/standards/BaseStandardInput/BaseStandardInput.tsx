/**
 * Input for entering (and viewing format of) Checkbox data that 
 * must match a certain schema.
 */
import { InputType } from '@local/shared';
import { createDefaultFieldData } from 'forms/generators';
import { FieldData } from 'forms/types';
import { useEffect, useMemo, useState } from 'react';
import { CheckboxStandardInput, DropzoneStandardInput, JsonStandardInput, MarkdownStandardInput, QuantityBoxStandardInput, RadioStandardInput, SwitchStandardInput, TextFieldStandardInput } from '../';
import { BaseStandardInputProps, CheckboxStandardInputProps, DropzoneStandardInputProps, JsonStandardInputProps, MarkdownStandardInputProps, QuantityBoxStandardInputProps, RadioStandardInputProps, SwitchStandardInputProps, TextFieldStandardInputProps } from '../types';

export const BaseStandardInput = ({
    isEditing,
    schema,
    onChange,
}: BaseStandardInputProps<FieldData>) => {

    // Random key
    const [key] = useState(`standard-create-schema-${Math.random().toString(36).substring(2, 15)}`);

    // Change schema when input type changes
    useEffect(() => {
        // Find schema is in local storage, or create a new one
        const storedData = localStorage.getItem(key);
        const newSchema = storedData ? JSON.parse(storedData) : createDefaultFieldData(schema?.type ?? InputType.JSON);
        // Update state
        localStorage.setItem(key, JSON.stringify(newSchema));
        onChange(newSchema)
    }, [key, onChange, schema?.type]);

    // Generate input component for type-specific fields
    const SchemaInput = useMemo(() => {
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