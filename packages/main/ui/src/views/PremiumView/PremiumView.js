import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { CompleteIcon } from "@local/icons";
import { Box, Button, Paper, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, useTheme } from "@mui/material";
import { loadStripe } from "@stripe/stripe-js";
import { useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { TopBar } from "../../components/navigation/TopBar/TopBar";
import { getCurrentUser } from "../../utils/authentication/session";
import { PubSub } from "../../utils/pubsub";
import { parseSearchParams, stringifySearchParams, useLocation } from "../../utils/route";
import { SessionContext } from "../../utils/SessionContext";
const stripePromise = loadStripe(process.env.VITE_STRIPE_PUBLISHABLE_KEY ?? "");
function createData(feature, nonPremium, premium) {
    return { feature, nonPremium, premium };
}
const rows = [
    createData("Routines and processes", "Up to 25 private, 100 public", "Very high limits"),
    createData("*AI-related features", "GPT API key required", "✔️"),
    createData("*Human and bot collaboration", "GPT API key required", "✔️"),
    createData("*Customize and replicate public organizations", "✔️", "✔️"),
    createData("*Copy and adapt public routines", "✔️", "✔️"),
    createData("*Analytics dashboard", "Essential", "Advanced"),
    createData("Customizable user experience", "✔️", "✔️"),
    createData("Community sharing", "✔️", "✔️"),
    createData("*Data import/export", "✔️", "✔️"),
    createData("Industry-standard templates", "✔️", "✔️"),
    createData("Mobile app", "✔️", "✔️"),
    createData("*Tutorial resources", "✔️", "✔️"),
    createData("Community support", "✔️", "✔️"),
    createData("Updates and improvements", "✔️", "Early access"),
    createData("Task management", "✔️", "✔️"),
    createData("*Calendar integration", "✔️", "✔️"),
    createData("Customized notifications", "✔️", "✔️"),
    createData("Provide feedback", "✔️", "✔️"),
    createData("Ad-free experience", "❌", "✔️"),
    createData("Enhanced focus modes", "❌", "✔️"),
    createData("*Premium API access", "❌", "✔️"),
];
export const PremiumView = ({ display = "page", }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const { hasPremium, id: userId } = useMemo(() => getCurrentUser(session), [session]);
    useEffect(() => {
        const searchParams = parseSearchParams();
        if (typeof searchParams.status === "string") {
            switch (searchParams.status) {
                case "success":
                    PubSub.get().publishAlertDialog({
                        messageKey: "PremiumPaymentSuccess",
                        buttons: [{
                                labelKey: "Ok",
                                onClick: () => setLocation(LINKS.Home),
                            }],
                    });
                    PubSub.get().publishCelebration();
                    break;
                case "canceled":
                    break;
            }
        }
    }, [setLocation]);
    const startCheckout = async (variant) => {
        const stripe = await stripePromise;
        if (!stripe) {
            console.error("Stripe failed to load");
            return;
        }
        let uri;
        const endpoint = "create-checkout-session";
        if (window.location.host.includes("localhost") || window.location.host.includes("192.168.0.")) {
            uri = `http://${window.location.hostname}:${process.env.VITE_PORT_SERVER ?? "5329"}/api/${endpoint}`;
        }
        else {
            uri = process.env.VITE_SERVER_URL && process.env.VITE_SERVER_URL.length > 0 ?
                `${process.env.VITE_SERVER_URL}/v2` :
                `http://${process.env.VITE_SITE_IP}:${process.env.VITE_PORT_SERVER ?? "5329"}/api/${endpoint}`;
        }
        try {
            const response = await fetch(uri, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    userId,
                    variant,
                }),
            });
            const session = await response.json();
            const result = await stripe.redirectToCheckout({
                sessionId: session.id,
            });
            if (result.error) {
                PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error", data: result.error });
            }
        }
        catch (error) {
            PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error", data: error });
        }
    };
    return (_jsxs(_Fragment, { children: [_jsx(TopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Premium",
                } }), _jsxs(Stack, { direction: "column", spacing: 4, mt: 2, mb: 2, justifyContent: "center", alignItems: "center", children: [_jsx(Typography, { variant: "h6", sx: { textAlign: "center", margin: 2 }, children: t("PremiumIntro1") }), _jsx(Typography, { variant: "h6", sx: { textAlign: "center", margin: 2 }, children: t("PremiumIntro2") }), _jsxs(Box, { children: [_jsxs(Typography, { variant: "body2", mb: 1, sx: { textAlign: "left", color: palette.error.main }, children: [_jsx("span", { style: { fontSize: "x-large" }, children: "*" }), " Coming soon"] }), _jsx(TableContainer, { component: Paper, sx: { maxWidth: 800 }, children: _jsxs(Table, { "aria-label": "features table", children: [_jsx(TableHead, { sx: { background: palette.primary.light }, children: _jsxs(TableRow, { children: [_jsx(TableCell, { sx: { color: palette.primary.contrastText }, children: "Feature" }), _jsx(TableCell, { align: "center", sx: { color: palette.primary.contrastText }, children: "Non-Premium" }), _jsx(TableCell, { align: "center", sx: { color: palette.primary.contrastText }, children: "Premium" })] }) }), _jsx(TableBody, { children: rows.map((row) => (_jsxs(TableRow, { children: [_jsx(TableCell, { component: "th", scope: "row", children: row.feature.startsWith("*") ? (_jsxs(_Fragment, { children: [_jsx("span", { style: { color: palette.error.main, fontSize: "x-large" }, children: "*" }), row.feature.slice(1)] })) : (row.feature) }), _jsx(TableCell, { align: "center", children: row.nonPremium === "✔️" ? _jsx(CompleteIcon, { fill: palette.mode === "light" ? palette.secondary.dark : palette.secondary.light }) : row.nonPremium }), _jsx(TableCell, { align: "center", children: row.premium === "✔️" ? _jsx(CompleteIcon, { fill: palette.mode === "light" ? palette.secondary.dark : palette.secondary.light }) : row.premium })] }, row.feature))) })] }) })] }), _jsx(Typography, { variant: "body1", sx: { textAlign: "center" }, children: "Upgrade to Vrooli Premium for an ad-free experience, AI-powered integrations, advanced analytics tools, and more. Maximize your potential \u2013 go Premium now!" }), userId && _jsxs(Stack, { direction: "column", spacing: 2, m: 2, sx: { width: "100%", maxWidth: "700px" }, children: [_jsx(Button, { disabled: hasPremium, fullWidth: true, onClick: () => { startCheckout("yearly"); }, children: "$149.99/year" }), _jsx(Button, { disabled: hasPremium, fullWidth: true, onClick: () => { startCheckout("monthly"); }, children: "$14.99/month" }), hasPremium && (_jsx(Typography, { variant: "body1", sx: { textAlign: "center" }, children: "You already have premium!" })), _jsx(Button, { fullWidth: true, onClick: () => { startCheckout("donation"); }, children: "One-time donation (no premium)" })] }), !userId && _jsx(Button, { fullWidth: true, onClick: () => { setLocation(`${LINKS.Start}${stringifySearchParams({ redirect: LINKS.Premium })}`); }, children: "Log in to upgrade" })] })] }));
};
//# sourceMappingURL=PremiumView.js.map