/* eslint-disable no-magic-numbers */
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { type Prisma, type PrismaClient } from "@prisma/client";
import { generatePK } from "@vrooli/shared";

/**
 * Database fixtures for Payment model - used for seeding transaction test data
 * These follow Prisma's shape for database operations
 */

// Cached IDs for consistent testing - lazy initialization pattern
let _paymentDbIds: Record<string, bigint> | null = null;
export function getPaymentDbIds() {
    if (!_paymentDbIds) {
        _paymentDbIds = {
            successfulUser: generatePK(),
            successfulTeam: generatePK(),
            pendingUser: generatePK(),
            failedUser: generatePK(),
            refundedUser: generatePK(),
            canceledTeam: generatePK(),
            enterpriseTeam: generatePK(),
        };
    }
    return _paymentDbIds;
}

// Cached IDs for consistent user/team relationships
const _paymentRelationIds: { [key: string]: bigint | undefined } = {};
export const paymentRelationIds = {
    get user1() { return _paymentRelationIds.user1 ??= generatePK(); },
    get user2() { return _paymentRelationIds.user2 ??= generatePK(); },
    get user3() { return _paymentRelationIds.user3 ??= generatePK(); },
    get user4() { return _paymentRelationIds.user4 ??= generatePK(); },
    get team1() { return _paymentRelationIds.team1 ??= generatePK(); },
    get team2() { return _paymentRelationIds.team2 ??= generatePK(); },
    get team3() { return _paymentRelationIds.team3 ??= generatePK(); },
};

/**
 * Successful user payment - Premium Monthly
 */
export const successfulUserPaymentDb: Prisma.paymentCreateInput = {
    id: getPaymentDbIds().successfulUser,
    amount: 999, // $9.99 in cents
    checkoutId: "cs_test_successful_123",
    currency: "USD",
    description: "Premium Monthly Subscription",
    paymentMethod: "card_visa",
    paymentType: "PremiumMonthly",
    status: "Paid",
    user: {
        connect: { id: paymentRelationIds.user1 },
    },
};

/**
 * Successful team payment - Premium Annual
 */
export const successfulTeamPaymentDb: Prisma.paymentCreateInput = {
    id: getPaymentDbIds().successfulTeam,
    amount: 9999, // $99.99 in cents
    checkoutId: "cs_test_team_annual_456",
    currency: "USD",
    description: "Team Premium Annual Subscription",
    paymentMethod: "card_mastercard",
    paymentType: "PremiumYearly",
    status: "Paid",
    team: {
        connect: { id: paymentRelationIds.team1 },
    },
};

/**
 * Pending user payment
 */
export const pendingUserPaymentDb: Prisma.paymentCreateInput = {
    id: getPaymentDbIds().pendingUser,
    amount: 1999, // $19.99 in cents
    checkoutId: "cs_test_pending_789",
    currency: "USD",
    description: "Premium Plus Monthly Subscription",
    paymentMethod: "bank_transfer",
    paymentType: "PremiumMonthly",
    status: "Pending",
    user: {
        connect: { id: paymentRelationIds.user2 },
    },
};

/**
 * Failed user payment
 */
export const failedUserPaymentDb: Prisma.paymentCreateInput = {
    id: getPaymentDbIds().failedUser,
    amount: 999, // $9.99 in cents
    checkoutId: "cs_test_failed_101",
    currency: "USD",
    description: "Premium Monthly Subscription",
    paymentMethod: "card_declined",
    paymentType: "PremiumMonthly",
    status: "Failed",
    user: {
        connect: { id: paymentRelationIds.user3 },
    },
};

/**
 * Refunded user payment
 */
export const refundedUserPaymentDb: Prisma.paymentCreateInput = {
    id: getPaymentDbIds().refundedUser,
    amount: 999, // $9.99 in cents
    checkoutId: "cs_test_refunded_202",
    currency: "USD",
    description: "Premium Monthly Subscription (Refunded)",
    paymentMethod: "card_visa",
    paymentType: "PremiumMonthly",
    status: "Failed", // Note: PaymentStatus enum doesn't have Refunded
    user: {
        connect: { id: paymentRelationIds.user4 },
    },
};

/**
 * Canceled team payment
 */
export const canceledTeamPaymentDb: Prisma.paymentCreateInput = {
    id: getPaymentDbIds().canceledTeam,
    amount: 29999, // $299.99 in cents
    checkoutId: "cs_test_canceled_303",
    currency: "USD",
    description: "Team Enterprise Monthly (Canceled)",
    paymentMethod: "invoice",
    paymentType: "PremiumYearly",
    status: "Failed", // Note: PaymentStatus enum doesn't have Canceled
    team: {
        connect: { id: paymentRelationIds.team2 },
    },
};

/**
 * Large enterprise payment
 */
export const enterprisePaymentDb: Prisma.paymentCreateInput = {
    id: getPaymentDbIds().enterpriseTeam,
    amount: 99999, // $999.99 in cents
    checkoutId: "cs_test_enterprise_404",
    currency: "USD",
    description: "Enterprise Annual Subscription - 100 seats",
    paymentMethod: "wire_transfer",
    paymentType: "PremiumYearly",
    status: "Paid",
    team: {
        connect: { id: paymentRelationIds.team3 },
    },
};

/**
 * Factory functions for creating payments dynamically
 */
export class PaymentDbFactory {
    /**
     * Create minimal payment
     */
    static createMinimal(overrides?: Partial<Prisma.paymentCreateInput>): Prisma.paymentCreateInput {
        return {
            id: generatePK(),
            amount: 999,
            checkoutId: `cs_test_${Date.now()}`,
            currency: "USD",
            description: "Test Payment",
            paymentMethod: "card_test",
            paymentType: "PremiumMonthly",
            status: "Pending",
            ...overrides,
        };
    }

    /**
     * Create user subscription payment
     */
    static createUserSubscription(
        userId: bigint,
        type: "PremiumMonthly" | "PremiumYearly" = "PremiumMonthly",
        status: "Pending" | "Paid" | "Failed" = "Paid",
        overrides?: Partial<Prisma.paymentCreateInput>,
    ): Prisma.paymentCreateInput {
        const amounts = {
            PremiumMonthly: 999,
            PremiumYearly: 9999,
        };

        const descriptions = {
            PremiumMonthly: "Premium Monthly Subscription",
            PremiumYearly: "Premium Annual Subscription",
        };

        return {
            id: generatePK(),
            amount: amounts[type],
            checkoutId: `cs_user_${type.toLowerCase()}_${Date.now()}`,
            currency: "USD",
            description: descriptions[type],
            paymentMethod: "card_visa",
            paymentType: type,
            status,
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    /**
     * Create team subscription payment
     */
    static createTeamSubscription(
        teamId: bigint,
        type: "PremiumMonthly" | "PremiumYearly" = "PremiumMonthly",
        status: "Pending" | "Paid" | "Failed" = "Paid",
        seats = 10,
        overrides?: Partial<Prisma.paymentCreateInput>,
    ): Prisma.paymentCreateInput {
        const baseAmounts = {
            PremiumMonthly: 1999, // $19.99 base
            PremiumYearly: 19999, // $199.99 base
        };

        const seatMultiplier = Math.max(1, Math.floor(seats / 5)); // Every 5 seats increases cost
        const amount = baseAmounts[type] * seatMultiplier;

        return {
            id: generatePK(),
            amount,
            checkoutId: `cs_team_${type.toLowerCase()}_${Date.now()}`,
            currency: "USD",
            description: `Team ${type.replace("Premium", "")} Subscription - ${seats} seats`,
            paymentMethod: "card_mastercard",
            paymentType: type,
            status,
            team: { connect: { id: teamId } },
            ...overrides,
        };
    }

    /**
     * Create credit purchase payment
     */
    static createCreditPurchase(
        entityId: bigint,
        entityType: "user" | "team",
        dollarAmount: number,
        creditsAmount: number,
        status: "Pending" | "Paid" | "Failed" = "Paid",
        overrides?: Partial<Prisma.paymentCreateInput>,
    ): Prisma.paymentCreateInput {
        return {
            id: generatePK(),
            amount: Math.round(dollarAmount * 100), // Convert to cents
            checkoutId: `cs_credits_${Date.now()}`,
            currency: "USD",
            description: `${creditsAmount.toLocaleString()} Credits Purchase`,
            paymentMethod: "card_visa",
            paymentType: "Credits",
            status,
            ...(entityType === "user"
                ? { user: { connect: { id: entityId } } }
                : { team: { connect: { id: entityId } } }
            ),
            ...overrides,
        };
    }

    /**
     * Create failed payment with specific reason
     */
    static createFailed(
        entityId: bigint,
        entityType: "user" | "team",
        failureReason: "card_declined" | "insufficient_funds" | "expired_card" | "invalid_cvc" = "card_declined",
        overrides?: Partial<Prisma.paymentCreateInput>,
    ): Prisma.paymentCreateInput {
        return {
            id: generatePK(),
            amount: 999,
            checkoutId: `cs_failed_${failureReason}_${Date.now()}`,
            currency: "USD",
            description: `Failed Payment - ${failureReason}`,
            paymentMethod: failureReason,
            paymentType: "PremiumMonthly",
            status: "Failed",
            ...(entityType === "user"
                ? { user: { connect: { id: entityId } } }
                : { team: { connect: { id: entityId } } }
            ),
            ...overrides,
        };
    }

    /**
     * Create international payment
     */
    static createInternational(
        entityId: bigint,
        entityType: "user" | "team",
        currency: "EUR" | "GBP" | "CAD" | "AUD" = "EUR",
        overrides?: Partial<Prisma.paymentCreateInput>,
    ): Prisma.paymentCreateInput {
        const amounts = {
            EUR: 899,  // €8.99
            GBP: 799,  // £7.99
            CAD: 1299, // C$12.99
            AUD: 1399, // A$13.99
        };

        return {
            id: generatePK(),
            amount: amounts[currency],
            checkoutId: `cs_intl_${currency.toLowerCase()}_${Date.now()}`,
            currency,
            description: `Premium Monthly Subscription (${currency})`,
            paymentMethod: "card_visa",
            paymentType: "PremiumMonthly",
            status: "Paid",
            ...(entityType === "user"
                ? { user: { connect: { id: entityId } } }
                : { team: { connect: { id: entityId } } }
            ),
            ...overrides,
        };
    }

    /**
     * Create enterprise payment
     */
    static createEnterprise(
        teamId: bigint,
        seats = 100,
        customAmount?: number,
        overrides?: Partial<Prisma.paymentCreateInput>,
    ): Prisma.paymentCreateInput {
        const amount = customAmount || (seats * 10 * 100); // $10 per seat per month, in cents

        return {
            id: generatePK(),
            amount,
            checkoutId: `cs_enterprise_${seats}seats_${Date.now()}`,
            currency: "USD",
            description: `Enterprise Annual Subscription - ${seats} seats`,
            paymentMethod: "wire_transfer",
            paymentType: "PremiumYearly",
            status: "Paid",
            team: { connect: { id: teamId } },
            ...overrides,
        };
    }
}

/**
 * Helper functions for seeding test data
 */

/**
 * Seed basic payment scenarios for testing
 */
export async function seedTestPayments(db: PrismaClient) {
    // Create users and teams first
    const user1 = await db.user.create({
        data: {
            id: generatePK(),
            publicId: "payment_user_1",
            name: "Payment Test User 1",
            handle: "paymentuser1",
            status: "Unlocked",
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
        },
    });

    const user2 = await db.user.create({
        data: {
            id: generatePK(),
            publicId: "payment_user_2",
            name: "Payment Test User 2",
            handle: "paymentuser2",
            status: "Unlocked",
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
        },
    });

    const team1 = await db.team.create({
        data: {
            id: generatePK(),
            publicId: "payment_team_1",
            handle: "paymentteam1",
            createdById: user1.id,
        },
    });

    // Create payments
    const payments = await Promise.all([
        db.payment.create({
            data: PaymentDbFactory.createUserSubscription(user1.id, "PremiumMonthly"),
        }),
        db.payment.create({
            data: PaymentDbFactory.createFailed(user2.id, "user", "card_declined"),
        }),
        db.payment.create({
            data: PaymentDbFactory.createTeamSubscription(team1.id, "PremiumYearly", "Paid", 25),
        }),
        db.payment.create({
            data: PaymentDbFactory.createCreditPurchase(user1.id, "user", 50.00, 5000000),
        }),
    ]);

    return { user1, user2, team1, payments };
}
