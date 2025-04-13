import { InputType, preventFormSubmit } from "@local/shared";
import { Box, IconButton, Stack, TextField, Tooltip, Typography, TypographyProps, styled, useTheme } from "@mui/material";
import { useFormikContext } from "formik";
import { ComponentType, Suspense, useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { lazily } from "react-lazily";
import { useEditableLabel } from "../../../hooks/useEditableLabel.js";
import { IconCommon } from "../../../icons/Icons.js";
import { PubSub } from "../../../utils/pubsub.js";
import { HelpButton } from "../../buttons/HelpButton.js";
import { FormInputCheckbox } from "./FormInputCheckbox.js";
import { FormInputInteger } from "./FormInputInteger.js";
import { FormInputLanguage } from "./FormInputLanguage.js";
import { FormInputLinkItem } from "./FormInputLinkItem.js";
import { FormInputLinkUrl } from "./FormInputLinkUrl.js";
import { FormInputRadio } from "./FormInputRadio.js";
import { FormInputSelector } from "./FormInputSelector.js";
import { FormInputSlider } from "./FormInputSlider.js";
import { FormInputSwitch } from "./FormInputSwitch.js";
import { FormInputTagSelector } from "./FormInputTagSelector.js";
import { FormInputText } from "./FormInputText.js";
import { FormInputProps } from "./types.js";

// Lazy-loading all input components is an option here, but it doesn't add much value except for the more complex components
const { FormInputCode } = lazily(() => import("./FormInputCode.js"));
const { FormInputDropzone } = lazily(() => import("./FormInputDropzone.js"));

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

const AsteriskLabel = styled(Typography)(({ theme }) => ({
    color: theme.palette.error.main,
    paddingLeft: "4px",
}));

interface FormInputLabelProps extends TypographyProps {
    disabled: boolean;
}

const FormInputLabel = styled(Typography, {
    shouldForwardProp: (prop) => prop !== "disabled",
})<FormInputLabelProps>(({ disabled, theme }) => ({
    color: disabled ? theme.palette.background.textSecondary : theme.palette.background.textPrimary,
}));

const formInputOuterBoxStyle = {
    paddingTop: 1,
    paddingBottom: 1,
    paddingLeft: 0,
    paddingRight: 0,
    border: 0,
    borderRadius: 1,
};
const textFieldStyle = {
    flexGrow: 1,
    width: "auto",
    "& .MuiInputBase-input": {
        padding: 0,
    },
} as const;
const copyButtonStyle = {
    paddingLeft: 0,
} as const;

export function FormInput({
    disabled,
    fieldData,
    index,
    isEditing,
    onConfigUpdate,
    onDelete,
    textPrimary,
    ...rest
}: FormInputProps) {
    const { t } = useTranslation();
    const { palette, typography } = useTheme();

    const formikContext = useFormikContext();
    const copyToClipboard = useCallback(function copyToClipboardCallback() {
        const fomikField = formikContext.getFieldProps(fieldData.fieldName);
        if (!fomikField) {
            console.error(`Field ${fieldData.fieldName} not found in formik context`);
            return;
        }
        navigator.clipboard.writeText(fomikField.value);
        PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
    }, [fieldData.fieldName, formikContext]);

    const updateLabel = useCallback((updatedLabel: string) => { onConfigUpdate?.({ ...fieldData, label: updatedLabel }); }, [fieldData, onConfigUpdate]);
    const {
        editedLabel,
        handleLabelChange,
        handleLabelKeyDown,
        isEditingLabel,
        labelEditRef,
        startEditingLabel,
        submitLabelChange,
    } = useEditableLabel({
        isEditable: isEditing,
        isMultiline: false,
        label: fieldData.label,
        onUpdate: updateLabel,
    });

    const handleDelete = useCallback(function handleDeleteClick() { onDelete(); }, [onDelete]);
    const onMarkdownChange = useMemo(function onMarkdownChangeMemo() {
        return isEditing ? (helpText: string) => { onConfigUpdate({ ...fieldData, helpText }); } : undefined;
    }, [isEditing, fieldData, onConfigUpdate]);

    const textFieldInputProps = useMemo(function textFieldInputPropsMemo() {
        return { style: (typography["h6"] as object || {}) };
    }, [typography]);
    const labelStyle = useMemo(function labelStyleMemo() {
        return {
            color: textPrimary,
            cursor: isEditing ? "pointer" : "default",
            // Ensure there's a clickable area, even if the text is empty
            minWidth: "20px",
            minHeight: "24px",
            "&:empty::before": {
                content: "\"\"",
                display: "inline-block",
                width: "100%",
                height: "100%",
            },
        };
    }, [textPrimary, isEditing]);

    const titleStack = useMemo(() => (
        <Stack direction="row" spacing={0} alignItems="center">
            {/* Delete button when editing form */}
            {isEditing && (
                <Tooltip title={t("Delete")}>
                    <IconButton
                        aria-label={t("Delete")}
                        onClick={handleDelete}
                    >
                        <IconCommon
                            decorative
                            fill={palette.error.main}
                            name="Delete"
                            size={24}
                        />
                    </IconButton>
                </Tooltip>
            )}
            {/* Red asterisk if required */}
            {fieldData.isRequired && <AsteriskLabel variant="h6">*</AsteriskLabel>}
            <legend aria-label="Input label">
                {isEditingLabel ? (
                    <TextField
                        ref={labelEditRef}
                        inputMode="text"
                        InputProps={textFieldInputProps}
                        onBlur={submitLabelChange}
                        onChange={handleLabelChange}
                        onKeyDown={handleLabelKeyDown}
                        placeholder={"Enter label..."}
                        size="small"
                        value={editedLabel}
                        sx={textFieldStyle}
                    />
                ) : (
                    <FormInputLabel
                        disabled={disabled ?? false}
                        id={`label-${fieldData.fieldName}`}
                        onClick={startEditingLabel}
                        sx={labelStyle}
                        variant="h6"
                    >
                        {editedLabel ?? (index && `Input ${index + 1}`) ?? t("Input", { count: 1 })}
                    </FormInputLabel>
                )}
            </legend>
            {/* Copy button for certain input types. Others include their own copy buttons or don't need one */}
            {[InputType.IntegerInput, InputType.LinkUrl, InputType.Text].includes(fieldData.type) && <Tooltip title={t("CopyToClipboard")}>
                <IconButton
                    aria-label={t("CopyToClipboard")}
                    onClick={copyToClipboard}
                    sx={copyButtonStyle}
                >
                    <IconCommon
                        decorative
                        fill={palette.background.textSecondary}
                        name="Copy"
                        size={24}
                    />
                </IconButton>
            </Tooltip>}
            {(fieldData.helpText || isEditing) ?
                <HelpButton
                    onMarkdownChange={onMarkdownChange}
                    markdown={fieldData.helpText ?? ""}
                /> : null}
        </Stack>
    ), [isEditing, t, handleDelete, palette.error.main, palette.background.textSecondary, isEditingLabel, labelEditRef, textFieldInputProps, submitLabelChange, handleLabelChange, handleLabelKeyDown, editedLabel, disabled, fieldData.fieldName, fieldData.isRequired, fieldData.type, fieldData.helpText, startEditingLabel, labelStyle, index, copyToClipboard, onMarkdownChange]);

    const inputElement = useMemo(() => {
        if (!fieldData.type || !typeMap[fieldData.type]) {
            console.error(`Unsupported input type: ${fieldData.type}`);
            return <div>Unsupported input type</div>;
        }
        const InputComponent = typeMap[fieldData.type];
        return (
            <Suspense fallback={<div>Loading...</div>}>
                <InputComponent
                    disabled={disabled}
                    fieldData={fieldData}
                    index={index}
                    isEditing={isEditing}
                    onConfigUpdate={onConfigUpdate}
                    onDelete={onDelete}
                    {...rest}
                />
            </Suspense>
        );
    }, [disabled, fieldData, index, isEditing, onConfigUpdate, onDelete, rest]);

    return (
        <Box
            aria-label={fieldData.fieldName}
            component="fieldset"
            key={`${fieldData.id}-label-box`}
            onSubmit={preventFormSubmit} // Input element should never submit a form
            sx={formInputOuterBoxStyle}
        >
            {titleStack}
            {inputElement}
        </Box>
    );
}
