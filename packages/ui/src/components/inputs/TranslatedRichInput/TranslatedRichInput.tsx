import { getDotNotationValue, setDotNotationValue } from "@local/shared";
import { useField } from "formik";
import { getTranslationData, handleTranslationChange } from "utils/display/translationTools";
import { RichInputBase } from "../RichInputBase/RichInputBase";
import { TranslatedRichInputProps } from "../types";

export const TranslatedRichInput = ({
    language,
    name,
    ...props
}: TranslatedRichInputProps) => {
    const [field, meta, helpers] = useField("translations");
    const translationData = getTranslationData(field, meta, language);

    const fieldValue = getDotNotationValue(translationData.value, name);
    const fieldError = getDotNotationValue(translationData.error, name);
    const fieldTouched = getDotNotationValue(translationData.touched, name);

    const handleBlur = (event) => {
        field.onBlur(event);
    };

    const handleChange = (newText: string) => {
        // Only use dot notation if the name has a dot in it
        if (name.includes(".")) {
            const updatedValue = setDotNotationValue(translationData.value ?? {}, name, newText);
            helpers.setValue(field.value.map((translation) => {
                if (translation.language === language) return updatedValue;
                return translation;
            }));
        }
        else handleTranslationChange(field, meta, helpers, { target: { name, value: newText } }, language);
    };

    return (
        <RichInputBase
            {...props}
            name={name}
            value={fieldValue || ""}
            error={fieldTouched && Boolean(fieldError)}
            helperText={fieldTouched && fieldError}
            onBlur={handleBlur}
            onChange={handleChange}
        />
    );
};
