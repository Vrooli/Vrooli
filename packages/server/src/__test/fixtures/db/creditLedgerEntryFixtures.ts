/* eslint-disable no-magic-numbers */
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { type Prisma, type PrismaClient } from "@prisma/client";
import { generatePK } from "@vrooli/shared";

/**
 * Database fixtures for CreditLedgerEntry model - used for seeding transaction history test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const creditLedgerDbIds = {
    purchase1: generatePK(),
    purchase2: generatePK(),
    spend1: generatePK(),
    spend2: generatePK(),
    transfer1: generatePK(),
    bonus1: generatePK(),
    refund1: generatePK(),
    rollover1: generatePK(),
};

/**
 * Purchase entry - user buys credits
 */
export const purchaseEntryDb: Prisma.credit_ledger_entryCreateInput = {
    id: creditLedgerDbIds.purchase1,
    idempotencyKey: "stripe_cs_test_123456",
    amount: 1000000n, // +1,000,000 credits
    type: "Purchase",
    source: "Stripe",
    meta: {
        stripeSessionId: "cs_test_123456",
        stripePaymentIntentId: "pi_test_123456",
        amount: 10.00,
        currency: "USD",
        packageType: "standard",
        creditsPerDollar: 100000,
    },
    account: {
        connect: { id: generatePK() },
    },
};

/**
 * Large enterprise purchase
 */
export const enterprisePurchaseDb: Prisma.credit_ledger_entryCreateInput = {
    id: creditLedgerDbIds.purchase2,
    idempotencyKey: "stripe_cs_enterprise_789",
    amount: 100000000n, // +100,000,000 credits
    type: "Purchase",
    source: "Stripe",
    meta: {
        stripeSessionId: "cs_enterprise_789",
        amount: 1000.00,
        currency: "USD",
        packageType: "enterprise",
        contractId: "ENT-2024-001",
        discountApplied: 0.15, // 15% enterprise discount
    },
    account: {
        connect: { id: generatePK() },
    },
};

/**
 * AI generation spending
 */
export const aiSpendEntryDb: Prisma.credit_ledger_entryCreateInput = {
    id: creditLedgerDbIds.spend1,
    idempotencyKey: "ai_gen_123456789",
    amount: -50000n, // -50,000 credits
    type: "Spend",
    source: "InternalAgent",
    meta: {
        operation: "ai_generation",
        model: "gpt-4",
        tokens: {
            input: 1000,
            output: 500,
            total: 1500,
        },
        taskId: "task_123456789",
        userId: "user_123",
        estimatedCost: 0.5,
    },
    account: {
        connect: { id: generatePK() },
    },
};

/**
 * API usage spending
 */
export const apiSpendEntryDb: Prisma.credit_ledger_entryCreateInput = {
    id: creditLedgerDbIds.spend2,
    idempotencyKey: "api_call_987654321",
    amount: -1000n, // -1,000 credits
    type: "Spend",
    source: "InternalAgent",
    meta: {
        operation: "api_call",
        endpoint: "/api/search",
        apiKeyId: "ak_123456",
        requestId: "req_987654321",
        complexity: "standard",
        executionTime: 250, // ms
    },
    account: {
        connect: { id: generatePK() },
    },
};

/**
 * Transfer between accounts
 */
export const transferInEntryDb: Prisma.credit_ledger_entryCreateInput = {
    id: creditLedgerDbIds.transfer1,
    idempotencyKey: "transfer_in_555",
    amount: 250000n, // +250,000 credits
    type: "TransferIn",
    source: "InternalAgent",
    meta: {
        operation: "account_transfer",
        fromAccountId: "acc_789",
        transferId: "tf_555",
        reason: "team_to_user_allocation",
        authorizedBy: "user_admin",
    },
    account: {
        connect: { id: generatePK() },
    },
};

/**
 * Bonus credits (promotion, referral, etc.)
 */
export const bonusEntryDb: Prisma.credit_ledger_entryCreateInput = {
    id: creditLedgerDbIds.bonus1,
    idempotencyKey: "bonus_ref_777",
    amount: 100000n, // +100,000 credits
    type: "Bonus",
    source: "Admin",
    meta: {
        operation: "referral_bonus",
        referralCode: "REF777",
        referredUserId: "user_999",
        bonusType: "successful_referral",
        campaignId: "camp_spring_2024",
    },
    account: {
        connect: { id: generatePK() },
    },
};

/**
 * Refund entry
 */
export const refundEntryDb: Prisma.credit_ledger_entryCreateInput = {
    id: creditLedgerDbIds.refund1,
    idempotencyKey: "refund_cs_444",
    amount: 500000n, // +500,000 credits
    type: "Refund",
    source: "Stripe",
    meta: {
        operation: "payment_refund",
        originalStripeSessionId: "cs_original_444",
        stripeRefundId: "re_refund_444",
        reason: "customer_request",
        refundAmount: 5.00,
        currency: "USD",
        supportTicketId: "ticket_12345",
    },
    account: {
        connect: { id: generatePK() },
    },
};

/**
 * Rollover entry (from previous billing period)
 */
export const rolloverEntryDb: Prisma.credit_ledger_entryCreateInput = {
    id: creditLedgerDbIds.rollover1,
    idempotencyKey: "rollover_2024_q1",
    amount: 75000n, // +75,000 credits
    type: "RollOver",
    source: "Scheduler",
    meta: {
        operation: "billing_rollover",
        fromPeriod: "2024-Q1",
        toPeriod: "2024-Q2",
        rolloverPolicy: "standard_50_percent",
        originalAmount: 150000,
        rolledOverAmount: 75000,
    },
    account: {
        connect: { id: generatePK() },
    },
};

/**
 * Migration import entry
 */
export const migrationEntryDb: Prisma.credit_ledger_entryCreateInput = {
    id: generatePK(),
    idempotencyKey: "migration_legacy_001",
    amount: 2000000n, // +2,000,000 credits
    type: "MigrationImport",
    source: "MigrationScript",
    meta: {
        operation: "legacy_import",
        legacySystemId: "old_sys_001",
        legacyBalance: 2000000,
        migrationDate: new Date().toISOString(),
        migrationBatch: "batch_001",
        verificationHash: "hash123456",
    },
    account: {
        connect: { id: generatePK() },
    },
};

/**
 * Factory functions for creating ledger entries dynamically
 */
export class CreditLedgerEntryDbFactory {
    /**
     * Create minimal ledger entry
     */
    static createMinimal(overrides?: Partial<Prisma.credit_ledger_entryCreateInput>): Prisma.credit_ledger_entryCreateInput {
        return {
            id: generatePK(),
            idempotencyKey: `entry_${Date.now()}`,
            amount: 0n,
            type: "Purchase",
            source: "Other",
            account: { connect: { id: generatePK() } },
            ...overrides,
        };
    }

    /**
     * Create purchase entry
     */
    static createPurchase(
        accountId: bigint,
        amount: bigint,
        dollarAmount: number,
        stripeSessionId?: string,
        overrides?: Partial<Prisma.credit_ledger_entryCreateInput>,
    ): Prisma.credit_ledger_entryCreateInput {
        return {
            id: generatePK(),
            idempotencyKey: stripeSessionId || `purchase_${Date.now()}`,
            amount,
            type: "Purchase",
            source: "Stripe",
            meta: {
                stripeSessionId: stripeSessionId || `cs_test_${Date.now()}`,
                amount: dollarAmount,
                currency: "USD",
                creditsPerDollar: Number(amount) / dollarAmount,
            },
            account: { connect: { id: accountId } },
            ...overrides,
        };
    }

    /**
     * Create spending entry
     */
    static createSpend(
        accountId: bigint,
        amount: bigint,
        operation: string,
        metadata?: any,
        overrides?: Partial<Prisma.credit_ledger_entryCreateInput>,
    ): Prisma.credit_ledger_entryCreateInput {
        return {
            id: generatePK(),
            idempotencyKey: `spend_${operation}_${Date.now()}`,
            amount: -Math.abs(Number(amount)), // Ensure negative
            type: "Spend",
            source: "InternalAgent",
            meta: {
                operation,
                timestamp: new Date().toISOString(),
                ...metadata,
            },
            account: { connect: { id: accountId } },
            ...overrides,
        };
    }

    /**
     * Create transfer entry
     */
    static createTransfer(
        accountId: bigint,
        amount: bigint,
        direction: "in" | "out",
        fromAccountId?: bigint,
        toAccountId?: bigint,
        overrides?: Partial<Prisma.credit_ledger_entryCreateInput>,
    ): Prisma.credit_ledger_entryCreateInput {
        const transferId = `tf_${Date.now()}`;

        return {
            id: generatePK(),
            idempotencyKey: `transfer_${direction}_${transferId}`,
            amount: direction === "in" ? amount : -Math.abs(Number(amount)),
            type: direction === "in" ? "TransferIn" : "TransferOut",
            source: "InternalAgent",
            meta: {
                operation: "account_transfer",
                transferId,
                fromAccountId: fromAccountId?.toString(),
                toAccountId: toAccountId?.toString(),
                timestamp: new Date().toISOString(),
            },
            account: { connect: { id: accountId } },
            ...overrides,
        };
    }

    /**
     * Create bonus entry
     */
    static createBonus(
        accountId: bigint,
        amount: bigint,
        bonusType: string,
        metadata?: any,
        overrides?: Partial<Prisma.credit_ledger_entryCreateInput>,
    ): Prisma.credit_ledger_entryCreateInput {
        return {
            id: generatePK(),
            idempotencyKey: `bonus_${bonusType}_${Date.now()}`,
            amount,
            type: "Bonus",
            source: "Admin",
            meta: {
                operation: bonusType,
                timestamp: new Date().toISOString(),
                ...metadata,
            },
            account: { connect: { id: accountId } },
            ...overrides,
        };
    }

    /**
     * Create refund entry
     */
    static createRefund(
        accountId: bigint,
        amount: bigint,
        originalSessionId: string,
        reason: string,
        overrides?: Partial<Prisma.credit_ledger_entryCreateInput>,
    ): Prisma.credit_ledger_entryCreateInput {
        return {
            id: generatePK(),
            idempotencyKey: `refund_${originalSessionId}`,
            amount,
            type: "Refund",
            source: "Stripe",
            meta: {
                operation: "payment_refund",
                originalStripeSessionId: originalSessionId,
                reason,
                timestamp: new Date().toISOString(),
            },
            account: { connect: { id: accountId } },
            ...overrides,
        };
    }

    /**
     * Create a sequence of entries for testing transaction flows
     */
    static createTransactionSequence(
        accountId: bigint,
        transactions: Array<{
            type: "purchase" | "spend" | "bonus" | "transfer_in" | "transfer_out" | "refund";
            amount: bigint;
            metadata?: any;
        }>,
    ): Prisma.credit_ledger_entryCreateInput[] {
        return transactions.map((tx, index) => {
            const baseKey = `seq_${Date.now()}_${index}`;

            switch (tx.type) {
                case "purchase":
                    return this.createPurchase(accountId, tx.amount, Number(tx.amount) / 100000, undefined, {
                        idempotencyKey: `${baseKey}_purchase`,
                    });
                case "spend":
                    return this.createSpend(accountId, tx.amount, "test_operation", tx.metadata, {
                        idempotencyKey: `${baseKey}_spend`,
                    });
                case "bonus":
                    return this.createBonus(accountId, tx.amount, "test_bonus", tx.metadata, {
                        idempotencyKey: `${baseKey}_bonus`,
                    });
                case "transfer_in":
                    return this.createTransfer(accountId, tx.amount, "in", undefined, undefined, {
                        idempotencyKey: `${baseKey}_transfer_in`,
                    });
                case "transfer_out":
                    return this.createTransfer(accountId, tx.amount, "out", undefined, undefined, {
                        idempotencyKey: `${baseKey}_transfer_out`,
                    });
                case "refund":
                    return this.createRefund(accountId, tx.amount, `original_${baseKey}`, "test_refund", {
                        idempotencyKey: `${baseKey}_refund`,
                    });
                default:
                    throw new Error(`Unknown transaction type: ${tx.type}`);
            }
        });
    }
}

/**
 * Helper functions for seeding test data
 */

/**
 * Seed comprehensive ledger entries for testing
 */
export async function seedTestLedgerEntries(db: PrismaClient, accountId: bigint) {
    const entries = await Promise.all([
        // Purchase
        db.creditLedgerEntry.create({
            data: CreditLedgerEntryDbFactory.createPurchase(accountId, 1000000n, 10.00),
        }),
        // AI spending
        db.creditLedgerEntry.create({
            data: CreditLedgerEntryDbFactory.createSpend(accountId, 50000n, "ai_generation", {
                model: "gpt-4",
                tokens: 1500,
            }),
        }),
        // Bonus
        db.creditLedgerEntry.create({
            data: CreditLedgerEntryDbFactory.createBonus(accountId, 25000n, "referral_bonus"),
        }),
        // Transfer in
        db.creditLedgerEntry.create({
            data: CreditLedgerEntryDbFactory.createTransfer(accountId, 100000n, "in"),
        }),
    ]);

    return entries;
}

/**
 * Create a complete transaction history for testing analytics
 */
export async function seedTransactionHistory(db: PrismaClient, accountId: bigint) {
    const sequence = CreditLedgerEntryDbFactory.createTransactionSequence(accountId, [
        { type: "purchase", amount: 2000000n }, // Initial purchase
        { type: "spend", amount: 100000n },     // Some usage
        { type: "bonus", amount: 50000n },      // Referral bonus
        { type: "spend", amount: 200000n },     // More usage
        { type: "transfer_in", amount: 300000n }, // Team allocation
        { type: "spend", amount: 150000n },     // Continued usage
        { type: "purchase", amount: 1000000n }, // Top-up purchase
        { type: "refund", amount: 100000n },    // Partial refund
    ]);

    const entries = await Promise.all(
        sequence.map(entry => db.credit_ledger_entry.create({ data: entry })),
    );

    return entries;
}
