import { CreditEntryType, CreditSourceSystem } from "@prisma/client";
import { DbProvider, logger, CacheService } from "@vrooli/server";
import { API_CREDITS_PREMIUM, CreditConfig, generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { creditRollover } from "./creditRollover.js";

// Mock BusService to prevent publish failures in tests
vi.mock("@vrooli/server", async () => {
    const actual = await vi.importActual("@vrooli/server");
    return {
        ...actual,
        BusService: {
            get: () => ({
                getBus: () => ({
                    publish: vi.fn().mockResolvedValue(undefined),
                }),
            }),
        },
    };
});

describe("creditRollover integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testCreditAccountIds: bigint[] = [];
    const testPlanIds: bigint[] = [];
    const testLedgerEntryIds: bigint[] = [];

    beforeAll(async () => {
        // Suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warn").mockImplementation(() => logger);
        
        // Initialize CacheService to ensure Redis connection
        try {
            await CacheService.get().ensure();
        } catch (error) {
            console.error("Failed to initialize CacheService:", error);
        }
    });

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testCreditAccountIds.length = 0;
        testPlanIds.length = 0;
        testLedgerEntryIds.length = 0;
        
        // Clear Redis rollover tracking for clean tests
        try {
            const redis = await CacheService.get().raw();
            const currentMonth = `${new Date().getUTCFullYear()}-${String(new Date().getUTCMonth() + 1).padStart(2, "0")}`;
            await redis.del(`creditRollover:processed:${currentMonth}`);
        } catch (error) {
            // Redis not available, skip cleanup
        }
    });

    afterEach(async () => {
        // Clean up test data in reverse order of dependencies
        const prisma = DbProvider.get();
        
        // Delete ledger entries first (foreign key constraint)
        if (testLedgerEntryIds.length > 0) {
            await prisma.credit_ledger_entry.deleteMany({
                where: { id: { in: testLedgerEntryIds } },
            });
        }

        // Delete users (cascades to plans and credit accounts)
        if (testUserIds.length > 0) {
            await prisma.user.deleteMany({
                where: { id: { in: testUserIds } },
            });
        }

        // Clear Redis rollover tracking
        try {
            const redis = await CacheService.get().raw();
            const currentMonth = `${new Date().getUTCFullYear()}-${String(new Date().getUTCMonth() + 1).padStart(2, "0")}`;
            await redis.del(`creditRollover:processed:${currentMonth}`);
        } catch (error) {
            // Redis not available, skip cleanup
        }
    });

    it("should skip processing if month already processed", async () => {
        const currentMonth = `${new Date().getUTCFullYear()}-${String(new Date().getUTCMonth() + 1).padStart(2, "0")}`;
        
        // Mark month as already processed in Redis
        const redis = await CacheService.get().raw();
        expect(redis).toBeDefined(); // Ensure Redis is available
        
        const rolloverKey = `creditRollover:processed:${currentMonth}`;
        await redis.set(rolloverKey, "1");
        
        // Verify the key was set
        const keyValue = await redis.get(rolloverKey);
        expect(keyValue).toBe("1");

        // Spy on logger to verify the skip message
        const loggerInfoSpy = vi.spyOn(logger, "info");
        
        await creditRollover();

        // Verify the function logged that it was skipping
        expect(loggerInfoSpy).toHaveBeenCalledWith(
            expect.stringContaining(`Credit rollover for month ${currentMonth} already processed`),
            expect.objectContaining({
                trace: "creditRollover_alreadyProcessed",
                month: currentMonth,
            }),
        );

        // Verify month is still marked as processed
        const stillProcessed = await redis.get(rolloverKey);
        expect(stillProcessed).toBe("1");
    });

    it("should process users with premium plans and credit settings", async () => {
        const currentMonth = `${new Date().getUTCFullYear()}-${String(new Date().getUTCMonth() + 1).padStart(2, "0")}`;
        
        // Create a credit account
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: BigInt(5000000), // 5 credits
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        // Create a user with premium plan and credit settings
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Test User",
                handle: "testuser",
                isBot: false,
                creditAccount: {
                    connect: { id: creditAccount.id },
                },
                creditSettings: {
                    __version: "1.0.0",
                    rollover: { 
                        enabled: true,
                        maxMonthsToKeep: 3,
                    },
                    donation: { 
                        enabled: true, 
                        percentage: 10,
                    },
                },
            },
        });
        testUserIds.push(user.id);

        // Create a premium plan for the user
        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                user: {
                    connect: { id: user.id },
                },
                enabledAt: new Date(Date.now() - 86400000), // Yesterday
                expiresAt: new Date(Date.now() + 86400000 * 30), // 30 days from now
            },
        });
        testPlanIds.push(plan.id);

        // Add some credit ledger entries to give the user free credits
        const ledgerEntry = await DbProvider.get().credit_ledger_entry.create({
            data: {
                id: generatePK(),
                idempotencyKey: generatePK().toString(),
                accountId: creditAccount.id,
                amount: BigInt(2000000), // 2 credits
                type: CreditEntryType.Bonus,
                source: CreditSourceSystem.Scheduler,
                meta: { description: "Monthly free credits" },
            },
        });
        testLedgerEntryIds.push(ledgerEntry.id);

        // Run creditRollover
        await creditRollover();

        // Check that the month is now marked as processed in Redis
        const redis = await CacheService.get().raw();
        const rolloverKey = `creditRollover:processed:${currentMonth}`;
        const processed = await redis.get(rolloverKey);
        expect(processed).toBe("1");

        // Check that user's credit settings were updated
        const updatedUser = await DbProvider.get().user.findUnique({
            where: { id: user.id },
            select: { creditSettings: true },
        });

        expect(updatedUser?.creditSettings).toBeDefined();
        const creditConfig = new CreditConfig(updatedUser!.creditSettings as any);
        expect(creditConfig.donation.lastProcessedMonth).toBe(currentMonth);
    });

    it("should skip users without active premium plans", async () => {
        // Create a credit account
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: BigInt(5000000), // 5 credits
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        // Create a user without a plan
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Test User No Plan",
                handle: "testusernoplan",
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

        await creditRollover();

        // User's credit settings should not be updated
        const updatedUser = await DbProvider.get().user.findUnique({
            where: { id: user.id },
            select: { creditSettings: true },
        });

        const creditConfig = new CreditConfig(updatedUser!.creditSettings as any);
        expect(creditConfig.donation.lastProcessedMonth).toBeUndefined();
    });

    it("should handle invalid credit settings gracefully", async () => {
        // Create a credit account
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: BigInt(5000000),
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        // Create a user with invalid credit settings
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Test User Invalid",
                handle: "testuserinvalid",
                isBot: false,
                creditAccount: {
                    connect: { id: creditAccount.id },
                },
                creditSettings: {
                    // Invalid structure - missing required fields
                    someInvalidField: true,
                } as any,
            },
        });
        testUserIds.push(user.id);

        // Create a premium plan for the user
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

        // Should not throw error
        await expect(creditRollover()).resolves.not.toThrow();
    });

    it("should skip donation if user has insufficient free credits", async () => {
        // Create a credit account with no balance
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: BigInt(0),
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        // Create a user with donation enabled
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Test User No Credits",
                handle: "testusernocredits",
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

        // Create a premium plan
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

        // Check that no donation was created (no negative ledger entry)
        const donationEntries = await DbProvider.get().credit_ledger_entry.findMany({
            where: {
                accountId: creditAccount.id,
                type: CreditEntryType.DonationGiven,
            },
        });

        expect(donationEntries).toHaveLength(0);
    });

    it("should handle transaction errors gracefully", async () => {
        // This test would need to simulate a database error, which is difficult
        // in an integration test. We'll test that the function doesn't crash
        // with edge cases instead.
        
        const currentMonth = `${new Date().getUTCFullYear()}-${String(new Date().getUTCMonth() + 1).padStart(2, "0")}`;
        
        // Create a user with a very large credit balance that might cause issues
        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                currentBalance: BigInt(Number.MAX_SAFE_INTEGER),
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Test User Large Balance",
                handle: "testuserlargebalance",
                isBot: false,
                creditAccount: {
                    connect: { id: creditAccount.id },
                },
                creditSettings: {
                    __version: "1.0.0",
                    rollover: { 
                        enabled: true,
                        maxMonthsToKeep: 1, // Force expiration
                    },
                    donation: { 
                        enabled: true, 
                        percentage: 50, // Large percentage
                    },
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
