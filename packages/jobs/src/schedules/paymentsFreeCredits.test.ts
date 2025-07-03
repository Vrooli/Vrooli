// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-24
import { CreditEntryType, CreditSourceSystem } from "@prisma/client";
import { API_CREDITS_PREMIUM, generatePK, generatePublicId, EventTypes } from "@vrooli/shared";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Mock the services at module level
vi.mock("@vrooli/server", async () => {
    const actual = await vi.importActual("@vrooli/server");
    return {
        ...actual,
        BusService: {
            get: vi.fn(),
        },
        SocketService: {
            get: vi.fn(),
        },
        Notify: vi.fn(),
    };
});

import { paymentsCreditsFreePremium } from "./paymentsFreeCredits.js";
import { DbProvider, BusService, SocketService, Notify } from "@vrooli/server";

describe("paymentsCreditsFreePremium integration tests", () => {
    // Store test entity IDs for cleanup
    const testUserIds: bigint[] = [];
    const testCreditAccountIds: bigint[] = [];
    const testPlanIds: bigint[] = [];

    // Mock services
    const mockBusPublish = vi.fn().mockResolvedValue(undefined);
    const mockSocketEmit = vi.fn();
    const mockNotify = vi.fn(() => ({
        pushFreeCreditsReceived: vi.fn().mockReturnValue({ 
            toUser: vi.fn().mockResolvedValue(undefined), 
        }),
    }));

    beforeEach(async () => {
        // Clear test ID arrays
        testUserIds.length = 0;
        testCreditAccountIds.length = 0;
        testPlanIds.length = 0;

        // Setup mocks
        vi.mocked(BusService.get).mockReturnValue({
            getBus: () => ({
                publish: mockBusPublish,
            }),
        } as any);

        vi.mocked(SocketService.get).mockReturnValue({
            emitSocketEvent: mockSocketEmit,
        } as any);

        vi.mocked(Notify).mockImplementation(mockNotify);

        // Reset mocks
        mockBusPublish.mockClear();
        mockSocketEmit.mockClear();
        mockNotify.mockClear();
    });

    afterEach(async () => {
        // Clean up test data
        const db = DbProvider.get();
        
        // Clean up in reverse dependency order
        if (testPlanIds.length > 0) {
            await db.plan.deleteMany({ where: { id: { in: testPlanIds } } });
        }
        if (testCreditAccountIds.length > 0) {
            await db.credit_account.deleteMany({ where: { id: { in: testCreditAccountIds } } });
        }
        if (testUserIds.length > 0) {
            await db.user.deleteMany({ where: { id: { in: testUserIds } } });
        }
    });

    it("should award free credits to premium users with low balance", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 86400000); // Yesterday
        const futureDate = new Date(now.getTime() + 86400000); // Tomorrow
        
        // Create user with active premium plan
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Premium User",
                handle: "premiumuser",
                isBot: false,
                languages: ["en"],
                stripeCustomerId: "cus_premium",
            },
        });
        testUserIds.push(user.id);

        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                userId: user.id,
                currentBalance: BigInt(100), // Low balance
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: user.id,
                enabledAt: pastDate,
                expiresAt: futureDate,
            },
        });
        testPlanIds.push(plan.id);

        await paymentsCreditsFreePremium();

        // Check that billing event was published
        expect(mockBusPublish).toHaveBeenCalledWith(
            expect.objectContaining({
                type: "billing:event",
                accountId: creditAccount.id.toString(),
                delta: API_CREDITS_PREMIUM.toString(),
                entryType: CreditEntryType.Bonus,
                source: CreditSourceSystem.Scheduler,
                meta: expect.objectContaining({
                    reason: "Monthly premium free credits",
                    jobName: "paymentsCreditsFreePremium",
                    originalUserId: user.id.toString(),
                }),
            }),
        );

        // Check that notification was sent
        expect(mockNotify).toHaveBeenCalledWith(["en"]);

        // Check that socket event was emitted
        expect(mockSocketEmit).toHaveBeenCalledWith(
            EventTypes.USER.CREDITS_UPDATED,
            user.id.toString(),
            { userId: user.id.toString(), credits: (BigInt(100) + API_CREDITS_PREMIUM).toString() },
        );
    });

    it("should cap credits at maximum allowed", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 86400000);
        const futureDate = new Date(now.getTime() + 86400000);
        
        const MAX_FREE_CREDITS = BigInt(6) * API_CREDITS_PREMIUM;
        const almostMaxCredits = MAX_FREE_CREDITS - BigInt(100);
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Almost Max User",
                handle: "almostmaxuser",
                isBot: false,
                languages: ["en"],
                stripeCustomerId: "cus_almostmax",
            },
        });
        testUserIds.push(user.id);

        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                userId: user.id,
                currentBalance: almostMaxCredits,
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: user.id,
                enabledAt: pastDate,
                expiresAt: futureDate,
            },
        });
        testPlanIds.push(plan.id);

        await paymentsCreditsFreePremium();

        // Check that only the difference to max was added
        expect(mockBusPublish).toHaveBeenCalledWith(
            expect.objectContaining({
                delta: BigInt(100).toString(), // Only add enough to reach max
            }),
        );

        // Check socket event shows max balance
        expect(mockSocketEmit).toHaveBeenCalledWith(
            EventTypes.USER.CREDITS_UPDATED,
            user.id.toString(),
            { userId: user.id.toString(), credits: MAX_FREE_CREDITS.toString() },
        );
    });

    it("should not award credits to users at max balance", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 86400000);
        const futureDate = new Date(now.getTime() + 86400000);
        
        const MAX_FREE_CREDITS = BigInt(6) * API_CREDITS_PREMIUM;
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Max Credits User",
                handle: "maxcreditsuser",
                isBot: false,
                languages: ["en"],
                stripeCustomerId: "cus_max",
            },
        });
        testUserIds.push(user.id);

        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                userId: user.id,
                currentBalance: MAX_FREE_CREDITS,
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: user.id,
                enabledAt: pastDate,
                expiresAt: futureDate,
            },
        });
        testPlanIds.push(plan.id);

        await paymentsCreditsFreePremium();

        // Check that no billing event was published
        expect(mockBusPublish).not.toHaveBeenCalled();

        // Check that no notification was sent
        expect(mockNotify).not.toHaveBeenCalled();

        // Check that socket event still emitted with current balance
        expect(mockSocketEmit).toHaveBeenCalledWith(
            EventTypes.USER.CREDITS_UPDATED,
            user.id.toString(),
            { userId: user.id.toString(), credits: MAX_FREE_CREDITS.toString() },
        );
    });

    it("should not award credits to users with expired plans", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 172800000); // 2 days ago
        const yesterdayDate = new Date(now.getTime() - 86400000); // Yesterday
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Expired Plan User",
                handle: "expiredplanuser",
                isBot: false,
                languages: ["en"],
                stripeCustomerId: "cus_expired",
            },
        });
        testUserIds.push(user.id);

        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                userId: user.id,
                currentBalance: BigInt(100),
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: user.id,
                enabledAt: pastDate,
                expiresAt: yesterdayDate, // Expired
            },
        });
        testPlanIds.push(plan.id);

        await paymentsCreditsFreePremium();

        // Check that no billing event was published
        expect(mockBusPublish).not.toHaveBeenCalled();
    });

    it("should not award credits to users without enabled plans", async () => {
        const now = new Date();
        const futureDate = new Date(now.getTime() + 86400000);
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Disabled Plan User",
                handle: "disabledplanuser",
                isBot: false,
                languages: ["en"],
                stripeCustomerId: "cus_disabled",
            },
        });
        testUserIds.push(user.id);

        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                userId: user.id,
                currentBalance: BigInt(100),
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: user.id,
                enabledAt: null, // Not enabled
                expiresAt: futureDate,
            },
        });
        testPlanIds.push(plan.id);

        await paymentsCreditsFreePremium();

        // Check that no billing event was published
        expect(mockBusPublish).not.toHaveBeenCalled();
    });

    it("should skip bots", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 86400000);
        const futureDate = new Date(now.getTime() + 86400000);
        
        const bot = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Bot User",
                handle: "botuser",
                isBot: true, // Bot flag
                languages: ["en"],
                stripeCustomerId: "cus_bot",
            },
        });
        testUserIds.push(bot.id);

        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                userId: bot.id,
                currentBalance: BigInt(100),
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: bot.id,
                enabledAt: pastDate,
                expiresAt: futureDate,
            },
        });
        testPlanIds.push(plan.id);

        await paymentsCreditsFreePremium();

        // Check that no billing event was published
        expect(mockBusPublish).not.toHaveBeenCalled();
    });

    it("should handle users without credit accounts", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 86400000);
        const futureDate = new Date(now.getTime() + 86400000);
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "No Credit Account User",
                handle: "nocredituser",
                isBot: false,
                languages: ["en"],
                stripeCustomerId: "cus_nocredit",
            },
        });
        testUserIds.push(user.id);

        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: user.id,
                enabledAt: pastDate,
                expiresAt: futureDate,
            },
        });
        testPlanIds.push(plan.id);

        // Should not throw
        await expect(paymentsCreditsFreePremium()).resolves.not.toThrow();

        // Check that no billing event was published
        expect(mockBusPublish).not.toHaveBeenCalled();
    });

    it("should handle users without plans", async () => {
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "No Plan User",
                handle: "noplanuser",
                isBot: false,
                languages: ["en"],
            },
        });
        testUserIds.push(user.id);

        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                userId: user.id,
                currentBalance: BigInt(100),
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        // Should not throw
        await expect(paymentsCreditsFreePremium()).resolves.not.toThrow();

        // Check that no billing event was published
        expect(mockBusPublish).not.toHaveBeenCalled();
    });

    it("should handle batch processing with multiple users", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 86400000);
        const futureDate = new Date(now.getTime() + 86400000);
        
        // Create multiple users with varying conditions
        const userPromises = [];
        for (let i = 0; i < 5; i++) {
            userPromises.push(
                DbProvider.get().user.create({
                    data: {
                        id: generatePK(),
                        publicId: generatePublicId(),
                        name: `Batch User ${i}`,
                        handle: `batchuser${i}`,
                        isBot: false,
                        languages: ["en"],
                        stripeCustomerId: `cus_batch${i}`,
                    },
                }),
            );
        }
        const users = await Promise.all(userPromises);
        users.forEach(u => testUserIds.push(u.id));

        // Create credit accounts and plans
        const creditPromises = [];
        const planPromises = [];
        for (let i = 0; i < users.length; i++) {
            creditPromises.push(
                DbProvider.get().credit_account.create({
                    data: {
                        id: generatePK(),
                        userId: users[i].id,
                        currentBalance: BigInt(i * 1000), // Varying balances
                    },
                }),
            );
            
            // Only create active plans for some users
            if (i % 2 === 0) {
                planPromises.push(
                    DbProvider.get().plan.create({
                        data: {
                            id: generatePK(),
                            userId: users[i].id,
                            enabledAt: pastDate,
                            expiresAt: futureDate,
                        },
                    }),
                );
            }
        }
        const creditAccounts = await Promise.all(creditPromises);
        creditAccounts.forEach(c => testCreditAccountIds.push(c.id));
        const plans = await Promise.all(planPromises);
        plans.forEach(p => testPlanIds.push(p.id));

        await paymentsCreditsFreePremium();

        // Check that billing events were published for eligible users
        
        // Should have events for users with active plans and low enough balance
        expect(mockBusPublish).toHaveBeenCalled();
        
        // Verify each call has proper structure
        mockBusPublish.mock.calls.forEach((call: unknown[]) => {
            expect(call[0]).toMatchObject({
                type: "billing:event",
                entryType: CreditEntryType.Bonus,
                source: CreditSourceSystem.Scheduler,
            });
        });
    });

    it("should handle publish errors gracefully", async () => {
        const now = new Date();
        const pastDate = new Date(now.getTime() - 86400000);
        const futureDate = new Date(now.getTime() + 86400000);
        
        // Mock publish to throw error
        mockBusPublish.mockRejectedValueOnce(new Error("Publish failed"));
        
        const user = await DbProvider.get().user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Error User",
                handle: "erroruser",
                isBot: false,
                languages: ["en"],
                stripeCustomerId: "cus_error",
            },
        });
        testUserIds.push(user.id);

        const creditAccount = await DbProvider.get().credit_account.create({
            data: {
                id: generatePK(),
                userId: user.id,
                currentBalance: BigInt(100),
            },
        });
        testCreditAccountIds.push(creditAccount.id);

        const plan = await DbProvider.get().plan.create({
            data: {
                id: generatePK(),
                userId: user.id,
                enabledAt: pastDate,
                expiresAt: futureDate,
            },
        });
        testPlanIds.push(plan.id);

        // Should not throw
        await expect(paymentsCreditsFreePremium()).resolves.not.toThrow();

        // Check that notification was NOT sent due to publish failure
        expect(mockNotify).not.toHaveBeenCalled();
    });
});
