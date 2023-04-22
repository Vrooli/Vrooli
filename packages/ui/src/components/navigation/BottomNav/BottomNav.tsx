import { BottomNavigation, useTheme } from "@mui/material";
import { useContext } from "react";
import { useKeyboardOpen } from "utils/hooks/useKeyboardOpen";
import { actionsToBottomNav, getUserActions } from "utils/navigation/userActions";
import { useLocation } from "utils/route";
import { SessionContext } from "utils/SessionContext";

export const BottomNav = () => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();

    let actions = actionsToBottomNav({
        actions: getUserActions({ session }),
        setLocation,
    });

    // Hide the nav if the keyboard is open. This is because fixed bottom navs 
    // will appear above the keyboard on Android for some reason.
    const invisible = useKeyboardOpen();

    if (invisible) return null;
    return (
        <BottomNavigation
            id="bottom-nav"
            showLabels
            sx={{
                background: palette.primary.dark,
                position: "fixed",
                zIndex: 6,
                bottom: 0,
                // env variables are used to account for iOS nav bar, notches, etc.
                paddingBottom: "env(safe-area-inset-bottom)",
                paddingLeft: "calc(4px + env(safe-area-inset-left))",
                paddingRight: "calc(4px + env(safe-area-inset-right))",
                height: "calc(56px + env(safe-area-inset-bottom))",
                width: "100%",
                display: { xs: "flex", md: "none" },
            }}
        >
            {actions}
        </BottomNavigation>
    );
}