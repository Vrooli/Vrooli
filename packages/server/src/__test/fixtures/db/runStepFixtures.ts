// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { generatePK } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";

/**
 * Database fixtures for RunStep model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const runStepDbIds = {
    step1: generatePK(),
    step2: generatePK(),
    step3: generatePK(),
    step4: generatePK(),
    step5: generatePK(),
    run1: generatePK(),
    run2: generatePK(),
    run3: generatePK(),
    resourceVersion1: generatePK(),
    resourceVersion2: generatePK(),
};

/**
 * Minimal run step data for database creation
 */
export const minimalRunStepDb: Prisma.run_stepCreateInput = {
    id: runStepDbIds.step1,
    name: "Test Step",
    nodeId: "node_test_123",
    resourceInId: "resource_input_123",
    order: 1,
    complexity: 1,
    contextSwitches: 0,
    status: "InProgress",
    run: {
        connect: { id: runStepDbIds.run1 },
    },
};

/**
 * Run step with resource version connection
 */
export const runStepWithResourceVersionDb: Prisma.run_stepCreateInput = {
    id: runStepDbIds.step2,
    name: "Step with Resource Version",
    nodeId: "node_resource_456",
    resourceInId: "resource_input_456",
    order: 2,
    complexity: 3,
    contextSwitches: 1,
    status: "InProgress",
    startedAt: new Date(),
    run: {
        connect: { id: runStepDbIds.run2 },
    },
    resourceVersion: {
        connect: { id: runStepDbIds.resourceVersion1 },
    },
};

/**
 * Complete run step with all features
 */
export const completeRunStepDb: Prisma.run_stepCreateInput = {
    id: runStepDbIds.step3,
    name: "Complete Test Step",
    nodeId: "node_complete_789",
    resourceInId: "resource_input_789",
    order: 3,
    complexity: 5,
    contextSwitches: 3,
    status: "Completed",
    startedAt: new Date(Date.now() - 120000), // Started 2 minutes ago
    completedAt: new Date(Date.now() - 60000), // Completed 1 minute ago
    timeElapsed: 60000, // 1 minute elapsed
    run: {
        connect: { id: runStepDbIds.run3 },
    },
    resourceVersion: {
        connect: { id: runStepDbIds.resourceVersion2 },
    },
};

/**
 * Skipped run step
 */
export const skippedRunStepDb: Prisma.run_stepCreateInput = {
    id: runStepDbIds.step4,
    name: "Skipped Step",
    nodeId: "node_skipped_999",
    resourceInId: "resource_input_999",
    order: 4,
    complexity: 2,
    contextSwitches: 0,
    status: "Skipped",
    run: {
        connect: { id: runStepDbIds.run1 },
    },
};

/**
 * Long-running step with high complexity
 */
export const complexRunStepDb: Prisma.run_stepCreateInput = {
    id: runStepDbIds.step5,
    name: "Complex Processing Step",
    nodeId: "node_complex_888",
    resourceInId: "resource_input_888",
    order: 5,
    complexity: 10,
    contextSwitches: 7,
    status: "InProgress",
    startedAt: new Date(Date.now() - 300000), // Started 5 minutes ago
    timeElapsed: 240000, // 4 minutes elapsed so far
    run: {
        connect: { id: runStepDbIds.run2 },
    },
    resourceVersion: {
        connect: { id: runStepDbIds.resourceVersion1 },
    },
};

/**
 * Factory for creating run step database fixtures with overrides
 */
export class RunStepDbFactory {
    static createMinimal(
        runId: string,
        overrides?: Partial<Prisma.run_stepCreateInput>,
    ): Prisma.run_stepCreateInput {
        return {
            ...minimalRunStepDb,
            id: generatePK(),
            name: `Test Step ${Date.now()}`,
            nodeId: `node_${Date.now()}`,
            resourceInId: `resource_input_${Date.now()}`,
            run: {
                connect: { id: runId },
            },
            ...overrides,
        };
    }

    static createWithResourceVersion(
        runId: string,
        resourceVersionId: string,
        overrides?: Partial<Prisma.run_stepCreateInput>,
    ): Prisma.run_stepCreateInput {
        return {
            ...runStepWithResourceVersionDb,
            id: generatePK(),
            name: `Step with Resource ${Date.now()}`,
            nodeId: `node_resource_${Date.now()}`,
            resourceInId: `resource_input_${Date.now()}`,
            run: {
                connect: { id: runId },
            },
            resourceVersion: {
                connect: { id: resourceVersionId },
            },
            ...overrides,
        };
    }

    static createComplete(
        runId: string,
        overrides?: Partial<Prisma.run_stepCreateInput>,
    ): Prisma.run_stepCreateInput {
        return {
            ...completeRunStepDb,
            id: generatePK(),
            name: `Complete Step ${Date.now()}`,
            nodeId: `node_complete_${Date.now()}`,
            resourceInId: `resource_input_${Date.now()}`,
            run: {
                connect: { id: runId },
            },
            ...overrides,
        };
    }

    static createSkipped(
        runId: string,
        overrides?: Partial<Prisma.run_stepCreateInput>,
    ): Prisma.run_stepCreateInput {
        return {
            ...skippedRunStepDb,
            id: generatePK(),
            name: `Skipped Step ${Date.now()}`,
            nodeId: `node_skipped_${Date.now()}`,
            resourceInId: `resource_input_${Date.now()}`,
            run: {
                connect: { id: runId },
            },
            ...overrides,
        };
    }

    static createComplex(
        runId: string,
        overrides?: Partial<Prisma.run_stepCreateInput>,
    ): Prisma.run_stepCreateInput {
        return {
            ...complexRunStepDb,
            id: generatePK(),
            name: `Complex Step ${Date.now()}`,
            nodeId: `node_complex_${Date.now()}`,
            resourceInId: `resource_input_${Date.now()}`,
            run: {
                connect: { id: runId },
            },
            ...overrides,
        };
    }

    /**
     * Create run step with specific status
     */
    static createWithStatus(
        runId: string,
        status: "InProgress" | "Completed" | "Skipped",
        overrides?: Partial<Prisma.run_stepCreateInput>,
    ): Prisma.run_stepCreateInput {
        const now = new Date();
        const baseData: Partial<Prisma.run_stepCreateInput> = {
            status,
        };

        // Set timestamps based on status
        if (status === "InProgress") {
            baseData.startedAt = new Date(now.getTime() - 60000); // Started 1 minute ago
        } else if (status === "Completed") {
            baseData.startedAt = new Date(now.getTime() - 120000); // Started 2 minutes ago
            baseData.completedAt = new Date(now.getTime() - 60000); // Completed 1 minute ago
            baseData.timeElapsed = 60000; // 1 minute elapsed
        }

        return this.createMinimal(runId, {
            ...baseData,
            ...overrides,
        });
    }

    /**
     * Create run step with specific complexity and context switches
     */
    static createWithComplexity(
        runId: string,
        complexity: number,
        contextSwitches = 0,
        overrides?: Partial<Prisma.run_stepCreateInput>,
    ): Prisma.run_stepCreateInput {
        return this.createMinimal(runId, {
            complexity,
            contextSwitches,
            ...overrides,
        });
    }

    /**
     * Create run step with specific order
     */
    static createWithOrder(
        runId: string,
        order: number,
        overrides?: Partial<Prisma.run_stepCreateInput>,
    ): Prisma.run_stepCreateInput {
        return this.createMinimal(runId, {
            order,
            name: `Step ${order}`,
            ...overrides,
        });
    }

    /**
     * Create a sequence of run steps for a run
     */
    static createSequence(
        runId: string,
        stepCount = 3,
        options?: {
            allCompleted?: boolean;
            lastInProgress?: boolean;
            complexityRange?: [number, number];
        },
    ): Prisma.run_stepCreateInput[] {
        const {
            allCompleted = false,
            lastInProgress = false,
            complexityRange = [1, 5],
        } = options || {};

        return Array.from({ length: stepCount }, (_, index) => {
            const order = index + 1;
            const isLast = index === stepCount - 1;
            
            let status: "InProgress" | "Completed" | "Skipped" = "InProgress";
            if (allCompleted) {
                status = "Completed";
            } else if (lastInProgress && isLast) {
                status = "InProgress";
            } else if (index < stepCount - 1) {
                status = "Completed";
            }

            const complexity = Math.floor(
                Math.random() * (complexityRange[1] - complexityRange[0] + 1),
            ) + complexityRange[0];
            
            const contextSwitches = Math.floor(Math.random() * complexity);

            return this.createWithStatus(runId, status, {
                order,
                name: `Step ${order}`,
                nodeId: `node_${order}_${Date.now()}`,
                resourceInId: `resource_input_${order}_${Date.now()}`,
                complexity,
                contextSwitches,
            });
        });
    }
}

/**
 * Helper to seed run steps for testing
 */
export async function seedRunSteps(
    prisma: any,
    options: {
        runId: string;
        stepCount?: number;
        statuses?: Array<"InProgress" | "Completed" | "Skipped">;
        withResourceVersions?: boolean;
        resourceVersionIds?: string[];
        complexityRange?: [number, number];
    },
) {
    const {
        runId,
        stepCount = 3,
        statuses = ["Completed", "Completed", "InProgress"],
        withResourceVersions = false,
        resourceVersionIds = [],
        complexityRange = [1, 5],
    } = options;

    const steps = [];

    for (let i = 0; i < stepCount; i++) {
        const order = i + 1;
        const status = statuses[i % statuses.length];
        const complexity = Math.floor(
            Math.random() * (complexityRange[1] - complexityRange[0] + 1),
        ) + complexityRange[0];
        
        const stepData = RunStepDbFactory.createWithStatus(runId, status, {
            order,
            name: `Seeded Step ${order}`,
            nodeId: `seeded_node_${order}`,
            resourceInId: `seeded_resource_input_${order}`,
            complexity,
            contextSwitches: Math.floor(Math.random() * complexity),
        });

        // Add resource version if requested and available
        if (withResourceVersions && resourceVersionIds[i]) {
            stepData.resourceVersion = {
                connect: { id: resourceVersionIds[i] },
            };
        }

        const createdStep = await prisma.run_step.create({
            data: stepData,
            include: {
                run: true,
                resourceVersion: true,
            },
        });

        steps.push(createdStep);
    }

    return steps;
}

/**
 * Helper to create run step that matches the shape expected by API responses
 */
export function createApiResponseRunStep(overrides?: Partial<Prisma.run_stepCreateInput>) {
    const stepData = RunStepDbFactory.createComplete("run_123", overrides);
    return {
        ...stepData,
        run: undefined, // Remove relation for API response
        resourceVersion: undefined, // Remove relation for API response
    };
}

/**
 * Helper to clean up test run steps
 */
export async function cleanupRunSteps(prisma: PrismaClient, stepIds: string[]) {
    await prisma.run_step.deleteMany({
        where: { id: { in: stepIds } },
    });
}

/**
 * Helper to verify run step state
 */
export async function verifyRunStepState(
    prisma: any,
    stepId: string,
    expected: Partial<{
        status: "InProgress" | "Completed" | "Skipped";
        complexity: number;
        contextSwitches: number;
        timeElapsed: number | null;
        completedAt: Date | null;
    }>,
) {
    const actual = await prisma.run_step.findUnique({
        where: { id: stepId },
        include: {
            run: true,
            resourceVersion: true,
        },
    });

    if (!actual) {
        throw new Error(`RunStep with id ${stepId} not found`);
    }

    // Check each expected field
    Object.entries(expected).forEach(([key, value]) => {
        if (key === "completedAt" && value === null) {
            expect(actual[key]).toBeNull();
        } else if (key === "completedAt" && value instanceof Date) {
            expect(actual[key]).toBeInstanceOf(Date);
        } else {
            expect(actual[key]).toBe(value);
        }
    });

    return actual;
}
