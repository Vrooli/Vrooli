import { CreditEntryType, CreditSourceSystem } from "@prisma/client";
import { generatePK, generatePublicId } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { DbProvider } from "../../db/provider.js";
import { calculateFreeCreditsBalance, getCreditBalancesBySource } from "./creditBalanceService.js";
import { CreditAccountDbFactory } from "../../__test/fixtures/db/creditAccountFixtures.js";

// Mock the logger
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
    },
}));

describe("calculateFreeCreditsBalance", () => {
    let testUserId: bigint | undefined;
    let testCreditAccountId: bigint;

    beforeAll(async () => {
        // Create a test user for all tests
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Credit Test User",
                handle: `credituser_${Date.now()}`,
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
            },
        });
        testUserId = user.id;
    });

    afterAll(async () => {
        if (!testUserId) return;
        
        // Clean up test data - use correct field name 'account'
        await DbProvider.get().credit_ledger_entry.deleteMany({
            where: { account: { userId: testUserId } },
        });
        await DbProvider.get().credit_account.deleteMany({
            where: { userId: testUserId },
        });
        await DbProvider.get().user.delete({
            where: { id: testUserId },
        });
    });

    beforeEach(async () => {
        if (!testUserId) return;
        
        // Clean up any existing credit data for this user
        await DbProvider.get().credit_ledger_entry.deleteMany({
            where: { account: { userId: testUserId } },
        });
        await DbProvider.get().credit_account.deleteMany({
            where: { userId: testUserId },
        });
        vi.clearAllMocks();
    });

    it("should calculate correct balance with only free credits", async () => {
        if (!testUserId) throw new Error("testUserId not initialized");
        
        // Create credit account with ledger entries
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: 1000000000n, // 1000 credits
                userId: testUserId,
                entries: {
                    create: [
                        {
                            id: generatePK(),
                            idempotencyKey: `bonus_${Date.now()}`,
                            amount: 1000000000n, // 1000 credits
                            type: CreditEntryType.Bonus,
                            source: CreditSourceSystem.Scheduler,
                            meta: {},
                        },
                    ],
                },
            },
        });
        testCreditAccountId = creditAccount.id;

        const balance = await calculateFreeCreditsBalance(testCreditAccountId);
        expect(balance).toBe(BigInt(1000000000));
    });

    it("should handle FIFO when spending credits", async () => {
        if (!testUserId) throw new Error("testUserId not initialized");
        
        // Create credit account with initial free credits
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: 900000000n, // 900 credits total after transactions
                userId: testUserId,
                entries: {
                    create: [
                        // Add 1000 free credits
                        {
                            id: generatePK(),
                            idempotencyKey: `bonus_${Date.now()}_1`,
                            amount: 1000000000n,
                            type: CreditEntryType.Bonus,
                            source: CreditSourceSystem.Scheduler,
                            meta: {},
                        },
                        // Spend 600 credits (should come from free)
                        {
                            id: generatePK(),
                            idempotencyKey: `spend_${Date.now()}_1`,
                            amount: -600000000n,
                            type: CreditEntryType.Spend,
                            source: CreditSourceSystem.InternalAgent,
                            meta: {},
                        },
                        // Add 500 purchased credits
                        {
                            id: generatePK(),
                            idempotencyKey: `purchase_${Date.now()}_1`,
                            amount: 500000000n,
                            type: CreditEntryType.Purchase,
                            source: CreditSourceSystem.Stripe,
                            meta: {},
                        },
                    ],
                },
            },
        });
        testCreditAccountId = creditAccount.id;

        const balance = await calculateFreeCreditsBalance(testCreditAccountId);
        // Should have 400 free credits remaining (1000 - 600)
        expect(balance).toBe(BigInt(400000000));
    });

    it("should handle spending more than free credits", async () => {
        if (!testUserId) throw new Error("testUserId not initialized");
        
        // Create credit account with transactions that use all free credits
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: 300000000n, // 300 credits remaining (500 purchased - 200 spent from purchased)
                userId: testUserId,
                entries: {
                    create: [
                        // Add 1000 free credits
                        {
                            id: generatePK(),
                            idempotencyKey: `bonus_${Date.now()}_2`,
                            amount: 1000000000n,
                            type: CreditEntryType.Bonus,
                            source: CreditSourceSystem.Scheduler,
                            meta: {},
                        },
                        // Add 500 purchased credits
                        {
                            id: generatePK(),
                            idempotencyKey: `purchase_${Date.now()}_2`,
                            amount: 500000000n,
                            type: CreditEntryType.Purchase,
                            source: CreditSourceSystem.Stripe,
                            meta: {},
                        },
                        // Spend 1200 credits (1000 from free, 200 from purchased)
                        {
                            id: generatePK(),
                            idempotencyKey: `spend_${Date.now()}_2`,
                            amount: -1200000000n,
                            type: CreditEntryType.Spend,
                            source: CreditSourceSystem.InternalAgent,
                            meta: {},
                        },
                    ],
                },
            },
        });
        testCreditAccountId = creditAccount.id;

        const balance = await calculateFreeCreditsBalance(testCreditAccountId);
        // Should have 0 free credits remaining (all spent)
        expect(balance).toBe(BigInt(0));
    });

    it("should return 0 on error", async () => {
        // Use an invalid credit account ID that doesn't exist
        const invalidAccountId = generatePK();
        
        const balance = await calculateFreeCreditsBalance(invalidAccountId);
        expect(balance).toBe(BigInt(0));
    });
});

describe("getCreditBalancesBySource", () => {
    let testUserId: bigint | undefined;
    let testCreditAccountId: bigint;

    beforeAll(async () => {
        // Create a test user for all tests
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Credit Source Test User",
                handle: `creditsourceuser_${Date.now()}`,
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
            },
        });
        testUserId = user.id;
    });

    afterAll(async () => {
        if (!testUserId) return;
        
        // Clean up test data - use correct field name 'account'
        await DbProvider.get().credit_ledger_entry.deleteMany({
            where: { account: { userId: testUserId } },
        });
        await DbProvider.get().credit_account.deleteMany({
            where: { userId: testUserId },
        });
        await DbProvider.get().user.deleteMany({
            where: { id: testUserId },
        });
    });

    beforeEach(async () => {
        if (!testUserId) return;
        
        // Clean up any existing credit data for this user
        await DbProvider.get().credit_ledger_entry.deleteMany({
            where: { account: { userId: testUserId } },
        });
        await DbProvider.get().credit_account.deleteMany({
            where: { userId: testUserId },
        });
        vi.clearAllMocks();
    });

    it("should return correct balance breakdown", async () => {
        if (!testUserId) throw new Error("testUserId not initialized");
        
        // Create credit account with mixed credit sources
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: 900000000n, // 400 free + 500 purchased
                userId: testUserId,
                entries: {
                    create: [
                        {
                            id: generatePK(),
                            idempotencyKey: `bonus_${Date.now()}_3`,
                            amount: 1000000000n,
                            type: CreditEntryType.Bonus,
                            source: CreditSourceSystem.Scheduler,
                            meta: {},
                        },
                        {
                            id: generatePK(),
                            idempotencyKey: `spend_${Date.now()}_3`,
                            amount: -600000000n,
                            type: CreditEntryType.Spend,
                            source: CreditSourceSystem.InternalAgent,
                            meta: {},
                        },
                        {
                            id: generatePK(),
                            idempotencyKey: `purchase_${Date.now()}_3`,
                            amount: 500000000n,
                            type: CreditEntryType.Purchase,
                            source: CreditSourceSystem.Stripe,
                            meta: {},
                        },
                    ],
                },
            },
        });
        testCreditAccountId = creditAccount.id;

        const balances = await getCreditBalancesBySource(testCreditAccountId);

        expect(balances.free).toBe(BigInt(400000000));
        expect(balances.purchased).toBe(BigInt(500000000));
        expect(balances.total).toBe(BigInt(900000000));
    });
});
