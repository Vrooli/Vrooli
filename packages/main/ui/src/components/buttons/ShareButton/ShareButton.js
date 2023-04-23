import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { ShareIcon } from "@local/icons";
import { IconButton, Tooltip, useTheme } from "@mui/material";
import { useCallback, useState } from "react";
import { ShareObjectDialog } from "../../dialogs/ShareObjectDialog/ShareObjectDialog";
export const ShareButton = ({ object, zIndex, }) => {
    const { palette } = useTheme();
    const [open, setOpen] = useState(false);
    const openDialog = useCallback(() => { setOpen(true); }, []);
    const closeDialog = useCallback(() => { setOpen(false); }, []);
    return (_jsxs(_Fragment, { children: [_jsx(ShareObjectDialog, { object: object, open: open, onClose: closeDialog, zIndex: zIndex + 1 }), _jsx(Tooltip, { title: "Share", children: _jsx(IconButton, { "aria-label": "Share", size: "small", onClick: openDialog, children: _jsx(ShareIcon, { fill: palette.background.textSecondary }) }) })] }));
};
//# sourceMappingURL=ShareButton.js.map