import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { InvisibleIcon, VisibleIcon } from "@local/icons";
import { Box, IconButton, Stack, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { useField } from "formik";
import Markdown from "markdown-to-jsx";
import { useEffect, useMemo, useState } from "react";
import { Status } from "../../../utils/consts";
import { isJson, jsonHelpText, jsonToMarkdown } from "../../../utils/shape/general";
import { HelpButton } from "../../buttons/HelpButton/HelpButton";
import { StatusButton } from "../../buttons/StatusButton/StatusButton";
export const JsonInput = ({ disabled = false, format, index, minRows = 4, name, placeholder = "", variables, }) => {
    const { palette } = useTheme();
    const [field, meta, helpers] = useField(name);
    useEffect(() => {
        if (disabled)
            return;
        if ((field.value ?? "").length > 0 || !format)
            return;
        const changedFormat = JSON.stringify(format);
        if (!variables) {
            helpers.setValue(changedFormat);
            return;
        }
        for (const [variableName, variableData] of Object.entries(variables)) {
            if (variableData.defaultValue === undefined)
                continue;
        }
    }, [disabled, format, field.value, helpers, variables]);
    useEffect(() => {
    }, [format, variables]);
    const [isPreviewOn, setIsPreviewOn] = useState(false);
    const togglePreview = () => setIsPreviewOn(!isPreviewOn);
    const { previewMarkdown, isValueValid } = useMemo(() => ({
        previewMarkdown: jsonToMarkdown(field.value),
        isValueValid: isJson(field.value),
    }), [field.value]);
    return (_jsxs(Stack, { direction: "column", spacing: 0, children: [_jsxs(Box, { sx: {
                    display: "flex",
                    width: "100%",
                    padding: "0.5rem",
                    borderBottom: "1px solid #e0e0e0",
                    background: palette.primary.light,
                    color: palette.primary.contrastText,
                    borderRadius: "0.5rem 0.5rem 0 0",
                }, children: [_jsx(StatusButton, { status: isValueValid ? Status.Valid : Status.Invalid, messages: isValueValid ? ["JSON is valid"] : ["JSON could not be parsed"], sx: {
                            marginLeft: 1,
                            marginRight: "auto",
                        } }), _jsx(Tooltip, { title: isPreviewOn ? "Preview mode" : "Edit mode", placement: "top", sx: { marginLeft: "auto" }, children: _jsx(IconButton, { size: "small", onClick: togglePreview, children: isPreviewOn ?
                                _jsx(InvisibleIcon, { fill: palette.primary.contrastText }) :
                                _jsx(VisibleIcon, { fill: palette.primary.contrastText }) }) }), _jsx(HelpButton, { markdown: jsonHelpText, sxRoot: { marginRight: 1 } })] }), isPreviewOn ?
                _jsx(Box, { sx: {
                        border: `1px solid ${palette.background.textPrimary}`,
                        color: isValueValid ? palette.background.textPrimary : palette.error.main,
                    }, children: previewMarkdown ?
                        _jsx(Markdown, { children: previewMarkdown }) :
                        _jsx("p", { children: `Error: Invalid JSON - ${field.value}` }) }) :
                _jsx(TextField, { id: `markdown-input-${name}`, autoFocus: index === 0, disabled: disabled, error: meta.touched && !!meta.error, name: name, placeholder: placeholder, rows: minRows, value: field.value ?? "", onBlur: field.onBlur, onChange: field.onChange, tabIndex: index, style: {
                        minWidth: "-webkit-fill-available",
                        maxWidth: "-webkit-fill-available",
                        minHeight: "50px",
                        maxHeight: "800px",
                        background: "transparent",
                        borderColor: (meta.touched && !!meta.error) ? "red" : "unset",
                        borderRadius: "0 0 0.5rem 0.5rem",
                        borderTop: "none",
                        fontFamily: "inherit",
                        fontSize: "inherit",
                    } }), meta.touched && !!meta.error && _jsx(Typography, { variant: "body1", sx: { color: "red" }, children: meta.error })] }));
};
//# sourceMappingURL=JsonInput.js.map