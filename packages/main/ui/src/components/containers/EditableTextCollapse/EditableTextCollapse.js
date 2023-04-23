import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { isOfType } from "@local/utils";
import { TextField, Typography, useTheme } from "@mui/material";
import { Field, useField } from "formik";
import Markdown from "markdown-to-jsx";
import { linkColors } from "../../../styles";
import { MarkdownInput } from "../../inputs/MarkdownInput/MarkdownInput";
import { TranslatedMarkdownInput } from "../../inputs/TranslatedMarkdownInput/TranslatedMarkdownInput";
import { TranslatedTextField } from "../../inputs/TranslatedTextField/TranslatedTextField";
import { ContentCollapse } from "../ContentCollapse/ContentCollapse";
export function EditableTextCollapse({ component, helpText, isEditing, isOpen, name, onOpenChange, props, showOnNoText, title, variant, }) {
    const { palette } = useTheme();
    const [field] = useField(name);
    console.log("editabletextcollapse field", field, name, component);
    if (!isEditing && (!field.value || field.value.trim().length === 0) && !showOnNoText)
        return null;
    return (_jsxs(ContentCollapse, { helpText: helpText, isOpen: isOpen, onOpenChange: onOpenChange, title: title, sxs: {
            root: { ...linkColors(palette) },
        }, children: [isEditing && component === "Markdown" && _jsx(MarkdownInput, { name: name, ...props }), isEditing && component === "TranslatedMarkdown" && _jsx(TranslatedMarkdownInput, { name: name, ...props }), isEditing && component === "TranslatedTextField" && _jsx(TranslatedTextField, { name: name, ...props }), isEditing && component === "TextField" && _jsx(Field, { name: name, as: TextField, ...props }), !isEditing && isOfType(component, "Markdown", "TranslatedMarkdown") && _jsx(Markdown, { variant: variant, children: field.value }), !isEditing && isOfType("TextField", "TranslatedTextField") && _jsx(Typography, { variant: variant, children: field.value })] }));
}
//# sourceMappingURL=EditableTextCollapse.js.map