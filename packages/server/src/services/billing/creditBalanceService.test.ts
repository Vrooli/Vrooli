import { CreditEntryType, CreditSourceSystem } from "@prisma/client";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { calculateFreeCreditsBalance, getCreditBalancesBySource } from "./creditBalanceService.js";
import { DbProvider } from "../../db/provider.js";

// Create mock functions that can be properly configured in tests
const mockFindMany = vi.fn();
const mockFindUnique = vi.fn();

// Mock the database provider
vi.mock("../../db/provider.js", () => ({
    DbProvider: {
        get: vi.fn(() => ({
            credit_ledger_entry: {
                findMany: mockFindMany,
            },
            credit_account: {
                findUnique: mockFindUnique,
            },
        })),
    },
}));

// Mock the logger
vi.mock("../../events/logger.js", () => ({
    logger: {
        info: vi.fn(),
        error: vi.fn(),
        warn: vi.fn(),
    },
}));

describe("calculateFreeCreditsBalance", () => {
    const mockCreditAccountId = BigInt(123);

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("should calculate correct balance with only free credits", async () => {
        const mockLedgerEntries = [
            {
                id: BigInt(1),
                amount: "1000000000", // 1000 credits
                type: CreditEntryType.Bonus,
                source: CreditSourceSystem.Scheduler,
                createdAt: new Date("2024-01-01"),
                meta: {},
            },
        ];

        mockFindMany.mockResolvedValue(mockLedgerEntries);

        const balance = await calculateFreeCreditsBalance(mockCreditAccountId);
        expect(balance).toBe(BigInt(1000000000));
    });

    it("should handle FIFO when spending credits", async () => {
        const mockLedgerEntries = [
            // Add 1000 free credits
            {
                id: BigInt(1),
                amount: "1000000000",
                type: CreditEntryType.Bonus,
                source: CreditSourceSystem.Scheduler,
                createdAt: new Date("2024-01-01"),
                meta: {},
            },
            // Spend 600 credits (should come from free)
            {
                id: BigInt(2),
                amount: "-600000000",
                type: CreditEntryType.Spend,
                source: CreditSourceSystem.InternalCall,
                createdAt: new Date("2024-01-02"),
                meta: {},
            },
            // Add 500 purchased credits
            {
                id: BigInt(3),
                amount: "500000000",
                type: CreditEntryType.Purchase,
                source: CreditSourceSystem.Stripe,
                createdAt: new Date("2024-01-03"),
                meta: {},
            },
        ];

        (DbProvider.get().credit_ledger_entry.findMany as any).mockResolvedValue(mockLedgerEntries);

        const balance = await calculateFreeCreditsBalance(mockCreditAccountId);
        // Should have 400 free credits remaining (1000 - 600)
        expect(balance).toBe(BigInt(400000000));
    });

    it("should handle spending more than free credits", async () => {
        const mockLedgerEntries = [
            // Add 1000 free credits
            {
                id: BigInt(1),
                amount: "1000000000",
                type: CreditEntryType.Bonus,
                source: CreditSourceSystem.Scheduler,
                createdAt: new Date("2024-01-01"),
                meta: {},
            },
            // Add 500 purchased credits
            {
                id: BigInt(2),
                amount: "500000000",
                type: CreditEntryType.Purchase,
                source: CreditSourceSystem.Stripe,
                createdAt: new Date("2024-01-02"),
                meta: {},
            },
            // Spend 1200 credits (1000 from free, 200 from purchased)
            {
                id: BigInt(3),
                amount: "-1200000000",
                type: CreditEntryType.Spend,
                source: CreditSourceSystem.InternalCall,
                createdAt: new Date("2024-01-03"),
                meta: {},
            },
        ];

        (DbProvider.get().credit_ledger_entry.findMany as any).mockResolvedValue(mockLedgerEntries);

        const balance = await calculateFreeCreditsBalance(mockCreditAccountId);
        // Should have 0 free credits remaining (all spent)
        expect(balance).toBe(BigInt(0));
    });

    it("should return 0 on error", async () => {
        mockFindMany.mockRejectedValue(new Error("Database error"));

        const balance = await calculateFreeCreditsBalance(mockCreditAccountId);
        expect(balance).toBe(BigInt(0));
    });
});

describe("getCreditBalancesBySource", () => {
    const mockCreditAccountId = BigInt(123);

    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("should return correct balance breakdown", async () => {
        // Mock ledger entries showing 400 free credits remaining
        const mockLedgerEntries = [
            {
                id: BigInt(1),
                amount: "1000000000",
                type: CreditEntryType.Bonus,
                source: CreditSourceSystem.Scheduler,
                createdAt: new Date("2024-01-01"),
                meta: {},
            },
            {
                id: BigInt(2),
                amount: "-600000000",
                type: CreditEntryType.Spend,
                source: CreditSourceSystem.InternalCall,
                createdAt: new Date("2024-01-02"),
                meta: {},
            },
            {
                id: BigInt(3),
                amount: "500000000",
                type: CreditEntryType.Purchase,
                source: CreditSourceSystem.Stripe,
                createdAt: new Date("2024-01-03"),
                meta: {},
            },
        ];

        mockFindMany.mockResolvedValue(mockLedgerEntries);
        mockFindUnique.mockResolvedValue({
            currentBalance: BigInt(900000000), // 400 free + 500 purchased
        });

        const balances = await getCreditBalancesBySource(mockCreditAccountId);
        
        expect(balances.free).toBe(BigInt(400000000));
        expect(balances.purchased).toBe(BigInt(500000000));
        expect(balances.total).toBe(BigInt(900000000));
    });
});