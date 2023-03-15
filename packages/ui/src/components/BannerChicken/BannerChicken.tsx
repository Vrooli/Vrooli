import { BannerChickenProps } from "components/types"
import { useMemo, useState } from "react"
import { getCurrentUser } from "utils/authentication"

/**
 * Displays a banner ad the bottom of the screen, above the BottomNav. 
 * Uses session to display no ads for premium users, and less ads if logged in.
 * 
 * NOTE: If we call this "BannerAd", ad blockers will cause the whole site to break. 
 * Hence the name "BannerChicken".
 */
export const BannerChicken = ({
    session
}: BannerChickenProps) => {
    const [adDisplayed, setAdDisplayed] = useState<boolean | null>(null);

    const adFrequency = useMemo(() => {
        const user = getCurrentUser(session);
        if (!user) return 'full';
        if (user.hasPremium) return 'none';
        if (session?.isLoggedIn) return 'half';
        return 'full';
    }, [session])

    const shouldDisplayAd = useMemo(() => {
        if (adFrequency === 'none') return false;
        if (adFrequency === 'full') return true;
        // Pick a random number between 0 and 1
        const random = Math.random();
        // If the random number is less than 0.5, display the ad
        return random < 0.5;
    }, [adFrequency])

    if (!shouldDisplayAd) return null;
    return (
        <>
            {/* AdSense script */}
            <script
                async
                src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-pub-7109871095821789"
                crossOrigin="anonymous"
                onLoad={() => setAdDisplayed(true)}
                onError={() => setAdDisplayed(false)}
            ></script>
            <ins className="adsbygoogle"
                style={{ display: 'block' }}
                // Disable ads for local development. Ads only work on live domains. 
                // You can use a test domain to test ads before deploying.
                data-adtest={window.location.host.includes('localhost')}
                data-ad-client="ca-pub-7109871095821789"
                data-ad-slot="9649766873"
                data-ad-format="auto"
                data-full-width-responsive="true"></ins>
        </>
    )
}