import { uuid } from "@local/shared";
import Stripe from "stripe";
import { RecursivePartial } from "../types.js";

type StripeGlobalStore = {
    billingPortalSessions: Stripe.BillingPortal.Session[];
    checkoutSessions: Stripe.Checkout.Session[];
    customers: Stripe.Customer[];
    prices: Stripe.Price[];
    subscriptions: Stripe.Subscription[];
}
const emptyGlobalStore: StripeGlobalStore = {
    billingPortalSessions: [],
    checkoutSessions: [],
    customers: [],
    prices: [],
    subscriptions: [],
};
let globalDataStore: StripeGlobalStore = emptyGlobalStore;

const createBillingPortalSession = (data: Partial<Stripe.BillingPortal.Session>) => ({
    id: uuid(),
    object: "billing_portal.session" as const,
    created: Date.now(),
    livemode: process.env.NODE_ENV === "production",
    ...data,
} as Stripe.BillingPortal.Session);

const createCheckoutSession = (data: Partial<Stripe.Checkout.Session>) => ({
    id: uuid(),
    object: "checkout.session",
    created: Date.now(),
    livemode: process.env.NODE_ENV === "production" as const,
    ...data,
} as Stripe.Checkout.Session);

const createCustomer = (data: Partial<Stripe.Customer>) => ({
    id: uuid(),
    object: "customer",
    created: Date.now(),
    livemode: process.env.NODE_ENV === "production" as const,
    ...data,
} as Stripe.Customer);

const createPrice = (data: Partial<Stripe.Price>) => ({
    id: uuid(),
    object: "price",
    created: Date.now(),
    currency: "usd",
    livemode: process.env.NODE_ENV === "production" as const,
    ...data,
} as Stripe.Price);

const createSubscription = (data: Partial<Stripe.Subscription>) => ({
    id: uuid(),
    object: "subscription",
    created: Date.now(),
    currency: "usd",
    livemode: process.env.NODE_ENV === "production" as const,
    ...data,
} as Stripe.Subscription);

class StripeMock {
    static instance = new StripeMock();
    static shouldFail = false;

    constructor() {
        return StripeMock.instance;
    }

    static injectData(data: {
        billingPortalSessions?: RecursivePartial<Stripe.BillingPortal.Session>[];
        checkoutSessions?: Partial<Stripe.Checkout.Session>[];
        customers?: Partial<Stripe.Customer>[];
        prices?: Partial<Stripe.Price>[];
        subscriptions?: Partial<Stripe.Subscription>[];
    }) {
        globalDataStore.billingPortalSessions = data.billingPortalSessions?.map(createBillingPortalSession) || [];
        globalDataStore.checkoutSessions = data.checkoutSessions?.map(createCheckoutSession) || [];
        globalDataStore.customers = data.customers?.map(createCustomer) || [];
        globalDataStore.prices = data.prices?.map(createPrice) || [];
        globalDataStore.subscriptions = data.subscriptions?.map(createSubscription) || [];
    }

    static clearData() {
        globalDataStore = emptyGlobalStore;
    }

    static simulateFailure(shouldFail: boolean) {
        StripeMock.shouldFail = shouldFail;
    }

    static resetMock() {
        StripeMock.clearData();
        StripeMock.simulateFailure(false);
    }

    _handlePromise = (operation) => {
        return StripeMock.shouldFail ? Promise.reject(new Error("Stripe operation failed")) : Promise.resolve(operation());
    };

    customers = {
        list: async ({
            email,
            limit = 10,
        }: {
            email?: string,
            limit?: number,
        }) => this._handlePromise(() => {
            const allCustomers = globalDataStore.customers || [];
            const filteredCustomers = email ? allCustomers.filter(customer => customer.email === email) : allCustomers;
            return {
                data: filteredCustomers.slice(0, limit),
                has_more: filteredCustomers.length > limit,
                object: "list",
            };
        }),
        retrieve: async (id: string) => this._handlePromise(() => {
            const customer = globalDataStore.customers?.find(c => c.id === id);
            return customer || null;
        }),
        create: async (data: Stripe.CustomerCreateParams) => this._handlePromise(() => {
            // Grab all parameters we currently care about
            const { email } = data;
            const customer = createCustomer({ email });
            globalDataStore.customers = globalDataStore.customers ? [...globalDataStore.customers, customer] : [customer];
            return customer;
        }),
    };

    checkout = {
        billingPortal: {
            sessions: {
                create: async (data: Stripe.BillingPortal.SessionCreateParams) => this._handlePromise(() => {
                    // Grab all parameters we currently care about
                    const { customer, return_url } = data;
                    const session = createBillingPortalSession({ customer, return_url });
                    globalDataStore.billingPortalSessions = globalDataStore.billingPortalSessions ? [...globalDataStore.billingPortalSessions, session] : [session];
                    return session;
                }),
            },
        },
        sessions: {
            list: async ({
                limit = 10,
                subscription,
            }: {
                limit?: number,
                subscription?: string,
            }) => this._handlePromise(() => {
                const allSessions = globalDataStore.checkoutSessions || [];
                const filteredSessions = subscription ? allSessions.filter(session => session.subscription === subscription) : allSessions;
                return {
                    data: filteredSessions.slice(0, limit),
                    has_more: filteredSessions.length > limit,
                    object: "list",
                };
            }),
            create: async (data: Stripe.Checkout.SessionCreateParams) => this._handlePromise(() => {
                // Grab all parameters we currently care about
                const {
                    cancel_url,
                    customer,
                    line_items,
                    metadata,
                    mode,
                    payment_method_types,
                    success_url,
                } = data;
                const session = createCheckoutSession({
                    cancel_url,
                    customer,
                    line_items: {
                        object: "list",
                        data: (line_items?.map(item => ({
                            id: uuid(),
                            object: "item" as const,
                            price: item.price,
                            quantity: item.quantity,
                        })) || []) as unknown as Stripe.LineItem[],
                        has_more: false,
                        url: "",
                    },
                    metadata: (metadata as Stripe.Metadata) ?? null,
                    mode,
                    payment_method_types,
                    success_url,
                });
                globalDataStore.checkoutSessions = globalDataStore.checkoutSessions ? [...globalDataStore.checkoutSessions, session] : [session];
                return session;
            }),
        },
    };

    prices = {
        retrieve: async (id: string) => this._handlePromise(() => {
            const price = globalDataStore.prices?.find(p => p.id === id);
            return price || null;
        }),
    };

    subscriptions = {
        list: async ({
            customer,
            limit = 10,
            status,
        }: {
            customer: string,
            limit?: number,
            status?: Stripe.SubscriptionListParams.Status,
        }) => this._handlePromise(() => {
            const allSubscriptions = globalDataStore.subscriptions || [];
            let filteredSubscriptions = customer ? allSubscriptions.filter(subscription => subscription.customer === customer) : allSubscriptions;
            filteredSubscriptions = (status && status !== "all") ? filteredSubscriptions.filter(subscription => subscription.status === status) : filteredSubscriptions;
            return {
                data: filteredSubscriptions.slice(0, limit),
                has_more: filteredSubscriptions.length > limit,
                object: "list",
            };
        }),
        retrieve: async (id: string) => this._handlePromise(() => {
            const subscription = globalDataStore.subscriptions?.find(s => s.id === id);
            return subscription || null;
        }),
    };
}

export default StripeMock;
