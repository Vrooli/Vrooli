// Mock version of useStripe for Storybook to prevent Stripe.js from loading
export function useStripe() {
    return {
        checkFailedCredits: async () => { },
        checkFailedSubscription: async () => { },
        loading: false,
        prices: null,
        redirectToCustomerPortal: async () => { },
        startCheckout: async () => { },
    } as const;
}
