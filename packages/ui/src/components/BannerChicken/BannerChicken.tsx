import { LINKS } from "@local/shared";
import { useTheme } from "@mui/material/styles";
import { SessionContext } from "contexts";
import { useWindowSize } from "hooks/useWindowSize";
import { useContext, useEffect, useMemo } from "react";
import { useLocation } from "route";
import { pagePaddingBottom } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";

const HALF = 0.5;
const BLACKLIST_ROUTES = [
    LINKS.About,
    LINKS.ForgotPassword,
    LINKS.Home,
    LINKS.Login,
    LINKS.Pro,
    LINKS.ResetPassword,
    LINKS.Signup,
] as string[];

/**
 * Displays a banner ad the bottom of the screen, above the BottomNav. 
 * Uses session to display no ads for premium users, and less ads if logged in.
 * 
 * NOTE: If we call this "BannerAd", ad blockers will cause the whole bundle to break. 
 * Hence the name "BannerChicken".
 */
export function BannerChicken() {
    const session = useContext(SessionContext);
    const { breakpoints, palette } = useTheme();
    const [location] = useLocation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.md);

    const adFrequency = useMemo(() => {
        const user = getCurrentUser(session);
        if (!user) return "full";
        if (user.hasPremium) return "none";
        if (session?.isLoggedIn) return "half";
        return "full";
    }, [session]);

    const shouldDisplayAd = useMemo(() => {
        // Don't display ads on certain routes
        if (BLACKLIST_ROUTES.includes(location.pathname)) return false;
        if (adFrequency === "none") return false;
        if (adFrequency === "full") return true;
        // Pick a random number between 0 and 1
        const random = Math.random();
        // If the random number is less than 0.5, display the ad
        return random < HALF;
    }, [adFrequency, location.pathname]);

    useEffect(function renderAdEffect() {
        if (!shouldDisplayAd || !process.env.VITE_GOOGLE_ADSENSE_PUBLISHER_ID) {
            console.warn("Conditions not met for displaying ads.");
            return;
        }

        const script = document.createElement("script");
        script.src = `https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-${process.env.VITE_GOOGLE_ADSENSE_PUBLISHER_ID}`;
        script.async = true;
        script.crossOrigin = "anonymous";
        script.onload = () => PubSub.get().publish("banner", { isDisplayed: true });
        script.onerror = () => PubSub.get().publish("banner", { isDisplayed: false });
        document.body.appendChild(script);

        return () => {
            document.body.removeChild(script);
        };
    }, [shouldDisplayAd]);

    const insStyle = useMemo(() => ({
        position: "absolute",
        // Display above the BottomNav, which is only displayed on mobile
        bottom: isMobile ? pagePaddingBottom : "env(safe-area-inset-bottom)",
        display: "block",
        background: palette.background.default,
    } as const), [isMobile, palette.background.default]);

    if (!shouldDisplayAd) return null;

    return (
        <ins
            className="adsbygoogle"
            style={insStyle}
            // Disable ads for local development. Ads only work on live domains. 
            // You can use a test domain to test ads before deploying.
            data-adtest={window.location.host.includes("localhost")}
            data-ad-client={`ca-${process.env.VITE_GOOGLE_ADSENSE_PUBLISHER_ID}`}
            data-ad-slot="9649766873"
            data-ad-format="auto"
            data-full-width-responsive="true"
        />
    );
}
