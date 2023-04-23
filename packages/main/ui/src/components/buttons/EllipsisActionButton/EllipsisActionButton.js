import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { CloseIcon, EllipsisIcon } from "@local/icons";
import { Collapse, Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { ColorIconButton } from "../ColorIconButton/ColorIconButton";
export function EllipsisActionButton({ children, }) {
    const { palette } = useTheme();
    const [isOpen, setIsOpen] = useState(false);
    const toggleOpen = useCallback(() => setIsOpen(!isOpen), [isOpen]);
    const Icon = useMemo(() => {
        if (isOpen)
            return CloseIcon;
        return EllipsisIcon;
    }, [isOpen]);
    return (_jsxs(_Fragment, { children: [_jsx(Collapse, { orientation: "horizontal", in: isOpen, children: _jsx(Stack, { spacing: { xs: 1, sm: 1.5, md: 2 }, direction: "row", alignItems: "center", justifyContent: "center", p: 1, sx: {
                        overflowX: "auto",
                    }, children: children }) }), _jsx(Tooltip, { title: "More options", placement: "top", children: _jsx(ColorIconButton, { "aria-label": "run-routine", background: palette.secondary.main, onClick: toggleOpen, sx: {
                        padding: 0,
                        width: "54px",
                        height: "54px",
                    }, children: _jsx(Icon, { fill: palette.secondary.contrastText, width: '36px', height: '36px' }) }) })] }));
}
//# sourceMappingURL=EllipsisActionButton.js.map