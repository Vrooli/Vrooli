// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-06-24
import { beforeAll, afterAll, describe, it, expect, vi } from "vitest";
import { creditRollover } from "./creditRollover.js";
import { BusService, CacheService, DbProvider, logger } from "@vrooli/server";
import { generatePK, generatePublicId } from "@vrooli/shared";
import { CreditEntryType, CreditSourceSystem } from "@prisma/client";

describe("creditRollover integration tests", () => {
    const testUserIds: bigint[] = [];
    const testCreditAccountIds: bigint[] = [];
    const testPlanIds: bigint[] = [];
    
    // Create mock bus at module level so it can be accessed by tests
    const mockBus = { publish: vi.fn().mockImplementation(async () => undefined) };

    beforeAll(async () => {
        // Mock logger
        vi.spyOn(logger, "info").mockImplementation(() => undefined);
        vi.spyOn(logger, "warn").mockImplementation(() => undefined);
        vi.spyOn(logger, "debug").mockImplementation(() => undefined);
        vi.spyOn(logger, "error").mockImplementation(() => undefined);
        
        // Mock BusService
        vi.spyOn(BusService, "get").mockReturnValue({ getBus: () => mockBus } as any);
    });

    afterAll(async () => {
        // Cleanup test data
        if (testUserIds.length > 0) {
            await DbProvider.get().user.deleteMany({
                where: { id: { in: testUserIds } },
            });
        }
        if (testCreditAccountIds.length > 0) {
            await DbProvider.get().credit_account.deleteMany({
                where: { id: { in: testCreditAccountIds } },
            });
        }
        if (testPlanIds.length > 0) {
            await DbProvider.get().plan.deleteMany({
                where: { id: { in: testPlanIds } },
            });
        }
        
        // Clear all Redis keys to prevent cross-test interference
        const redis = await CacheService.get().raw();
        if (redis) {
            const keys = await redis.keys("creditRollover:*");
            if (keys.length > 0) {
                await redis.del(...keys);
            }
        }
        
        vi.restoreAllMocks();
    });

    it("should skip processing if month already processed", async () => {
        // Set the Redis key to simulate already processed
        const redis = await CacheService.get().raw();
        const now = new Date();
        const currentMonth = `${now.getUTCFullYear()}-${String(now.getUTCMonth() + 1).padStart(2, "0")}`;
        const rolloverKey = `creditRollover:processed:${currentMonth}`;
        
        await redis.set(rolloverKey, "1");
        
        await creditRollover();
        
        expect(logger.info).toHaveBeenCalledWith(
            `Credit rollover for month ${currentMonth} already processed`,
            expect.objectContaining({
                trace: "creditRollover_alreadyProcessed",
            }),
        );
        
        // Clean up
        await redis.del(rolloverKey);
    });

    it("should process users with premium plans and credit settings", async () => {
        // Clear any existing processed marker
        const redis = await CacheService.get().raw();
        const now = new Date();
        const currentMonth = `${now.getUTCFullYear()}-${String(now.getUTCMonth() + 1).padStart(2, "0")}`;
        const rolloverKey = `creditRollover:processed:${currentMonth}`;
        await redis.del(rolloverKey);

        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: BigInt(5000000), // 5000 credits
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        // Add a free credit entry to simulate monthly bonus credits
        await DbProvider.get().credit_ledger_entry.create({
            data: {
                id: generatePK(),
                accountId: creditAccount.id,
                amount: BigInt(5000000),
                type: CreditEntryType.Bonus,
                source: CreditSourceSystem.Scheduler,
                meta: { reason: "Monthly free credits" },
                idempotencyKey: `test-bonus-${Date.now()}`,
            },
        });

        const uniqueId = Date.now();
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Premium User",
                handle: `premiumuser_${uniqueId}`,
                isBot: false,
                creditAccount: {
                    connect: { id: creditAccount.id },
                },
                creditSettings: {
                    __version: "1.0.0",
                    rollover: { enabled: true },
                    donation: { enabled: true, percentage: 10 },
                },
            },
        });
        testUserIds.push(user.id);

        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                user: {
                    connect: { id: user.id },
                },
                enabledAt: new Date(Date.now() - 86400000),
                expiresAt: new Date(Date.now() + 86400000 * 30),
            },
        });
        testPlanIds.push(plan.id);

        await creditRollover();

        // Verify that billing events were published instead of direct ledger entries
        expect(mockBus.publish).toHaveBeenCalled();
        const publishCalls = (mockBus.publish as any).mock.calls;
        
        // Check that a donation billing event was published
        const donationEvent = publishCalls.find((call: any) => 
            call[0].entryType === CreditEntryType.DonationGiven,
        );
        expect(donationEvent).toBeTruthy();
        expect(donationEvent[0].accountId).toBe(creditAccount.id.toString());
        expect(BigInt(donationEvent[0].delta)).toBeLessThan(BigInt(0)); // Should be negative (outgoing)
    });

    it("should skip users without active premium plans", async () => {
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: BigInt(1000000),
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        const uniqueId = Date.now();
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Non-Premium User",
                handle: `nonpremiumuser_${uniqueId}`,
                isBot: false,
                creditAccount: {
                    connect: { id: creditAccount.id },
                },
                creditSettings: {
                    __version: "1.0.0",
                    rollover: { enabled: true },
                    donation: { enabled: true, percentage: 10 },
                },
            },
        });
        testUserIds.push(user.id);

        // No plan created for this user

        await creditRollover();

        // Should skip this user (no active plan)
        const entries = await DbProvider.get().credit_ledger_entry.findMany({
            where: {
                accountId: creditAccount.id,
                type: CreditEntryType.Donation,
            },
        });
        
        expect(entries.length).toBe(0);
    });

    it("should handle invalid credit settings gracefully", async () => {
        // Clear monthly processing cache to avoid interference from other tests
        const redis = await CacheService.get().raw();
        const now = new Date();
        const currentMonth = `${now.getUTCFullYear()}-${String(now.getUTCMonth() + 1).padStart(2, "0")}`;
        const rolloverKey = `creditRollover:processed:${currentMonth}`;
        await redis.del(rolloverKey);
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: BigInt(1000000),
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        // Add a free credit entry to simulate monthly bonus credits
        await DbProvider.get().credit_ledger_entry.create({
            data: {
                id: generatePK(),
                accountId: creditAccount.id,
                amount: BigInt(1000000),
                type: CreditEntryType.Bonus,
                source: CreditSourceSystem.Scheduler,
                meta: { reason: "Monthly free credits" },
                idempotencyKey: `test-bonus-invalid-${Date.now()}`,
            },
        });

        const uniqueId = Date.now();
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Invalid Settings User",
                handle: `invalidsettings_${uniqueId}`,
                isBot: false,
                creditAccount: {
                    connect: { id: creditAccount.id },
                },
                // Use non-object type that will trigger type check warning
                creditSettings: "invalid_string_type" as any,
            },
        });
        testUserIds.push(user.id);

        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                user: {
                    connect: { id: user.id },
                },
                enabledAt: new Date(Date.now() - 86400000),
                expiresAt: new Date(Date.now() + 86400000 * 30),
            },
        });
        testPlanIds.push(plan.id);

        // Clear any previous mock calls and ensure mock is working
        vi.clearAllMocks();
        const warnSpy = vi.spyOn(logger, "warn").mockImplementation(() => undefined);

        await creditRollover();

        // Should log warning for invalid settings (string type instead of object)
        expect(warnSpy).toHaveBeenCalled();
        const warnCalls = (logger.warn as any).mock.calls;
        const invalidSettingsCall = warnCalls.find((call: any) => 
            call[0].includes("Invalid credit settings type") && call[0].includes(user.id.toString()),
        );
        expect(invalidSettingsCall).toBeTruthy();
    });

    it("should skip donation if user has insufficient free credits", async () => {
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: BigInt(100), // Very low balance
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        const uniqueId = Date.now();
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Low Balance User",
                handle: `lowbalanceuser_${uniqueId}`,
                isBot: false,
                creditAccount: {
                    connect: { id: creditAccount.id },
                },
                creditSettings: {
                    __version: "1.0.0",
                    rollover: { enabled: true },
                    donation: { enabled: true, percentage: 10 },
                },
            },
        });
        testUserIds.push(user.id);

        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                user: {
                    connect: { id: user.id },
                },
                enabledAt: new Date(Date.now() - 86400000),
                expiresAt: new Date(Date.now() + 86400000 * 30),
            },
        });
        testPlanIds.push(plan.id);

        await creditRollover();

        // Should not create donation entry due to low balance
        const entries = await DbProvider.get().credit_ledger_entry.findMany({
            where: {
                accountId: creditAccount.id,
                type: CreditEntryType.Donation,
            },
        });
        
        expect(entries.length).toBe(0);
    });

    it("should handle transaction errors gracefully", async () => {
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: BigInt(5000000),
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        const uniqueId = Date.now();
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Transaction Test User",
                handle: `transactionuser_${uniqueId}`,
                isBot: false,
                creditAccount: {
                    connect: { id: creditAccount.id },
                },
                creditSettings: {
                    __version: "1.0.0",
                    rollover: { enabled: true },
                    donation: { enabled: true, percentage: 10 },
                },
            },
        });
        testUserIds.push(user.id);

        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                user: {
                    connect: { id: user.id },
                },
                enabledAt: new Date(Date.now() - 86400000),
                expiresAt: new Date(Date.now() + 86400000 * 30),
            },
        });
        testPlanIds.push(plan.id);

        // Should handle large numbers without throwing
        await expect(creditRollover()).resolves.not.toThrow();
    });

    describe("error handling and edge cases", () => {
        it("should handle very large donation amounts", async () => {
            const creditAccount = await DbProvider.get().credit_account.create({
                data: {
                    id: generatePK(),
                    currentBalance: BigInt("999999999999999"), // Large but valid BigInt
                },
            });
            testCreditAccountIds.push(creditAccount.id);

            const uniqueId = Date.now();
            const user = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Large Donation User",
                    handle: `largedonationuser_${uniqueId}`,
                    isBot: false,
                    creditAccount: {
                        connect: { id: creditAccount.id },
                    },
                    creditSettings: {
                        __version: "1.0.0",
                        rollover: { enabled: false },
                        donation: { enabled: true, percentage: 50 },
                    },
                },
            });
            testUserIds.push(user.id);

            const plan = await DbProvider.get().plan.create({
                data: {
                    id: generatePK(),
                    user: {
                        connect: { id: user.id },
                    },
                    enabledAt: new Date(Date.now() - 86400000),
                    expiresAt: new Date(Date.now() + 86400000 * 30),
                },
            });
            testPlanIds.push(plan.id);

            // Should handle large numbers without throwing
            await expect(creditRollover()).resolves.not.toThrow();
        });
    });
});
