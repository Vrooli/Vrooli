import { describe, expect, it, vi, beforeEach } from "vitest";
import { RunStatus } from "../api/types.js";
import { RunLimitsManager } from "./limits.js";
import { RunStatusChangeReason, type RunProgress, type RunRequestLimits } from "./types.js";

describe("RunLimitsManager", () => {
    let limitsManager: RunLimitsManager;
    let mockLogger: any;
    let mockRun: RunProgress;

    beforeEach(() => {
        mockLogger = {
            log: vi.fn(),
            info: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        };

        limitsManager = new RunLimitsManager(mockLogger);

        mockRun = {
            runId: "test-run-123",
            status: RunStatus.Running,
            metrics: {
                creditsSpent: "100",
                stepsRun: 5,
            },
        } as RunProgress;
    });

    describe("checkLimits", () => {
        it("should return undefined when no limits are reached", () => {
            const runLimits: RunRequestLimits = {
                maxTime: 60000, // 1 minute
                maxCredits: "1000",
                maxSteps: 100,
            };
            const startTime = Date.now() - 5000; // 5 seconds ago

            const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

            expect(result).toBeUndefined();
            expect(mockRun.status).toBe(RunStatus.Running);
        });

        describe("time limit", () => {
            it("should detect time limit exceeded", () => {
                const runLimits: RunRequestLimits = {
                    maxTime: 5000, // 5 seconds
                    onMaxTime: "Stop",
                };
                const startTime = Date.now() - 10000; // 10 seconds ago

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxTime);
                expect(mockRun.status).toBe(RunStatus.Failed);
                expect(mockLogger.error).toHaveBeenCalledWith(
                    expect.stringContaining("Run test-run-123 reached time limit")
                );
            });

            it("should pause run when onMaxTime is 'Pause'", () => {
                const runLimits: RunRequestLimits = {
                    maxTime: 5000,
                    onMaxTime: "Pause",
                };
                const startTime = Date.now() - 10000;

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxTime);
                expect(mockRun.status).toBe(RunStatus.Paused);
            });

            it("should not check time limit when maxTime is not set", () => {
                const runLimits: RunRequestLimits = {};
                const startTime = Date.now() - 100000; // Long time ago

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBeUndefined();
                expect(mockRun.status).toBe(RunStatus.Running);
            });

            it("should allow execution at the time limit boundary", () => {
                const runLimits: RunRequestLimits = {
                    maxTime: 5000,
                    onMaxTime: "Stop",
                };
                // Use a fixed time to avoid timing precision issues
                const fixedNow = 1000000; // Fixed timestamp
                const startTime = fixedNow - 5000; // Exactly 5 seconds ago
                
                // Mock Date.now to return our fixed time
                const originalNow = Date.now;
                Date.now = vi.fn(() => fixedNow);

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                // Restore Date.now
                Date.now = originalNow;

                // When elapsed time equals max time, it should still be within limits
                // Users should get the full time duration they configured
                expect(result).toBeUndefined();
                expect(mockRun.status).toBe(RunStatus.Running);
            });
        });

        describe("credits limit", () => {
            it("should detect credits limit exceeded", () => {
                mockRun.metrics.creditsSpent = "1000";
                const runLimits: RunRequestLimits = {
                    maxCredits: "500",
                    onMaxCredits: "Stop",
                };
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxCredits);
                expect(mockRun.status).toBe(RunStatus.Failed);
                expect(mockLogger.error).toHaveBeenCalledWith(
                    expect.stringContaining("Run test-run-123 reached credits limit")
                );
            });

            it("should pause run when onMaxCredits is 'Pause'", () => {
                mockRun.metrics.creditsSpent = "1000";
                const runLimits: RunRequestLimits = {
                    maxCredits: "500",
                    onMaxCredits: "Pause",
                };
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxCredits);
                expect(mockRun.status).toBe(RunStatus.Paused);
            });

            it("should handle large credit numbers", () => {
                mockRun.metrics.creditsSpent = "999999999999999999";
                const runLimits: RunRequestLimits = {
                    maxCredits: "999999999999999998",
                    onMaxCredits: "Stop",
                };
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxCredits);
                expect(mockRun.status).toBe(RunStatus.Failed);
            });

            it("should handle zero credits spent", () => {
                mockRun.metrics.creditsSpent = "0";
                const runLimits: RunRequestLimits = {
                    maxCredits: "100",
                };
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBeUndefined();
                expect(mockRun.status).toBe(RunStatus.Running);
            });

            it("should handle undefined credits spent", () => {
                mockRun.metrics.creditsSpent = undefined;
                const runLimits: RunRequestLimits = {
                    maxCredits: "100",
                };
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBeUndefined();
                expect(mockRun.status).toBe(RunStatus.Running);
            });

            it("should not check credits limit when maxCredits is not set", () => {
                mockRun.metrics.creditsSpent = "1000000";
                const runLimits: RunRequestLimits = {};
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBeUndefined();
                expect(mockRun.status).toBe(RunStatus.Running);
            });

            it("should allow execution at the credit limit boundary", () => {
                mockRun.metrics.creditsSpent = "500";
                const runLimits: RunRequestLimits = {
                    maxCredits: "500",
                    onMaxCredits: "Stop",
                };
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                // When credits spent equals max credits, it should still be within limits
                // Users should get the full credit amount they paid for
                expect(result).toBeUndefined();
                expect(mockRun.status).toBe(RunStatus.Running);
            });
        });

        describe("steps limit", () => {
            it("should detect steps limit exceeded", () => {
                mockRun.metrics.stepsRun = 100;
                const runLimits: RunRequestLimits = {
                    maxSteps: 50,
                    onMaxSteps: "Stop",
                };
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxSteps);
                expect(mockRun.status).toBe(RunStatus.Failed);
                expect(mockLogger.error).toHaveBeenCalledWith(
                    expect.stringContaining("Run test-run-123 reached steps limit")
                );
            });

            it("should pause run when onMaxSteps is 'Pause'", () => {
                mockRun.metrics.stepsRun = 100;
                const runLimits: RunRequestLimits = {
                    maxSteps: 50,
                    onMaxSteps: "Pause",
                };
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxSteps);
                expect(mockRun.status).toBe(RunStatus.Paused);
            });

            it("should not check steps limit when maxSteps is undefined", () => {
                mockRun.metrics.stepsRun = 1000;
                const runLimits: RunRequestLimits = {};
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBeUndefined();
                expect(mockRun.status).toBe(RunStatus.Running);
            });

            it("should allow execution at the step limit boundary", () => {
                mockRun.metrics.stepsRun = 50;
                const runLimits: RunRequestLimits = {
                    maxSteps: 50,
                    onMaxSteps: "Stop",
                };
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                // When steps run equals max steps, execution should still be allowed 
                // The limit is triggered only when exceeded (stepsRun > maxSteps)
                // This gives users the full range they paid for
                expect(result).toBeUndefined();
                expect(mockRun.status).toBe(RunStatus.Running);
            });

            it("should handle zero steps run", () => {
                mockRun.metrics.stepsRun = 0;
                const runLimits: RunRequestLimits = {
                    maxSteps: 10,
                };
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBeUndefined();
                expect(mockRun.status).toBe(RunStatus.Running);
            });
        });

        describe("limit priority", () => {
            it("should return time limit when multiple limits are exceeded", () => {
                mockRun.metrics.creditsSpent = "1000";
                mockRun.metrics.stepsRun = 100;
                const runLimits: RunRequestLimits = {
                    maxTime: 5000,
                    maxCredits: "500",
                    maxSteps: 50,
                    onMaxTime: "Stop",
                    onMaxCredits: "Pause",
                    onMaxSteps: "Stop",
                };
                const startTime = Date.now() - 10000;

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                // Time limit should be checked first
                expect(result).toBe(RunStatusChangeReason.MaxTime);
                expect(mockRun.status).toBe(RunStatus.Failed);
            });

            it("should return credits limit when time limit not exceeded but credits limit is", () => {
                mockRun.metrics.creditsSpent = "1000";
                mockRun.metrics.stepsRun = 100;
                const runLimits: RunRequestLimits = {
                    maxTime: 60000, // Not exceeded
                    maxCredits: "500",
                    maxSteps: 50,
                    onMaxCredits: "Pause",
                    onMaxSteps: "Stop",
                };
                const startTime = Date.now() - 5000;

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                // Credits limit should be checked second
                expect(result).toBe(RunStatusChangeReason.MaxCredits);
                expect(mockRun.status).toBe(RunStatus.Paused);
            });

            it("should return steps limit when only steps limit is exceeded", () => {
                mockRun.metrics.creditsSpent = "100";
                mockRun.metrics.stepsRun = 100;
                const runLimits: RunRequestLimits = {
                    maxTime: 60000, // Not exceeded
                    maxCredits: "500", // Not exceeded
                    maxSteps: 50,
                    onMaxSteps: "Stop",
                };
                const startTime = Date.now() - 5000;

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                // Steps limit should be checked last
                expect(result).toBe(RunStatusChangeReason.MaxSteps);
                expect(mockRun.status).toBe(RunStatus.Failed);
            });
        });

        describe("behavior handling", () => {
            it("should default to 'Stop' behavior when behavior is undefined", () => {
                const runLimits: RunRequestLimits = {
                    maxTime: 5000,
                    // onMaxTime is undefined
                };
                const startTime = Date.now() - 10000;

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxTime);
                expect(mockRun.status).toBe(RunStatus.Failed);
            });

            it("should handle 'Pause' behavior correctly", () => {
                const runLimits: RunRequestLimits = {
                    maxTime: 5000,
                    onMaxTime: "Pause",
                };
                const startTime = Date.now() - 10000;

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxTime);
                expect(mockRun.status).toBe(RunStatus.Paused);
            });

            it("should handle 'Stop' behavior correctly", () => {
                const runLimits: RunRequestLimits = {
                    maxTime: 5000,
                    onMaxTime: "Stop",
                };
                const startTime = Date.now() - 10000;

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxTime);
                expect(mockRun.status).toBe(RunStatus.Failed);
            });
        });

        describe("edge cases", () => {
            it("should handle empty metrics object", () => {
                mockRun.metrics = {} as any; // Missing stepsRun property
                const runLimits: RunRequestLimits = {
                    maxCredits: "100",
                    maxSteps: 10,
                    onMaxSteps: "Stop",
                };
                const startTime = Date.now();

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                // When stepsRun is undefined, it should be treated as 0 and not trigger limits
                expect(result).toBeUndefined();
                expect(mockRun.status).toBe(RunStatus.Running);
            });

            it("should handle very large time differences", () => {
                const runLimits: RunRequestLimits = {
                    maxTime: 1000,
                    onMaxTime: "Stop",
                };
                const startTime = 0; // Very long time ago

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxTime);
                expect(mockRun.status).toBe(RunStatus.Failed);
            });

            it("should handle negative start time", () => {
                const runLimits: RunRequestLimits = {
                    maxTime: 1000,
                    onMaxTime: "Stop",
                };
                const startTime = -1000;

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxTime);
                expect(mockRun.status).toBe(RunStatus.Failed);
            });

            it("should handle run with different initial status", () => {
                mockRun.status = RunStatus.Paused;
                const runLimits: RunRequestLimits = {
                    maxTime: 5000,
                    onMaxTime: "Stop",
                };
                const startTime = Date.now() - 10000;

                const result = limitsManager.checkLimits(mockRun, runLimits, startTime);

                expect(result).toBe(RunStatusChangeReason.MaxTime);
                expect(mockRun.status).toBe(RunStatus.Failed); // Should change to Failed
            });
        });
    });

    describe("logging", () => {
        it("should log error for time limit", () => {
            const runLimits: RunRequestLimits = {
                maxTime: 5000,
                onMaxTime: "Stop",
            };
            const startTime = Date.now() - 10000;

            limitsManager.checkLimits(mockRun, runLimits, startTime);

            expect(mockLogger.error).toHaveBeenCalledWith(
                expect.stringContaining("checkLimits: Run test-run-123 reached time limit")
            );
        });

        it("should log error for credits limit", () => {
            mockRun.metrics.creditsSpent = "1000";
            const runLimits: RunRequestLimits = {
                maxCredits: "500",
                onMaxCredits: "Stop",
            };
            const startTime = Date.now();

            limitsManager.checkLimits(mockRun, runLimits, startTime);

            expect(mockLogger.error).toHaveBeenCalledWith(
                expect.stringContaining("checkLimits: Run test-run-123 reached credits limit")
            );
        });

        it("should log error for steps limit", () => {
            mockRun.metrics.stepsRun = 100;
            const runLimits: RunRequestLimits = {
                maxSteps: 50,
                onMaxSteps: "Stop",
            };
            const startTime = Date.now();

            limitsManager.checkLimits(mockRun, runLimits, startTime);

            expect(mockLogger.error).toHaveBeenCalledWith(
                expect.stringContaining("checkLimits: Run test-run-123 reached steps limit")
            );
        });

        it("should not log when no limits are reached", () => {
            const runLimits: RunRequestLimits = {
                maxTime: 60000,
                maxCredits: "1000",
                maxSteps: 100,
            };
            const startTime = Date.now() - 5000;

            limitsManager.checkLimits(mockRun, runLimits, startTime);

            expect(mockLogger.error).not.toHaveBeenCalled();
        });
    });
});