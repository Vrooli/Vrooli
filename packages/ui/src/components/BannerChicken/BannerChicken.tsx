import { SessionContext } from "contexts/SessionContext";
import { useContext, useEffect, useMemo } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";

/**
 * Displays a banner ad the bottom of the screen, above the BottomNav. 
 * Uses session to display no ads for premium users, and less ads if logged in.
 * 
 * NOTE: If we call this "BannerAd", ad blockers will cause the whole bundle to break. 
 * Hence the name "BannerChicken".
 */
export const BannerChicken = () => {
    const session = useContext(SessionContext);

    const adFrequency = useMemo(() => {
        const user = getCurrentUser(session);
        if (!user) return "full";
        if (user.hasPremium) return "none";
        if (session?.isLoggedIn) return "half";
        return "full";
    }, [session]);

    const shouldDisplayAd = useMemo(() => {
        if (adFrequency === "none") return false;
        if (adFrequency === "full") return true;
        // Pick a random number between 0 and 1
        const random = Math.random();
        // If the random number is less than 0.5, display the ad
        return random < 0.5;
    }, [adFrequency]);

    useEffect(() => {
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

    if (!shouldDisplayAd) return null;

    return (
        <ins className="adsbygoogle"
            style={{ display: "block" }}
            // Disable ads for local development. Ads only work on live domains. 
            // You can use a test domain to test ads before deploying.
            data-adtest={window.location.host.includes("localhost")}
            data-ad-client={`ca-${process.env.VITE_GOOGLE_ADSENSE_PUBLISHER_ID}`}
            data-ad-slot="9649766873"
            data-ad-format="auto"
            data-full-width-responsive="true"></ins>
    );
};
