import { jsx as _jsx } from "react/jsx-runtime";
import { Stack } from "@mui/material";
export const SideActionButtons = ({ children, display, hasGridActions = false, isLeftHanded, sx, zIndex, }) => {
    const gridActionsHeight = hasGridActions ? "70px" : "0px";
    const bottomNavHeight = display === "page" ? "56px" : "0px";
    return (_jsx(Stack, { direction: "row", spacing: 2, sx: {
            position: "absolute",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            zIndex,
            bottom: 0,
            left: isLeftHanded ? 0 : "auto",
            right: isLeftHanded ? "auto" : 0,
            marginBottom: {
                xs: `calc(${bottomNavHeight} + ${gridActionsHeight} + 16px + env(safe-area-inset-bottom))`,
                md: `calc(${gridActionsHeight} + 16px + env(safe-area-inset-bottom))`,
            },
            marginLeft: "calc(16px + env(safe-area-inset-left))",
            marginRight: "calc(16px + env(safe-area-inset-right))",
            height: "calc(64px + env(safe-area-inset-bottom))",
            ...(sx ?? {}),
        }, children: children }));
};
//# sourceMappingURL=SideActionButtons.js.map