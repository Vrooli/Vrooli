import { useField } from "formik";
import { getTranslationData, handleTranslationChange } from "utils/display/translationTools";
import { TextInput } from "../TextInput/TextInput";
import { TranslatedTextInputProps } from "../types";

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
