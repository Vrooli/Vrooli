import { Box, useTheme } from "@mui/material";
import { GeneratedInputComponent } from "components/inputs/generated";
import { PreviewSwitch } from "components/inputs/PreviewSwitch/PreviewSwitch";
import { SelectorBase } from "components/inputs/SelectorBase/SelectorBase";
import { useField } from "formik";
import { FieldData } from "forms/types";
import { useCallback, useEffect, useState } from "react";
import { InputTypeOption, InputTypeOptions } from "utils/consts";
import { BaseStandardInput } from "../BaseStandardInput/BaseStandardInput";
import { StandardInputProps } from "../types";

/**
 * Used to preview and edit standards of various types
 */
export const StandardInput = ({
    disabled = false,
    fieldName,
    zIndex,
}: StandardInputProps) => {
    const { palette } = useTheme();

    const [field, , helpers] = useField<FieldData | null>(fieldName);

    // Toggle preview/edit mode
    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);

    // Handle input type selector
    const [inputType, setInputType] = useState<InputTypeOption>(InputTypeOptions[1]);
    const handleInputTypeSelect = useCallback((selected: InputTypeOption) => {
        setInputType(selected);
    }, []);

    useEffect(() => {
        if (field.value && field.value.type !== inputType.value) {
            helpers.setValue({ ...field.value, type: inputType.value as any, props: {} });
        }
    }, [inputType, field.value, helpers]);

    const [schemaKey] = useState(`standard-create-schema-preview-${Math.random().toString(36).substring(2, 15)}`);
    console.log('standardinput field value', field.value)

    return (
        <Box sx={{
            background: palette.background.paper,
            borderRadius: 2,
            padding: 2,
        }}>
            {!disabled && <PreviewSwitch
                isPreviewOn={isPreviewOn}
                onChange={onPreviewChange}
                sx={{ marginBottom: 2 }}
            />}
            {
                (isPreviewOn || disabled) ?
                    (field.value && <GeneratedInputComponent
                        disabled={true} // Always disabled, since this is a preview
                        fieldData={field.value}
                        onUpload={() => { }}
                        zIndex={zIndex}
                    />) :
                    <Box>
                        <SelectorBase<InputTypeOption>
                            fullWidth
                            options={InputTypeOptions}
                            value={inputType}
                            onChange={handleInputTypeSelect}
                            getOptionLabel={(option: InputTypeOption) => option.label}
                            name="inputType"
                            inputAriaLabel='input-type-selector'
                            label="Type"
                            sx={{ marginBottom: 2 }}
                        />
                        <BaseStandardInput
                            fieldName={fieldName}
                            inputType={inputType.value}
                            isEditing={true}
                            storageKey={schemaKey}
                        />
                    </Box>
            }
        </Box>
    )
}