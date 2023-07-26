import { LINKS } from "@local/shared";
import { Box, Button, CircularProgress, Stack, Typography, useTheme } from "@mui/material";
import { loadStripe } from "@stripe/stripe-js";
import { fetchData } from "api";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { CancelIcon, OpenInNewIcon } from "icons";
import { useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { openLink, parseSearchParams, useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { useDisplayServerError } from "utils/hooks/useDisplayServerError";
import { useFetch } from "utils/hooks/useFetch";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { SettingsPaymentViewProps } from "../types";

const stripePromise = loadStripe(import.meta.env.VITE_STRIPE_PUBLISHABLE_KEY);

/** Stripe page to update plan and payment details */
const PORTAL_LINK = (environment: "development" | "production") =>
    environment === "development" ?
        "https://billing.stripe.com/p/login/test_7sI3gd8Ihd8j5lCeUU" :
        "https://billing.stripe.com/p/login/00g7vW7H2adJ7Xq3cc";

export const SettingsPaymentView = ({
    display = "page",
    onClose,
    zIndex,
}: SettingsPaymentViewProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { hasPremium, id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const [loading, setLoading] = useState(false);

    const { data: prices, errors } = useFetch<undefined, { monthly: number, yearly: number }>({
        endpoint: "/premium-prices",
        method: "GET",
        omitRestBase: true,
    });
    useDisplayServerError(errors);

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

    /** Creates stripe checkout session and redirects to checkout page */
    const startCheckout = async (variant: "yearly" | "monthly" | "donation") => {
        setLoading(true);
        // Initialize Stripe
        const stripe = await stripePromise;
        if (!stripe) {
            console.error("Stripe failed to load");
            PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error" });
            return;
        }
        // Call server to create checkout session
        await fetchData({
            endpoint: "/create-checkout-session",
            inputs: {
                userId,
                variant,
            },
            method: "POST",
            omitRestBase: true,
        }).then(async (session: any) => {
            const result = await stripe.redirectToCheckout({
                sessionId: session.id,
            });
            if (result.error) {
                PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error", data: result.error });
            }
        }).catch((error) => {
            PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error", data: error });
        }).finally(() => {
            setLoading(false);
        });
    };

    /** Creates stripe portal session and redirects to portal page */
    const redirectToCustomerPortal = async () => {
        setLoading(true);
        // Call server to create portal session
        await fetchData({
            endpoint: "/create-portal-session",
            inputs: {
                userId,
            },
            method: "POST",
            omitRestBase: true,
        }).then(async (portalSession: any) => {
            // Redirect to portal page
            window.location.href = portalSession.url;
        }).catch((error) => {
            PubSub.get().publishSnack({ messageKey: "ErrorUnknown", severity: "Error", data: error });
        }).finally(() => {
            setLoading(false);
        });
    };

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Payment", { count: 1 })}
                zIndex={zIndex}
            />
            <Stack direction="row" mt={2}>
                <SettingsList />
                <Box m="auto">
                    <Typography variant="h6" textAlign="center">{t(hasPremium ? "AlreadyHavePremium" : "DoNotHavePremium")}</Typography>
                    <Stack direction="column" spacing={2}>
                        <Button
                            color="secondary"
                            fullWidth
                            startIcon={<OpenInNewIcon />}
                            onClick={() => openLink(setLocation, LINKS.Premium)}
                            variant={!hasPremium ? "outlined" : "contained"}
                            sx={{ marginTop: 2, marginBottom: 2 }}
                        >{t("ViewBenefits")}</Button>
                        {!hasPremium && <Button
                            fullWidth
                            onClick={() => { startCheckout("yearly"); }}
                            startIcon={loading ? <CircularProgress size={24} sx={{ color: "white" }} /> : undefined}
                            variant="contained"
                        >
                            <Box display="flex" justifyContent="center" alignItems="center" width="100%">
                                ${(prices?.yearly ?? 0) / 100}/{t("Year")}
                                <Box component="span" fontStyle="italic" color="green" pl={1}>
                                    {t("BestDeal")}
                                </Box>
                            </Box>
                        </Button>}
                        {!hasPremium && <Button
                            fullWidth
                            onClick={() => { startCheckout("monthly"); }}
                            startIcon={loading ? <CircularProgress size={24} sx={{ color: "white" }} /> : undefined}
                            variant="outlined"
                        >${(prices?.monthly ?? 0) / 100}/{t("Month")}</Button>}
                        {hasPremium && <Button
                            fullWidth
                            onClick={redirectToCustomerPortal}
                            startIcon={loading ? <CircularProgress size={24} sx={{ color: "white" }} /> : undefined}
                            variant="outlined"
                        >Change Plan</Button>}
                        <Button
                            fullWidth
                            onClick={() => { startCheckout("donation"); }}
                            startIcon={loading ? <CircularProgress size={24} sx={{ color: "white" }} /> : undefined}
                            variant="outlined"
                        >{t("DonationButton")}</Button>
                        {hasPremium && <Button
                            fullWidth
                            onClick={redirectToCustomerPortal}
                            startIcon={loading ? <CircularProgress size={24} sx={{ color: "white" }} /> : <CancelIcon />}
                            variant="outlined"
                            sx={{ color: palette.error.main, borderColor: palette.error.main, marginTop: "48px!important" }}
                        >Cancel Premium</Button>}
                    </Stack>
                </Box>
            </Stack>
        </>
    );
};
