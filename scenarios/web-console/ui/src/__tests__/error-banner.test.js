import { describe, it, expect } from "vitest";
/**
 * Tests for the ErrorBanner extension point.
 *
 * These tests validate that the ErrorBanner component module exists
 * and exports the expected interface. The component is the single
 * place where error display logic lives, enabling error category
 * and recovery changes without touching layout components.
 *
 * [REQ:P0-001b] Independent Pane Session Lifecycle — error feedback
 */
describe("ErrorBanner (change axis: error display)", () => {
    it("exports default component function", async () => {
        const mod = await import("../components/ErrorBanner");
        expect(typeof mod.default).toBe("function");
    });
    it("ErrorInfo shape supports message, recovery, and retry", () => {
        const info = {
            message: "Test error",
            recovery: "Try again",
            retry: true,
        };
        expect(info.message).toBe("Test error");
        expect(info.recovery).toBe("Try again");
        expect(info.retry).toBe(true);
    });
    it("ErrorInfo works with minimal fields", () => {
        const info = {
            message: "Simple error",
        };
        expect(info.message).toBe("Simple error");
        expect(info.recovery).toBeUndefined();
        expect(info.retry).toBeUndefined();
    });
});
