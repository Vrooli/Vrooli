// AI_CHECK: TEST_QUALITY=1,TEST_COVERAGE=1 | LAST: 2025-06-24
import { CreditEntryType, CreditSourceSystem } from "@prisma/client";
import { CreditConfig, generatePK, generatePublicId } from "@vrooli/shared";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { creditRollover } from "./creditRollover.js";

const { DbProvider, logger, CacheService } = await import("@vrooli/server");

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
            // CacheService not available in test, continue without it
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

    describe("error handling and edge cases", () => {
        it("should handle Redis errors when checking processed status", async () => {
            // Mock Redis to throw error
            const originalRaw = CacheService.get().raw;
            CacheService.get().raw = vi.fn().mockRejectedValue(new Error("Redis connection failed"));
            
            // Should continue processing even if Redis is down
            await expect(creditRollover()).resolves.not.toThrow();
            
            // Restore original
            CacheService.get().raw = originalRaw;
        });

        it("should handle Redis errors when updating job status", async () => {
            const redis = await CacheService.get().raw();
            const originalSet = redis.set;
            
            // Create test user
            const creditAccount = await DbProvider.get().credit_account.create({
                data: {
                    id: generatePK(),
                    currentBalance: BigInt(5000000),
                },
            });
            testCreditAccountIds.push(creditAccount.id);

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

            // Mock Redis set to fail on job status update
            redis.set = vi.fn().mockImplementation((key, value) => {
                if (key.includes("job:creditRollover:lastRun")) {
                    throw new Error("Redis write failed");
                }
                return originalSet.call(redis, key, value);
            });

            // Should complete without throwing
            await expect(creditRollover()).resolves.not.toThrow();

            // Verify error was logged
            expect(logger.error).toHaveBeenCalledWith(
                "Failed to update Redis job status",
                expect.objectContaining({
                    trace: "creditRollover_redisError",
                })
            );

            // Restore
            redis.set = originalSet;
        });

        it("should handle batch processing errors for individual users", async () => {
            // Create multiple users, one with invalid settings
            const users = [];
            
            // Valid user
            const creditAccount1 = await DbProvider.get().credit_account.create({
                data: {
                    id: generatePK(),
                    currentBalance: BigInt(1000000),
                },
            });
            testCreditAccountIds.push(creditAccount1.id);

            const user1 = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Valid User",
                    handle: "validuser",
                    isBot: false,
                    creditAccount: {
                        connect: { id: creditAccount1.id },
                    },
                    creditSettings: {
                        __version: "1.0.0",
                        rollover: { enabled: true },
                        donation: { enabled: true, percentage: 10 },
                    },
                },
            });
            testUserIds.push(user1.id);
            users.push(user1);

            const plan1 = await DbProvider.get().plan.create({
                data: {
                    id: generatePK(),
                    user: {
                        connect: { id: user1.id },
                    },
                    enabledAt: new Date(Date.now() - 86400000),
                    expiresAt: new Date(Date.now() + 86400000 * 30),
                },
            });
            testPlanIds.push(plan1.id);

            // User with creditSettings as string (invalid)
            const creditAccount2 = await DbProvider.get().credit_account.create({
                data: {
                    id: generatePK(),
                    currentBalance: BigInt(1000000),
                },
            });
            testCreditAccountIds.push(creditAccount2.id);

            const user2 = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Invalid Settings User",
                    handle: "invalidsettings",
                    isBot: false,
                    creditAccount: {
                        connect: { id: creditAccount2.id },
                    },
                    creditSettings: "invalid string" as any,
                },
            });
            testUserIds.push(user2.id);
            users.push(user2);

            const plan2 = await DbProvider.get().plan.create({
                data: {
                    id: generatePK(),
                    user: {
                        connect: { id: user2.id },
                    },
                    enabledAt: new Date(Date.now() - 86400000),
                    expiresAt: new Date(Date.now() + 86400000 * 30),
                },
            });
            testPlanIds.push(plan2.id);

            await creditRollover();

            // Should log warning for invalid settings
            expect(logger.warn).toHaveBeenCalledWith(
                expect.stringContaining(`Invalid credit settings type for user ${user2.id}`),
                expect.objectContaining({
                    trace: "creditRollover_invalidSettingsType",
                })
            );

            // Valid user should still be processed
            const updatedUser1 = await DbProvider.get().user.findUnique({
                where: { id: user1.id },
                select: { creditSettings: true },
            });
            const creditConfig = new CreditConfig(updatedUser1!.creditSettings as any);
            expect(creditConfig.donation.lastProcessedMonth).toBeDefined();
        });

        it("should process credit rollover expiration correctly", async () => {
            const creditAccount = await DbProvider.get().credit_account.create({
                data: {
                    id: generatePK(),
                    currentBalance: BigInt(10000000000), // 10,000 credits (way over limit)
                },
            });
            testCreditAccountIds.push(creditAccount.id);

            const user = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Excess Credits User",
                    handle: "excessuser",
                    isBot: false,
                    creditAccount: {
                        connect: { id: creditAccount.id },
                    },
                    creditSettings: {
                        __version: "1.0.0",
                        rollover: { 
                            enabled: true,
                            maxMonthsToKeep: 2, // Only keep 2 months worth
                        },
                        donation: { enabled: false },
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

            const { BusService } = await import("@vrooli/server");
            const publishSpy = vi.spyOn(BusService.get().getBus(), "publish");

            await creditRollover();

            // Should have published expiration event
            expect(publishSpy).toHaveBeenCalledWith(
                expect.objectContaining({
                    type: "billing:event",
                    entryType: CreditEntryType.Expire,
                    source: CreditSourceSystem.Scheduler,
                })
            );
        });

        it("should handle publish errors gracefully", async () => {
            const creditAccount = await DbProvider.get().credit_account.create({
                data: {
                    id: generatePK(),
                    currentBalance: BigInt(5000000),
                },
            });
            testCreditAccountIds.push(creditAccount.id);

            const user = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Publish Error User",
                    handle: "publisherror",
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

            // Add free credits
            const ledgerEntry = await DbProvider.get().credit_ledger_entry.create({
                data: {
                    id: generatePK(),
                    idempotencyKey: generatePK().toString(),
                    accountId: creditAccount.id,
                    amount: BigInt(2000000),
                    type: CreditEntryType.Bonus,
                    source: CreditSourceSystem.Scheduler,
                    meta: { description: "Free credits" },
                },
            });
            testLedgerEntryIds.push(ledgerEntry.id);

            // Mock BusService to throw error
            const { BusService } = await import("@vrooli/server");
            BusService.get().getBus().publish = vi.fn().mockRejectedValue(new Error("Publish failed"));

            await creditRollover();

            // Should log error
            expect(logger.error).toHaveBeenCalledWith(
                expect.stringContaining("Failed to publish donation BillingEvent"),
                expect.objectContaining({
                    trace: "creditRollover_donationPublishError",
                })
            );
        });

        it("should handle updateCreditSettingsProcessedMonth retry logic", async () => {
            const creditAccount = await DbProvider.get().credit_account.create({
                data: {
                    id: generatePK(),
                    currentBalance: BigInt(1000000),
                },
            });
            testCreditAccountIds.push(creditAccount.id);

            const user = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Retry Test User",
                    handle: "retryuser",
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

            // Mock transaction to fail twice then succeed
            const originalTransaction = DbProvider.get().$transaction;
            let attemptCount = 0;
            DbProvider.get().$transaction = vi.fn().mockImplementation(async (fn) => {
                attemptCount++;
                if (attemptCount <= 2) {
                    throw new Error("Transaction failed");
                }
                return originalTransaction.call(DbProvider.get(), fn);
            });

            await creditRollover();

            // Should have retried and eventually succeeded
            expect(attemptCount).toBeGreaterThan(2);

            // Restore
            DbProvider.get().$transaction = originalTransaction;
        });

        it("should handle users with expired premium plans", async () => {
            const creditAccount = await DbProvider.get().credit_account.create({
                data: {
                    id: generatePK(),
                    currentBalance: BigInt(5000000),
                },
            });
            testCreditAccountIds.push(creditAccount.id);

            const user = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Expired Premium User",
                    handle: "expiredpremium",
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

            // Create expired plan
            const plan = await DbProvider.get().plan.create({
                data: {
                    id: generatePK(),
                    user: {
                        connect: { id: user.id },
                    },
                    enabledAt: new Date(Date.now() - 86400000 * 60), // 60 days ago
                    expiresAt: new Date(Date.now() - 86400000), // Expired yesterday
                },
            });
            testPlanIds.push(plan.id);

            await creditRollover();

            // User should not be processed
            const updatedUser = await DbProvider.get().user.findUnique({
                where: { id: user.id },
                select: { creditSettings: true },
            });
            const creditConfig = new CreditConfig(updatedUser!.creditSettings as any);
            expect(creditConfig.donation.lastProcessedMonth).toBeUndefined();
        });

        it("should process users with no rollover expiration needed", async () => {
            const creditAccount = await DbProvider.get().credit_account.create({
                data: {
                    id: generatePK(),
                    currentBalance: BigInt(1000000), // Small balance, under limit
                },
            });
            testCreditAccountIds.push(creditAccount.id);

            const user = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Small Balance User",
                    handle: "smallbalance",
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
                        donation: { enabled: false },
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

            // Should still mark rollover as processed even without expiration
            const updatedUser = await DbProvider.get().user.findUnique({
                where: { id: user.id },
                select: { creditSettings: true },
            });
            const creditConfig = new CreditConfig(updatedUser!.creditSettings as any);
            const currentMonth = `${new Date().getUTCFullYear()}-${String(new Date().getUTCMonth() + 1).padStart(2, "0")}`;
            expect(creditConfig.rollover.lastProcessedMonth).toBe(currentMonth);
        });

        it("should handle very large donation amounts", async () => {
            const creditAccount = await DbProvider.get().credit_account.create({
                data: {
                    id: generatePK(),
                    currentBalance: BigInt("999999999999999999999"), // Very large balance
                },
            });
            testCreditAccountIds.push(creditAccount.id);

            const user = await DbProvider.get().user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Large Donation User",
                    handle: "largedonation",
                    isBot: false,
                    creditAccount: {
                        connect: { id: creditAccount.id },
                    },
                    creditSettings: {
                        __version: "1.0.0",
                        rollover: { enabled: false },
                        donation: { 
                            enabled: true, 
                            percentage: 50, // 50% of huge balance
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

            // Add free credits ledger entry
            const ledgerEntry = await DbProvider.get().credit_ledger_entry.create({
                data: {
                    id: generatePK(),
                    idempotencyKey: generatePK().toString(),
                    accountId: creditAccount.id,
                    amount: BigInt("999999999999999999999"),
                    type: CreditEntryType.Bonus,
                    source: CreditSourceSystem.Scheduler,
                    meta: { description: "Large free credits" },
                },
            });
            testLedgerEntryIds.push(ledgerEntry.id);

            await creditRollover();

            // Should log warning about large donation
            expect(logger.warn).toHaveBeenCalledWith(
                expect.stringContaining("Large donation amount"),
                expect.objectContaining({
                    trace: "creditRollover_largeDonation",
                })
            );
        });
    });
});
