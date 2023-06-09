import { CompleteIcon, LINKS, parseSearchParams, stringifySearchParams, useLocation } from "@local/shared";
import { Box, Button, CircularProgress, Paper, Stack, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Typography, useTheme } from "@mui/material";
import { loadStripe } from "@stripe/stripe-js";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { PremiumViewProps } from "../types";

const stripePromise = loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY);

// Features comparison table data
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

export const PremiumView = ({
    display = "page",
    onClose,
}: PremiumViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const { hasPremium, id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // This view is also used to check for successful/failed payments, 
    // so we need to check URL search params
    useEffect(() => {
        const searchParams = parseSearchParams();
        // Check for status
        if (typeof searchParams.status === "string") {
            switch (searchParams.status) {
                case "success":
                    // Show alert and confetti
                    PubSub.get().publishAlertDialog({
                        messageKey: "PremiumPaymentSuccess",
                        buttons: [{
                            labelKey: "Ok",
                            // Redirect to home page
                            onClick: () => setLocation(LINKS.Home),
                        }],
                    });
                    PubSub.get().publishCelebration();
                    break;
                case "canceled":
                    // Do nothing
                    break;
            }
        }
    }, [setLocation]);

    const [loading, setLoading] = useState(false);

    /**
     * Creates stripe checkout session and redirects to checkout page
     */
    const startCheckout = async (variant: "yearly" | "monthly" | "donation") => {
        setLoading(true);
        // Initialize Stripe
        const stripe = await stripePromise;
        if (!stripe) {
            console.error("Stripe failed to load");
            return;
        }
        // Determine server URL
        // Determine origin of API server
        let uri: string;
        // If running locally
        const endpoint = "create-checkout-session";
        if (window.location.host.includes("localhost") || window.location.host.includes("192.168.0.")) {
            uri = `http://${window.location.hostname}:${import.meta.env.VITE_PORT_SERVER ?? "5329"}/api/${endpoint}`;
        }
        // If running on server
        else {
            uri = import.meta.env.VITE_SERVER_URL && import.meta.env.VITE_SERVER_URL.length > 0 ?
                `${import.meta.env.VITE_SERVER_URL}/v2` :
                `http://${import.meta.env.VITE_SITE_IP}:${import.meta.env.VITE_PORT_SERVER ?? "5329"}/api/${endpoint}`;
        }
        // Create checkout session
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
        } catch (error) {
            PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error", data: error });
        } finally {
            setLoading(false);
        }
    };

    // TODO convert MaxObjects to list of limit increases 
    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={t("Premium")}
            />
            <Stack direction="column" spacing={4} mt={2} mb={2} justifyContent="center" alignItems="center">
                {/* Introduction to premium */}
                <Typography variant="h6" sx={{ textAlign: "center", margin: 2 }}>{t("PremiumIntro1")}</Typography>
                <Typography variant="h6" sx={{ textAlign: "center", margin: 2 }}>{t("PremiumIntro2")}</Typography>
                {/* Main features as table */}
                <Box>
                    <Typography variant="body2" mb={1} sx={{ textAlign: "left", color: palette.error.main }}><span style={{ fontSize: "x-large" }}>*</span> Coming soon</Typography>
                    <TableContainer component={Paper} sx={{ maxWidth: 800 }}>
                        <Table aria-label="features table">
                            <TableHead sx={{ background: palette.primary.light }}>
                                <TableRow>
                                    <TableCell sx={{ color: palette.primary.contrastText }}>Feature</TableCell>
                                    <TableCell align="center" sx={{ color: palette.primary.contrastText }}>Non-Premium</TableCell>
                                    <TableCell align="center" sx={{ color: palette.primary.contrastText }}>Premium</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {rows.map((row) => (
                                    <TableRow key={row.feature}>
                                        <TableCell component="th" scope="row">
                                            {row.feature.startsWith("*") ? (
                                                <>
                                                    <span style={{ color: palette.error.main, fontSize: "x-large" }}>*</span>
                                                    {row.feature.slice(1)}
                                                </>
                                            ) : (
                                                row.feature
                                            )}
                                        </TableCell>
                                        <TableCell align="center">
                                            {row.nonPremium === "✔️" ? <CompleteIcon fill={palette.mode === "light" ? palette.secondary.dark : palette.secondary.light} /> : row.nonPremium}
                                        </TableCell>
                                        <TableCell align="center">
                                            {row.premium === "✔️" ? <CompleteIcon fill={palette.mode === "light" ? palette.secondary.dark : palette.secondary.light} /> : row.premium}
                                        </TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </TableContainer>
                </Box>
                <Typography variant="body1" sx={{ textAlign: "center" }}>
                    Upgrade to Vrooli Premium for an ad-free experience, AI-powered integrations, advanced analytics tools, and more. Maximize your potential – go Premium now!
                </Typography>
                {/* Link to open popup that displays all limit increases */}
                {/* TODO */}
                {/* Button row for different subscriptions, with donation option at bottom. Should have dialog like on matthalloran.info */}
                {userId && <Stack direction="column" spacing={2} m={2} sx={{ width: "100%", maxWidth: "700px" }}>
                    <Button
                        disabled={hasPremium}
                        fullWidth
                        onClick={() => { startCheckout("yearly"); }}
                        startIcon={loading ? <CircularProgress size={24} sx={{ color: "white" }} /> : undefined}
                    >
                        <Box display="flex" justifyContent="center" alignItems="center" width="100%">
                            $149.99/year
                            <Box component="span" fontStyle="italic" color="green" pl={1}>
                                Best Deal!
                            </Box>
                        </Box>
                    </Button>
                    <Button
                        disabled={hasPremium}
                        fullWidth
                        onClick={() => { startCheckout("monthly"); }}
                        startIcon={loading ? <CircularProgress size={24} sx={{ color: "white" }} /> : undefined}
                    >$14.99/month</Button>
                    {hasPremium && (
                        // TODO need way to change from monthly to yearly and vice versa
                        <Typography variant="body1" sx={{ textAlign: "center" }}>
                            You already have premium!
                        </Typography>
                    )}
                    <Button
                        fullWidth
                        onClick={() => { startCheckout("donation"); }}
                        startIcon={loading ? <CircularProgress size={24} sx={{ color: "white" }} /> : undefined}
                    >One-time donation (no premium)</Button>
                </Stack>}
                {/* If not logged in, button to log in first */}
                {!userId && <Button
                    fullWidth
                    onClick={() => { setLocation(`${LINKS.Start}${stringifySearchParams({ redirect: LINKS.Premium })}`); }}
                >Log in to upgrade</Button>}
            </Stack>
        </>
    );
};
