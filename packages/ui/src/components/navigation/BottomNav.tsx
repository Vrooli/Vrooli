/**
 * Bottom navigation component for mobile devices
 * 
 * This component has been migrated to Tailwind CSS for improved performance.
 * It uses CSS variables for theming to maintain compatibility with the MUI theme system.
 */
import { LINKS } from "@vrooli/shared";
import { useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { SessionContext } from "../../contexts/session.js";
import { useKeyboardOpen } from "../../hooks/useKeyboardOpen.js";
import { Icon } from "../../icons/Icons.js";
import { openLink } from "../../route/openLink.js";
import { useLocation } from "../../route/router.js";
import { checkIfLoggedIn } from "../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { getUserActions } from "../../utils/navigation/userActions.js";
import { cn } from "../../utils/tailwind-theme.js";
import {
    badgeClasses,
    badgeWrapperClasses,
    bottomNavClasses,
    navActionClasses,
    navActionLabelClasses,
} from "./bottomNavStyles.js";

/**
 * Hide the bottom nav in the following cases:
 * - Keyboard is open (fixed bottom navs appear above the keyboard on Android)
 * - User is logged in at the home page ("/")
 * - User is logged in and at the chat page ("/chat")
 */
export function useIsBottomNavVisible() {
    const session = useContext(SessionContext);
    const isLoggedIn = checkIfLoggedIn(session);
    const isKeyboardOpen = useKeyboardOpen();
    const [location] = useLocation();
    const pathname = location?.pathname || "/";

    const shouldHideNav = isKeyboardOpen
        || (isLoggedIn && (pathname === LINKS.Home))
        || pathname.startsWith(LINKS.Chat);

    return !shouldHideNav;
}

/**
 * Custom Badge component for notification counts
 */
function NotificationBadge({ count, children }: { count: number; children: React.ReactNode }) {
    if (count === 0) return <>{children}</>;
    
    return (
        <div className={badgeWrapperClasses}>
            {children}
            <span className={badgeClasses}>
                {count > 99 ? "99+" : count}
            </span>
        </div>
    );
}

export function BottomNav() {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const isLoggedIn = checkIfLoggedIn(session);
    const isVisible = useIsBottomNavVisible();

    const actions = getUserActions({ session });

    const handleNavActionClick = useCallback((link: string) => (e: React.MouseEvent) => {
        e.preventDefault();
        // Check if link is different from current location
        const shouldScroll = link === window.location.pathname;
        // If same, scroll to top of page instead of navigating
        if (shouldScroll) window.scrollTo({ top: 0, behavior: "smooth" });
        // Otherwise, navigate to link
        else openLink(setLocation, link);
    }, [setLocation]);

    if (!isVisible) return null;
    
    return (
        <nav
            id={ELEMENT_IDS.BottomNav}
            className={bottomNavClasses}
        >
            {actions.map(({ label, value, iconInfo, link, numNotifications }) => (
                <a
                    key={value}
                    href={link}
                    onClick={handleNavActionClick(link)}
                    className={navActionClasses}
                    aria-label={t(label, { count: 2 })}
                >
                    <NotificationBadge count={numNotifications}>
                        <Icon
                            info={iconInfo}
                            decorative
                            size={24}
                            className="tw-text-current"
                        />
                    </NotificationBadge>
                    {!isLoggedIn && (
                        <span className={navActionLabelClasses}>
                            {t(label, { count: 2 })}
                        </span>
                    )}
                </a>
            ))}
        </nav>
    );
}
