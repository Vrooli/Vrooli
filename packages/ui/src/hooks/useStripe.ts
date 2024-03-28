import { CheckSubscriptionParams, CheckSubscriptionResponse, CommonKey, CreateCheckoutSessionParams, CreateCheckoutSessionResponse, LINKS, PaymentType, SubscriptionPricesResponse } from "@local/shared";
import { loadStripe } from "@stripe/stripe-js";
import { fetchData } from "api";
import { SessionContext } from "contexts/SessionContext";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { addSearchParams, openLink, parseSearchParams, removeSearchParams, useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { useFetch } from "./useFetch";

const stripePromise = loadStripe(process.env.VITE_STRIPE_PUBLISHABLE_KEY ?? "");

const paymentTypeToSuccessMessage: Record<PaymentType, CommonKey> = {
    [PaymentType.Credits]: "CreditsPaymentSuccess",
    [PaymentType.Donation]: "DonationPaymentSuccess",
    [PaymentType.PremiumMonthly]: "ProPaymentSuccess",
    [PaymentType.PremiumYearly]: "ProPaymentSuccess",
}

export const useStripe = () => {
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const currentUser = useMemo(() => getCurrentUser(session), [session]);
    const [loading, setLoading] = useState(false);

    const { data: prices } = useFetch<undefined, SubscriptionPricesResponse>({
        endpoint: "/subscription-prices",
        method: "GET",
        omitRestBase: true,
    });

    // We use the URL to check for successful/failed payments
    const handleUrlParams = useCallback(() => {
        const { paymentType, status } = parseSearchParams();
        removeSearchParams(setLocation, ["status"]);
        // Check for status
        if (typeof status === "string") {
            switch (status) {
                case "success": {
                    // Show alert and confetti
                    PubSub.get().publish("alertDialog", {
                        messageKey: typeof paymentType === "string"
                            ? paymentTypeToSuccessMessage[paymentType as PaymentType]
                            : "PaymentSuccess",
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
                    console.info("Payment canceled");
                    break;
            }
        }
    }, [setLocation]);

    // Automatically check URL params on mount
    useEffect(() => {
        handleUrlParams();
    }, [handleUrlParams]);

    /** Creates stripe checkout session and redirects to checkout page */
    const startCheckout = async (variant: PaymentType) => {
        // If not logged in and trying to get premium, redirect to signup page
        if (!currentUser.id && variant !== PaymentType.Donation) {
            openLink(setLocation, LINKS.Signup, { redirect: window.location.pathname });
            return;
        }

        const finish = () => {
            setLoading(false);
            PubSub.get().publish("loading", false);
        };

        setLoading(true);
        PubSub.get().publish("loading", true);

        // Make sure stripe is loaded, so a Stripe object is in the window. 
        // I believe this is used for security purposes.
        const stripe = await stripePromise;
        if (!stripe) {
            PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error", data: "Stripe failed to load" });
            finish();
            return;
        }

        // Call server to create checkout session
        await fetchData<CreateCheckoutSessionParams, CreateCheckoutSessionResponse>({
            endpoint: "/create-checkout-session",
            inputs: {
                userId: currentUser.id,
                variant,
            },
            method: "POST",
            omitRestBase: true,
        }).then(async ({ data }) => {
            if (!data) {
                throw new Error("Failed to create checkout session");
            }
            // Redirect to checkout page
            window.location.href = data.url;
        }).catch((error) => {
            PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error", data: error });
        }).finally(() => {
            finish();
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

    /** Checks if the user has a subscription that never went through */
    const checkSubscription = async () => {
        // Must be logged in to check subscription
        if (!currentUser.id) {
            openLink(setLocation, LINKS.Signup, { redirect: window.location.pathname });
            return;
        }

        setLoading(true);
        PubSub.get().publish("loading", true);
        // Call server to check subscription status
        await fetchData<CheckSubscriptionParams, CheckSubscriptionResponse>({
            endpoint: "/check-subscription",
            inputs: {
                userId: currentUser.id,
            },
            method: "POST",
            omitRestBase: true,
        }).then(async ({ data }) => {
            console.log("checkSubscription response", data);
            if (!data) {
                throw new Error("Did not receive subscription status");
            }
            switch (data.status) {
                // Let user know if they already have pro
                case "already_subscribed": {
                    PubSub.get().publish("snack", { messageKey: "AlreadyHavePro", severity: "Info" });
                    break;
                }
                // If we fixed their pro status, update the URL and trigger handleUrlParams
                case "now_subscribed": {
                    addSearchParams(setLocation, { paymentType: data.paymentType, status: "success" });
                    handleUrlParams();
                    break;
                }
                // If the user is still not subscribed, show an alert with more information
                case "not_subscribed": {
                    PubSub.get().publish("alertDialog", {
                        messageKey: "StillNoPro",
                        buttons: [{
                            labelKey: "Ok",
                            onClick: () => { },
                        }],
                    });
                    break;
                }
            }
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
        checkSubscription,
    } as const;
};
