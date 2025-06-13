import { CreditEntryType, CreditSourceSystem, type Prisma } from "@prisma/client";
import { DbProvider, batch, BusService, logger } from "@vrooli/server";
import { API_CREDITS_PREMIUM, CreditConfig, type CreditConfigObject, generatePK } from "@vrooli/shared";
import { afterAll, beforeAll, beforeEach, describe, expect, it, vi, type MockedFunction } from "vitest";
import { creditRollover } from "./creditRollover.js";
import { calculateFreeCreditsBalance } from "@vrooli/server";

// Mock dependencies
vi.mock("@vrooli/server", async () => {
    const actual = await vi.importActual("@vrooli/server");
    return {
        ...actual,
        DbProvider: {
            get: vi.fn(),
        },
        batch: vi.fn(),
        BusService: {
            get: vi.fn(() => ({
                pub: vi.fn(),
            })),
        },
        logger: {
            info: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        },
        calculateFreeCreditsBalance: vi.fn(),
    };
});

describe("creditRollover", () => {
    let mockPrisma: any;
    let mockBus: any;

    beforeAll(() => {
        // Suppress logger output during tests
        vi.spyOn(logger, "error").mockImplementation(() => logger);
        vi.spyOn(logger, "info").mockImplementation(() => logger);
        vi.spyOn(logger, "warn").mockImplementation(() => logger);
    });

    beforeEach(() => {
        vi.clearAllMocks();
        
        // Mock prisma client
        mockPrisma = {
            billing_event: {
                findFirst: vi.fn(),
            },
            $transaction: vi.fn((fn) => fn(mockPrisma)),
            credit_ledger_entry: {
                createMany: vi.fn(),
            },
            user: {
                update: vi.fn(),
            },
        };
        
        // Mock bus service
        mockBus = {
            pub: vi.fn(),
        };
        
        (DbProvider.get as MockedFunction<typeof DbProvider.get>).mockReturnValue(mockPrisma as any);
        (BusService.get as MockedFunction<typeof BusService.get>).mockReturnValue(mockBus as any);
    });

    afterAll(() => {
        vi.restoreAllMocks();
    });

    it("should skip processing if month already processed", async () => {
        const currentMonth = `${new Date().getUTCFullYear()}-${String(new Date().getUTCMonth() + 1).padStart(2, '0')}`;
        
        // Mock existing rollover event
        mockPrisma.billing_event.findFirst.mockResolvedValue({
            id: BigInt(1),
            type: "CREDIT_ROLLOVER_MONTHLY",
            meta: { month: currentMonth },
        });

        await creditRollover();

        expect(mockPrisma.billing_event.findFirst).toHaveBeenCalledWith({
            where: {
                type: "CREDIT_ROLLOVER_MONTHLY",
                meta: {
                    path: ["month"],
                    equals: currentMonth,
                },
            },
        });
        expect(batch).not.toHaveBeenCalled();
        expect(logger.info).toHaveBeenCalledWith(
            expect.stringContaining("already processed"),
            expect.objectContaining({ month: currentMonth })
        );
    });

    it("should process users with premium plans and credit settings", async () => {
        const currentMonth = `${new Date().getUTCFullYear()}-${String(new Date().getUTCMonth() + 1).padStart(2, '0')}`;
        
        // Mock no existing rollover
        mockPrisma.billing_event.findFirst.mockResolvedValue(null);
        
        // Mock batch processing
        const mockUser = {
            id: BigInt(123),
            creditSettings: {
                __version: "1.0.0",
                rollover: { enabled: true },
                donation: { enabled: true, percentage: 10 },
            },
            languages: ["en"],
            plan: {
                id: BigInt(1),
                enabledAt: new Date(Date.now() - 86400000), // Yesterday
                expiresAt: new Date(Date.now() + 86400000), // Tomorrow
            },
            creditAccount: {
                id: BigInt(456),
                currentBalance: BigInt(5000000), // 5 credits
            },
        };
        
        // Mock calculateFreeCreditsBalance to return 2 credits (2M microcredits)
        (calculateFreeCreditsBalance as MockedFunction<typeof calculateFreeCreditsBalance>)
            .mockResolvedValue(BigInt(2000000));
        
        // Mock batch to immediately process the mock user
        (batch as MockedFunction<typeof batch>).mockImplementation(async ({ processBatch }) => {
            await processBatch([mockUser]);
        });

        await creditRollover();

        expect(batch).toHaveBeenCalledWith(expect.objectContaining({
            objectType: "User",
            batchSize: 100,
            where: {
                isBot: false,
                plan: {
                    enabledAt: { not: null },
                },
                creditAccount: {
                    isNot: null,
                },
                creditSettings: {
                    not: null,
                },
            },
        }));

        // Verify donation calculation (10% of 2 credits = 0.2 credits = 200000 microcredits)
        expect(mockPrisma.credit_ledger_entry.createMany).toHaveBeenCalledWith({
            data: [
                {
                    id: expect.any(String),
                    creditAccountId: BigInt(456),
                    amount: BigInt(-200000), // 10% of 2M
                    type: CreditEntryType.DonationGiven,
                    source: CreditSourceSystem.User,
                    description: "Monthly donation to swarm operations",
                    meta: {
                        month: currentMonth,
                        rolloverReason: "Monthly premium benefit",
                    },
                },
                {
                    id: expect.any(String),
                    creditAccountId: BigInt(0), // System account
                    amount: BigInt(200000),
                    type: CreditEntryType.DonationReceived,
                    source: CreditSourceSystem.User,
                    description: "Monthly donation from premium users",
                    meta: {
                        month: currentMonth,
                        donorUserId: "123",
                    },
                },
            ],
        });

        // Verify rollover notification
        expect(mockBus.pub).toHaveBeenCalledWith("user-notification", expect.objectContaining({
            type: "Rollover",
            recipientId: BigInt(123),
        }));

        // Verify month marked as processed
        expect(mockBus.pub).toHaveBeenCalledWith("billing", expect.objectContaining({
            type: "CREDIT_ROLLOVER_MONTHLY",
            meta: { month: currentMonth },
        }));
    });

    it("should skip users without active premium plans", async () => {
        mockPrisma.billing_event.findFirst.mockResolvedValue(null);
        
        const mockUser = {
            id: BigInt(123),
            creditSettings: {
                __version: "1.0.0",
                rollover: { enabled: true },
                donation: { enabled: true, percentage: 10 },
            },
            languages: ["en"],
            plan: {
                id: BigInt(1),
                enabledAt: new Date(Date.now() - 86400000),
                expiresAt: new Date(Date.now() - 3600000), // Expired an hour ago
            },
            creditAccount: {
                id: BigInt(456),
                currentBalance: BigInt(5000000),
            },
        };
        
        (batch as MockedFunction<typeof batch>).mockImplementation(async ({ processBatch }) => {
            await processBatch([mockUser]);
        });

        await creditRollover();

        // Should not process any donations
        expect(mockPrisma.credit_ledger_entry.createMany).not.toHaveBeenCalled();
        expect(mockBus.pub).toHaveBeenCalledTimes(1); // Only the completion marker
    });

    it("should handle invalid credit settings gracefully", async () => {
        mockPrisma.billing_event.findFirst.mockResolvedValue(null);
        
        const mockUser = {
            id: BigInt(123),
            creditSettings: "invalid-json", // Invalid settings
            languages: ["en"],
            plan: {
                id: BigInt(1),
                enabledAt: new Date(Date.now() - 86400000),
                expiresAt: null, // No expiration
            },
            creditAccount: {
                id: BigInt(456),
                currentBalance: BigInt(5000000),
            },
        };
        
        (batch as MockedFunction<typeof batch>).mockImplementation(async ({ processBatch }) => {
            await processBatch([mockUser]);
        });

        await creditRollover();

        // Should log warning but not crash
        expect(logger.warn).toHaveBeenCalledWith(
            expect.stringContaining("Invalid credit settings type"),
            expect.any(Object)
        );
        expect(mockPrisma.credit_ledger_entry.createMany).not.toHaveBeenCalled();
    });

    it("should skip donation if user has insufficient free credits", async () => {
        mockPrisma.billing_event.findFirst.mockResolvedValue(null);
        
        const mockUser = {
            id: BigInt(123),
            creditSettings: {
                __version: "1.0.0",
                rollover: { enabled: true },
                donation: { enabled: true, percentage: 50 },
            },
            languages: ["en"],
            plan: {
                id: BigInt(1),
                enabledAt: new Date(Date.now() - 86400000),
                expiresAt: null,
            },
            creditAccount: {
                id: BigInt(456),
                currentBalance: BigInt(5000000),
            },
        };
        
        // Mock no free credits available
        (calculateFreeCreditsBalance as MockedFunction<typeof calculateFreeCreditsBalance>)
            .mockResolvedValue(BigInt(0));
        
        (batch as MockedFunction<typeof batch>).mockImplementation(async ({ processBatch }) => {
            await processBatch([mockUser]);
        });

        await creditRollover();

        // Should not process donation
        expect(mockPrisma.credit_ledger_entry.createMany).not.toHaveBeenCalled();
        expect(logger.warn).toHaveBeenCalledWith(
            expect.stringContaining("Insufficient free credits"),
            expect.any(Object)
        );
    });

    it("should handle transaction errors gracefully", async () => {
        mockPrisma.billing_event.findFirst.mockResolvedValue(null);
        
        const mockUser = {
            id: BigInt(123),
            creditSettings: {
                __version: "1.0.0",
                rollover: { enabled: true },
                donation: { enabled: true, percentage: 10 },
            },
            languages: ["en"],
            plan: {
                id: BigInt(1),
                enabledAt: new Date(Date.now() - 86400000),
                expiresAt: null,
            },
            creditAccount: {
                id: BigInt(456),
                currentBalance: BigInt(5000000),
            },
        };
        
        (calculateFreeCreditsBalance as MockedFunction<typeof calculateFreeCreditsBalance>)
            .mockResolvedValue(BigInt(2000000));
        
        // Mock transaction failure
        mockPrisma.$transaction.mockRejectedValue(new Error("Database error"));
        
        (batch as MockedFunction<typeof batch>).mockImplementation(async ({ processBatch }) => {
            await processBatch([mockUser]);
        });

        await creditRollover();

        // Should log error but continue processing
        expect(logger.error).toHaveBeenCalledWith(
            expect.stringContaining("Failed to process donation"),
            expect.any(Object)
        );
        
        // Should still mark month as processed
        expect(mockBus.pub).toHaveBeenCalledWith("billing", expect.objectContaining({
            type: "CREDIT_ROLLOVER_MONTHLY",
        }));
    });
});