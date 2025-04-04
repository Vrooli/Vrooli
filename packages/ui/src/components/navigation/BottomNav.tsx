import { LINKS } from "@local/shared";
import { Badge, BottomNavigation, BottomNavigationAction, styled, useTheme } from "@mui/material";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../contexts/session.js";
import { useKeyboardOpen } from "../../hooks/useKeyboardOpen.js";
import { Icon } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { bottomNavHeight } from "../../styles.js";
import { checkIfLoggedIn } from "../../utils/authentication/session.js";
import { Z_INDEX } from "../../utils/consts.js";
import { getUserActions } from "../../utils/navigation/userActions.js";

const StyledBottomNavigationAction = styled(BottomNavigationAction)(({ theme }) => ({
    color: theme.palette.primary.contrastText,
    minWidth: "58px", // Default min width is too big for some screens, like closed Galaxy Fold 
}));

export function BottomNav() {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();

    const actions = getUserActions({ session });

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

    const isKeyboardOpen = useKeyboardOpen();

    // Hide the nav in the following cases:
    // - Keyboard is open (fixed bottom navs appear above the keyboard on Android)
    // - User is logged in at the home page ("/")
    // - User is logged in and at the chat page ("/chat")
    const isLoggedIn = checkIfLoggedIn(session);
    const shouldHideNav = isKeyboardOpen
        || (isLoggedIn && (window.location.pathname === LINKS.Home))
        || window.location.pathname.startsWith(LINKS.Chat);

    if (shouldHideNav) return null;
    return (
        <BottomNavigation
            id="bottom-nav"
            sx={bottomNavSx}
        >
            {actions.map(({ label, value, iconInfo, link, numNotifications }) => {
                function handleNavActionClick(e: React.MouseEvent) {
                    e.preventDefault();
                    // Check if link is different from current location
                    const shouldScroll = link === window.location.pathname;
                    // If same, scroll to top of page instead of navigating
                    if (shouldScroll) window.scrollTo({ top: 0, behavior: "smooth" });
                    // Otherwise, navigate to link
                    else openLink(setLocation, link);
                }
                return (
                    <StyledBottomNavigationAction
                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                        // @ts-ignore
                        href={link}
                        icon={<Badge badgeContent={numNotifications} color="error">
                            <Icon
                                decorative
                                fill="primary.contrastText"
                                info={iconInfo}
                                size={24}
                            />
                        </Badge>}
                        key={value}
                        label={t(label, { count: 2 })}
                        onClick={handleNavActionClick}
                        value={value}
                        showLabel={isLoggedIn !== true}
                    />
                );
            })}
        </BottomNavigation>
    );
}
