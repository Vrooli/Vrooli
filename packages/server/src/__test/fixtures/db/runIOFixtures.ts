import { type Prisma } from "@prisma/client";
import { generatePK } from "@vrooli/shared";

/**
 * Database fixtures for RunIO model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing - using hardcoded bigint values
export const runIODbIds = {
    runIO1: BigInt("123456789012345670"),
    runIO2: BigInt("123456789012345671"),
    runIO3: BigInt("123456789012345672"),
    runIO4: BigInt("123456789012345673"),
    runIO5: BigInt("123456789012345674"),
};

/**
 * Minimal RunIO data for database creation
 */
export const minimalRunIODb: Omit<Prisma.run_ioCreateInput, "run"> = {
    id: runIODbIds.runIO1,
    data: "test input data",
    nodeInputName: "input1",
    nodeName: "TestNode",
};

/**
 * Complete RunIO with all fields
 */
export const completeRunIODb: Omit<Prisma.run_ioCreateInput, "run"> = {
    id: runIODbIds.runIO2,
    data: JSON.stringify({
        value: "complex test data",
        type: "string",
        metadata: { source: "test" },
    }),
    nodeInputName: "complexInput",
    nodeName: "ComplexTestNode",
};

/**
 * RunIO with JSON data
 */
export const jsonRunIODb: Omit<Prisma.run_ioCreateInput, "run"> = {
    id: runIODbIds.runIO3,
    data: JSON.stringify({
        numbers: [1, 2, 3, 4, 5],
        nested: {
            property: "value",
            flag: true,
        },
    }),
    nodeInputName: "jsonInput",
    nodeName: "JsonNode",
};

/**
 * RunIO with large text data
 */
export const largeDataRunIODb: Omit<Prisma.run_ioCreateInput, "run"> = {
    id: runIODbIds.runIO4,
    data: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. ".repeat(50),
    nodeInputName: "textInput",
    nodeName: "TextProcessingNode",
};

/**
 * Factory for creating RunIO database fixtures with overrides
 */
export class RunIODbFactory {
    static createMinimal(
        runId: string,
        overrides?: Partial<Prisma.run_ioCreateInput>,
    ): Prisma.run_ioCreateInput {
        return {
            ...minimalRunIODb,
            id: generatePK(),
            run: { connect: { id: BigInt(runId) } },
            ...overrides,
        };
    }

    static createComplete(
        runId: string,
        overrides?: Partial<Prisma.run_ioCreateInput>,
    ): Prisma.run_ioCreateInput {
        return {
            ...completeRunIODb,
            id: generatePK(),
            run: { connect: { id: BigInt(runId) } },
            ...overrides,
        };
    }

    static createWithJsonData(
        runId: string,
        jsonData: any,
        overrides?: Partial<Prisma.run_ioCreateInput>,
    ): Prisma.run_ioCreateInput {
        return {
            ...jsonRunIODb,
            id: generatePK(),
            data: JSON.stringify(jsonData),
            run: { connect: { id: BigInt(runId) } },
            ...overrides,
        };
    }

    static createWithTextData(
        runId: string,
        textData: string,
        overrides?: Partial<Prisma.run_ioCreateInput>,
    ): Prisma.run_ioCreateInput {
        return {
            ...largeDataRunIODb,
            id: generatePK(),
            data: textData,
            run: { connect: { id: BigInt(runId) } },
            ...overrides,
        };
    }

    /**
     * Create RunIO for specific node and input combination
     */
    static createForNode(
        runId: string,
        nodeName: string,
        nodeInputName: string,
        data: any,
        overrides?: Partial<Prisma.run_ioCreateInput>,
    ): Prisma.run_ioCreateInput {
        return {
            id: generatePK(),
            data: typeof data === "string" ? data : JSON.stringify(data),
            nodeInputName,
            nodeName,
            run: { connect: { id: BigInt(runId) } },
            ...overrides,
        };
    }

    /**
     * Create multiple RunIO entries for a single run
     */
    static createMultiple(
        runId: string,
        ioData: Array<{
            nodeName: string;
            nodeInputName: string;
            data: any;
            overrides?: Partial<Prisma.run_ioCreateInput>;
        }>,
    ): Prisma.run_ioCreateInput[] {
        return ioData.map(item =>
            this.createForNode(
                runId,
                item.nodeName,
                item.nodeInputName,
                item.data,
                item.overrides,
            ),
        );
    }
}

/**
 * Helper to seed RunIO data for testing
 */
export async function seedRunIO(
    prisma: any,
    options: {
        runId: string;
        count?: number;
        withJsonData?: boolean;
        withLargeData?: boolean;
        customData?: Array<{
            nodeName: string;
            nodeInputName: string;
            data: any;
        }>;
    },
) {
    const runIOs = [];

    if (options.customData) {
        // Create custom RunIO entries
        for (const item of options.customData) {
            const runIO = await prisma.run_io.create({
                data: RunIODbFactory.createForNode(
                    options.runId,
                    item.nodeName,
                    item.nodeInputName,
                    item.data,
                ),
            });
            runIOs.push(runIO);
        }
    } else {
        // Create standard test RunIO entries
        const count = options.count || 3;

        for (let i = 0; i < count; i++) {
            let runIOData;

            if (options.withJsonData && i === 0) {
                runIOData = RunIODbFactory.createWithJsonData(
                    options.runId,
                    { testData: `value_${i}`, index: i },
                );
            } else if (options.withLargeData && i === 1) {
                runIOData = RunIODbFactory.createWithTextData(
                    options.runId,
                    `Large test data entry ${i}. `.repeat(100),
                );
            } else {
                runIOData = RunIODbFactory.createMinimal(options.runId, {
                    nodeInputName: `input${i}`,
                    nodeName: `Node${i}`,
                    data: `test data ${i}`,
                });
            }

            const runIO = await prisma.run_io.create({ data: runIOData });
            runIOs.push(runIO);
        }
    }

    return runIOs;
}

/**
 * Helper to create RunIO entries that simulate a routine execution flow
 */
export async function seedRoutineExecutionIO(
    prisma: any,
    runId: string,
) {
    const executionFlow = [
        {
            nodeName: "StartNode",
            nodeInputName: "initialInput",
            data: { message: "Starting routine execution" },
        },
        {
            nodeName: "DataProcessingNode",
            nodeInputName: "dataInput",
            data: {
                rawData: [1, 2, 3, 4, 5],
                processedData: [2, 4, 6, 8, 10],
            },
        },
        {
            nodeName: "ConditionalNode",
            nodeInputName: "conditionInput",
            data: { condition: true, result: "continue" },
        },
        {
            nodeName: "OutputNode",
            nodeInputName: "finalInput",
            data: {
                success: true,
                message: "Routine completed successfully",
                results: { count: 5, total: 30 },
            },
        },
    ];

    return seedRunIO(prisma, {
        runId,
        customData: executionFlow,
    });
}
