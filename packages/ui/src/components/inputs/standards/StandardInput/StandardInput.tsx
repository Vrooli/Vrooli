import { Box } from "@mui/material";
import { GeneratedInputComponent } from "components/inputs/generated";
import { PreviewSwitch } from "components/inputs/PreviewSwitch/PreviewSwitch";
import { Selector } from "components/inputs/Selector/Selector";
import { useField } from "formik";
import { FieldData } from "forms/types";
import { useCallback, useState } from "react";
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
    const [field, meta, helpers] = useField<FieldData | null>(fieldName);

    // Toggle preview/edit mode
    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);

    // Handle input type selector
    const [inputType, setInputType] = useState<InputTypeOption>(InputTypeOptions[1]);
    const handleInputTypeSelect = useCallback((selected: InputTypeOption) => {
        setInputType(selected);
    }, []);

    const [schemaKey] = useState(`standard-create-schema-preview-${Math.random().toString(36).substring(2, 15)}`);

    return (
        <>
            {!disabled && <PreviewSwitch
                isPreviewOn={isPreviewOn}
                onChange={onPreviewChange}
                sx={{ marginBottom: 2 }}
            />}
            {
                isPreviewOn ?
                    (field.value && <GeneratedInputComponent
                        disabled={true} // Always disabled, since this is a preview
                        fieldName={fieldName}
                        inputType={inputType.value}
                        onUpload={() => { }}
                        zIndex={zIndex}
                    />) :
                    <Box>
                        <Selector
                            fullWidth
                            options={InputTypeOptions}
                            selected={inputType}
                            handleChange={handleInputTypeSelect}
                            getOptionLabel={(option: InputTypeOption) => option.label}
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
        </>
    )
}