import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { DeleteIcon } from "@local/icons";
import { IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback } from "react";
import { multiLineEllipsis } from "../../../../../styles";
const Status = {
    NotVerified: "#a71c2d",
    Verified: "#19972b",
};
export function PushListItem({ handleDelete, handleUpdate, index, data, }) {
    const { palette } = useTheme();
    const onDelete = useCallback(() => {
        handleDelete(data);
    }, [data, handleDelete]);
    return (_jsxs(ListItem, { disablePadding: true, sx: {
            display: "flex",
            padding: 1,
        }, children: [_jsx(Stack, { direction: "column", spacing: 1, pl: 2, sx: { marginRight: "auto" }, children: _jsx(ListItemText, { primary: data.name ?? data.id, sx: { ...multiLineEllipsis(1) } }) }), _jsx(Stack, { direction: "row", spacing: 1, children: _jsx(Tooltip, { title: "Delete Email", children: _jsx(IconButton, { onClick: onDelete, children: _jsx(DeleteIcon, { fill: palette.secondary.main }) }) }) })] }));
}
//# sourceMappingURL=PushListItem.js.map