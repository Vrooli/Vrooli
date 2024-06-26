import { TextField, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import { RefObject } from "react";
import { getTranslationData, handleTranslationChange } from "utils/display/translationTools";
import { TextInputProps, TranslatedTextInputProps } from "../types";

export function TextInput({
    enterWillSubmit,
    helperText,
    label,
    isRequired,
    onSubmit,
    placeholder,
    ref,
    ...props
}: TextInputProps) {
    const { palette } = useTheme();

    function StyledLabel() {
        return (
            <>
                {label}
                {isRequired && <Typography component="span" variant="h6" sx={{ color: palette.error.main, paddingLeft: "4px" }}>*</Typography>}
            </>
        );
    }

    function onKeyDown(event: React.KeyboardEvent<HTMLInputElement>) {
        if (enterWillSubmit !== true || typeof onSubmit !== "function") return;
        if (event.key === "Enter" && !event.shiftKey) {
            event.preventDefault();
            onSubmit();
        }
    }

    return <TextField
        helperText={helperText ? typeof helperText === "string" ? helperText : JSON.stringify(helperText) : undefined}
        InputLabelProps={(label && placeholder) ? { shrink: true } : {}}
        label={label ? <StyledLabel /> : ""}
        onKeyDown={onKeyDown}
        placeholder={placeholder}
        ref={ref as RefObject<HTMLInputElement>}
        variant="outlined"
        {...props}
    />;
}

export function TranslatedTextInput({
    language,
    name,
    ...props
}: TranslatedTextInputProps) {
    const [field, meta, helpers] = useField("translations");
    const { value, error, touched } = getTranslationData(field, meta, language);

    function handleBlur(event) {
        field.onBlur(event);
    }

    function handleChange(event) {
        handleTranslationChange(field, meta, helpers, event, language);
    }

    return (
        <TextInput
            {...props}
            id={name}
            name={name}
            value={value?.[name] || ""}
            error={touched?.[name] && Boolean(error?.[name])}
            helperText={touched?.[name] && error?.[name]}
            onBlur={handleBlur}
            onChange={handleChange}
        />
    );
}
