import { describe, it, expect, vi, beforeEach } from "vitest";
import { RunProgressConfig, type RunProgressConfigObject } from "./run.js";
import { InputGenerationStrategy, PathSelectionStrategy, SubroutineExecutionStrategy } from "../../run/enums.js";
import { type RunProgress } from "../../run/types.js";
import { type Run } from "../../api/types.js";

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
            const testTime = new Date().toISOString();
            const data: RunProgressConfigObject = {
                __version: "1.0",
                branches: [
                    {
                        id: "branch1",
                        parentId: null,
                        status: "Running",
                        depth: 0,
                        path: ["node1", "node2"],
                        completedNodes: ["node1"],
                        currentNode: "node2",
                        nextNodes: ["node3"],
                        waitingForSubroutines: [],
                        inputValues: { input1: "value1" },
                        outputValues: {},
                        createdAt: testTime,
                        isActive: true,
                    },
                ],
                config: {
                    botConfig: { model: "gpt-4" },
                    decisionConfig: {
                        inputGeneration: InputGenerationStrategy.LlmGenerate,
                        pathSelection: PathSelectionStrategy.LlmPick,
                        subroutineExecution: SubroutineExecutionStrategy.Parallel,
                    },
                    limits: {
                        maxTokens: 1000,
                        maxSteps: 100,
                    },
                    loopConfig: {
                        maxIterations: 10,
                    },
                    onBranchFailure: "Continue",
                    onGatewayForkFailure: "Stop",
                    onNormalNodeFailure: "Skip",
                    onOnlyWaitingBranches: "Stop",
                    testMode: true,
                },
                decisions: [
                    {
                        id: "decision1",
                        type: "InputGeneration",
                        timestamp: Date.now(),
                        branchId: "branch1",
                        nodeId: "node1",
                        status: "Resolved",
                        result: { generated: true },
                    },
                ],
                metrics: {
                    creditsSpent: "50000",
                },
                subcontexts: {
                    "subroutine1": {
                        inputs: { data: "test" },
                        outputs: { result: "success" },
                        metadata: { startTime: testTime },
                    },
                },
            };

            const config = new RunProgressConfig(data);

            expect(config.__version).toBe("1.0");
            expect(config.branches).toHaveLength(1);
            expect(config.branches[0].id).toBe("branch1");
            expect(config.config.botConfig?.model).toBe("gpt-4");
            expect(config.config.decisionConfig.inputGeneration).toBe(InputGenerationStrategy.LlmGenerate);
            expect(config.config.testMode).toBe(true);
            expect(config.decisions).toHaveLength(1);
            expect(config.metrics.creditsSpent).toBe("50000");
            expect(config.subcontexts["subroutine1"]).toBeDefined();
        });

        it("should create RunProgressConfig with minimal data and defaults", () => {
            const data: Partial<RunProgressConfigObject> = {};

            const config = new RunProgressConfig(data as RunProgressConfigObject);

            expect(config.__version).toBe("1.0");
            expect(config.branches).toEqual([]);
            expect(config.config).toEqual(RunProgressConfig.defaultRunConfig());
            expect(config.decisions).toEqual([]);
            expect(config.metrics).toEqual(RunProgressConfig.defaultMetrics());
            expect(config.subcontexts).toEqual({});
        });
    });

    describe("parse", () => {
        it("should parse valid run data", () => {
            const runData: Pick<Run, "data"> = {
                data: JSON.stringify({
                    __version: "1.0",
                    branches: [],
                    config: {
                        botConfig: {},
                        decisionConfig: {
                            inputGeneration: InputGenerationStrategy.Manual,
                            pathSelection: PathSelectionStrategy.ManualPick,
                            subroutineExecution: SubroutineExecutionStrategy.Sequential,
                        },
                        limits: {},
                        loopConfig: {},
                        onBranchFailure: "Stop",
                        onGatewayForkFailure: "Fail",
                        onNormalNodeFailure: "Fail",
                        onOnlyWaitingBranches: "Continue",
                        testMode: false,
                    },
                    decisions: [],
                    metrics: {
                        creditsSpent: "100000",
                    },
                    subcontexts: {},
                }),
            };

            const config = RunProgressConfig.parse(runData, mockLogger);

            expect(config.__version).toBe("1.0");
            expect(config.config.decisionConfig.inputGeneration).toBe(InputGenerationStrategy.Manual);
            expect(config.metrics.creditsSpent).toBe("100000");
        });

        it("should parse with different stringify modes", () => {
            const configData: RunProgressConfigObject = {
                __version: "1.0",
                branches: [],
                config: RunProgressConfig.defaultRunConfig(),
                decisions: [],
                metrics: { creditsSpent: "0" },
                subcontexts: {},
            };

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
            const data: RunProgressConfigObject = {
                __version: "1.0",
                branches: [],
                config: {
                    ...RunProgressConfig.defaultRunConfig(),
                    isPrivate: false, // This should be removed in export
                },
                decisions: [],
                metrics: {
                    creditsSpent: "25000",
                },
                subcontexts: {},
            };

            const config = new RunProgressConfig(data);
            const exported = config.export();

            expect(exported.config.isPrivate).toBeUndefined();
            expect(exported.__version).toBe("1.0");
            expect(exported.metrics).toEqual({ creditsSpent: "25000" });
        });

        it("should only include creditsSpent in metrics", () => {
            const fullMetrics: RunProgress["metrics"] = {
                complexityCompleted: 50,
                complexityTotal: 100,
                creditsSpent: "75000",
                startedAt: Date.now(),
                stepsRun: 25,
                timeElapsed: 60000,
            };

            const data: RunProgressConfigObject = {
                __version: "1.0",
                branches: [],
                config: RunProgressConfig.defaultRunConfig(),
                decisions: [],
                metrics: fullMetrics as any, // Cast to bypass type checking
                subcontexts: {},
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
            const branches: RunProgress["branches"] = [
                {
                    id: "main",
                    parentId: null,
                    status: "Completed",
                    depth: 0,
                    path: ["start", "middle", "end"],
                    completedNodes: ["start", "middle", "end"],
                    currentNode: null,
                    nextNodes: [],
                    waitingForSubroutines: [],
                    inputValues: { initial: "data" },
                    outputValues: { final: "result" },
                    createdAt: new Date().toISOString(),
                    completedAt: new Date().toISOString(),
                    isActive: false,
                },
                {
                    id: "fork1",
                    parentId: "main",
                    status: "Running",
                    depth: 1,
                    path: ["fork_start", "fork_middle"],
                    completedNodes: ["fork_start"],
                    currentNode: "fork_middle",
                    nextNodes: ["fork_end"],
                    waitingForSubroutines: ["sub1", "sub2"],
                    inputValues: { fork_input: "value" },
                    outputValues: {},
                    createdAt: new Date().toISOString(),
                    isActive: true,
                },
            ];

            const data: RunProgressConfigObject = {
                __version: "1.0",
                branches,
                config: RunProgressConfig.defaultRunConfig(),
                decisions: [],
                metrics: { creditsSpent: "0" },
                subcontexts: {},
            };

            const config = new RunProgressConfig(data);

            expect(config.branches).toHaveLength(2);
            expect(config.branches[0].status).toBe("Completed");
            expect(config.branches[1].status).toBe("Running");
            expect(config.branches[1].waitingForSubroutines).toEqual(["sub1", "sub2"]);
        });

        it("should handle complex decision history", () => {
            const decisions: RunProgress["decisions"] = [
                {
                    id: "d1",
                    type: "PathSelection",
                    timestamp: Date.now() - 60000,
                    branchId: "branch1",
                    nodeId: "gateway1",
                    status: "Resolved",
                    result: { selectedPath: "path_a" },
                },
                {
                    id: "d2",
                    type: "InputGeneration",
                    timestamp: Date.now() - 30000,
                    branchId: "branch1",
                    nodeId: "input_node",
                    status: "Deferred",
                    deferredUntil: Date.now() + 30000,
                },
                {
                    id: "d3",
                    type: "SubroutineExecution",
                    timestamp: Date.now() - 10000,
                    branchId: "branch2",
                    nodeId: "routine_node",
                    status: "Failed",
                    error: "Subroutine timeout",
                },
            ];

            const data: RunProgressConfigObject = {
                __version: "1.0",
                branches: [],
                config: RunProgressConfig.defaultRunConfig(),
                decisions,
                metrics: { creditsSpent: "0" },
                subcontexts: {},
            };

            const config = new RunProgressConfig(data);

            expect(config.decisions).toHaveLength(3);
            expect(config.decisions[0].status).toBe("Resolved");
            expect(config.decisions[1].status).toBe("Deferred");
            expect(config.decisions[2].status).toBe("Failed");
        });

        it("should handle nested subcontexts", () => {
            const subcontexts: RunProgress["subcontexts"] = {
                "routine1": {
                    inputs: { data: [1, 2, 3] },
                    outputs: { sum: 6 },
                    metadata: { executionTime: 123 },
                },
                "routine2": {
                    inputs: { 
                        complex: {
                            nested: {
                                value: "deep",
                                array: [1, 2, 3],
                            },
                        },
                    },
                    outputs: { 
                        processed: true,
                        results: ["a", "b", "c"],
                    },
                    metadata: {
                        startTime: new Date().toISOString(),
                        endTime: new Date().toISOString(),
                        creditsUsed: 50,
                    },
                },
            };

            const data: RunProgressConfigObject = {
                __version: "1.0",
                branches: [],
                config: RunProgressConfig.defaultRunConfig(),
                decisions: [],
                metrics: { creditsSpent: "150000" },
                subcontexts,
            };

            const config = new RunProgressConfig(data);

            expect(config.subcontexts["routine1"].outputs.sum).toBe(6);
            expect(config.subcontexts["routine2"].inputs.complex.nested.value).toBe("deep");
            expect(config.subcontexts["routine2"].metadata.creditsUsed).toBe(50);
        });

        it("should handle different failure handling strategies", () => {
            const configs = [
                { onBranchFailure: "Stop", onGatewayForkFailure: "Fail", onNormalNodeFailure: "Fail" },
                { onBranchFailure: "Continue", onGatewayForkFailure: "Stop", onNormalNodeFailure: "Skip" },
                { onBranchFailure: "Skip", onGatewayForkFailure: "Continue", onNormalNodeFailure: "Stop" },
            ];

            for (const testConfig of configs) {
                const data: RunProgressConfigObject = {
                    __version: "1.0",
                    branches: [],
                    config: {
                        ...RunProgressConfig.defaultRunConfig(),
                        ...testConfig,
                    },
                    decisions: [],
                    metrics: { creditsSpent: "0" },
                    subcontexts: {},
                };

                const config = new RunProgressConfig(data);

                expect(config.config.onBranchFailure).toBe(testConfig.onBranchFailure);
                expect(config.config.onGatewayForkFailure).toBe(testConfig.onGatewayForkFailure);
                expect(config.config.onNormalNodeFailure).toBe(testConfig.onNormalNodeFailure);
            }
        });
    });
});