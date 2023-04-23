import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { CompleteIcon, DeleteIcon } from "@local/icons";
import { Box, IconButton, ListItem, ListItemText, Stack, Tooltip, useTheme } from "@mui/material";
import { useCallback } from "react";
import { multiLineEllipsis } from "../../../../../styles";
const Status = {
    NotVerified: "#a71c2d",
    Verified: "#19972b",
};
export function EmailListItem({ handleDelete, handleVerify, index, data, }) {
    const { palette } = useTheme();
    const onDelete = useCallback(() => {
        handleDelete(data);
    }, [data, handleDelete]);
    const onVerify = useCallback(() => {
        handleVerify(data);
    }, [data, handleVerify]);
    return (_jsxs(ListItem, { disablePadding: true, sx: {
            display: "flex",
            padding: 1,
        }, children: [_jsxs(Stack, { direction: "column", spacing: 1, pl: 2, sx: { marginRight: "auto" }, children: [_jsx(ListItemText, { primary: data.emailAddress, sx: { ...multiLineEllipsis(1) } }), _jsx(Box, { sx: {
                            borderRadius: 1,
                            border: `2px solid ${data.verified ? Status.Verified : Status.NotVerified}`,
                            color: data.verified ? Status.Verified : Status.NotVerified,
                            height: "fit-content",
                            fontWeight: "bold",
                            marginTop: "auto",
                            marginBottom: "auto",
                            textAlign: "center",
                            padding: 0.25,
                            width: "fit-content",
                        }, children: data.verified ? "Verified" : "Not Verified" })] }), _jsxs(Stack, { direction: "row", spacing: 1, children: [!data.verified && _jsx(Tooltip, { title: "Resend email verification", children: _jsx(IconButton, { onClick: onVerify, children: _jsx(CompleteIcon, { fill: Status.NotVerified }) }) }), _jsx(Tooltip, { title: "Delete Email", children: _jsx(IconButton, { onClick: onDelete, children: _jsx(DeleteIcon, { fill: palette.secondary.main }) }) })] })] }));
}
//# sourceMappingURL=EmailListItem.js.map