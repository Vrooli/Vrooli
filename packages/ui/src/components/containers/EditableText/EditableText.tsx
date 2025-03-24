import { isOfType } from "@local/shared";
import { Typography } from "@mui/material";
import { Field, useField } from "formik";
import { RichInput, TranslatedRichInput } from "../../inputs/RichInput/RichInput.js";
import { TextInput, TranslatedTextInput } from "../../inputs/TextInput/TextInput.js";
import { MarkdownDisplay } from "../../text/MarkdownDisplay.js";
import { EditTextComponent, EditableTextProps, PropsByComponentType } from "../types.js";

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
            {isEditing && component === "Markdown" && <RichInput name="name" {...(props as PropsByComponentType["Markdown"])} />}
            {isEditing && component === "TranslatedMarkdown" && <TranslatedRichInput name="name" {...(props as PropsByComponentType["TranslatedMarkdown"])} />}
            {isEditing && component === "TranslatedTextInput" && <TranslatedTextInput name={name} {...(props as PropsByComponentType["TranslatedTextInput"])} />}
            {isEditing && component === "TextInput" && <Field name={name} as={TextInput} {...(props as PropsByComponentType["TextInput"])} />}
            {/* Display components */}
            {!isEditing && isOfType(component, "Markdown", "TranslatedMarkdown") && <MarkdownDisplay variant={variant} content={field.value} />}
            {!isEditing && isOfType("TextInput", "TranslatedTextInput") && <Typography variant={variant}>{field.value}</Typography>}
        </>
    );
}
