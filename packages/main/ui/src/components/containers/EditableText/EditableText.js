import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { isOfType } from "@local/utils";
import { TextField, Typography } from "@mui/material";
import { Field, useField } from "formik";
import Markdown from "markdown-to-jsx";
import { MarkdownInput } from "../../inputs/MarkdownInput/MarkdownInput";
import { TranslatedMarkdownInput } from "../../inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "../../inputs/TranslatedTextField/TranslatedTextField";
export function EditableText({ component, isEditing, name, props, showOnNoText, variant, }) {
    const [field] = useField(component.startsWith("Translated") ? `translations.${name}` : name);
    if (!isEditing && (!field.value || field.value.trim().length === 0) && !showOnNoText)
        return null;
    return (_jsxs(_Fragment, { children: [isEditing && component === "Markdown" && _jsx(MarkdownInput, { name: "name", ...props }), isEditing && component === "TranslatedMarkdown" && _jsx(TranslatedMarkdownInput, { name: "name", ...props }), isEditing && component === "TranslatedTextField" && _jsx(TranslatedTextField, { name: name, ...props }), isEditing && component === "TextField" && _jsx(Field, { name: name, as: TextField, ...props }), !isEditing && isOfType(component, "Markdown", "TranslatedMarkdown") && _jsx(Markdown, { variant: variant, children: field.value }), !isEditing && isOfType("TextField", "TranslatedTextField") && _jsx(Typography, { variant: variant, children: field.value })] }));
}
//# sourceMappingURL=EditableText.js.map