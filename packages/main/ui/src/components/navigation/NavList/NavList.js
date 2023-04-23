import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { LogInIcon, ProfileIcon } from "@local/icons";
import { Button, Container, IconButton, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { checkIfLoggedIn, getCurrentUser } from "../../../utils/authentication/session";
import { useWindowSize } from "../../../utils/hooks/useWindowSize";
import { actionsToMenu, ACTION_TAGS, getUserActions } from "../../../utils/navigation/userActions";
import { openLink, useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
import { PopupMenu } from "../../buttons/PopupMenu/PopupMenu";
import { AccountMenu } from "../../dialogs/AccountMenu/AccountMenu";
import { ContactInfo } from "../ContactInfo/ContactInfo";
const navItemStyle = (palette) => ({
    background: "transparent",
    color: palette.primary.contrastText,
    textTransform: "none",
    fontSize: "1.4em",
    "&:hover": {
        color: palette.secondary.light,
    },
});
export const NavList = () => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { breakpoints, palette } = useTheme();
    const [, setLocation] = useLocation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);
    const nav_actions = useMemo(() => getUserActions({ session, exclude: [ACTION_TAGS.Home, ACTION_TAGS.LogIn] }), [session]);
    const [accountMenuAnchor, setAccountMenuAnchor] = useState(null);
    const openAccountMenu = useCallback((ev) => {
        ev.stopPropagation();
        setAccountMenuAnchor(ev.currentTarget);
    }, [setAccountMenuAnchor]);
    const closeAccountMenu = useCallback((ev) => {
        ev.stopPropagation();
        setAccountMenuAnchor(null);
    }, []);
    return (_jsxs(Container, { sx: {
            display: "flex",
            height: "100%",
            paddingBottom: "0",
            paddingRight: "0 !important",
            right: "0",
        }, children: [!isMobile && !getCurrentUser(session).id && _jsx(PopupMenu, { text: t("Contact"), variant: "text", size: "large", sx: navItemStyle(palette), children: _jsx(ContactInfo, {}) }), _jsx(AccountMenu, { anchorEl: accountMenuAnchor, onClose: closeAccountMenu }), !isMobile && actionsToMenu({
                actions: nav_actions,
                setLocation,
                sx: navItemStyle(palette),
            }), !checkIfLoggedIn(session) && (_jsx(Button, { href: LINKS.Start, onClick: (e) => { e.preventDefault(); openLink(setLocation, LINKS.Start); }, startIcon: _jsx(LogInIcon, {}), sx: {
                    background: "#387e30",
                    borderRadius: "10px",
                    whiteSpace: "nowrap",
                    fontSize: { xs: "0px", sm: "1em", md: "1.4em" },
                    "& .MuiButton-startIcon": {
                        marginLeft: { xs: "0px", sm: "-4px" },
                        marginRight: { xs: "0px", sm: "8px" },
                    },
                }, children: t("LogIn") })), checkIfLoggedIn(session) && (_jsx(IconButton, { color: "inherit", onClick: openAccountMenu, "aria-label": "profile", children: _jsx(ProfileIcon, { fill: 'white', width: '40px', height: '40px' }) }))] }));
};
//# sourceMappingURL=NavList.js.map