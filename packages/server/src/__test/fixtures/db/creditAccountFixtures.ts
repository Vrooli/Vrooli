// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";

/**
 * Database fixtures for CreditAccount model - used for seeding billing test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const creditAccountDbIds = {
    userAccount1: generatePK(),
    userAccount2: generatePK(),
    teamAccount1: generatePK(),
    teamAccount2: generatePK(),
    zeroBalance: generatePK(),
    highBalance: generatePK(),
    negativeBalance: generatePK(),
};

/**
 * User credit account with positive balance
 */
export const userCreditAccountDb: Prisma.credit_accountCreateInput = {
    id: creditAccountDbIds.userAccount1,
    currentBalance: 1000000n, // 1,000,000 credits
    user: {
        connect: { id: generatePK() },
    },
};

/**
 * User credit account with low balance
 */
export const lowBalanceUserAccountDb: Prisma.credit_accountCreateInput = {
    id: creditAccountDbIds.userAccount2,
    currentBalance: 5000n, // 5,000 credits
    user: {
        connect: { id: generatePK() },
    },
};

/**
 * Team credit account with high balance
 */
export const teamCreditAccountDb: Prisma.credit_accountCreateInput = {
    id: creditAccountDbIds.teamAccount1,
    currentBalance: 10000000n, // 10,000,000 credits
    team: {
        connect: { id: generatePK() },
    },
};

/**
 * Team credit account with enterprise balance
 */
export const enterpriseTeamAccountDb: Prisma.credit_accountCreateInput = {
    id: creditAccountDbIds.teamAccount2,
    currentBalance: 100000000n, // 100,000,000 credits
    team: {
        connect: { id: generatePK() },
    },
};

/**
 * Account with zero balance
 */
export const zeroBalanceAccountDb: Prisma.credit_accountCreateInput = {
    id: creditAccountDbIds.zeroBalance,
    currentBalance: 0n,
    user: {
        connect: { id: generatePK() },
    },
};

/**
 * Account with very high balance (testing limits)
 */
export const highBalanceAccountDb: Prisma.credit_accountCreateInput = {
    id: creditAccountDbIds.highBalance,
    currentBalance: 999999999999n, // Nearly 1 trillion credits
    user: {
        connect: { id: generatePK() },
    },
};

/**
 * Account with negative balance (debt scenario)
 */
export const negativeBalanceAccountDb: Prisma.credit_accountCreateInput = {
    id: creditAccountDbIds.negativeBalance,
    currentBalance: -50000n, // -50,000 credits (in debt)
    user: {
        connect: { id: generatePK() },
    },
};

/**
 * Account with ledger entries included
 */
export const accountWithEntriesDb: Prisma.credit_accountCreateInput = {
    id: generatePK(),
    currentBalance: 500000n, // 500,000 credits
    user: {
        connect: { id: generatePK() },
    },
    entries: {
        create: [
            {
                id: generatePK(),
                idempotencyKey: "purchase_001",
                amount: 1000000n, // +1M credits (purchase)
                type: "Purchase",
                source: "Stripe",
                meta: {
                    stripeSessionId: "cs_test_123",
                    amount: 10.00,
                    currency: "USD",
                },
            },
            {
                id: generatePK(),
                idempotencyKey: "spend_001",
                amount: -500000n, // -500K credits (spending)
                type: "Spend",
                source: "InternalAgent",
                meta: {
                    operation: "ai_generation",
                    tokens: 50000,
                    model: "gpt-4",
                },
            },
        ],
    },
};

/**
 * Factory functions for creating credit accounts dynamically
 */
export class CreditAccountDbFactory {
    /**
     * Create minimal credit account
     */
    static createMinimal(overrides?: Partial<Prisma.credit_accountCreateInput>): Prisma.credit_accountCreateInput {
        return {
            id: generatePK(),
            currentBalance: 0n,
            ...overrides,
        };
    }

    /**
     * Create user credit account with specified balance
     */
    static createUserAccount(
        userId: bigint,
        balance = 100000n,
        overrides?: Partial<Prisma.credit_accountCreateInput>,
    ): Prisma.credit_accountCreateInput {
        return {
            id: generatePK(),
            currentBalance: balance,
            user: { connect: { id: userId } },
            ...overrides,
        };
    }

    /**
     * Create team credit account with specified balance
     */
    static createTeamAccount(
        teamId: bigint,
        balance = 1000000n,
        overrides?: Partial<Prisma.credit_accountCreateInput>,
    ): Prisma.credit_accountCreateInput {
        return {
            id: generatePK(),
            currentBalance: balance,
            team: { connect: { id: teamId } },
            ...overrides,
        };
    }

    /**
     * Create account with initial purchase entry
     */
    static createWithPurchase(
        entityId: bigint,
        entityType: "user" | "team",
        purchaseAmount: bigint,
        stripeSessionId?: string,
        overrides?: Partial<Prisma.credit_accountCreateInput>,
    ): Prisma.credit_accountCreateInput {
        return {
            id: generatePK(),
            currentBalance: purchaseAmount,
            ...(entityType === "user" 
                ? { user: { connect: { id: entityId } } }
                : { team: { connect: { id: entityId } } }
            ),
            entries: {
                create: [
                    {
                        id: generatePK(),
                        idempotencyKey: stripeSessionId || `purchase_${Date.now()}`,
                        amount: purchaseAmount,
                        type: "Purchase",
                        source: "Stripe",
                        meta: {
                            stripeSessionId: stripeSessionId || "cs_test_default",
                            amount: Number(purchaseAmount) / 100000, // Assume 100K credits per dollar
                            currency: "USD",
                        },
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Create account with transaction history
     */
    static createWithHistory(
        entityId: bigint,
        entityType: "user" | "team",
        transactions: Array<{
            amount: bigint;
            type: "Purchase" | "Spend" | "TransferIn" | "TransferOut" | "Bonus" | "Refund";
            source: "Stripe" | "InternalAgent" | "Admin" | "Other";
            meta?: any;
        }>,
        overrides?: Partial<Prisma.credit_accountCreateInput>,
    ): Prisma.credit_accountCreateInput {
        const currentBalance = transactions.reduce((sum, tx) => sum + tx.amount, 0n);

        return {
            id: generatePK(),
            currentBalance,
            ...(entityType === "user" 
                ? { user: { connect: { id: entityId } } }
                : { team: { connect: { id: entityId } } }
            ),
            entries: {
                create: transactions.map((tx, index) => ({
                    id: generatePK(),
                    idempotencyKey: `tx_${Date.now()}_${index}`,
                    amount: tx.amount,
                    type: tx.type,
                    source: tx.source,
                    meta: tx.meta || {},
                })),
            },
            ...overrides,
        };
    }

    /**
     * Create account for testing spending limits
     */
    static createForLimitTesting(
        entityId: bigint,
        entityType: "user" | "team",
        lowBalance = false,
        overrides?: Partial<Prisma.credit_accountCreateInput>,
    ): Prisma.credit_accountCreateInput {
        const balance = lowBalance ? 1000n : 100000n; // Low: 1K, Normal: 100K

        return {
            id: generatePK(),
            currentBalance: balance,
            ...(entityType === "user" 
                ? { user: { connect: { id: entityId } } }
                : { team: { connect: { id: entityId } } }
            ),
            ...overrides,
        };
    }

    /**
     * Create enterprise account with high balance
     */
    static createEnterprise(
        teamId: bigint,
        balance = 100000000n, // 100M credits
        overrides?: Partial<Prisma.credit_accountCreateInput>,
    ): Prisma.credit_accountCreateInput {
        return {
            id: generatePK(),
            currentBalance: balance,
            team: { connect: { id: teamId } },
            entries: {
                create: [
                    {
                        id: generatePK(),
                        idempotencyKey: `enterprise_setup_${Date.now()}`,
                        amount: balance,
                        type: "Purchase",
                        source: "Stripe",
                        meta: {
                            plan: "enterprise",
                            contractValue: Number(balance) / 100000, // Credits to dollars
                            currency: "USD",
                        },
                    },
                ],
            },
            ...overrides,
        };
    }
}

/**
 * Helper functions for seeding test data
 */

/**
 * Seed basic credit account scenarios for testing
 */
export async function seedTestCreditAccounts(db: PrismaClient) {
    // Create users and teams first
    const user1 = await db.user.create({
        data: {
            id: generatePK(),
            publicId: "credit_user_1",
            name: "Credit Test User 1",
            handle: "credituser1",
            status: "Unlocked",
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
        },
    });

    const user2 = await db.user.create({
        data: {
            id: generatePK(),
            publicId: "credit_user_2",
            name: "Credit Test User 2", 
            handle: "credituser2",
            status: "Unlocked",
            isBot: false,
            isBotDepictingPerson: false,
            isPrivate: false,
        },
    });

    const team1 = await db.team.create({
        data: {
            id: generatePK(),
            publicId: "credit_team_1",
            handle: "creditteam1",
            createdById: user1.id,
        },
    });

    // Create credit accounts
    const accounts = await Promise.all([
        db.creditAccount.create({
            data: CreditAccountDbFactory.createUserAccount(user1.id, 500000n),
        }),
        db.creditAccount.create({
            data: CreditAccountDbFactory.createForLimitTesting(user2.id, "user", true), // Low balance
        }),
        db.creditAccount.create({
            data: CreditAccountDbFactory.createEnterprise(team1.id),
        }),
    ]);

    return { user1, user2, team1, accounts };
}
