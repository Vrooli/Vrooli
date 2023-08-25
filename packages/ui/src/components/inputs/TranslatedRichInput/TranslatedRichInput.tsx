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
    const { value, error, touched } = getTranslationData(field, meta, language);

    const handleBlur = (event) => {
        field.onBlur(event);
    };

    const handleChange = (newText: string) => {
        handleTranslationChange(field, meta, helpers, { target: { name, value: newText } }, language);
    };

    return (
        <RichInputBase
            {...props}
            name={name}
            value={value?.[name] || ""}
            error={touched?.[name] && Boolean(error?.[name])}
            helperText={touched?.[name] && error?.[name]}
            onBlur={handleBlur}
            onChange={handleChange}
        />
    );
};
