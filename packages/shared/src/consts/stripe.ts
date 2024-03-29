import { PaymentType } from "../api/generated/graphqlTypes";

export type SubscriptionPricesResponse = {
    monthly: number;
    yearly: number;
};

export type CreateCheckoutSessionParams = {
    userId: string | undefined;
    variant: PaymentType;
};

export type CreateCheckoutSessionResponse = {
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
    SubscriptionPrices = "/api/subscription-prices",
    CreateCheckoutSession = "/api/create-checkout-session",
    CreatePortalSession = "/api/create-portal-session",
    CheckSubscription = "/api/check-subscription",
    CheckCreditsPayment = "/api/check-credits-payment",
}
