import { CopyIcon } from ":local/icons";
import { Box, IconButton, Stack, Tooltip, Typography } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { HelpButton } from "../../../buttons/HelpButton/HelpButton";
import { GeneratedInputComponent } from "../GeneratedInputComponent/GeneratedInputComponent";
import { GeneratedInputComponentWithLabelProps } from "../types";

export const GeneratedInputComponentWithLabel = ({
    copyInput,
    disabled,
    fieldData,
    index,
    onUpload,
    textPrimary,
    zIndex,
}: GeneratedInputComponentWithLabelProps) => {
    console.log("rendering input component with label");
    const { t } = useTranslation();

    const inputComponent = useMemo(() => {
        <GeneratedInputComponent
            fieldData={fieldData}
            disabled={false}
            index={index}
            onUpload={() => { }}
            zIndex={zIndex}
        />;
    }, [fieldData, index, zIndex]);

    return (
        <Box key={index} sx={{
            paddingTop: 1,
            paddingBottom: 1,
            borderRadius: 1,
        }}>
            <>
                {/* Label, help button, and copy iput icon */}
                <Stack direction="row" spacing={0} sx={{ alignItems: "center" }}>
                    <Tooltip title={t("CopyToClipboard")}>
                        <IconButton onClick={() => copyInput && copyInput(fieldData.fieldName)}>
                            <CopyIcon fill={textPrimary} />
                        </IconButton>
                    </Tooltip>
                    <Typography variant="h6" sx={{ color: textPrimary }}>{fieldData.label ?? (index && `Input ${index + 1}`) ?? t("Input")}</Typography>
                    {fieldData.helpText && <HelpButton markdown={fieldData.helpText} />}
                </Stack>
                {inputComponent}
            </>
        </Box>
    );
};
