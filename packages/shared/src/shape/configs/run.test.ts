import { describe, it, expect, vi, beforeEach } from "vitest";
import { RunProgressConfig, type RunProgressConfigObject } from "./run.js";
import { InputGenerationStrategy, PathSelectionStrategy, SubroutineExecutionStrategy } from "../../run/enums.js";
import { type RunProgress } from "../../run/types.js";
import { type Run } from "../../api/types.js";
import { runConfigFixtures } from "../../__test/fixtures/config/runConfigFixtures.js";

describe("RunProgressConfig", () => {
    let mockLogger: any;

    beforeEach(() => {
        mockLogger = {
            trace: vi.fn(),
            debug: vi.fn(),
            info: vi.fn(),
            warn: vi.fn(),
            error: vi.fn(),
        };
    });

    describe("constructor", () => {
        it("should create RunProgressConfig with complete data", () => {
            const data = runConfigFixtures.complete;

            const config = new RunProgressConfig(data);

            expect(config.__version).toBe("1.0");
            expect(config.branches).toHaveLength(2);
            expect(config.branches[0].branchId).toBe("branch_1");
            expect(config.config.botConfig?.model).toBe("gpt-4");
            expect(config.config.decisionConfig.inputGeneration).toBe(InputGenerationStrategy.Manual);
            expect(config.config.testMode).toBe(true);
            expect(config.decisions).toHaveLength(1);
            expect(config.metrics.creditsSpent).toBe("1000");
            expect(config.subcontexts["sub_1"]).toBeDefined();
        });

        it("should create RunProgressConfig with minimal data and defaults", () => {
            const data = runConfigFixtures.minimal;

            const config = new RunProgressConfig(data);

            expect(config.__version).toBe("1.0");
            expect(config.branches).toEqual([]);
            expect(config.config.botConfig).toEqual({});
            expect(config.config.decisionConfig).toEqual({
                inputGeneration: InputGenerationStrategy.Auto,
                pathSelection: PathSelectionStrategy.AutoPickFirst,
                subroutineExecution: SubroutineExecutionStrategy.Auto,
            });
            expect(config.config.testMode).toBe(false);
            expect(config.decisions).toEqual([]);
            expect(config.metrics).toEqual({ creditsSpent: "0" });
            expect(config.subcontexts).toEqual({});
        });
    });

    describe("parse", () => {
        it("should parse valid run data", () => {
            const testData = runConfigFixtures.variants.manualExecutionConfig;
            const runData: Pick<Run, "data"> = {
                data: JSON.stringify(testData),
            };

            const config = RunProgressConfig.parse(runData, mockLogger);

            expect(config.__version).toBe("1.0");
            expect(config.config.decisionConfig.inputGeneration).toBe(InputGenerationStrategy.Manual);
            expect(config.metrics.creditsSpent).toBe("0");
        });

        it("should parse with different stringify modes", () => {
            const configData = runConfigFixtures.minimal;

            // Test JSON mode (default)
            const jsonRun: Pick<Run, "data"> = {
                data: JSON.stringify(configData),
            };
            const jsonConfig = RunProgressConfig.parse(jsonRun, mockLogger);
            expect(jsonConfig.__version).toBe("1.0");

            // Test base64 mode
            const base64Run: Pick<Run, "data"> = {
                data: Buffer.from(JSON.stringify(configData)).toString("base64"),
            };
            const base64Config = RunProgressConfig.parse(base64Run, mockLogger, { mode: "base64" });
            expect(base64Config.__version).toBe("1.0");
        });

        it("should return default config when data is null", () => {
            const runData: Pick<Run, "data"> = {
                data: null,
            };

            const config = RunProgressConfig.parse(runData, mockLogger);

            expect(config).toEqual(RunProgressConfig.default());
        });

        it("should return default config when data is invalid", () => {
            const runData: Pick<Run, "data"> = {
                data: "invalid json",
            };

            const config = RunProgressConfig.parse(runData, mockLogger);

            expect(config).toEqual(RunProgressConfig.default());
            expect(mockLogger.error).toHaveBeenCalled();
        });
    });

    describe("default", () => {
        it("should create default RunProgressConfig", () => {
            const config = RunProgressConfig.default();

            expect(config.__version).toBe("1.0");
            expect(config.branches).toEqual([]);
            expect(config.config).toEqual({
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Auto,
                    pathSelection: PathSelectionStrategy.AutoPickFirst,
                    subroutineExecution: SubroutineExecutionStrategy.Auto,
                },
                isPrivate: true,
                limits: {},
                loopConfig: {},
                onBranchFailure: "Stop",
                onGatewayForkFailure: "Fail",
                onNormalNodeFailure: "Fail",
                onOnlyWaitingBranches: "Continue",
                testMode: false,
            });
            expect(config.decisions).toEqual([]);
            expect(config.metrics).toEqual({
                complexityCompleted: 0,
                complexityTotal: 0,
                creditsSpent: "0",
                startedAt: null,
                stepsRun: 0,
                timeElapsed: 0,
            });
            expect(config.subcontexts).toEqual({});
        });
    });

    describe("serialize", () => {
        it("should serialize to JSON by default", () => {
            const config = RunProgressConfig.default();
            const serialized = config.serialize("json");

            const parsed = JSON.parse(serialized);
            expect(parsed.__version).toBe("1.0");
            expect(parsed.branches).toEqual([]);
        });

        // Note: base64 serialization is not currently supported in utils.ts
        // Add this test when base64 mode is implemented in StringifyMode
    });

    describe("export", () => {
        it("should export without isPrivate in config", () => {
            const data = {
                ...runConfigFixtures.minimal,
                metrics: {
                    creditsSpent: "25000",
                },
            };

            const config = new RunProgressConfig(data);
            const exported = config.export();

            expect(exported.config.isPrivate).toBeUndefined();
            expect(exported.__version).toBe("1.0");
            expect(exported.metrics).toEqual({ creditsSpent: "25000" });
        });

        it("should only include creditsSpent in metrics", () => {
            const data = {
                ...runConfigFixtures.minimal,
                metrics: {
                    creditsSpent: "75000",
                },
            };

            const config = new RunProgressConfig(data);
            const exported = config.export();

            expect(exported.metrics).toEqual({ creditsSpent: "75000" });
            expect(exported.metrics).not.toHaveProperty("complexityCompleted");
            expect(exported.metrics).not.toHaveProperty("stepsRun");
        });
    });

    describe("defaultBranches", () => {
        it("should return empty array", () => {
            expect(RunProgressConfig.defaultBranches()).toEqual([]);
        });
    });

    describe("defaultRunConfig", () => {
        it("should return expected default run config", () => {
            const config = RunProgressConfig.defaultRunConfig();

            expect(config).toEqual({
                botConfig: {},
                decisionConfig: {
                    inputGeneration: InputGenerationStrategy.Auto,
                    pathSelection: PathSelectionStrategy.AutoPickFirst,
                    subroutineExecution: SubroutineExecutionStrategy.Auto,
                },
                isPrivate: true,
                limits: {},
                loopConfig: {},
                onBranchFailure: "Stop",
                onGatewayForkFailure: "Fail",
                onNormalNodeFailure: "Fail",
                onOnlyWaitingBranches: "Continue",
                testMode: false,
            });
        });
    });

    describe("defaultDecisions", () => {
        it("should return empty array", () => {
            expect(RunProgressConfig.defaultDecisions()).toEqual([]);
        });
    });

    describe("defaultMetrics", () => {
        it("should return expected default metrics", () => {
            const metrics = RunProgressConfig.defaultMetrics();

            expect(metrics).toEqual({
                complexityCompleted: 0,
                complexityTotal: 0,
                creditsSpent: "0",
                startedAt: null,
                stepsRun: 0,
                timeElapsed: 0,
            });
        });
    });

    describe("defaultSubcontexts", () => {
        it("should return empty object", () => {
            expect(RunProgressConfig.defaultSubcontexts()).toEqual({});
        });
    });

    describe("Complex scenarios", () => {
        it("should handle multiple branches with different states", () => {
            const data = runConfigFixtures.variants.withActiveBranches;

            const config = new RunProgressConfig(data);

            expect(config.branches).toHaveLength(3);
            expect(config.branches[0].status).toBe("Active");
            expect(config.branches[1].status).toBe("Active");
            expect(config.branches[2].status).toBe("Active");
        });

        it("should handle complex decision history", () => {
            const data = runConfigFixtures.variants.withCompletedDecisions;

            const config = new RunProgressConfig(data);

            expect(config.decisions).toHaveLength(2);
            expect(config.decisions[0].__type).toBe("Resolved");
            expect(config.decisions[1].__type).toBe("Resolved");
            expect(config.decisions[0].decisionType).toBe("chooseOne");
            expect(config.decisions[1].decisionType).toBe("chooseOne");
        });

        it("should handle nested subcontexts", () => {
            const data = runConfigFixtures.variants.withSubcontexts;

            const config = new RunProgressConfig(data);

            expect(config.subcontexts["sub_process_1"]).toBeDefined();
            expect(config.subcontexts["sub_validate_1"]).toBeDefined();
            expect(config.subcontexts["sub_transform_1"]).toBeDefined();
            expect(config.subcontexts["sub_process_1"].allOutputsMap.result).toBe("processed");
            expect(config.subcontexts["sub_validate_1"].allOutputsMap.valid).toBe(true);
            expect(config.subcontexts["sub_transform_1"].allOutputsMap.transformed).toEqual({ key: "value" });
        });

        it("should handle different failure handling strategies", () => {
            const testConfigs = [
                runConfigFixtures.minimal, // Stop, Fail, Fail
                runConfigFixtures.complete, // Continue, Wait, Continue
                runConfigFixtures.variants.autoExecutionConfig, // Continue, Fail, Fail
            ];

            const expectedStrategies = [
                { onBranchFailure: "Stop", onGatewayForkFailure: "Fail", onNormalNodeFailure: "Fail" },
                { onBranchFailure: "Continue", onGatewayForkFailure: "Wait", onNormalNodeFailure: "Continue" },
                { onBranchFailure: "Continue", onGatewayForkFailure: "Fail", onNormalNodeFailure: "Fail" },
            ];

            for (let i = 0; i < testConfigs.length; i++) {
                const config = new RunProgressConfig(testConfigs[i]);
                const expected = expectedStrategies[i];

                expect(config.config.onBranchFailure).toBe(expected.onBranchFailure);
                expect(config.config.onGatewayForkFailure).toBe(expected.onGatewayForkFailure);
                expect(config.config.onNormalNodeFailure).toBe(expected.onNormalNodeFailure);
            }
        });
    });
});
