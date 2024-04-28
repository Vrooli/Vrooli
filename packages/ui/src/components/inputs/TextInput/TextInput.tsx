import { TextField, Typography } from "@mui/material";
import { useField } from "formik";
import { RefObject } from "react";
import { useTranslation } from "react-i18next";
import { getTranslationData, handleTranslationChange } from "utils/display/translationTools";
import { TextInputProps, TranslatedTextInputProps } from "../types";

export const TextInput = ({
    enterWillSubmit,
    label,
    isOptional,
    onSubmit,
    placeholder,
    ref,
    ...props
}: TextInputProps) => {
    const { t } = useTranslation();

    // Custom label component with optional styling
    const LabelWithOptional = () => (
        <>
            {label}
            {isOptional && (
                <>
                    <Typography component="span" variant="body1" style={{ marginLeft: "4px" }}>
                        -
                    </Typography>
                    <Typography component="span" variant="body1" style={{ margin: "0 4px", fontStyle: "italic" }}>
                        {t("Optional")}
                    </Typography>
                </>
            )}
        </>
    );

    const onKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
        if (enterWillSubmit !== true || typeof onSubmit !== "function") return;
        if (event.key === "Enter" && !event.shiftKey) {
            event.preventDefault();
            onSubmit();
        }
    };


    return <TextField
        label={label ? <LabelWithOptional /> : ""}
        onKeyDown={onKeyDown}
        placeholder={placeholder}
        ref={ref as RefObject<HTMLInputElement>}
        InputLabelProps={(label && placeholder) ? { shrink: true } : {}}
        variant="outlined"
        {...props}
    />;
};

export const TranslatedTextInput = ({
    language,
    name,
    ...props
}: TranslatedTextInputProps) => {
    const [field, meta, helpers] = useField("translations");
    const { value, error, touched } = getTranslationData(field, meta, language);

    const handleBlur = (event) => {
        field.onBlur(event);
    };

    const handleChange = (event) => {
        handleTranslationChange(field, meta, helpers, event, language);
    };

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
};
