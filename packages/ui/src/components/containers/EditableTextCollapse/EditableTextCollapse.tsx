import { isOfType } from "@local/shared";
import { Typography, useTheme } from "@mui/material";
import { RichInput, TranslatedRichInput } from "components/inputs/RichInput/RichInput";
import { TextInput, TranslatedTextInput } from "components/inputs/TextInput/TextInput";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Field, useField } from "formik";
import { linkColors } from "styles";
import { ContentCollapse } from "../ContentCollapse/ContentCollapse";
import { EditTextComponent, EditableTextCollapseProps, PropsByComponentType } from "../types";

/**
 * A text collapse that supports editing mode, either with 
 * a TextInput or RichInput
 */
export function EditableTextCollapse<T extends EditTextComponent>({
    component,
    helpText,
    isEditing,
    isOpen,
    name,
    onOpenChange,
    props,
    showOnNoText,
    title,
    variant,
}: EditableTextCollapseProps<T>) {
    const { palette } = useTheme();
    const [field] = useField(name);

    if (!isEditing && (!field.value || field.value.trim().length === 0) && !showOnNoText) return null; //TODO won't work with translations
    return (
        <ContentCollapse
            helpText={helpText}
            isOpen={isOpen}
            onOpenChange={onOpenChange}
            title={title}
            sxs={{
                root: { ...linkColors(palette) },
            }}
        >
            {/* Editing components */}
            {isEditing && component === "Markdown" && <RichInput name={name} {...(props as PropsByComponentType["Markdown"])} />}
            {isEditing && component === "TranslatedMarkdown" && <TranslatedRichInput name={name} {...(props as PropsByComponentType["TranslatedMarkdown"])} />}
            {isEditing && component === "TranslatedTextInput" && <TranslatedTextInput name={name} {...(props as PropsByComponentType["TranslatedTextInput"])} />}
            {isEditing && component === "TextInput" && <Field name={name} as={TextInput} {...(props as PropsByComponentType["TextInput"])} />}
            {/* Display components */}
            {!isEditing && isOfType(component, "Markdown", "TranslatedMarkdown") && <MarkdownDisplay variant={variant} content={field.value} />}
            {!isEditing && isOfType("TextInput", "TranslatedTextInput") && <Typography variant={variant}>{field.value}</Typography>}
        </ContentCollapse>
    );
}
