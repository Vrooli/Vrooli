import Typography from "@mui/material/Typography";
import { isOfType } from "@vrooli/shared";
import { Field, useField } from "formik";
import { type EditTextComponent, type EditableTextProps, type PropsByComponentType } from "../containers/types.js";
import { MarkdownDisplay } from "../text/MarkdownDisplay.js";
import { AdvancedInput, TranslatedAdvancedInput } from "./AdvancedInput/AdvancedInput.js";
import { TextInput, TranslatedTextInput } from "./TextInput/TextInput.js";

export function EditableText<T extends EditTextComponent>({
    component,
    isEditing,
    name,
    props,
    showOnNoText,
    variant,
}: EditableTextProps<T>) {
    const [field] = useField(component.startsWith("Translated") ? `translations.${name}` : name);

    if (!isEditing && (!field.value || field.value.trim().length === 0) && !showOnNoText) return null;
    return (
        <>
            {/* Editing components */}
            {isEditing && component === "Markdown" && <AdvancedInput name="name" {...(props as PropsByComponentType["Markdown"])} />}
            {isEditing && component === "TranslatedMarkdown" && <TranslatedAdvancedInput name="name" language={props.language || "en"} {...(props as PropsByComponentType["TranslatedMarkdown"])} />}
            {isEditing && component === "TranslatedTextInput" && <TranslatedTextInput name={name} {...(props as PropsByComponentType["TranslatedTextInput"])} />}
            {isEditing && component === "TextInput" && <Field name={name} as={TextInput} {...(props as PropsByComponentType["TextInput"])} />}
            {/* Display components */}
            {!isEditing && isOfType(component, "Markdown", "TranslatedMarkdown") && <MarkdownDisplay variant={variant} content={field.value} />}
            {!isEditing && isOfType("TextInput", "TranslatedTextInput") && <Typography variant={variant}>{field.value}</Typography>}
        </>
    );
}
