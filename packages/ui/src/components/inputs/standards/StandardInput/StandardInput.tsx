import { Box, Typography, useTheme } from "@mui/material";
import { SelectorBase } from "components/inputs/Selector/Selector";
import { ToggleSwitch } from "components/inputs/ToggleSwitch/ToggleSwitch";
import { GeneratedInputComponent } from "components/inputs/generated";
import { useField } from "formik";
import { FieldData } from "forms/types";
import { BuildIcon, VisibleIcon } from "icons";
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
}: StandardInputProps) => {
    const { palette } = useTheme();

    const [field, , helpers] = useField<FieldData | null>(fieldName);
    useEffect(() => {
        console.log("standardinput field changed", field.value);
    }, [field.value]);

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
                    field.value ? <GeneratedInputComponent
                        disabled={true} // Always disabled, since this is a preview
                        fieldData={field.value}
                        // eslint-disable-next-line @typescript-eslint/no-empty-function
                        onUpload={() => { }}
                    /> :
                        <Typography
                            variant="body2"
                            color="textSecondary"
                            sx={{ textAlign: "center", fontStyle: "italic" }}
                        >
                            No input data
                        </Typography> :
                    <Box>
                        <SelectorBase<InputTypeOption>
                            fullWidth
                            options={InputTypeOptions}
                            value={inputType}
                            onChange={handleInputTypeSelect}
                            getOptionLabel={(option: InputTypeOption) => option.label}
                            getOptionDescription={(option: InputTypeOption) => option.description}
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
