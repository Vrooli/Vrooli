import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { ReportIcon } from "@local/icons";
import { IconButton, Tooltip, useTheme } from "@mui/material";
import { useCallback, useState } from "react";
import { ReportDialog } from "../../dialogs/ReportDialog/ReportDialog";
export const ReportButton = ({ forId, reportFor, zIndex, }) => {
    const { palette } = useTheme();
    const [open, setOpen] = useState(false);
    const openDialog = useCallback(() => { setOpen(true); }, []);
    const closeDialog = useCallback(() => { setOpen(false); }, []);
    return (_jsxs(_Fragment, { children: [_jsx(ReportDialog, { forId: forId, onClose: closeDialog, open: open, reportFor: reportFor, zIndex: zIndex + 1 }), _jsx(Tooltip, { title: "Report", children: _jsx(IconButton, { "aria-label": "Report", size: "small", onClick: openDialog, children: _jsx(ReportIcon, { fill: palette.background.textSecondary }) }) })] }));
};
//# sourceMappingURL=ReportButton.js.map