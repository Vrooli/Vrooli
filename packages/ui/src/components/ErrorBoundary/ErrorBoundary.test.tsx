import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ErrorBoundary } from "./ErrorBoundary.js";
import { server } from "../../__test/mocks/server.js";
import { http, HttpResponse } from "msw";

// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-06-18

// Mock components that throw errors for testing
const ThrowError = ({ shouldThrow }: { shouldThrow: boolean }) => {
    if (shouldThrow) {
        throw new Error("Test error message");
    }
    return <div>No error</div>;
};

const ThrowErrorWithStack = () => {
    const error = new Error("Test error with stack");
    error.stack = "Error: Test error with stack\n    at ThrowErrorWithStack\n    at Component";
    throw error;
};

const ThrowMalformedError = () => {
    // Simulate an error without proper properties
    const error: any = {};
    error.toString = () => "Malformed error";
    throw error;
};

// Lightweight mocking for performance
const mockLocalStorage = { getItem: vi.fn(), setItem: vi.fn() };
const mockLocation = { href: "http://test.example.com/test-page", assign: vi.fn(), reload: vi.fn() };

vi.stubGlobal("localStorage", mockLocalStorage); 
vi.stubGlobal("location", mockLocation);
vi.stubGlobal("navigator", { userAgent: "Test User Agent" });

describe("ErrorBoundary", () => {
    beforeEach(() => {
        mockLocalStorage.getItem.mockClear().mockReturnValue(null);
        mockLocalStorage.setItem.mockClear();
        mockLocation.assign.mockClear();
        mockLocation.reload.mockClear();
    });

    it("displays child components normally when no errors occur", () => {
        // GIVEN: A component that doesn't throw an error
        // WHEN: The component is rendered within an ErrorBoundary
        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={false} />
            </ErrorBoundary>,
        );

        // THEN: The child component should be displayed normally
        expect(screen.getByText("No error")).toBeDefined();
    });

    it("displays error UI with recovery options when a child component crashes", () => {
        // GIVEN: A component that will throw an error
        // WHEN: The component crashes while rendering
        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>,
        );

        // THEN: The user should see an error message and recovery options
        expect(screen.getByText(/Something went wrong/)).toBeDefined();
        expect(screen.getByRole("button", { name: "Refresh" })).toBeDefined();
        expect(screen.getByRole("button", { name: "Go to Home" })).toBeDefined();
    });

    it("remembers user's error reporting preference across sessions", () => {
        // GIVEN: User previously opted out of error reporting
        mockLocalStorage.getItem.mockReturnValue("false");

        // WHEN: A new error occurs
        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>,
        );

        // THEN: The checkbox should reflect the saved preference
        const checkbox = screen.getByRole("checkbox");
        expect(checkbox.getAttribute("checked")).toBeNull();
    });

    it("allows user to opt out of automatic error reporting", () => {
        // GIVEN: An error has occurred with default error reporting enabled
        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>,
        );

        // WHEN: User unchecks the error reporting checkbox
        const checkbox = screen.getByRole("checkbox");
        fireEvent.click(checkbox);

        // THEN: The preference should be saved for future errors
        expect(mockLocalStorage.setItem).toHaveBeenCalledWith("shouldSendReport", "false");
    });

    it("automatically reports errors and refreshes page when user chooses to refresh", async () => {
        // Track if the error report was sent
        let errorReportSent = false;
        server.use(
            http.post("*/api/error-reports", async ({ request }) => {
                const body = await request.json();
                // Verify the error report contains expected data
                expect(body).toMatchObject({
                    error: expect.stringContaining("Test error message"),
                    userAgent: "Test User Agent",
                    url: "http://test.example.com/test-page",
                    timestamp: expect.any(String),
                });
                errorReportSent = true;
                return HttpResponse.json({ success: true });
            }),
        );

        // GIVEN: An error has occurred with error reporting enabled (default)
        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>,
        );

        // WHEN: User clicks the refresh button to try again
        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        // THEN: The error should be reported and page should refresh
        await waitFor(() => {
            expect(errorReportSent).toBe(true);
        });

        await waitFor(() => {
            expect(mockLocation.reload).toHaveBeenCalled();
        });
    });

    it("navigates user to home page with error reporting when home button is clicked", async () => {
        // Track if the error report was sent
        let errorReportSent = false;
        server.use(
            http.post("*/api/error-reports", async ({ request }) => {
                const body = await request.json();
                expect(body).toMatchObject({
                    error: expect.stringContaining("Test error message"),
                    userAgent: "Test User Agent",
                    url: "http://test.example.com/test-page",
                    timestamp: expect.any(String),
                });
                errorReportSent = true;
                return HttpResponse.json({ success: true });
            }),
        );

        // GIVEN: An error has occurred with error reporting enabled
        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>,
        );

        // WHEN: User clicks "Go to Home" to escape the error
        const homeButton = screen.getByRole("button", { name: "Go to Home" });
        fireEvent.click(homeButton);

        // THEN: Error should be reported and user navigated home
        await waitFor(() => {
            expect(errorReportSent).toBe(true);
        });

        await waitFor(() => {
            expect(mockLocation.assign).toHaveBeenCalledWith("/");
        });
    });

    it("respects user privacy by not sending error reports when opted out", async () => {
        // Track if any error report was sent
        let errorReportSent = false;
        server.use(
            http.post("*/api/error-reports", () => {
                errorReportSent = true;
                return HttpResponse.json({ success: true });
            }),
        );

        // GIVEN: User has opted out of error reporting
        mockLocalStorage.getItem.mockReturnValue("false");

        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>,
        );

        // WHEN: User clicks refresh to recover from error
        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        // THEN: Page should refresh without sending error report
        await waitFor(() => {
            expect(mockLocation.reload).toHaveBeenCalled();
        });

        expect(errorReportSent).toBe(false);
    });

    it("gracefully handles all types of JavaScript errors including those with stack traces", async () => {
        // Track if the error report was sent with stack trace
        let errorReportSent = false;
        server.use(
            http.post("*/api/error-reports", async ({ request }) => {
                const body = await request.json();
                expect(body).toMatchObject({
                    error: expect.stringContaining("Test error with stack"),
                    stack: expect.stringContaining("at ThrowErrorWithStack"),
                    userAgent: "Test User Agent",
                    url: "http://test.example.com/test-page",
                    timestamp: expect.any(String),
                });
                errorReportSent = true;
                return HttpResponse.json({ success: true });
            }),
        );

        // GIVEN: An error with full stack trace information occurs
        render(
            <ErrorBoundary>
                <ThrowErrorWithStack />
            </ErrorBoundary>,
        );

        // WHEN: User attempts to recover
        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        // THEN: The error report should include the stack trace for debugging
        await waitFor(() => {
            expect(errorReportSent).toBe(true);
        });
    });

    it("provides fallback error handling for unusual error objects", async () => {
        // Track if the error report was sent
        let errorReportSent = false;
        server.use(
            http.post("*/api/error-reports", async ({ request }) => {
                const body = await request.json();
                expect(body).toMatchObject({
                    error: expect.stringContaining("Malformed error"),
                    userAgent: "Test User Agent",
                    url: "http://test.example.com/test-page",
                    timestamp: expect.any(String),
                });
                errorReportSent = true;
                return HttpResponse.json({ success: true });
            }),
        );

        // GIVEN: A non-standard error object is thrown
        render(
            <ErrorBoundary>
                <ThrowMalformedError />
            </ErrorBoundary>,
        );

        // WHEN: User attempts to recover
        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        // THEN: The error should still be reported with available information
        await waitFor(() => {
            expect(errorReportSent).toBe(true);
        });
    });

    it("continues to function when error reporting service is unavailable", async () => {
        // GIVEN: The error reporting service is down
        const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
        server.use(
            http.post("*/api/error-reports", () => {
                return HttpResponse.error();
            }),
        );

        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>,
        );

        // WHEN: User attempts to refresh after an error
        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        // THEN: The page should still refresh despite reporting failure
        await waitFor(() => {
            expect(mockLocation.reload).toHaveBeenCalled();
        });

        // AND: The failure should be logged for debugging
        await waitFor(() => {
            expect(consoleErrorSpy).toHaveBeenCalledWith("Failed to send error report:", expect.any(Error));
        });
        
        consoleErrorSpy.mockRestore();
    });

});
