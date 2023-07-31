import { isOfType } from "@local/shared";
import { TextField, Typography, useTheme } from "@mui/material";
import { MarkdownInput } from "components/inputs/MarkdownInput/MarkdownInput";
import { TranslatedMarkdownInput } from "components/inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { MarkdownDisplay } from "components/text/MarkdownDisplay/MarkdownDisplay";
import { Field, useField } from "formik";
import { linkColors } from "styles";
import { ContentCollapse } from "../ContentCollapse/ContentCollapse";
import { EditableTextCollapseProps, EditTextComponent, PropsByComponentType } from "../types";

/**
 * A text collapse that supports editing mode, either with 
 * a TextField or MarkdownInput
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
    zIndex,
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
            zIndex={zIndex}
        >
            {/* Editing components */}
            {isEditing && component === "Markdown" && <MarkdownInput name={name} zIndex={zIndex} {...(props as PropsByComponentType["Markdown"])} />}
            {isEditing && component === "TranslatedMarkdown" && <TranslatedMarkdownInput name={name} zIndex={zIndex} {...(props as PropsByComponentType["TranslatedMarkdown"])} />}
            {isEditing && component === "TranslatedTextField" && <TranslatedTextField name={name} {...(props as PropsByComponentType["TranslatedTextField"])} />}
            {isEditing && component === "TextField" && <Field name={name} as={TextField} {...(props as PropsByComponentType["TextField"])} />}
            {/* Display components */}
            {!isEditing && isOfType(component, "Markdown", "TranslatedMarkdown") && <MarkdownDisplay variant={variant} content={field.value} zIndex={zIndex} />}
            {!isEditing && isOfType("TextField", "TranslatedTextField") && <Typography variant={variant}>{field.value}</Typography>}
        </ContentCollapse>
    );
}
