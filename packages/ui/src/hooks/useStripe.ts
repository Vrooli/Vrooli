import { CheckCreditsPaymentParams, CheckCreditsPaymentResponse, CheckSubscriptionParams, CheckSubscriptionResponse, CreateCheckoutSessionParams, CreateCheckoutSessionResponse, CreatePortalSessionParams, CreatePortalSessionResponse, LINKS, PaymentType, StripeEndpoint, SubscriptionPricesResponse, TranslationKeyCommon, parseSearchParams } from "@local/shared";
import { loadStripe } from "@stripe/stripe-js";
import { fetchData } from "api/fetchData";
import { SessionContext } from "contexts";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { addSearchParams, openLink, removeSearchParams, useLocation } from "route";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { useFetch } from "./useFetch";

const stripePromise = loadStripe(process.env.VITE_STRIPE_PUBLISHABLE_KEY ?? "");

const paymentTypeToSuccessMessage: Record<PaymentType, TranslationKeyCommon> = {
    [PaymentType.Credits]: "CreditsPaymentSuccess",
    [PaymentType.Donation]: "DonationPaymentSuccess",
    [PaymentType.PremiumMonthly]: "ProPaymentSuccess",
    [PaymentType.PremiumYearly]: "ProPaymentSuccess",
};

export function useStripe() {
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const currentUser = useMemo(() => getCurrentUser(session), [session]);
    const [loading, setLoading] = useState(false);

    const { data: prices } = useFetch<undefined, SubscriptionPricesResponse>({
        endpoint: StripeEndpoint.SubscriptionPrices,
        method: "GET",
        omitRestBase: true,
    });

    const toggleLoading = useCallback(function toggleLoadingCallback(isLoading: boolean) {
        setLoading(isLoading);
        PubSub.get().publish("loading", isLoading);
    }, []);

    const handleError = useCallback(function handleErrorCallback(error: unknown) {
        PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error", data: error });
    }, []);

    const handleUrlParams = useCallback(() => {
        const { paymentType, status } = parseSearchParams();
        removeSearchParams(setLocation, ["status"]);
        if (status === "success") {
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
        }
        else if (status === "canceled") console.info("Payment canceled");
    }, [setLocation]);

    // Automatically check URL params on mount
    useEffect(() => {
        handleUrlParams();
    }, [handleUrlParams]);

    /** 
     * Creates stripe checkout session and redirects to checkout page 
     * @param variant The type of payment to start
     * @param amount The amount to pay, if variant is "Donation" or "Credits"
     */
    const startCheckout = async (variant: PaymentType, amount?: number) => {
        // If not logged in and trying to get premium, redirect to signup page
        if (!currentUser.id && variant !== PaymentType.Donation) {
            openLink(setLocation, LINKS.Signup, { redirect: window.location.pathname });
            return;
        }

        toggleLoading(true);

        // Make sure stripe is loaded, so a Stripe object is in the window. 
        // I believe this is used for security purposes.
        const stripe = await stripePromise;
        if (!stripe) {
            handleError("Stripe failed to load");
            toggleLoading(false);
            return;
        }

        // Call server to create checkout session
        await fetchData<CreateCheckoutSessionParams, CreateCheckoutSessionResponse>({
            endpoint: StripeEndpoint.CreateCheckoutSession,
            inputs: {
                amount,
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
            handleError(error);
        }).finally(() => {
            toggleLoading(false);
        });
    };

    /** Creates stripe portal session and redirects to portal page */
    const redirectToCustomerPortal = async () => {
        // If not logged in, redirect to login page
        if (!currentUser.id) {
            openLink(setLocation, LINKS.Login, { redirect: window.location.pathname });
            return;
        }

        toggleLoading(true);
        // Call server to create portal session
        await fetchData<CreatePortalSessionParams, CreatePortalSessionResponse>({
            endpoint: StripeEndpoint.CreatePortalSession,
            inputs: {
                userId: currentUser.id,
                returnUrl: window.location.href,
            },
            method: "POST",
            omitRestBase: true,
        }).then(async ({ data }) => {
            if (!data) {
                throw new Error("Failed to create portal session");
            }
            // Redirect to portal page
            window.location.href = data.url;
        }).catch((error) => {
            handleError(error);
        }).finally(() => {
            toggleLoading(false);
        });
    };

    /** Checks if the user has a subscription that never went through */
    const checkFailedSubscription = async () => {
        // Must be logged in
        if (!currentUser.id) {
            openLink(setLocation, LINKS.Signup, { redirect: window.location.pathname });
            return;
        }

        toggleLoading(true);

        await fetchData<CheckSubscriptionParams, CheckSubscriptionResponse>({
            endpoint: StripeEndpoint.CheckSubscription,
            inputs: { userId: currentUser.id },
            method: "POST",
            omitRestBase: true,
        }).then(async ({ data }) => {
            if (!data) {
                throw new Error("Did not receive status");
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
                        }],
                    });
                    break;
                }
            }
        }).catch((error) => {
            handleError(error);
        }).finally(() => {
            toggleLoading(false);
        });
    };

    /** Checks if the user has a failed credits payment that never went through */
    const checkFailedCredits = async () => {
        // Must be logged in
        if (!currentUser.id) {
            openLink(setLocation, LINKS.Signup, { redirect: window.location.pathname });
            return;
        }

        toggleLoading(true);

        await fetchData<CheckCreditsPaymentParams, CheckCreditsPaymentResponse>({
            endpoint: StripeEndpoint.CheckCreditsPayment,
            inputs: { userId: currentUser.id },
            method: "POST",
            omitRestBase: true,
        }).then(async ({ data }) => {
            if (!data) {
                throw new Error("Did not receive status");
            }
            switch (data.status) {
                // Let user know if they already received all credtis
                case "already_received_all_credits": {
                    PubSub.get().publish("snack", { messageKey: "AlreadyReceivedAllCredits", severity: "Info" });
                    break;
                }
                // If we fixed their credits, update the URL and trigger handleUrlParams
                case "new_credits_received": {
                    addSearchParams(setLocation, { paymentType: PaymentType.Credits, status: "success" });
                    handleUrlParams();
                    break;
                }
            }
        }).catch((error) => {
            handleError(error);
        }).finally(() => {
            toggleLoading(false);
        });
    };

    return {
        checkFailedCredits,
        checkFailedSubscription,
        currentUser,
        loading,
        prices,
        redirectToCustomerPortal,
        startCheckout,
    } as const;
}
