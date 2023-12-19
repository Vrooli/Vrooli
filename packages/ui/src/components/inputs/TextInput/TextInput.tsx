import { TextField, Typography } from "@mui/material";
import { useTranslation } from "react-i18next";
import { TextInputProps } from "../types";

export const TextInput = ({
    enterWillSubmit,
    label,
    isOptional,
    onSubmit,
    placeholder,
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
        InputLabelProps={(label && placeholder) ? { shrink: true } : {}}
        variant="outlined"
        {...props}
    />;
};
