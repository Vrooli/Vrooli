import { InputType } from "@local/shared";
import { Box, IconButton, Stack, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { useEditableLabel } from "hooks/useEditableLabel";
import { CopyIcon, DeleteIcon } from "icons";
import { ComponentType, Suspense, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { lazily } from "react-lazily";
import { FormInputCheckbox } from "../FormInputCheckbox/FormInputCheckbox";
import { FormInputInteger } from "../FormInputInteger/FormInputInteger";
import { FormInputLanguage } from "../FormInputLanguage/FormInputLanguage";
import { FormInputLinkItem } from "../FormInputLinkItem/FormInputLinkItem";
import { FormInputLinkUrl } from "../FormInputLinkUrl/FormInputLinkUrl";
import { FormInputRadio } from "../FormInputRadio/FormInputRadio";
import { FormInputSelector } from "../FormInputSelector/FormInputSelector";
import { FormInputSlider } from "../FormInputSlider/FormInputSlider";
import { FormInputSwitch } from "../FormInputSwitch/FormInputSwitch";
import { FormInputTagSelector } from "../FormInputTagSelector/FormInputTagSelector";
import { FormInputText } from "../FormInputText/FormInputText";
import { FormInputProps } from "../types";

// Lazy-loading all input components is an option here, but it doesn't add much value except for the more complex components
const { FormInputCode } = lazily(() => import("../FormInputCode/FormInputCode"));
const { FormInputDropzone } = lazily(() => import("../FormInputDropzone/FormInputDropzone"));

/**
 * Maps a data input type string to its corresponding component generator function
 */
const typeMap: { [key in InputType]: ComponentType<FormInputProps<any>> } = {
    [InputType.Checkbox]: FormInputCheckbox,
    [InputType.Dropzone]: FormInputDropzone,
    [InputType.JSON]: FormInputCode,
    [InputType.IntegerInput]: FormInputInteger,
    [InputType.LanguageInput]: FormInputLanguage,
    [InputType.LinkItem]: FormInputLinkItem,
    [InputType.LinkUrl]: FormInputLinkUrl,
    [InputType.Radio]: FormInputRadio,
    [InputType.Selector]: FormInputSelector,
    [InputType.Slider]: FormInputSlider,
    [InputType.Switch]: FormInputSwitch,
    [InputType.TagSelector]: FormInputTagSelector,
    [InputType.Text]: FormInputText,
};

export function FormInput({
    copyInput,
    fieldData,
    index,
    onConfigUpdate,
    onDelete,
    textPrimary,
    ...rest
}: FormInputProps) {
    const { t } = useTranslation();
    const { palette, typography } = useTheme();

    const updateLabel = useCallback((updatedLabel: string) => { onConfigUpdate?.({ ...fieldData, label: updatedLabel }); }, []);
    const {
        editedLabel,
        handleLabelChange,
        handleLabelKeyDown,
        isEditingLabel,
        labelEditRef,
        startEditingLabel,
        submitLabelChange,
    } = useEditableLabel({
        isEditable: typeof onConfigUpdate === "function",
        label: fieldData.label,
        onUpdate: updateLabel,
    });

    const titleStack = useMemo(() => (
        <Stack direction="row" spacing={0} alignItems="center">
            {/* Delete button when editing form, and copy button when using form */}
            {typeof onDelete === "function" && typeof onConfigUpdate === "function" ? (
                <Tooltip title={t("Delete")}>
                    <IconButton onClick={function deleteElementClick() { onDelete(); }}>
                        <DeleteIcon fill={palette.error.main} width="24px" height="24px" />
                    </IconButton>
                </Tooltip>
            ) : typeof copyInput === "function" ? (
                <Tooltip title={t("CopyToClipboard")}>
                    <IconButton onClick={() => copyInput(fieldData.fieldName)}>
                        <CopyIcon fill={textPrimary} />
                    </IconButton>
                </Tooltip>
            ) : null}
            <legend aria-label="Input label">
                {isEditingLabel ? (
                    <TextField
                        ref={labelEditRef}
                        autoFocus
                        InputProps={{ style: (typography["h6"] as object || {}) }}
                        onBlur={submitLabelChange}
                        onChange={handleLabelChange}
                        onKeyDown={handleLabelKeyDown}
                        size="small"
                        value={editedLabel}
                        sx={{
                            flexGrow: 1,
                            width: "auto",
                            "& .MuiInputBase-input": {
                                padding: 0,
                            },
                        }}
                    />
                ) : (
                    <Typography
                        id={`label-${fieldData.fieldName}`}
                        onClick={startEditingLabel}
                        sx={{ color: textPrimary, cursor: typeof onConfigUpdate === "function" ? "pointer" : "default" }}
                        variant="h6"
                    >
                        {editedLabel ?? (index && `Input ${index + 1}`) ?? t("Input")}
                    </Typography>
                )}
            </legend>
            {/* Red asterisk if required */}
            {fieldData.isRequired && <Typography variant="h6" sx={{ color: palette.error.main, paddingLeft: "4px" }}>*</Typography>}
            {(fieldData.helpText || typeof onConfigUpdate === "function") ?
                <HelpButton
                    onMarkdownChange={typeof onConfigUpdate === "function" ? (helpText) => { onConfigUpdate({ ...fieldData, helpText }); } : undefined}
                    markdown={fieldData.helpText ?? ""}
                /> : null}
        </Stack>
    ), [onDelete, onConfigUpdate, t, palette.error.main, copyInput, textPrimary, isEditingLabel, typography, submitLabelChange, handleLabelChange, handleLabelKeyDown, editedLabel, fieldData, startEditingLabel, index]);

    const inputElement = useMemo(() => {
        if (!fieldData.type || !typeMap[fieldData.type]) {
            console.error(`Unsupported input type: ${fieldData.type}`);
            return <div>Unsupported input type</div>;
        }
        const InputComponent = typeMap[fieldData.type];
        return (
            <Suspense fallback={<div>Loading...</div>}>
                <InputComponent
                    fieldData={fieldData}
                    index={index}
                    onConfigUpdate={onConfigUpdate}
                    {...rest}
                />
            </Suspense>
        );
    }, [fieldData, index, onConfigUpdate, rest]);

    return (
        <Box
            aria-label={fieldData.fieldName}
            component="fieldset"
            key={`${fieldData.id}-label-box`}
            sx={{
                paddingTop: 1,
                paddingBottom: 1,
                border: 0,
                borderRadius: 1,
            }}
        >
            {titleStack}
            {inputElement}
        </Box>
    );
}
