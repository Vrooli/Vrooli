import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { ExpandLessIcon, ExpandMoreIcon } from "@local/icons";
import { Box, Collapse, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { HelpButton } from "../../buttons/HelpButton/HelpButton";
export function ContentCollapse({ children, helpText, id, isOpen = true, onOpenChange, sxs, title, titleComponent, titleKey, titleVariables, }) {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [internalIsOpen, setInternalIsOpen] = useState(isOpen);
    useEffect(() => {
        setInternalIsOpen(isOpen);
    }, [isOpen]);
    const toggleOpen = useCallback(() => {
        setInternalIsOpen(!internalIsOpen);
        if (onOpenChange) {
            onOpenChange(!internalIsOpen);
        }
    }, [internalIsOpen, onOpenChange]);
    const fillColor = sxs?.root?.color ?? (Boolean(children) ? palette.background.textPrimary : palette.background.textSecondary);
    const titleText = useMemo(() => {
        if (titleKey)
            return t(titleKey, { ...titleVariables, defaultValue: title ?? "" });
        if (title)
            return title;
        return "";
    }, [title, titleKey, titleVariables, t]);
    return (_jsxs(Box, { id: id, sx: {
            color: Boolean(children) ? palette.background.textPrimary : palette.background.textSecondary,
            ...(sxs?.root ?? {}),
        }, children: [_jsxs(Stack, { direction: "row", alignItems: "center", sx: sxs?.titleContainer ?? {}, children: [_jsx(Typography, { component: titleComponent ?? "h6", variant: "h6", children: titleText }), helpText && _jsx(HelpButton, { markdown: helpText }), _jsx(IconButton, { id: `toggle-expand-icon-button-${title}`, "aria-label": t(internalIsOpen ? "Collapse" : "Expand"), onClick: toggleOpen, children: internalIsOpen ?
                            _jsx(ExpandMoreIcon, { id: `toggle-expand-icon-${title}`, fill: fillColor }) :
                            _jsx(ExpandLessIcon, { id: `toggle-expand-icon-${title}`, fill: fillColor }) })] }), _jsx(Collapse, { in: internalIsOpen, children: children })] }));
}
//# sourceMappingURL=ContentCollapse.js.map