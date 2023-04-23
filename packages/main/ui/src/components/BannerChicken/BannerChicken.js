import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { useContext, useMemo, useState } from "react";
import { getCurrentUser } from "../../utils/authentication/session";
import { SessionContext } from "../../utils/SessionContext";
export const BannerChicken = () => {
    const session = useContext(SessionContext);
    const [adDisplayed, setAdDisplayed] = useState(null);
    const adFrequency = useMemo(() => {
        const user = getCurrentUser(session);
        if (!user)
            return "full";
        if (user.hasPremium)
            return "none";
        if (session?.isLoggedIn)
            return "half";
        return "full";
    }, [session]);
    const shouldDisplayAd = useMemo(() => {
        if (adFrequency === "none")
            return false;
        if (adFrequency === "full")
            return true;
        const random = Math.random();
        return random < 0.5;
    }, [adFrequency]);
    if (!shouldDisplayAd)
        return null;
    return (_jsxs(_Fragment, { children: [_jsx("script", { async: true, src: `https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=ca-${process.env.VITE_GOOGLE_ADSENSE_PUBLISHER_ID}`, crossOrigin: "anonymous", onLoad: () => setAdDisplayed(true), onError: () => setAdDisplayed(false) }), _jsx("ins", { className: "adsbygoogle", style: { display: "block" }, "data-adtest": window.location.host.includes("localhost"), "data-ad-client": `ca-${process.env.VITE_GOOGLE_ADSENSE_PUBLISHER_ID}`, "data-ad-slot": "9649766873", "data-ad-format": "auto", "data-full-width-responsive": "true" })] }));
};
//# sourceMappingURL=BannerChicken.js.map