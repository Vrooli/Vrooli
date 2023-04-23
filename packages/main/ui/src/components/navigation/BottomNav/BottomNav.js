import { jsx as _jsx } from "react/jsx-runtime";
import { BottomNavigation, useTheme } from "@mui/material";
import { useContext } from "react";
import { useKeyboardOpen } from "../../../utils/hooks/useKeyboardOpen";
import { actionsToBottomNav, getUserActions } from "../../../utils/navigation/userActions";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
export const BottomNav = () => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const actions = actionsToBottomNav({
        actions: getUserActions({ session }),
        setLocation,
    });
    const invisible = useKeyboardOpen();
    if (invisible)
        return null;
    return (_jsx(BottomNavigation, { id: "bottom-nav", showLabels: true, sx: {
            background: palette.primary.dark,
            position: "fixed",
            zIndex: 6,
            bottom: 0,
            paddingBottom: "env(safe-area-inset-bottom)",
            paddingLeft: "calc(4px + env(safe-area-inset-left))",
            paddingRight: "calc(4px + env(safe-area-inset-right))",
            height: "calc(56px + env(safe-area-inset-bottom))",
            width: "100%",
            display: { xs: "flex", md: "none" },
        }, children: actions }));
};
//# sourceMappingURL=BottomNav.js.map