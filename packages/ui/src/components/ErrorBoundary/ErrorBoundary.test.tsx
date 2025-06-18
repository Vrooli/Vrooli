import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ErrorBoundary } from "./ErrorBoundary.js";

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
const mockFetch = vi.fn();
const mockLocalStorage = { getItem: vi.fn(), setItem: vi.fn() };
const mockLocation = { href: 'http://test.example.com/test-page', assign: vi.fn(), reload: vi.fn() };

vi.stubGlobal('fetch', mockFetch);
vi.stubGlobal('localStorage', mockLocalStorage); 
vi.stubGlobal('location', mockLocation);
vi.stubGlobal('navigator', { userAgent: 'Test User Agent' });

describe("ErrorBoundary", () => {
    beforeEach(() => {
        mockFetch.mockClear().mockResolvedValue({ ok: true });
        mockLocalStorage.getItem.mockClear().mockReturnValue(null);
        mockLocalStorage.setItem.mockClear();
        mockLocation.assign.mockClear();
        mockLocation.reload.mockClear();
    });

    it("renders children when there is no error", () => {
        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={false} />
            </ErrorBoundary>
        );

        expect(screen.getByText("No error")).toBeInTheDocument();
    });

    it("renders error UI when child component throws", () => {
        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>
        );

        expect(screen.getByText(/Something went wrong/)).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Refresh" })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Go to Home" })).toBeInTheDocument();
    });

    it("handles localStorage setting for shouldSendReport", () => {
        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>
        );

        const checkbox = screen.getByRole("checkbox");
        fireEvent.click(checkbox);

        expect(mockLocalStorage.setItem).toHaveBeenCalledWith("shouldSendReport", "false");
    });

    it("reads initial shouldSendReport from localStorage", () => {
        mockLocalStorage.getItem.mockReturnValue("false");

        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>
        );

        const checkbox = screen.getByRole("checkbox");
        expect(checkbox).not.toBeChecked();
    });

    it("sends error report when shouldSendReport is true and refresh is clicked", async () => {
        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>
        );

        // shouldSendReport defaults to true, so no need to click checkbox
        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        // Use shorter timeout since this should be fast
        await waitFor(() => {
            expect(mockFetch).toHaveBeenCalledWith("/api/error-reports", expect.objectContaining({
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: expect.stringContaining("Test error message"),
            }));
        });

        await waitFor(() => {
            expect(mockLocation.reload).toHaveBeenCalled();
        });
    });

    it("sends error report when shouldSendReport is true and home is clicked", async () => {
        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>
        );

        // shouldSendReport defaults to true, so no need to click checkbox
        const homeButton = screen.getByRole("button", { name: "Go to Home" });
        fireEvent.click(homeButton);

        await waitFor(() => {
            expect(mockFetch).toHaveBeenCalledWith("/api/error-reports", expect.objectContaining({
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: expect.stringContaining("Test error message"),
            }));
        });

        await waitFor(() => {
            expect(mockLocation.assign).toHaveBeenCalledWith("/");
        });
    });

    it("does not send error report when shouldSendReport is false", async () => {
        mockLocalStorage.getItem.mockReturnValue("false");

        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>
        );

        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        await waitFor(() => {
            expect(mockLocation.reload).toHaveBeenCalled();
        });

        expect(mockFetch).not.toHaveBeenCalled();
    });

    it("handles errors with stack traces properly", () => {
        render(
            <ErrorBoundary>
                <ThrowErrorWithStack />
            </ErrorBoundary>
        );

        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        expect(mockFetch).toHaveBeenCalledWith("/api/error-reports", expect.objectContaining({
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: expect.stringContaining("Test error with stack"),
        }));
    });

    it("handles malformed error objects", () => {
        render(
            <ErrorBoundary>
                <ThrowMalformedError />
            </ErrorBoundary>
        );

        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        expect(mockFetch).toHaveBeenCalledWith("/api/error-reports", expect.objectContaining({
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: expect.stringContaining("Malformed error"),
        }));
    });

    it("handles failed error report gracefully", async () => {
        const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
        mockFetch.mockRejectedValue(new Error("Network error"));

        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>
        );

        // shouldSendReport defaults to true, so no need to click checkbox
        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        await waitFor(() => {
            expect(mockLocation.reload).toHaveBeenCalled();
        });

        // Wait for async error logging to complete
        await waitFor(() => {
            expect(consoleErrorSpy).toHaveBeenCalledWith("Failed to send error report:", expect.any(Error));
        });
        
        consoleErrorSpy.mockRestore();
    });

    it("applies size limits to error data", () => {
        // Create an error with a very long message and stack
        const longMessage = "x".repeat(3000);
        const longStack = "y".repeat(6000);
        
        const LongError = () => {
            const error = new Error(longMessage);
            error.stack = longStack;
            throw error;
        };

        render(
            <ErrorBoundary>
                <LongError />
            </ErrorBoundary>
        );

        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        expect(mockFetch).toHaveBeenCalledWith("/api/error-reports", expect.objectContaining({
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: expect.any(String),
        }));
    });

    it("handles missing navigator properties gracefully", () => {
        // Mock a scenario where navigator properties might be missing
        Object.defineProperty(window, 'navigator', {
            value: {},
        });

        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>
        );

        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        expect(mockFetch).toHaveBeenCalledWith("/api/error-reports", expect.objectContaining({
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: expect.stringContaining("Unknown"),
        }));
    });

    it("validates timestamp format", () => {
        render(
            <ErrorBoundary>
                <ThrowError shouldThrow={true} />
            </ErrorBoundary>
        );

        const refreshButton = screen.getByRole("button", { name: "Refresh" });
        fireEvent.click(refreshButton);

        expect(mockFetch).toHaveBeenCalledWith("/api/error-reports", expect.objectContaining({
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: expect.stringMatching(/"timestamp":"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z"/),
        }));
    });
});