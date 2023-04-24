import { BuildIcon, VisibleIcon } from ":local/icons";
import { Box, useTheme } from "@mui/material";
import { useField } from "formik";
import { useCallback, useState } from "react";
import { FieldData } from "../../../../forms/types";
import { InputTypeOption, InputTypeOptions } from "../../../../utils/consts";
import { GeneratedInputComponent } from "../../generated";
import { SelectorBase } from "../../SelectorBase/SelectorBase";
import { ToggleSwitch } from "../../ToggleSwitch/ToggleSwitch";
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
    const onPreviewChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => { setIsPreviewOn(event.target.checked); }, []);

    // Handle input type selector
    const [inputType, setInputType] = useState<InputTypeOption>(InputTypeOptions[1]);
    const handleInputTypeSelect = useCallback((selected: InputTypeOption) => {
        setInputType(selected);
    }, []);

    const [schemaKey] = useState(`standard-create-schema-preview-${Math.random().toString(36).substring(2, 15)}`);
    console.log("standardinput field value", fieldName, field.value);

    return (
        <Box sx={{
            background: palette.background.paper,
            borderRadius: 2,
            padding: 2,
        }}>
            {!disabled && <ToggleSwitch
                checked={isPreviewOn}
                onChange={onPreviewChange}
                OffIcon={BuildIcon}
                OnIcon={VisibleIcon}
                label={isPreviewOn ? "Preview" : "Edit"}
                tooltip={isPreviewOn ? "Switch to edit" : "Switch to preview"}
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
    );
};
