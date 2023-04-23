import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { CopyIcon } from "@local/icons";
import { Box, IconButton, Stack, Tooltip, Typography } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { HelpButton } from "../../../buttons/HelpButton/HelpButton";
import { GeneratedInputComponent } from "../GeneratedInputComponent/GeneratedInputComponent";
export const GeneratedInputComponentWithLabel = ({ copyInput, disabled, fieldData, index, onUpload, textPrimary, zIndex, }) => {
    console.log("rendering input component with label");
    const { t } = useTranslation();
    const inputComponent = useMemo(() => {
        _jsx(GeneratedInputComponent, { fieldData: fieldData, disabled: false, index: index, onUpload: () => { }, zIndex: zIndex });
    }, [fieldData, index, zIndex]);
    return (_jsx(Box, { sx: {
            paddingTop: 1,
            paddingBottom: 1,
            borderRadius: 1,
        }, children: _jsxs(_Fragment, { children: [_jsxs(Stack, { direction: "row", spacing: 0, sx: { alignItems: "center" }, children: [_jsx(Tooltip, { title: t("CopyToClipboard"), children: _jsx(IconButton, { onClick: () => copyInput && copyInput(fieldData.fieldName), children: _jsx(CopyIcon, { fill: textPrimary }) }) }), _jsx(Typography, { variant: "h6", sx: { color: textPrimary }, children: fieldData.label ?? (index && `Input ${index + 1}`) ?? t("Input") }), fieldData.helpText && _jsx(HelpButton, { markdown: fieldData.helpText })] }), inputComponent] }) }, index));
};
//# sourceMappingURL=GeneratedInputComponentWithLabel.js.map