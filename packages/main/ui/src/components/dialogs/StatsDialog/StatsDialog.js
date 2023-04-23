import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { Box } from "@mui/material";
import { TopBar } from "../../navigation/TopBar/TopBar";
import { StatsCompact } from "../../text/StatsCompact/StatsCompact";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const titleId = "stats-object-dialog-title";
export const StatsDialog = ({ handleObjectUpdate, isOpen, object, onClose, zIndex, }) => {
    return (_jsxs(LargeDialog, { id: "object-stats-dialog", onClose: onClose, isOpen: isOpen, titleId: titleId, zIndex: zIndex, children: [_jsx(TopBar, { display: "dialog", onClose: onClose, titleData: { titleId, titleKey: "Share" } }), _jsx(Box, { sx: { padding: 2 }, children: _jsx(StatsCompact, { handleObjectUpdate: handleObjectUpdate, loading: false, object: object }) })] }));
};
//# sourceMappingURL=StatsDialog.js.map