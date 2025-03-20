import { BottomNavigation, useTheme } from "@mui/material";
import { useContext, useMemo } from "react";
import { SessionContext } from "../../contexts.js";
import { useKeyboardOpen } from "../../hooks/useKeyboardOpen.js";
import { useLocation } from "../../route/router.js";
import { bottomNavHeight } from "../../styles.js";
import { Z_INDEX } from "../../utils/consts.js";
import { actionsToBottomNav, getUserActions } from "../../utils/navigation/userActions.js";

export function BottomNav() {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();

    const actions = actionsToBottomNav({
        actions: getUserActions({ session }),
        setLocation,
    });

    const bottomNavSx = useMemo(() => ({
        background: palette.primary.dark,
        position: "fixed",
        zIndex: Z_INDEX.BottomNav,
        bottom: 0,
        // env variables are used to account for iOS nav bar, notches, etc.
        paddingBottom: "env(safe-area-inset-bottom)",
        paddingLeft: "calc(4px + env(safe-area-inset-left))",
        paddingRight: "calc(4px + env(safe-area-inset-right))",
        height: bottomNavHeight,
        width: "100%",
        display: { xs: "flex", md: "none" },
    }), [palette.primary.dark]);

    // Hide the nav if the keyboard is open. This is because fixed bottom navs 
    // will appear above the keyboard on Android for some reason.
    const invisible = useKeyboardOpen();

    if (invisible) return null;
    return (
        <BottomNavigation
            id="bottom-nav"
            sx={bottomNavSx}
        >
            {actions}
        </BottomNavigation>
    );
}
