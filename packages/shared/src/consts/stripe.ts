import { type PaymentType } from "../api/types.js";

export type SubscriptionPricesResponse = {
    monthly: number;
    yearly: number;
};

export type CreateCheckoutSessionParams = {
    /** Only used if variant is "Credits" */
    amount?: number | undefined;
    userId: string | undefined;
    variant: PaymentType;
};

export type CreateCheckoutSessionResponse = {
    url: string;
};

export type CreatePortalSessionParams = {
    userId: string;
    returnUrl: string;
}

export type CreatePortalSessionResponse = {
    url: string;
};

export type CheckoutSessionMetadata = {
    userId: string | null;
    paymentType: PaymentType;
};

export type CheckSubscriptionParams = {
    userId: string
};

export type CheckSubscriptionResponse = {
    paymentType?: PaymentType.PremiumMonthly | PaymentType.PremiumYearly;
    status: "already_subscribed" | "now_subscribed" | "not_subscribed";
};

export type CheckCreditsPaymentParams = {
    userId: string
};

export type CheckCreditsPaymentResponse = {
    status: "already_received_all_credits" | "new_credits_received";
};

export enum StripeEndpoint {
    SubscriptionPrices = "/subscription-prices",
    CreateCheckoutSession = "/create-checkout-session",
    CreatePortalSession = "/create-portal-session",
    CheckSubscription = "/check-subscription",
    CheckCreditsPayment = "/check-credits-payment",
}
