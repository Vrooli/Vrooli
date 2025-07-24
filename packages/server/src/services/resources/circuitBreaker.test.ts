/**
 * Tests for Circuit Breaker functionality
 */

import { describe, expect, it, vi } from "vitest";
import { CircuitBreaker, CircuitBreakerFactory, CircuitState } from "./circuitBreaker.js";

describe("CircuitBreaker", () => {
    describe("Basic functionality", () => {
        it("should start in closed state", () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 3,
                recoveryTimeout: 1000,
                halfOpenTimeout: 500,
            });

            const stats = breaker.getStats();
            expect(stats.state).toBe(CircuitState.Closed);
            expect(stats.failureCount).toBe(0);
            expect(stats.successCount).toBe(0);
        });

        it("should allow calls when closed", () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 3,
                recoveryTimeout: 1000,
                halfOpenTimeout: 500,
            });

            expect(breaker.isCallAllowed()).toBe(true);
        });

        it("should execute successful functions", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 3,
                recoveryTimeout: 1000,
                halfOpenTimeout: 500,
            });

            const mockFn = vi.fn().mockResolvedValue("success");
            const result = await breaker.execute(mockFn);

            expect(result).toBe("success");
            expect(mockFn).toHaveBeenCalledTimes(1);

            const stats = breaker.getStats();
            expect(stats.successCount).toBe(1);
            expect(stats.failureCount).toBe(0);
        });

        it("should track failures", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 3,
                recoveryTimeout: 1000,
                halfOpenTimeout: 500,
            });

            const mockFn = vi.fn().mockRejectedValue(new Error("test error"));

            try {
                await breaker.execute(mockFn);
            } catch (error) {
                expect(error).toBeInstanceOf(Error);
            }

            const stats = breaker.getStats();
            expect(stats.failureCount).toBe(1);
            expect(stats.successCount).toBe(0);
        });

        it("should open circuit after threshold failures", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 3,
                recoveryTimeout: 1000,
                halfOpenTimeout: 500,
            });

            const mockFn = vi.fn().mockRejectedValue(new Error("test error"));

            // Cause 3 failures to open the circuit
            for (let i = 0; i < 3; i++) {
                try {
                    await breaker.execute(mockFn);
                } catch (error) {
                    // Expected to fail
                }
            }

            const stats = breaker.getStats();
            expect(stats.state).toBe(CircuitState.Open);
            expect(stats.failureCount).toBe(3);
        });

        it("should not execute function when open", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 2,
                recoveryTimeout: 1000,
                halfOpenTimeout: 500,
            });

            const mockFn = vi.fn().mockRejectedValue(new Error("test error"));

            // Open the circuit
            for (let i = 0; i < 2; i++) {
                try {
                    await breaker.execute(mockFn);
                } catch (error) {
                    // Expected to fail
                }
            }

            // Reset mock to track further calls
            mockFn.mockClear();

            // Try to execute when circuit is open
            try {
                await breaker.execute(mockFn);
                throw new Error("Should have thrown");
            } catch (error) {
                expect(error).toBeInstanceOf(Error);
                expect((error as Error).message).toContain("Circuit breaker");
            }

            // Function should not have been called
            expect(mockFn).not.toHaveBeenCalled();
        });
    });

    describe("State transitions", () => {
        it("should transition from open to half-open after timeout", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 2,
                recoveryTimeout: 100, // Short timeout for testing
                halfOpenTimeout: 500,
            });

            const mockFn = vi.fn().mockRejectedValue(new Error("test error"));

            // Open the circuit
            for (let i = 0; i < 2; i++) {
                try {
                    await breaker.execute(mockFn);
                } catch (error) {
                    // Expected to fail
                }
            }

            expect(breaker.getStats().state).toBe(CircuitState.Open);

            // Wait for recovery timeout
            await new Promise(resolve => setTimeout(resolve, 150));

            // Now should be in half-open state
            expect(breaker.isCallAllowed()).toBe(true);
            expect(breaker.getStats().state).toBe(CircuitState.HalfOpen);
        });

        it("should close circuit on successful call in half-open state", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 2,
                recoveryTimeout: 100,
                halfOpenTimeout: 500,
            });

            // Open the circuit
            const failingFn = vi.fn().mockRejectedValue(new Error("test error"));
            for (let i = 0; i < 2; i++) {
                try {
                    await breaker.execute(failingFn);
                } catch (error) {
                    // Expected to fail
                }
            }

            // Wait for recovery timeout to enter half-open
            await new Promise(resolve => setTimeout(resolve, 150));

            // Execute successful function
            const successFn = vi.fn().mockResolvedValue("success");
            const result = await breaker.execute(successFn);

            expect(result).toBe("success");
            expect(breaker.getStats().state).toBe(CircuitState.Closed);
            expect(breaker.getStats().failureCount).toBe(0); // Reset on close
        });

        it("should reopen circuit on failure in half-open state", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 2,
                recoveryTimeout: 100,
                halfOpenTimeout: 500,
            });

            // Open the circuit
            const failingFn = vi.fn().mockRejectedValue(new Error("test error"));
            for (let i = 0; i < 2; i++) {
                try {
                    await breaker.execute(failingFn);
                } catch (error) {
                    // Expected to fail
                }
            }

            // Wait for recovery timeout to enter half-open
            await new Promise(resolve => setTimeout(resolve, 150));
            expect(breaker.getStats().state).toBe(CircuitState.HalfOpen);

            // Execute failing function in half-open state
            try {
                await breaker.execute(failingFn);
                throw new Error("Should have thrown");
            } catch (error) {
                expect(error).toBeInstanceOf(Error);
            }

            expect(breaker.getStats().state).toBe(CircuitState.Open);
        });
    });

    describe("Manual control", () => {
        it("should allow manual reset", () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 2,
                recoveryTimeout: 1000,
                halfOpenTimeout: 500,
            });

            // Open the circuit
            breaker.execute(vi.fn().mockRejectedValue(new Error("fail"))).catch(() => {
                // Ignore error
            });

            // Manually reset
            breaker.forceReset();

            expect(breaker.getStats().state).toBe(CircuitState.Closed);
            expect(breaker.getStats().failureCount).toBe(0);
            expect(breaker.getStats().successCount).toBe(0);
        });
    });

    describe("Configuration validation", () => {
        it("should throw on invalid configuration", () => {
            expect(() => {
                new CircuitBreaker({
                    name: "",
                    failureThreshold: -1,
                    recoveryTimeout: 1000,
                    halfOpenTimeout: 500,
                });
            }).toThrow();

            expect(() => {
                new CircuitBreaker({
                    name: "test",
                    failureThreshold: 0,
                    recoveryTimeout: 1000,
                    halfOpenTimeout: 500,
                });
            }).toThrow();

            expect(() => {
                new CircuitBreaker({
                    name: "test",
                    failureThreshold: 5,
                    recoveryTimeout: -1,
                    halfOpenTimeout: 500,
                });
            }).toThrow();
        });
    });
});

describe("CircuitBreakerFactory", () => {
    it("should create circuit breaker for resource discovery", () => {
        const breaker = CircuitBreakerFactory.forResourceDiscovery("test-resource");

        const stats = breaker.getStats();
        expect(stats.state).toBe(CircuitState.Closed);
        expect(breaker.isCallAllowed()).toBe(true);
    });

    it("should create circuit breaker for health checks", () => {
        const breaker = CircuitBreakerFactory.forHealthCheck("test-resource");

        const stats = breaker.getStats();
        expect(stats.state).toBe(CircuitState.Closed);
        expect(breaker.isCallAllowed()).toBe(true);
    });

    it("should create circuit breaker for resource operations", () => {
        const breaker = CircuitBreakerFactory.forResourceOperation("test-resource");

        const stats = breaker.getStats();
        expect(stats.state).toBe(CircuitState.Closed);
        expect(breaker.isCallAllowed()).toBe(true);
    });
});
