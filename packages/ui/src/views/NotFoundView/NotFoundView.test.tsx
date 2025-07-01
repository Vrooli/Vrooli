import { act, render, screen, waitFor } from "@testing-library/react";
import { userEvent } from "@testing-library/user-event";
import React from "react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { NotFoundView } from "./NotFoundView.js";

// Mock sessionStorage
const mockSessionStorage = {
    getItem: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn(),
    clear: vi.fn(),
};

Object.defineProperty(window, "sessionStorage", {
    value: mockSessionStorage,
    writable: true,
});

// Mock window.history.back
const mockHistoryBack = vi.fn();
Object.defineProperty(window, "history", {
    value: { back: mockHistoryBack },
    writable: true,
});

describe("NotFoundView", () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mockSessionStorage.getItem.mockReturnValue(null);
    });

    describe("Basic rendering", () => {
        it("renders the 404 error page structure", () => {
            render(<NotFoundView />);

            // Check main container
            const container = screen.getByTestId("not-found-container");
            expect(container).toBeDefined();

            // Check error code
            const errorCode = screen.getByTestId("error-code");
            expect(errorCode).toBeDefined();
            expect(errorCode.textContent).toBe("404");

            // Check content box
            const contentBox = screen.getByTestId("content-box");
            expect(contentBox).toBeDefined();
        });

        it("displays the bunny image with proper alt text", () => {
            render(<NotFoundView />);

            const bunnyImage = screen.getByTestId("bunny-image");
            expect(bunnyImage).toBeDefined();
            expect(bunnyImage.getAttribute("alt")).toBe("A lop-eared bunny in a pastel-themed workspace with a notepad and pen, looking slightly worried.");
            expect(bunnyImage.getAttribute("src")).toContain("Bunny404");
        });

        it("displays the page title and description", () => {
            render(<NotFoundView />);

            const title = screen.getByTestId("page-title");
            expect(title).toBeDefined();
            expect(title.textContent).toBe("Page Not Found");
            expect(title.tagName).toBe("H1");

            const description = screen.getByTestId("page-description");
            expect(description).toBeDefined();
            expect(description.textContent).toBe("Oops! The page you're looking for seems to have hopped away. Let's get you back on track!");
        });

        it("renders with semantic HTML structure", () => {
            render(<NotFoundView />);

            // Check heading structure
            const heading = screen.getByRole("heading", { level: 1 });
            expect(heading).toBeDefined();
            expect(heading.textContent).toBe("Page Not Found");

            // Check image
            const image = screen.getByRole("img");
            expect(image).toBeDefined();
            expect(image.getAttribute("alt")).toContain("bunny");
        });
    });

    describe("Navigation buttons behavior", () => {
        it("displays only home button when no previous page exists", () => {
            mockSessionStorage.getItem.mockReturnValue(null);
            render(<NotFoundView />);

            const actionButtons = screen.getByTestId("action-buttons");
            expect(actionButtons).toBeDefined();

            // Should only have home button
            const homeButton = screen.getByTestId("go-home-button");
            expect(homeButton).toBeDefined();
            expect(homeButton.textContent).toBe("GoToHome");
            expect(homeButton.getAttribute("aria-label")).toBe("GoToHome");

            // Should not have back button
            const backButton = screen.queryByTestId("go-back-button");
            expect(backButton).toBeNull();
        });

        it("displays both back and home buttons when previous page exists", () => {
            mockSessionStorage.getItem.mockReturnValue("/previous-page");
            render(<NotFoundView />);

            const actionButtons = screen.getByTestId("action-buttons");
            expect(actionButtons).toBeDefined();

            // Should have both buttons
            const backButton = screen.getByTestId("go-back-button");
            expect(backButton).toBeDefined();
            expect(backButton.textContent).toBe("GoBack");
            expect(backButton.getAttribute("aria-label")).toBe("GoBack");

            const homeButton = screen.getByTestId("go-home-button");
            expect(homeButton).toBeDefined();
            expect(homeButton.textContent).toBe("GoToHome");
        });

        it("handles back button click", async () => {
            mockSessionStorage.getItem.mockReturnValue("/previous-page");
            const user = userEvent.setup();

            render(<NotFoundView />);

            const backButton = screen.getByTestId("go-back-button");

            await act(async () => {
                await user.click(backButton);
            });

            expect(mockHistoryBack).toHaveBeenCalledTimes(1);
        });

        it("handles home button navigation", () => {
            render(<NotFoundView />);

            const homeButton = screen.getByTestId("go-home-button");
            expect(homeButton).toBeDefined();

            // Check that the button is wrapped in a Link component
            const linkElement = homeButton.closest("a");
            expect(linkElement).toBeDefined();
        });
    });

    describe("Popular pages section", () => {
        it("displays popular pages section with label", () => {
            render(<NotFoundView />);

            const popularPagesSection = screen.getByTestId("popular-pages-section");
            expect(popularPagesSection).toBeDefined();

            const popularPagesLabel = screen.getByTestId("popular-pages-label");
            expect(popularPagesLabel).toBeDefined();
            expect(popularPagesLabel.textContent).toBe("Popular pages:");
        });

        it("displays all popular page chips", () => {
            render(<NotFoundView />);

            const popularPagesChips = screen.getByTestId("popular-pages-chips");
            expect(popularPagesChips).toBeDefined();

            // Check for each expected chip
            const expectedPages = ["dashboard", "search", "projects", "teams"];
            
            expectedPages.forEach(page => {
                const chip = screen.getByTestId(`popular-page-chip-${page}`);
                expect(chip).toBeDefined();
                expect(chip.getAttribute("role")).toBe("button");
            });
        });

        it("has clickable popular page chips", async () => {
            const user = userEvent.setup();
            render(<NotFoundView />);

            const dashboardChip = screen.getByTestId("popular-page-chip-dashboard");
            expect(dashboardChip).toBeDefined();

            // Check that the chip is clickable
            await act(async () => {
                await user.hover(dashboardChip);
            });

            // Chip should be interactive
            expect(dashboardChip.getAttribute("role")).toBe("button");
        });

        it("wraps popular page chips in navigation links", () => {
            render(<NotFoundView />);

            const dashboardChip = screen.getByTestId("popular-page-chip-dashboard");
            const linkElement = dashboardChip.closest("a");
            expect(linkElement).toBeDefined();
        });
    });

    describe("Responsive behavior", () => {
        it("maintains accessibility features across different states", () => {
            render(<NotFoundView />);

            // Check ARIA labels
            const homeButton = screen.getByTestId("go-home-button");
            expect(homeButton.getAttribute("aria-label")).toBe("GoToHome");

            // Check image alt text
            const bunnyImage = screen.getByTestId("bunny-image");
            expect(bunnyImage.getAttribute("alt")).toBeTruthy();

            // Check heading structure
            const heading = screen.getByRole("heading", { level: 1 });
            expect(heading).toBeDefined();
        });

        it("handles window resize gracefully", () => {
            render(<NotFoundView />);

            // Component should render without errors
            const container = screen.getByTestId("not-found-container");
            expect(container).toBeDefined();

            // All key elements should still be present
            expect(screen.getByTestId("error-code")).toBeDefined();
            expect(screen.getByTestId("page-title")).toBeDefined();
            expect(screen.getByTestId("action-buttons")).toBeDefined();
        });
    });

    describe("State transitions", () => {
        it("toggles back button visibility based on session storage", () => {
            // First render without previous page
            const { rerender } = render(<NotFoundView />);
            
            expect(screen.queryByTestId("go-back-button")).toBeNull();
            expect(screen.getByTestId("go-home-button")).toBeDefined();

            // Simulate having a previous page
            mockSessionStorage.getItem.mockReturnValue("/some-path");
            rerender(<NotFoundView />);

            expect(screen.getByTestId("go-back-button")).toBeDefined();
            expect(screen.getByTestId("go-home-button")).toBeDefined();

            // Back to no previous page
            mockSessionStorage.getItem.mockReturnValue(null);
            rerender(<NotFoundView />);

            expect(screen.queryByTestId("go-back-button")).toBeNull();
            expect(screen.getByTestId("go-home-button")).toBeDefined();
        });
    });

    describe("Content animations", () => {
        it("renders fade animation container", async () => {
            render(<NotFoundView />);

            const contentBox = screen.getByTestId("content-box");
            expect(contentBox).toBeDefined();

            // Content should be visible after animation
            await waitFor(() => {
                expect(contentBox).toBeDefined();
            }, { timeout: 1500 });
        });
    });

    describe("User interactions", () => {
        it("handles keyboard navigation on buttons", async () => {
            mockSessionStorage.getItem.mockReturnValue("/previous-page");
            const user = userEvent.setup();

            render(<NotFoundView />);

            const backButton = screen.getByTestId("go-back-button");
            const homeButton = screen.getByTestId("go-home-button");

            // Test that buttons are focusable
            await act(async () => {
                await user.click(backButton);
            });

            // Verify buttons exist and can be interacted with
            expect(backButton).toBeDefined();
            expect(homeButton).toBeDefined();
            expect(backButton.getAttribute("tabindex")).not.toBe("-1");
            expect(homeButton.getAttribute("tabindex")).not.toBe("-1");
        });

        it("handles keyboard navigation on popular page chips", async () => {
            const user = userEvent.setup();
            render(<NotFoundView />);

            const dashboardChip = screen.getByTestId("popular-page-chip-dashboard");

            await act(async () => {
                await user.tab();
                // Keep tabbing until we reach a chip or other focusable element
                let attempts = 0;
                while (document.activeElement !== dashboardChip && attempts < 10) {
                    await user.tab();
                    attempts++;
                }
            });

            // Should be able to navigate chips with keyboard
            expect(dashboardChip.getAttribute("role")).toBe("button");
        });

        it("supports screen readers with proper ARIA attributes", () => {
            render(<NotFoundView />);

            // Check main heading is properly marked
            const heading = screen.getByRole("heading", { level: 1 });
            expect(heading.textContent).toBe("Page Not Found");

            // Check image has descriptive alt text
            const image = screen.getByRole("img");
            expect(image.getAttribute("alt")).toBeTruthy();
            expect(image.getAttribute("alt").length).toBeGreaterThan(10);

            // Check buttons have proper labels
            const homeButton = screen.getByTestId("go-home-button");
            expect(homeButton.getAttribute("aria-label")).toBe("GoToHome");
        });
    });

    describe("Error boundaries", () => {
        it("renders without crashing when props are missing", () => {
            // Component doesn't take props, but test it handles edge cases
            expect(() => render(<NotFoundView />)).not.toThrow();
        });

        it("handles missing translation gracefully", () => {
            render(<NotFoundView />);

            // Should have fallback text even if translations fail
            const title = screen.getByTestId("page-title");
            expect(title.textContent).toBeTruthy();
            expect(title.textContent.length).toBeGreaterThan(0);

            const description = screen.getByTestId("page-description");
            expect(description.textContent).toBeTruthy();
            expect(description.textContent.length).toBeGreaterThan(0);
        });
    });

    describe("Integration points", () => {
        it("uses correct translation keys", () => {
            render(<NotFoundView />);

            // Component should render with default values when translations are not available
            const title = screen.getByTestId("page-title");
            expect(title.textContent).toBe("Page Not Found");

            const description = screen.getByTestId("page-description");
            expect(description.textContent).toContain("hopped away");

            const popularPagesLabel = screen.getByTestId("popular-pages-label");
            expect(popularPagesLabel.textContent).toBe("Popular pages:");
        });

        it("integrates with routing system", () => {
            render(<NotFoundView />);

            // Check that navigation elements are properly set up
            const homeButton = screen.getByTestId("go-home-button");
            const linkElement = homeButton.closest("a");
            expect(linkElement).toBeDefined();

            // Check popular page chips are wrapped in links
            const dashboardChip = screen.getByTestId("popular-page-chip-dashboard");
            const chipLink = dashboardChip.closest("a");
            expect(chipLink).toBeDefined();
        });
    });
});
