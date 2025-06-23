import { generatePK, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface RunIORelationConfig extends RelationConfig {
    withRun?: { runId: string };
}

/**
 * Enhanced database fixture factory for RunIO model
 * Provides comprehensive testing capabilities for run input/output data
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for various data types
 * - Node-based input/output tracking
 * - JSON data validation
 * - Predefined test scenarios
 * - Large data handling
 */
export class RunIODbFactory extends EnhancedDatabaseFactory<
    Prisma.run_ioCreateInput,
    Prisma.run_ioCreateInput,
    Prisma.run_ioInclude,
    Prisma.run_ioUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("RunIO", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.run_io;
    }

    /**
     * Get complete test fixtures for RunIO model
     */
    protected getFixtures(): DbTestFixtures<Prisma.run_ioCreateInput, Prisma.run_ioUpdateInput> {
        return {
            minimal: {
                id: generatePK().toString(),
                nodeInputName: "input1",
                nodeName: "Node1",
                data: JSON.stringify({ value: "test" }),
                run: {
                    connect: { id: "run_id" },
                },
            },
            complete: {
                id: generatePK().toString(),
                nodeInputName: "complexInput",
                nodeName: "ProcessingNode",
                data: JSON.stringify({
                    type: "object",
                    value: {
                        string: "test string",
                        number: 42,
                        boolean: true,
                        array: [1, 2, 3],
                        nested: {
                            key: "value",
                            timestamp: new Date().toISOString(),
                        },
                    },
                    metadata: {
                        source: "user",
                        version: "1.0",
                        processed: true,
                    },
                }),
                run: {
                    connect: { id: "run_id" },
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, nodeInputName, nodeName, data, run
                    createdAt: new Date(),
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    nodeInputName: null, // Should be string
                    nodeName: 123, // Should be string
                    data: { object: "not-string" }, // Should be string
                    runId: "string-not-bigint", // Should be BigInt
                },
                tooLongNodeInputName: {
                    id: generatePK().toString(),
                    nodeInputName: "a".repeat(129), // Exceeds 128 character limit
                    nodeName: "Node",
                    data: "{}",
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                tooLongNodeName: {
                    id: generatePK().toString(),
                    nodeInputName: "input",
                    nodeName: "a".repeat(129), // Exceeds 128 character limit
                    data: "{}",
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                tooLongData: {
                    id: generatePK().toString(),
                    nodeInputName: "input",
                    nodeName: "Node",
                    data: JSON.stringify({ value: "a".repeat(8193) }), // Exceeds 8192 character limit
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                invalidJson: {
                    id: generatePK().toString(),
                    nodeInputName: "input",
                    nodeName: "Node",
                    data: "not-json{", // Invalid JSON
                    run: {
                        connect: { id: "run_id" },
                    },
                },
            },
            edgeCases: {
                emptyData: {
                    id: generatePK().toString(),
                    nodeInputName: "emptyInput",
                    nodeName: "EmptyNode",
                    data: "{}",
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                nullValueData: {
                    id: generatePK().toString(),
                    nodeInputName: "nullInput",
                    nodeName: "NullNode",
                    data: JSON.stringify({ value: null }),
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                arrayData: {
                    id: generatePK().toString(),
                    nodeInputName: "arrayInput",
                    nodeName: "ArrayNode",
                    data: JSON.stringify([1, 2, 3, 4, 5]),
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                booleanData: {
                    id: generatePK().toString(),
                    nodeInputName: "booleanInput",
                    nodeName: "BooleanNode",
                    data: JSON.stringify(true),
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                numberData: {
                    id: generatePK().toString(),
                    nodeInputName: "numberInput",
                    nodeName: "NumberNode",
                    data: JSON.stringify(3.14159),
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                stringData: {
                    id: generatePK().toString(),
                    nodeInputName: "stringInput",
                    nodeName: "StringNode",
                    data: JSON.stringify("Simple string value"),
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                largeData: {
                    id: generatePK().toString(),
                    nodeInputName: "largeInput",
                    nodeName: "LargeDataNode",
                    data: JSON.stringify({
                        items: Array.from({ length: 100 }, (_, i) => ({
                            id: i,
                            name: `Item ${i}`,
                            value: Math.random(),
                            timestamp: new Date().toISOString(),
                        })),
                    }),
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                specialCharactersData: {
                    id: generatePK().toString(),
                    nodeInputName: "specialInput",
                    nodeName: "SpecialNode",
                    data: JSON.stringify({
                        text: "Special characters: \"quotes\", 'apostrophes', \n newlines, \t tabs",
                        unicode: "Emoji: 😀 Unicode: 你好",
                        escaped: "Backslash: \\, Forward slash: /",
                    }),
                    run: {
                        connect: { id: "run_id" },
                    },
                },
            },
            updates: {
                minimal: {
                    data: JSON.stringify({ updated: true }),
                },
                complete: {
                    nodeInputName: "updatedInput",
                    nodeName: "UpdatedNode",
                    data: JSON.stringify({
                        value: "updated value",
                        timestamp: new Date().toISOString(),
                        version: 2,
                    }),
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.run_ioCreateInput>): Prisma.run_ioCreateInput {
        return {
            id: generatePK().toString(),
            nodeInputName: `input_${nanoid(6)}`,
            nodeName: `Node_${nanoid(6)}`,
            data: JSON.stringify({ value: "default" }),
            run: {
                connect: { id: "default_run_id" },
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.run_ioCreateInput>): Prisma.run_ioCreateInput {
        return {
            id: generatePK().toString(),
            nodeInputName: `complete_input_${nanoid(6)}`,
            nodeName: `CompleteNode_${nanoid(6)}`,
            data: JSON.stringify({
                type: "complete",
                value: {
                    string: "test data",
                    number: 123,
                    boolean: false,
                    array: ["a", "b", "c"],
                    object: {
                        nested: true,
                        level: 2,
                    },
                },
                metadata: {
                    created: new Date().toISOString(),
                    source: "test",
                },
            }),
            run: {
                connect: { id: "default_run_id" },
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            simpleInput: {
                name: "simpleInput",
                description: "Simple input data for a node",
                config: {
                    overrides: {
                        nodeInputName: "userInput",
                        nodeName: "InputNode",
                        data: JSON.stringify({ text: "User provided input" }),
                    },
                },
            },
            complexOutput: {
                name: "complexOutput",
                description: "Complex output data from processing",
                config: {
                    overrides: {
                        nodeInputName: "processedOutput",
                        nodeName: "OutputNode",
                        data: JSON.stringify({
                            result: "success",
                            data: {
                                processed: 100,
                                failed: 0,
                                summary: {
                                    total: 100,
                                    duration: 5432,
                                },
                            },
                            timestamp: new Date().toISOString(),
                        }),
                    },
                },
            },
            errorData: {
                name: "errorData",
                description: "Error information from failed node",
                config: {
                    overrides: {
                        nodeInputName: "errorOutput",
                        nodeName: "ErrorNode",
                        data: JSON.stringify({
                            error: true,
                            message: "Processing failed",
                            code: "PROC_ERR_001",
                            stack: "Error: Processing failed\n    at processData()",
                        }),
                    },
                },
            },
            transformationData: {
                name: "transformationData",
                description: "Data transformation between nodes",
                config: {
                    overrides: {
                        nodeInputName: "transformedData",
                        nodeName: "TransformNode",
                        data: JSON.stringify({
                            input: { format: "csv", rows: 1000 },
                            output: { format: "json", objects: 1000 },
                            transformation: "CSV to JSON",
                        }),
                    },
                },
            },
            apiResponseData: {
                name: "apiResponseData",
                description: "API response data",
                config: {
                    overrides: {
                        nodeInputName: "apiResponse",
                        nodeName: "APINode",
                        data: JSON.stringify({
                            status: 200,
                            headers: {
                                "content-type": "application/json",
                                "x-request-id": nanoid(),
                            },
                            body: {
                                success: true,
                                data: [1, 2, 3],
                            },
                        }),
                    },
                },
            },
        };
    }

    /**
     * Create specific IO types
     */
    async createInput(runId: string, nodeName: string, data: any): Promise<Prisma.run_io> {
        return await this.createMinimal({
            nodeInputName: "input",
            nodeName,
            data: JSON.stringify(data),
            run: { connect: { id: runId } },
        });
    }

    async createOutput(runId: string, nodeName: string, data: any): Promise<Prisma.run_io> {
        return await this.createMinimal({
            nodeInputName: "output",
            nodeName,
            data: JSON.stringify(data),
            run: { connect: { id: runId } },
        });
    }

    async createError(runId: string, nodeName: string, error: Error): Promise<Prisma.run_io> {
        return await this.createMinimal({
            nodeInputName: "error",
            nodeName,
            data: JSON.stringify({
                error: true,
                message: error.message,
                stack: error.stack,
                timestamp: new Date().toISOString(),
            }),
            run: { connect: { id: runId } },
        });
    }

    protected getDefaultInclude(): Prisma.run_ioInclude {
        return {
            run: {
                select: {
                    id: true,
                    name: true,
                    status: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.run_ioCreateInput,
        config: RunIORelationConfig,
        tx: any,
    ): Promise<Prisma.run_ioCreateInput> {
        const data = { ...baseData };

        // Handle run relationship
        if (config.withRun) {
            data.run = {
                connect: { id: config.withRun.runId },
            };
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.run_io): Promise<string[]> {
        const violations: string[] = [];
        
        // Check name lengths
        if (record.nodeInputName.length > 128) {
            violations.push("Node input name exceeds 128 character limit");
        }

        if (record.nodeName.length > 128) {
            violations.push("Node name exceeds 128 character limit");
        }

        // Check data length
        if (record.data.length > 8192) {
            violations.push("Data exceeds 8192 character limit");
        }

        // Check data validity
        try {
            JSON.parse(record.data);
        } catch {
            violations.push("Data must be valid JSON");
        }

        // Check run association
        if (!record.runId) {
            violations.push("RunIO must belong to a run");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {};
    }

    protected async deleteRelatedRecords(
        record: Prisma.run_io,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[],
    ): Promise<void> {
        // RunIO has no dependent records to delete
    }

    /**
     * Create multiple IO records for a run
     */
    async createRunIOSet(
        runId: string,
        ioData: Array<{ nodeInputName: string; nodeName: string; data: any }>,
    ): Promise<Prisma.run_io[]> {
        return await this.createMany(
            ioData.map(io => ({
                run: { connect: { id: runId } },
                nodeInputName: io.nodeInputName,
                nodeName: io.nodeName,
                data: JSON.stringify(io.data),
            })),
        );
    }

    /**
     * Create IO flow for a complete node execution
     */
    async createNodeExecutionIO(
        runId: string,
        nodeName: string,
        input: any,
        output: any,
    ): Promise<{ input: Prisma.run_io; output: Prisma.run_io }> {
        const [inputIO, outputIO] = await Promise.all([
            this.createInput(runId, nodeName, input),
            this.createOutput(runId, nodeName, output),
        ]);

        return { input: inputIO, output: outputIO };
    }
}

// Export factory creator function
export const createRunIODbFactory = (prisma: PrismaClient) => 
    RunIODbFactory.getInstance("RunIO", prisma);

// Export the class for type usage
export { RunIODbFactory as RunIODbFactoryClass };
