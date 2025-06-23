import { generatePK, DAYS_180_MS } from "@vrooli/shared";
import { DbProvider } from "../../db/provider.js";

export interface StripeTestUser {
    id: bigint;
    name: string;
    stripeCustomerId?: string;
    emails?: { emailAddress: string }[];
    hasPlan?: boolean;
}

export interface StripeTestPayment {
    id: bigint;
    userId: bigint;
    checkoutId: string;
    amount: number;
    status: string;
}

/**
 * Creates standard test users for Stripe tests
 */
export async function createStripeTestUsers() {
    const user1Id = generatePK();
    const user2Id = generatePK();
    const user3Id = generatePK();
    const user4Id = generatePK();
    const user5Id = generatePK();
    const user6Id = generatePK();
    const user7Id = generatePK();

    // User 1: Has Stripe customer ID and premium plan
    await DbProvider.get().user.create({
        data: {
            id: user1Id,
            publicId: user1Id.toString(),
            name: "user1",
            stripeCustomerId: "cus_123",
            emails: {
                create: [{ id: generatePK(), emailAddress: "user@example.com" }],
            },
            plan: {
                create: {
                    id: generatePK(),
                    enabledAt: new Date(),
                    expiresAt: new Date(Date.now() + DAYS_180_MS),
                },
            },
        },
    });

    // User 2: No Stripe customer ID, but email matches StripeMock customer
    await DbProvider.get().user.create({
        data: {
            id: user2Id,
            publicId: user2Id.toString(),
            name: "user2",
            emails: {
                create: [{ id: generatePK(), emailAddress: "user2@example.com" }],
            },
        },
    });

    // User 3: No Stripe customer ID, email doesn't match any customer
    await DbProvider.get().user.create({
        data: {
            id: user3Id,
            publicId: user3Id.toString(),
            name: "user3",
            emails: {
                create: [{ id: generatePK(), emailAddress: "missing@example.com" }],
            },
        },
    });

    // User 4: Invalid Stripe customer ID
    await DbProvider.get().user.create({
        data: {
            id: user4Id,
            publicId: user4Id.toString(),
            name: "user4",
            stripeCustomerId: "invalid_customer_id",
        },
    });

    // User 5: Stripe customer ID for deleted customer
    await DbProvider.get().user.create({
        data: {
            id: user5Id,
            publicId: user5Id.toString(),
            name: "user5",
            stripeCustomerId: "cus_deleted",
        },
    });

    // User 6: Email matches deleted customer
    await DbProvider.get().user.create({
        data: {
            id: user6Id,
            publicId: user6Id.toString(),
            name: "user6",
            emails: {
                create: [{ id: generatePK(), emailAddress: "emailForDeletedCustomer@example.com" }],
            },
        },
    });

    // User 7: Has Stripe customer ID that will be updated
    await DbProvider.get().user.create({
        data: {
            id: user7Id,
            publicId: user7Id.toString(),
            name: "user7",
            stripeCustomerId: "cus_777",
            emails: {
                create: [{ id: generatePK(), emailAddress: "emailWithSubscription@example.com" }],
            },
        },
    });

    return {
        user1Id,
        user2Id,
        user3Id,
        user4Id,
        user5Id,
        user6Id,
        user7Id,
    };
}

/**
 * Creates a user with multiple email addresses
 */
export async function createUserWithMultipleEmails(stripeCustomerId: string) {
    const userId = generatePK();
    
    await DbProvider.get().user.create({
        data: {
            id: userId,
            publicId: userId.toString(),
            name: "multiEmailUser",
            stripeCustomerId,
            emails: {
                create: [
                    { id: generatePK(), emailAddress: "user1@example.com" },
                    { id: generatePK(), emailAddress: "user2@example.com" },
                ],
            },
        },
    });
    
    return { userId };
}

/**
 * Creates a user with a plan
 */
export async function createUserWithPlan(stripeCustomerId: string, name = "userWithPlan") {
    const userId = generatePK();
    
    await DbProvider.get().user.create({
        data: {
            id: userId,
            publicId: userId.toString(),
            name,
            stripeCustomerId,
            emails: {
                create: [{ id: generatePK(), emailAddress: `${name}@example.com` }],
            },
            plan: {
                create: {
                    id: generatePK(),
                    enabledAt: new Date(),
                    expiresAt: new Date(Date.now() + DAYS_180_MS),
                },
            },
        },
    });
    
    return { userId };
}

/**
 * Creates a payment record
 */
export async function createPayment({
    userId,
    checkoutId,
    amount = 1000,
    status = "Pending",
    paymentType = "PremiumMonthly",
}: {
    userId: bigint;
    checkoutId: string;
    amount?: number;
    status?: string;
    paymentType?: string;
}) {
    const paymentId = generatePK();
    
    await DbProvider.get().payment.create({
        data: {
            id: paymentId,
            userId,
            checkoutId,
            amount,
            currency: "usd",
            description: "Test payment",
            paymentMethod: "Stripe",
            status,
            paymentType,
        },
    });
    
    return { paymentId };
}

/**
 * Creates basic users for webhook handler tests
 */
export async function createWebhookTestUsers() {
    const user1Id = generatePK();
    const user2Id = generatePK();

    await DbProvider.get().user.create({
        data: {
            id: user1Id,
            publicId: user1Id.toString(),
            name: "user1",
            stripeCustomerId: "cus_123",
            emails: {
                create: [{ id: generatePK(), emailAddress: "user@example.com" }],
            },
            plan: {
                create: {
                    id: generatePK(),
                    enabledAt: new Date(),
                    expiresAt: new Date(Date.now() + DAYS_180_MS),
                },
            },
        },
    });

    await DbProvider.get().user.create({
        data: {
            id: user2Id,
            publicId: user2Id.toString(),
            name: "user2",
            stripeCustomerId: "cus_234",
        },
    });

    return { user1Id, user2Id };
}

/**
 * Creates users for customer deletion tests
 */
export async function createCustomerDeletionTestUsers() {
    const user1Id = generatePK();
    const user2Id = generatePK();
    const customerId1 = `deletion_customer1_${user1Id}`;
    const customerId2 = `deletion_customer2_${user2Id}`;

    await DbProvider.get().user.create({
        data: {
            id: user1Id,
            publicId: user1Id.toString(),
            name: "user1",
            stripeCustomerId: customerId1,
        },
    });

    await DbProvider.get().user.create({
        data: {
            id: user2Id,
            publicId: user2Id.toString(),
            name: "user2",
            stripeCustomerId: customerId2,
        },
    });

    return { user1Id, user2Id, customerId1, customerId2 };
}

/**
 * Creates users and payments for invoice handler tests
 */
export async function createInvoiceTestData() {
    const userId = generatePK();
    const paymentId = generatePK();
    
    await DbProvider.get().user.create({
        data: {
            id: userId,
            publicId: userId.toString(),
            name: "user1",
            stripeCustomerId: "cus_123",
        },
    });
    
    await DbProvider.get().payment.create({
        data: {
            id: paymentId,
            userId,
            checkoutId: "inv_123",
            amount: 1000,
            currency: "usd",
            description: "Test payment",
            paymentMethod: "Stripe",
            status: "Pending",
            paymentType: "PremiumMonthly",
        },
    });
    
    return { userId, paymentId };
}

/**
 * Creates users and payments for invoice handler tests (with emails for notifications)
 */
export async function createInvoiceTestDataWithEmail() {
    const userId = generatePK();
    const paymentId = generatePK();
    
    await DbProvider.get().user.create({
        data: {
            id: userId,
            publicId: userId.toString(),
            name: "user1",
            stripeCustomerId: "cus_123",
            emails: {
                create: [{ id: generatePK(), emailAddress: "user@example.com" }],
            },
        },
    });
    
    await DbProvider.get().payment.create({
        data: {
            id: paymentId,
            userId,
            checkoutId: "inv_123",
            amount: 1000,
            currency: "usd",
            description: "Test payment",
            paymentMethod: "Stripe",
            status: "Pending",
            paymentType: "PremiumMonthly",
        },
    });
    
    return { userId, paymentId };
}

/**
 * Creates a simple user without stripe customer ID for createStripeCustomerId tests
 */
export async function createSimpleTestUser() {
    const userId = generatePK();

    await DbProvider.get().user.create({
        data: {
            id: userId,
            publicId: userId.toString(),
            name: "user1",
            stripeCustomerId: null,
        },
    });

    return { userId };
}

/**
 * Creates test data for handleCheckoutSessionExpired tests
 */
export async function createCheckoutSessionTestData() {
    const payment1Id = generatePK();
    const user1Id = generatePK();
    const user2Id = generatePK();
    const customerId1 = `checkout_customer1_${user1Id}`;
    const customerId2 = `checkout_customer2_${user2Id}`;

    await DbProvider.get().user.create({
        data: {
            id: user1Id,
            publicId: user1Id.toString(),
            name: "user1",
            stripeCustomerId: customerId1,
        },
    });
    await DbProvider.get().user.create({
        data: {
            id: user2Id,
            publicId: user2Id.toString(),
            name: "user2",
            stripeCustomerId: customerId2,
        },
    });
    await DbProvider.get().payment.create({
        data: {
            id: payment1Id,
            amount: 1000,
            currency: "usd",
            description: "Test payment",
            checkoutId: "session1",
            paymentMethod: "Stripe",
            status: "Pending",
            userId: user1Id,
        },
    });

    return { payment1Id, user1Id, user2Id, customerId1, customerId2 };
}

/**
 * Creates user with multiple emails for source expiring tests
 */
export async function createSourceExpiringTestUser() {
    const user1Id = generatePK();
    const customerId = `source_expiring_cus_123_${user1Id}`;

    await DbProvider.get().user.create({
        data: {
            id: user1Id,
            publicId: user1Id.toString(),
            name: "user1",
            stripeCustomerId: customerId,
            emails: {
                create: [
                    { id: generatePK(), emailAddress: "user1@example.com" }, 
                    { id: generatePK(), emailAddress: "user2@example.com" },
                ],
            },
        },
    });

    return { user1Id, customerId };
}

/**
 * Clean up all test data
 */
export async function cleanupStripeTestData() {
    await DbProvider.get().payment.deleteMany({});
    await DbProvider.get().plan.deleteMany({});
    await DbProvider.get().user.deleteMany({});
}
