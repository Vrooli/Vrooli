import { LINKS, PaymentType } from "@local/shared";
import { loadStripe } from "@stripe/stripe-js";
import { fetchData } from "api";
import { SessionContext } from "contexts/SessionContext";
import { useContext, useEffect, useMemo, useState } from "react";
import { parseSearchParams, removeSearchParams, useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { useFetch } from "./useFetch";

const stripePromise = loadStripe(process.env.VITE_STRIPE_PUBLISHABLE_KEY ?? "");

export const useStripe = () => {
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const currentUser = useMemo(() => getCurrentUser(session), [session]);
    const [loading, setLoading] = useState(false);

    const { data: prices } = useFetch<undefined, { monthly: number, yearly: number }>({
        endpoint: "/premium-prices",
        method: "GET",
        omitRestBase: true,
    });

    // This view is also used to check for successful/failed payments, 
    // so we need to check URL search params
    useEffect(() => {
        const searchParams = parseSearchParams();
        // Remove status from URL
        removeSearchParams(setLocation, ["status"]);
        // Check for status
        if (typeof searchParams.status === "string") {
            switch (searchParams.status) {
                case "success": {
                    // Retrieve the paymentType from localStorage
                    const paymentType = window.localStorage.getItem("paymentType");
                    // Show alert and confetti
                    PubSub.get().publish("alertDialog", {
                        messageKey: paymentType === PaymentType.Donation ? "DonationPaymentSuccess" : "PremiumPaymentSuccess",
                        buttons: [{
                            labelKey: "Reload",
                            // Reload the page
                            onClick: () => {
                                setLocation(LINKS.Home);
                                window.location.reload();
                            },
                        }, {
                            labelKey: "Ok",
                            // Redirect to home page
                            onClick: () => setLocation(LINKS.Home),
                        }],
                    });
                    PubSub.get().publish("celebration");
                    break;
                }
                case "canceled":
                    // Do nothing
                    break;
            }
        }
    }, [setLocation]);

    /** Creates stripe checkout session and redirects to checkout page */
    const startCheckout = async (variant: PaymentType) => {
        setLoading(true);
        PubSub.get().publish("loading", true);
        // Initialize Stripe
        const stripe = await stripePromise;
        if (!stripe) {
            console.error("Stripe failed to load");
            PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error" });
            return;
        }
        // Call server to create checkout session
        await fetchData({
            endpoint: "/create-checkout-session",
            inputs: {
                userId: currentUser.id,
                variant,
            },
            method: "POST",
            omitRestBase: true,
        }).then(async (session: any) => {
            // Set the variant in localStorage
            window.localStorage.setItem("paymentType", variant);
            const result = await stripe.redirectToCheckout({
                sessionId: session.id,
            });
            if (result.error) {
                PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error", data: result.error });
            }
        }).catch((error) => {
            PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error", data: error });
        }).finally(() => {
            setLoading(false);
            PubSub.get().publish("loading", false);
        });
    };

    /** Creates stripe portal session and redirects to portal page */
    const redirectToCustomerPortal = async () => {
        setLoading(true);
        PubSub.get().publish("loading", true);
        // Call server to create portal session
        await fetchData({
            endpoint: "/create-portal-session",
            inputs: {
                userId: currentUser.id,
                returnUrl: window.location.href,
            },
            method: "POST",
            omitRestBase: true,
        }).then(async (portalSession: any) => {
            console.log("portalSession", portalSession);
            if (portalSession.error) {
                PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error", data: portalSession });
            } else {
                // Redirect to portal page
                window.location.href = portalSession.url;
            }
            // Redirect to portal page
            // window.location.href = portalSession.url;
        }).catch((error) => {
            PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error", data: error });
        }).finally(() => {
            setLoading(false);
            PubSub.get().publish("loading", false);
        });
    };

    return {
        currentUser,
        loading,
        prices,
        startCheckout,
        redirectToCustomerPortal,
    } as const;
};
