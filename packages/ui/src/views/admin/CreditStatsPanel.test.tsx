import { type AdminSiteStatsOutput } from "@vrooli/shared";
import { afterAll, beforeEach, describe, expect, it, vi, type Mock } from "vitest";
import { render, screen } from "../../__test/testUtils.js";
import { useLazyFetch } from "../../hooks/useFetch.js";
import { CreditStatsPanel } from "./CreditStatsPanel.js";

// Custom matchers removed - using Vitest built-in matchers

// Mock the useLazyFetch hook
vi.mock("../../hooks/useFetch.js", () => ({
    useLazyFetch: vi.fn(),
}));

// Mock console methods to suppress error logs in tests
const originalError = console.error;
const originalWarn = console.warn;

beforeEach(() => {
    console.error = vi.fn();
    console.warn = vi.fn();
});

afterAll(() => {
    console.error = originalError;
    console.warn = originalWarn;
});

describe("CreditStatsPanel", () => {
    const mockFetchSiteStats = vi.fn();

    const mockSiteStatsData: AdminSiteStatsOutput = {
        totalUsers: 1000,
        activeUsers: 750,
        newUsersToday: 25,
        totalRoutines: 500,
        activeRoutines: 400,
        totalApiKeys: 200,
        totalApiCalls: 0,
        apiCallsToday: 0,
        totalStorage: 0,
        usedStorage: 0,
        creditStats: {
            totalCreditsInCirculation: "50000000000", // 50,000 credits
            totalCreditsDonatedThisMonth: "5000000000", // 5,000 credits
            totalCreditsDonatedAllTime: "100000000000", // 100,000 credits
            activeDonorsThisMonth: 150,
            averageDonationPercentage: 15.5,
            lastRolloverJobStatus: "success",
            lastRolloverJobTime: "2024-01-02T02:00:00.000Z",
            nextScheduledRollover: "2024-02-02T02:00:00.000Z",
            donationsByMonth: [
                { month: "2024-01", amount: "5000000000", donors: 150 },
                { month: "2023-12", amount: "4500000000", donors: 140 },
                { month: "2023-11", amount: "4000000000", donors: 130 },
                { month: "2023-10", amount: "3500000000", donors: 120 },
                { month: "2023-09", amount: "3000000000", donors: 110 },
                { month: "2023-08", amount: "2500000000", donors: 100 },
            ],
        },
    };

    beforeEach(() => {
        vi.clearAllMocks();
        mockFetchSiteStats.mockClear();
    });

    it("should display loading state initially", () => {
        (useLazyFetch as Mock).mockReturnValue([
            mockFetchSiteStats,
            { data: null, loading: true, errors: null },
        ]);

        render(<CreditStatsPanel />);

        expect(screen.getByRole("progressbar")).toBeTruthy();
    });

    it("should display error message when fetch fails", () => {
        (useLazyFetch as Mock).mockReturnValue([
            mockFetchSiteStats,
            {
                data: null,
                loading: false,
                errors: [{ message: "Failed to fetch data" }],
            },
        ]);

        render(
            <CreditStatsPanel />,
        );

        expect(screen.getByRole("alert").textContent).toContain("ErrorLoadingStats");
        expect(screen.getByRole("alert").textContent).toContain("Failed to fetch data");
    });

    it("should display no data message when data is unavailable", () => {
        (useLazyFetch as Mock).mockReturnValue([
            mockFetchSiteStats,
            { data: {}, loading: false, errors: null },
        ]);

        render(<CreditStatsPanel />);

        expect(screen.getByRole("alert").textContent).toContain("NoDataAvailable");
    });

    it("should display credit statistics when data is loaded", async () => {
        (useLazyFetch as Mock).mockReturnValue([
            mockFetchSiteStats,
            { data: mockSiteStatsData, loading: false, errors: null },
        ]);

        render(<CreditStatsPanel />);

        // Check job status banner
        expect(screen.getByText("Job Status:")).toBeTruthy();
        expect(screen.getByText("Success")).toBeTruthy();
        expect(screen.getByText(/Last run:/)).toBeTruthy();
        expect(screen.getByText(/Next run in -?\d+ days/)).toBeTruthy();

        // Check overview cards
        expect(screen.getByText("TotalInCirculation")).toBeTruthy();
        expect(screen.getByText("50,000")).toBeTruthy();

        expect(screen.getByText("DonatedThisMonth")).toBeTruthy();
        // Use getAllByText since "5,000" appears in both the card and table
        const fiveThousandElements = screen.getAllByText("5,000");
        expect(fiveThousandElements.length).toBeGreaterThan(0);

        expect(screen.getByText("ActiveDonors")).toBeTruthy();
        // Use getAllByText since "150" appears in both the card and table
        const oneFiftyElements = screen.getAllByText("150");
        expect(oneFiftyElements.length).toBeGreaterThan(0);

        expect(screen.getByText("AvgDonationPercent")).toBeTruthy();
        expect(screen.getByText("16%")).toBeTruthy(); // Rounded from 15.5

        // Check donation history table
        expect(screen.getByText("DonationHistory")).toBeTruthy();
        expect(screen.getByText("Month")).toBeTruthy();
        expect(screen.getByText("Credits")).toBeTruthy();
        expect(screen.getByText("Donors")).toBeTruthy();
        expect(screen.getByText("AvgPerDonor")).toBeTruthy();

        // Check all-time stats
        expect(screen.getByText("AllTimeStats")).toBeTruthy();
        expect(screen.getByText("TotalDonated")).toBeTruthy();
        expect(screen.getByText("100,000 credits")).toBeTruthy();
        expect(screen.getByText("EstimatedValue")).toBeTruthy();
        // The calculation: 100,000,000,000 (totalCreditsDonatedAllTime) / 1e8 / 100 = 10.00
        expect(screen.getByText("$10.00")).toBeTruthy();
    });

    it("should calculate and display month-over-month trend", () => {
        (useLazyFetch as Mock).mockReturnValue([
            mockFetchSiteStats,
            { data: mockSiteStatsData, loading: false, errors: null },
        ]);

        render(<CreditStatsPanel />);

        // Should show +11% trend (5000 - 4500) / 4500 * 100
        expect(screen.getByText("+11% from last month")).toBeTruthy();
    });

    it("should handle failed job status correctly", () => {
        const failedData = {
            ...mockSiteStatsData,
            creditStats: {
                ...mockSiteStatsData.creditStats,
                lastRolloverJobStatus: "failed" as const,
            },
        };

        (useLazyFetch as Mock).mockReturnValue([
            mockFetchSiteStats,
            { data: failedData, loading: false, errors: null },
        ]);

        render(<CreditStatsPanel />);

        expect(screen.getByText("Job Status:")).toBeTruthy();
        expect(screen.getByText("Failed")).toBeTruthy();
    });

    it("should handle never run job status correctly", () => {
        const neverRunData = {
            ...mockSiteStatsData,
            creditStats: {
                ...mockSiteStatsData.creditStats,
                lastRolloverJobStatus: "never_run" as const,
                lastRolloverJobTime: null,
            },
        };

        (useLazyFetch as Mock).mockReturnValue([
            mockFetchSiteStats,
            { data: neverRunData, loading: false, errors: null },
        ]);

        render(<CreditStatsPanel />);

        expect(screen.getByText("Job Status:")).toBeTruthy();
        expect(screen.getByText("Never Run")).toBeTruthy();
        expect(screen.queryByText(/Last run:/)).not.toBeTruthy();
    });

    it("should fetch data on mount", () => {
        (useLazyFetch as Mock).mockReturnValue([
            mockFetchSiteStats,
            { data: null, loading: true, errors: null },
        ]);

        render(<CreditStatsPanel />);

        expect(mockFetchSiteStats).toHaveBeenCalledTimes(1);
    });

    it("should set up auto-refresh interval", () => {
        vi.useFakeTimers();
        const setIntervalSpy = vi.spyOn(global, "setInterval");

        (useLazyFetch as Mock).mockReturnValue([
            mockFetchSiteStats,
            { data: mockSiteStatsData, loading: false, errors: null },
        ]);

        const { unmount } = render(<CreditStatsPanel />);

        // Check that setInterval was called with 2 minutes (120000ms)
        expect(setIntervalSpy).toHaveBeenCalledWith(
            expect.any(Function),
            120000,
        );

        // Clean up
        unmount();
        setIntervalSpy.mockRestore();
        vi.clearAllTimers();
        vi.useRealTimers();
    });

    it("should handle formatting errors gracefully", () => {
        const invalidData = {
            ...mockSiteStatsData,
            creditStats: {
                ...mockSiteStatsData.creditStats,
                totalCreditsInCirculation: "invalid-bigint",
                donationsByMonth: [
                    { month: "2024-01", amount: "invalid-amount", donors: 150 },
                ],
            },
        };

        (useLazyFetch as Mock).mockReturnValue([
            mockFetchSiteStats,
            { data: invalidData, loading: false, errors: null },
        ]);

        render(<CreditStatsPanel />);

        // Should display "0" for invalid amounts
        expect(screen.getByText("0")).toBeTruthy();

        // Should log errors to console
        expect(console.error).toHaveBeenCalledWith(
            expect.stringContaining("Error formatting credits:"),
            expect.any(Error),
        );
    });

    it("should calculate average donation per donor correctly", () => {
        (useLazyFetch as Mock).mockReturnValue([
            mockFetchSiteStats,
            { data: mockSiteStatsData, loading: false, errors: null },
        ]);

        render(<CreditStatsPanel />);

        // For January: 5,000 credits / 150 donors = 33.33, rounded to 33
        const avgCells = screen.getAllByText(/33/);
        expect(avgCells.length).toBeGreaterThan(0);
    });

    it("should display zero for months with no donors", () => {
        const dataWithNoDonors = {
            ...mockSiteStatsData,
            creditStats: {
                ...mockSiteStatsData.creditStats,
                donationsByMonth: [
                    { month: "2024-01", amount: "0", donors: 0 },
                ],
            },
        };

        (useLazyFetch as Mock).mockReturnValue([
            mockFetchSiteStats,
            { data: dataWithNoDonors, loading: false, errors: null },
        ]);

        render(<CreditStatsPanel />);

        // Should show "-" for average when no donors
        expect(screen.getByText("-")).toBeTruthy();
    });
});
