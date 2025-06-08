import { useEffect, useMemo } from "react";

/**
 * Routes that should never display ads.
 * 
 * NOTE: These are hard-coded instead of using the LINKS object because 
 * we need to minimize imports to reduce the bundle size.
 */
const BLACKLIST_ROUTES = [
    "/about",
    "/auth/forgot-password",
    "/",
    "/auth/login",
    "/pro",
    "/auth/password-reset",
    "/auth/signup",
] as const;

const HALF = 0.5;

type BannerChickenProps = {
    backgroundColor: string;
    isMobile: boolean;
    isLoggedIn: boolean;
    hasPremium: boolean;
}

/**
 * Displays a banner ad the bottom of the screen, above the BottomNav. 
 * Uses session data to display no ads for premium users, and less ads if logged in.
 * 
 * NOTE 1: If we call this "BannerAd", ad blockers will cause the whole bundle to break. 
 * Hence the name "BannerChicken".
 * 
 * NOTE 2: This is setup to use a minimal amount of imports. We want this to be the only 
 * file in its bundle, in case it is blocked by an ad blocker.
 * 
 * NOTE 3: Session data is passed as props to avoid importing heavy dependencies.
 */
export function BannerChicken({
    backgroundColor,
    isMobile,
    isLoggedIn,
    hasPremium,
}: BannerChickenProps) {

    const adFrequency = useMemo(() => {
        if (!isLoggedIn) return "full";
        if (hasPremium) return "none";
        return "half";
    }, [isLoggedIn, hasPremium]);

    const shouldDisplayAd = useMemo(() => {
        // Don't display ads on certain routes
        if (BLACKLIST_ROUTES.includes(window.location.pathname as any)) return false;
        if (adFrequency === "none") return false;
        if (adFrequency === "full") return true;
        // Pick a random number between 0 and 1
        const random = Math.random();
        // If the random number is less than 0.5, display the ad
        return random < HALF;
    }, [adFrequency]);

    useEffect(function renderAdEffect() {
        if (!shouldDisplayAd || !process.env.VITE_GOOGLE_ADSENSE_PUBLISHER_ID) {
            console.warn("Conditions not met for displaying ads.");
            return;
        }

        const script = document.createElement("script");
        script.src = `https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-${process.env.VITE_GOOGLE_ADSENSE_PUBLISHER_ID}`;
        script.async = true;
        script.crossOrigin = "anonymous";
        document.body.appendChild(script);

        return () => {
            document.body.removeChild(script);
        };
    }, [shouldDisplayAd]);

    const insStyle = useMemo(() => ({
        position: "absolute",
        // Display above the BottomNav, which is only displayed on mobile
        bottom: isMobile ? "var(--page-padding-bottom)" : "env(safe-area-inset-bottom)",
        display: "block",
        background: backgroundColor,
    } as const), [isMobile, backgroundColor]);

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
