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
}