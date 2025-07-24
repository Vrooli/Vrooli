/**
 * Tests for Circuit Breaker functionality
 */

import { CircuitBreaker, CircuitState, CircuitBreakerFactory } from "../circuitBreaker.js";

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

            const mockFn = jest.fn().mockResolvedValue("success");
            const result = await breaker.execute(mockFn);

            expect(result).toBe("success");
            expect(mockFn).toHaveBeenCalledTimes(1);
            
            const stats = breaker.getStats();
            expect(stats.successCount).toBe(1);
            expect(stats.failureCount).toBe(0);
        });
    });

    describe("Failure handling", () => {
        it("should track failures", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 3,
                recoveryTimeout: 1000,
                halfOpenTimeout: 500,
            });

            const mockFn = jest.fn().mockRejectedValue(new Error("test error"));

            try {
                await breaker.execute(mockFn);
            } catch (error) {
                expect(error).toBeInstanceOf(Error);
            }

            const stats = breaker.getStats();
            expect(stats.failureCount).toBe(1);
            expect(stats.successCount).toBe(0);
            expect(stats.state).toBe(CircuitState.Closed);
        });

        it("should open circuit after threshold failures", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 3,
                recoveryTimeout: 1000,
                halfOpenTimeout: 500,
            });

            const mockFn = jest.fn().mockRejectedValue(new Error("test error"));

            // Trigger failures to reach threshold
            for (let i = 0; i < 3; i++) {
                try {
                    await breaker.execute(mockFn);
                } catch (error) {
                    // Expected
                }
            }

            const stats = breaker.getStats();
            expect(stats.state).toBe(CircuitState.Open);
            expect(stats.failureCount).toBe(3);
            expect(breaker.isCallAllowed()).toBe(false);
        });

        it("should block calls when circuit is open", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 2,
                recoveryTimeout: 1000,
                halfOpenTimeout: 500,
            });

            const mockFn = jest.fn().mockRejectedValue(new Error("test error"));

            // Trigger failures to open circuit
            for (let i = 0; i < 2; i++) {
                try {
                    await breaker.execute(mockFn);
                } catch (error) {
                    // Expected
                }
            }

            // Now circuit should be open and block calls
            try {
                await breaker.execute(jest.fn());
                fail("Should have thrown circuit breaker error");
            } catch (error) {
                expect(error).toBeInstanceOf(Error);
                expect((error as Error).message).toContain("Circuit breaker is OPEN");
            }

            const stats = breaker.getStats();
            expect(stats.blockedCalls).toBe(1);
        });
    });

    describe("Recovery mechanism", () => {
        it("should transition to half-open after recovery timeout", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 2,
                recoveryTimeout: 100, // Short timeout for test
                halfOpenTimeout: 500,
            });

            const mockFn = jest.fn().mockRejectedValue(new Error("test error"));

            // Open the circuit
            for (let i = 0; i < 2; i++) {
                try {
                    await breaker.execute(mockFn);
                } catch (error) {
                    // Expected
                }
            }

            expect(breaker.getStats().state).toBe(CircuitState.Open);

            // Wait for recovery timeout
            await new Promise(resolve => setTimeout(resolve, 150));

            // Circuit should allow one call to test recovery
            expect(breaker.isCallAllowed()).toBe(true);

            // Executing should transition to half-open
            const successFn = jest.fn().mockResolvedValue("success");
            const result = await breaker.execute(successFn);

            expect(result).toBe("success");
            expect(breaker.getStats().state).toBe(CircuitState.Closed);
        });

        it("should reset to closed state on successful half-open call", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 1,
                recoveryTimeout: 100,
                halfOpenTimeout: 500,
            });

            // Open the circuit
            try {
                await breaker.execute(jest.fn().mockRejectedValue(new Error("fail")));
            } catch (error) {
                // Expected
            }

            // Wait for recovery
            await new Promise(resolve => setTimeout(resolve, 150));

            // Successful call should close circuit
            const result = await breaker.execute(jest.fn().mockResolvedValue("success"));
            expect(result).toBe("success");

            const stats = breaker.getStats();
            expect(stats.state).toBe(CircuitState.Closed);
            expect(stats.failureCount).toBe(0); // Should reset failure count
        });

        it("should reopen circuit on half-open failure", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 1,
                recoveryTimeout: 100,
                halfOpenTimeout: 500,
            });

            // Open the circuit
            try {
                await breaker.execute(jest.fn().mockRejectedValue(new Error("fail")));
            } catch (error) {
                // Expected
            }

            // Wait for recovery
            await new Promise(resolve => setTimeout(resolve, 150));

            // Failed call should reopen circuit
            try {
                await breaker.execute(jest.fn().mockRejectedValue(new Error("fail again")));
            } catch (error) {
                // Expected
            }

            expect(breaker.getStats().state).toBe(CircuitState.Open);
        });
    });

    describe("Manual reset", () => {
        it("should allow manual reset of circuit", () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 1,
                recoveryTimeout: 10000, // Long timeout
                halfOpenTimeout: 500,
            });

            // Open the circuit
            breaker.execute(jest.fn().mockRejectedValue(new Error("fail"))).catch(() => {});

            // Manually reset
            breaker.forceReset();

            const stats = breaker.getStats();
            expect(stats.state).toBe(CircuitState.Closed);
            expect(stats.failureCount).toBe(0);
            expect(breaker.isCallAllowed()).toBe(true);
        });
    });

    describe("Factory methods", () => {
        it("should create circuit breaker for resource discovery", () => {
            const breaker = CircuitBreakerFactory.forResourceDiscovery("test-resource");
            
            const stats = breaker.getStats();
            expect(stats.state).toBe(CircuitState.Closed);
            
            // Test that it has reasonable defaults
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

    describe("Timeout handling", () => {
        it("should timeout half-open calls", async () => {
            const breaker = new CircuitBreaker({
                name: "test",
                failureThreshold: 1,
                recoveryTimeout: 100,
                halfOpenTimeout: 100, // Short timeout
            });

            // Open the circuit
            try {
                await breaker.execute(jest.fn().mockRejectedValue(new Error("fail")));
            } catch (error) {
                // Expected
            }

            // Wait for recovery
            await new Promise(resolve => setTimeout(resolve, 150));

            // Slow function that should timeout in half-open state
            const slowFn = jest.fn().mockImplementation(
                () => new Promise(resolve => setTimeout(resolve, 200))
            );

            try {
                await breaker.execute(slowFn);
                fail("Should have timed out");
            } catch (error) {
                expect(error).toBeInstanceOf(Error);
                expect((error as Error).message).toContain("Half-open timeout");
            }

            expect(breaker.getStats().state).toBe(CircuitState.Open);
        });
    });
});