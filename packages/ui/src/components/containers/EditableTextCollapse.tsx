import { isOfType } from "@local/shared";
import { Typography, useTheme } from "@mui/material";
import { Field, useField } from "formik";
import { useMemo } from "react";
import { linkColors } from "../../styles.js";
import { RichInput, TranslatedRichInput } from "../inputs/RichInput/RichInput.js";
import { TextInput, TranslatedTextInput } from "../inputs/TextInput/TextInput.js";
import { MarkdownDisplay } from "../text/MarkdownDisplay.js";
import { ContentCollapse } from "./ContentCollapse.js";
import { type EditTextComponent, type EditableTextCollapseProps, type PropsByComponentType } from "./types.js";

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

    const collapseStyle = useMemo(function collapseStyleMemo() {
        return {
            root: { ...linkColors(palette) },
        } as const;
    }, [palette]);

    if (!isEditing && (!field.value || field.value.trim().length === 0) && !showOnNoText) return null; //TODO won't work with translations
    return (
        <ContentCollapse
            helpText={helpText}
            isOpen={isOpen}
            onOpenChange={onOpenChange}
            title={title}
            sxs={collapseStyle}
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
