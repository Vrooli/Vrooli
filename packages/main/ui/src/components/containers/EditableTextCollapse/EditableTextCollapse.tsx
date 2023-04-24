import { isOfType } from "@local/shared;";
import { TextField, Typography, useTheme } from "@mui/material";
import { Field, useField } from "formik";
import Markdown from "markdown-to-jsx";
import { linkColors } from "../../../styles";
import { MarkdownInput } from "../../inputs/MarkdownInput/MarkdownInput";
import { TranslatedMarkdownInput } from "../../inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "../../inputs/TranslatedTextField/TranslatedTextField";
import { ContentCollapse } from "../ContentCollapse/ContentCollapse";
import { EditTextComponent, EditableTextCollapseProps, PropsByComponentType } from "../types";

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
}: EditableTextCollapseProps<T>) {
    const { palette } = useTheme();
    const [field] = useField(name);
    console.log("editabletextcollapse field", field, name, component);

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
            {isEditing && component === "Markdown" && <MarkdownInput name={name} {...(props as PropsByComponentType["Markdown"])} />}
            {isEditing && component === "TranslatedMarkdown" && <TranslatedMarkdownInput name={name} {...(props as PropsByComponentType["TranslatedMarkdown"])} />}
            {isEditing && component === "TranslatedTextField" && <TranslatedTextField name={name} {...(props as PropsByComponentType["TranslatedTextField"])} />}
            {isEditing && component === "TextField" && <Field name={name} as={TextField} {...(props as PropsByComponentType["TextField"])} />}
            {/* Display components */}
            {!isEditing && isOfType(component, "Markdown", "TranslatedMarkdown") && <Markdown variant={variant}>{field.value}</Markdown>}
            {!isEditing && isOfType("TextField", "TranslatedTextField") && <Typography variant={variant}>{field.value}</Typography>}
        </ContentCollapse>
    );
}
