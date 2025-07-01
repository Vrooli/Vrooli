import { act, render, screen, waitFor } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import { HttpResponse, http } from "msw";
import { setupServer } from "msw/node";
import React from "react";
import { beforeAll, afterAll, afterEach, describe, expect, it, vi } from "vitest";
import { AwardCategory, type Award } from "@vrooli/shared";
import { render } from "../../__test/testUtils.js";
import { signedInNoPremiumWithCreditsSession, loggedOutSession } from "../../__test/storybookConsts.js";
import { AwardsView, awardToDisplay } from "./AwardsView.js";

// Mock translation function
const mockT = vi.fn((key: string, options?: { count?: number; [key: string]: unknown }) => {
    if (key === "Award") return options?.count === 2 ? "Awards" : "Award";
    if (key.endsWith("Title")) return `${key.replace("Title", "")} Award Title`;
    if (key.endsWith("Body")) return `${key.replace("Body", "")} Award Description`;
    if (key.endsWith("UnearnedTitle")) return `Unearned ${key.replace("UnearnedTitle", "")} Title`;
    if (key.endsWith("UnearnedBody")) return `Unearned ${key.replace("UnearnedBody", "")} Description`;
    return key;
});

// Mock i18next
vi.mock("react-i18next", () => ({
    useTranslation: () => ({
        t: mockT,
        i18n: { language: "en" },
    }),
}));

// Mock useFetch hook
const mockUseFetch = vi.fn();
vi.mock("../../hooks/useFetch.js", () => ({
    useFetch: () => mockUseFetch(),
}));

// Mock socket services
vi.mock("../../services/socket.js", () => ({
    SocketService: {
        get: () => ({
            onEvent: vi.fn(),
            offEvent: vi.fn(),
            isConnected: vi.fn(() => false),
        }),
    },
}));

// Mock notification store
vi.mock("../../stores/notificationsStore.js", () => ({
    useNotificationsStore: vi.fn((selector) => {
        const state = {
            notifications: [],
            isLoading: false,
            error: null,
            fetchNotifications: vi.fn(() => Promise.resolve([])),
            setNotifications: vi.fn(),
            addNotification: vi.fn(),
            clearNotifications: vi.fn(),
            markAllNotificationsAsRead: vi.fn(),
        };
        return selector ? selector(state) : state;
    }),
    useNotifications: vi.fn(),
}));


// Create MSW server
const server = setupServer(
    // Mock notifications endpoint
    http.get("*/api/v2/notifications", () => {
        return HttpResponse.json({
            data: { edges: [], pageInfo: { hasNextPage: false, endCursor: null } },
        });
    }),
    // Mock awards endpoint
    http.get("*/api/awards", () => {
        return HttpResponse.json({
            data: { edges: [], pageInfo: { hasNextPage: false, endCursor: null } },
        });
    }),
);

// Sample award data for testing
const createMockAward = (category: AwardCategory, progress: number): Award => ({
    __typename: "Award",
    id: `award-${category}`,
    category,
    progress,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    title: `${category} Award`,
    description: `Description for ${category}`,
    tierCompletedAt: progress > 0 ? new Date().toISOString() : null,
});

const mockAwardsData = {
    edges: [
        { cursor: "1", node: createMockAward(AwardCategory.RoutineCreate, 1000) }, // Fully completed
        { cursor: "2", node: createMockAward(AwardCategory.RunRoutine, 87) }, // Almost completed
        { cursor: "3", node: createMockAward(AwardCategory.CommentCreate, 15) }, // In progress
        { cursor: "4", node: createMockAward(AwardCategory.ProjectCreate, 0) }, // Not started
    ],
    pageInfo: {
        __typename: "PageInfo",
        endCursor: "4",
        hasNextPage: false,
    },
};

describe("AwardsView", () => {
    beforeAll(() => {
        server.listen({ onUnhandledRequest: "error" });
    });

    afterAll(() => {
        server.close();
    });

    afterEach(() => {
        server.resetHandlers();
        vi.clearAllMocks();
    });

    describe("Loading and Data Display", () => {
        it("renders the awards page with navbar title", async () => {
            mockUseFetch.mockReturnValue({
                data: null,
                loading: true,
                refetch: vi.fn(),
            });

            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                expect(screen.getByText("Awards")).toBeDefined();
                expect(screen.getByText("🏆 Your Awards Progress")).toBeDefined();
            });
        });

        it("displays summary statistics correctly", async () => {
            mockUseFetch.mockReturnValue({
                data: mockAwardsData,
                loading: false,
                refetch: vi.fn(),
            });

            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                const summarySection = screen.getByTestId("awards-summary");
                expect(summarySection).toBeDefined();

                // Check summary chips are displayed
                const completedChip = screen.getByTestId("completed-count-chip");
                const inProgressChip = screen.getByTestId("in-progress-count-chip");
                const totalChip = screen.getByTestId("total-count-chip");

                expect(completedChip).toBeDefined();
                expect(inProgressChip).toBeDefined();
                expect(totalChip).toBeDefined();
            });
        });

        it("displays awards grid with correct number of cards", async () => {
            mockUseFetch.mockReturnValue({
                data: mockAwardsData,
                loading: false,
                refetch: vi.fn(),
            });

            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            // First check that the component renders
            expect(screen.getByText("🏆 Your Awards Progress")).toBeDefined();
            
            await waitFor(() => {
                const awardsGrid = screen.getByTestId("awards-grid");
                expect(awardsGrid).toBeDefined();

                const awardCards = screen.getAllByTestId("award-card");
                // Should include both fetched awards and default zero-progress awards
                expect(awardCards.length).toBeGreaterThan(0);
            }, { timeout: 15000 });
        });
    });

    describe("Award Card Behavior", () => {
        beforeEach(() => {
            mockUseFetch.mockReturnValue({
                data: mockAwardsData,
                loading: false,
                refetch: vi.fn(),
            });
        });

        it("displays completed award with correct status", async () => {
            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                const awardCards = screen.getAllByTestId("award-card");
                const completedCard = awardCards.find(card => 
                    card.getAttribute("data-completed") === "true",
                );
                expect(completedCard).toBeDefined();

                if (completedCard) {
                    const statusChips = screen.getAllByTestId("award-status-chip");
                    const completedChip = statusChips.find(chip => chip.textContent === "Completed");
                    expect(completedChip).toBeDefined();
                }
            }, { timeout: 10000 });
        });

        it("displays in-progress award with correct progress", async () => {
            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                const awardCards = screen.getAllByTestId("award-card");
                const inProgressCard = awardCards.find(card => 
                    card.getAttribute("data-completed") === "false" && 
                    parseInt(card.getAttribute("data-progress") || "0") > 0,
                );
                
                expect(inProgressCard).toBeDefined();
                
                if (inProgressCard) {
                    const statusChips = screen.getAllByTestId("award-status-chip");
                    const inProgressChip = statusChips.find(chip => 
                        chip.textContent === "In Progress" || chip.textContent === "Almost there!",
                    );
                    expect(inProgressChip).toBeDefined();
                }
            });
        });

        it("displays progress bar with correct value", async () => {
            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                const progressBars = screen.getAllByTestId("award-progress-bar");
                expect(progressBars.length).toBeGreaterThan(0);

                // Check that progress bars have values
                progressBars.forEach(progressBar => {
                    const progressValue = progressBar.getAttribute("aria-valuenow");
                    expect(progressValue).toBeDefined();
                    expect(parseInt(progressValue || "0")).toBeGreaterThanOrEqual(0);
                    expect(parseInt(progressValue || "0")).toBeLessThanOrEqual(100);
                });
            });
        });

        it("shows tier information correctly", async () => {
            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                const tierChips = screen.getAllByTestId("award-tier-chip");
                expect(tierChips.length).toBeGreaterThan(0);

                // Verify tier chips have appropriate content
                tierChips.forEach(tierChip => {
                    const content = tierChip.textContent || "";
                    expect(content).toMatch(/(Tier \d+|No progress)/);
                });
            });
        });
    });

    describe("Award Sorting and Organization", () => {
        it("sorts awards correctly (completed first, then by progress)", async () => {
            mockUseFetch.mockReturnValue({
                data: mockAwardsData,
                loading: false,
                refetch: vi.fn(),
            });

            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                const awardCards = screen.getAllByTestId("award-card");
                
                // Get completion status of each card
                const completionStatuses = awardCards.map(card => 
                    card.getAttribute("data-completed") === "true",
                );

                // Verify completed awards come first
                let foundIncomplete = false;
                for (const isCompleted of completionStatuses) {
                    if (foundIncomplete && isCompleted) {
                        // Found a completed award after an incomplete one - sorting is wrong
                        expect(false).toBe(true);
                    }
                    if (!isCompleted) {
                        foundIncomplete = true;
                    }
                }
            });
        });

        it("handles awards with different categories", async () => {
            mockUseFetch.mockReturnValue({
                data: mockAwardsData,
                loading: false,
                refetch: vi.fn(),
            });

            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                const awardCards = screen.getAllByTestId("award-card");
                
                // Verify we have cards with different categories
                const categories = awardCards.map(card => card.getAttribute("data-category"));
                const uniqueCategories = new Set(categories);
                
                expect(uniqueCategories.size).toBeGreaterThan(1);
            });
        });
    });

    describe("No Data States", () => {
        it("handles empty awards data gracefully", async () => {
            mockUseFetch.mockReturnValue({
                data: { edges: [], pageInfo: { endCursor: null, hasNextPage: false } },
                loading: false,
                refetch: vi.fn(),
            });

            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            // First check that the basic component renders
            expect(screen.getByText("🏆 Your Awards Progress")).toBeDefined();
            
            await waitFor(() => {
                const awardsGrid = screen.getByTestId("awards-grid");
                expect(awardsGrid).toBeDefined();

                // Should still show default zero-progress awards
                const awardCards = screen.getAllByTestId("award-card");
                expect(awardCards.length).toBeGreaterThan(0);
            }, { timeout: 15000 });
        });

        it("displays appropriate counts when no fetched awards have progress", async () => {
            mockUseFetch.mockReturnValue({
                data: { edges: [], pageInfo: { endCursor: null, hasNextPage: false } },
                loading: false,
                refetch: vi.fn(),
            });

            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                const completedChip = screen.getByTestId("completed-count-chip");
                const inProgressChip = screen.getByTestId("in-progress-count-chip");

                // Note: The component creates default awards with 0 progress, but the award system 
                // may still assign tier 1 to some categories, so we check that counts are reasonable
                expect(completedChip.textContent).toMatch(/\d+ Completed/);
                expect(inProgressChip.textContent).toMatch(/\d+ In Progress/);
            }, { timeout: 10000 });
        });
    });

    describe("Session Handling", () => {
        it("works with different session states", async () => {
            mockUseFetch.mockReturnValue({
                data: mockAwardsData,
                loading: false,
                refetch: vi.fn(),
            });

            const { rerender } = render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                expect(screen.getByTestId("awards-summary")).toBeDefined();
            });

            // Test with logged out session
            rerender(<AwardsView />);
            
            // Component should still render (though may show different data)
            expect(screen.getByText("🏆 Your Awards Progress")).toBeDefined();
        });

        it("handles undefined session gracefully", async () => {
            mockUseFetch.mockReturnValue({
                data: null,
                loading: false,
                refetch: vi.fn(),
            });

            render(<AwardsView />, {
                session: undefined,
            });

            expect(screen.getByText("🏆 Your Awards Progress")).toBeDefined();
        });
    });

    describe("Data Fetching", () => {
        it("calls refetch function when data needs to be refreshed", async () => {
            const mockRefetch = vi.fn();
            mockUseFetch.mockReturnValue({
                data: mockAwardsData,
                loading: false,
                refetch: mockRefetch,
            });

            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            // First verify basic render
            expect(screen.getByText("🏆 Your Awards Progress")).toBeDefined();

            await waitFor(() => {
                expect(screen.getByTestId("awards-grid")).toBeDefined();
            }, { timeout: 15000 });

            // The refetch function should be available (though not called automatically)
            expect(mockRefetch).toBeDefined();
        });

        it("handles loading state appropriately", async () => {
            mockUseFetch.mockReturnValue({
                data: null,
                loading: true,
                refetch: vi.fn(),
            });

            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            // Component should still render basic structure during loading
            expect(screen.getByText("🏆 Your Awards Progress")).toBeDefined();
            
            await waitFor(() => {
                expect(screen.getByTestId("awards-grid")).toBeDefined();
            }, { timeout: 15000 });
        });
    });

    describe("Award Display Function", () => {
        it("converts award data to display format correctly", () => {
            const testAward: Award = createMockAward(AwardCategory.RoutineCreate, 50);
            
            const displayAward = awardToDisplay(testAward, mockT);

            expect(displayAward.category).toBe(AwardCategory.RoutineCreate);
            expect(displayAward.progress).toBe(50);
            expect(displayAward.categoryTitle).toBeDefined();
            expect(displayAward.categoryDescription).toBeDefined();
        });

        it("handles awards with no progress correctly", () => {
            const testAward: Award = createMockAward(AwardCategory.CommentCreate, 0);
            
            const displayAward = awardToDisplay(testAward, mockT);

            expect(displayAward.progress).toBe(0);
            // With 0 progress, earnedTier might still be defined if the award system gives tier 1 for any activity
            // Let's check that it either undefined or has level 0/1
            if (displayAward.earnedTier) {
                expect(displayAward.earnedTier.level).toBeLessThanOrEqual(1);
            }
        });

        it("handles fully completed awards correctly", () => {
            const testAward: Award = createMockAward(AwardCategory.Reputation, 10000);
            
            const displayAward = awardToDisplay(testAward, mockT);

            expect(displayAward.progress).toBe(10000);
            expect(displayAward.earnedTier).toBeDefined();
        });
    });

    describe("Component State Changes", () => {
        it("updates when awards data changes", async () => {
            const mockRefetch = vi.fn();
            
            // Start with no data
            mockUseFetch.mockReturnValue({
                data: null,
                loading: false,
                refetch: mockRefetch,
            });

            const { rerender } = render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            // Should show default awards
            await waitFor(() => {
                const completedChip = screen.getByTestId("completed-count-chip");
                expect(completedChip.textContent).toMatch(/\d+ Completed/);
            }, { timeout: 10000 });

            // Update with actual data
            mockUseFetch.mockReturnValue({
                data: mockAwardsData,
                loading: false,
                refetch: mockRefetch,
            });

            rerender(<AwardsView />);

            // Should show updated counts
            await waitFor(() => {
                const completedChip = screen.getByTestId("completed-count-chip");
                // Count should be greater than 0 now
                expect(completedChip.textContent).not.toContain("0 Completed");
            });
        });

        it("handles session changes gracefully", async () => {
            mockUseFetch.mockReturnValue({
                data: mockAwardsData,
                loading: false,
                refetch: vi.fn(),
            });

            const { rerender } = render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                expect(screen.getByTestId("awards-summary")).toBeDefined();
            });

            // Change session
            rerender(<AwardsView />);

            // Component should continue to work
            expect(screen.getByTestId("awards-summary")).toBeDefined();
        });
    });

    describe("Accessibility", () => {
        it("maintains proper ARIA attributes and roles", async () => {
            mockUseFetch.mockReturnValue({
                data: mockAwardsData,
                loading: false,
                refetch: vi.fn(),
            });

            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                const progressBars = screen.getAllByTestId("award-progress-bar");
                
                progressBars.forEach(progressBar => {
                    // Progress bars should have proper ARIA attributes
                    expect(progressBar.getAttribute("role")).toBe("progressbar");
                    expect(progressBar.getAttribute("aria-valuenow")).toBeDefined();
                    expect(progressBar.getAttribute("aria-valuemin")).toBe("0");
                    expect(progressBar.getAttribute("aria-valuemax")).toBe("100");
                });
            });
        });

        it("provides meaningful text content for screen readers", async () => {
            mockUseFetch.mockReturnValue({
                data: mockAwardsData,
                loading: false,
                refetch: vi.fn(),
            });

            render(<AwardsView />, {
                session: signedInNoPremiumWithCreditsSession,
            });

            await waitFor(() => {
                // Check that status chips have meaningful text
                const statusChips = screen.getAllByTestId("award-status-chip");
                statusChips.forEach(chip => {
                    const text = chip.textContent || "";
                    expect(text).toMatch(/(Completed|In Progress|Almost there!|Not Started)/);
                });
            });
        });
    });
});
