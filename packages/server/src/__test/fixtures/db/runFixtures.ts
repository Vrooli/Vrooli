import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for Run model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const runDbIds = {
    run1: generatePK(),
    run2: generatePK(),
    run3: generatePK(),
    run4: generatePK(),
    schedule1: generatePK(),
    schedule2: generatePK(),
    resourceVersion1: generatePK(),
    resourceVersion2: generatePK(),
    team1: generatePK(),
    user1: generatePK(),
    user2: generatePK(),
    step1: generatePK(),
    step2: generatePK(),
    step3: generatePK(),
    io1: generatePK(),
    io2: generatePK(),
};

/**
 * Minimal run data for database creation
 */
export const minimalRunDb: Prisma.RunCreateInput = {
    id: runDbIds.run1,
    name: "Test Run",
    status: "Scheduled",
    isPrivate: false,
    completedComplexity: 0,
    contextSwitches: 0,
    wasRunAutomatically: false,
};

/**
 * Run with resource version connection
 */
export const runWithResourceVersionDb: Prisma.RunCreateInput = {
    id: runDbIds.run2,
    name: "Run with Resource Version",
    status: "InProgress",
    isPrivate: false,
    completedComplexity: 5,
    contextSwitches: 2,
    wasRunAutomatically: false,
    startedAt: new Date(),
    data: JSON.stringify({ input: "test data" }),
    resourceVersion: {
        connect: { id: runDbIds.resourceVersion1 },
    },
};

/**
 * Complete run with all features
 */
export const completeRunDb: Prisma.RunCreateInput = {
    id: runDbIds.run3,
    name: "Complete Test Run",
    status: "Completed",
    isPrivate: false,
    completedComplexity: 10,
    contextSwitches: 5,
    wasRunAutomatically: true,
    startedAt: new Date(Date.now() - 300000), // 5 minutes ago
    completedAt: new Date(Date.now() - 60000), // 1 minute ago
    timeElapsed: 240000, // 4 minutes
    data: JSON.stringify({ 
        input: "complex test data",
        parameters: { verbose: true },
        result: "success",
    }),
    resourceVersion: {
        connect: { id: runDbIds.resourceVersion2 },
    },
    user: {
        connect: { id: runDbIds.user1 },
    },
    team: {
        connect: { id: runDbIds.team1 },
    },
    schedule: {
        connect: { id: runDbIds.schedule1 },
    },
    steps: {
        create: [
            {
                id: runDbIds.step1,
                name: "Initialize",
                nodeId: "node_1",
                status: "Completed",
                order: 1,
                complexity: 3,
                contextSwitches: 1,
                timeElapsed: 60000,
                completedAt: new Date(Date.now() - 240000),
            },
            {
                id: runDbIds.step2,
                name: "Process",
                nodeId: "node_2", 
                status: "Completed",
                order: 2,
                complexity: 5,
                contextSwitches: 3,
                timeElapsed: 120000,
                completedAt: new Date(Date.now() - 120000),
            },
            {
                id: runDbIds.step3,
                name: "Finalize",
                nodeId: "node_3",
                status: "Completed", 
                order: 3,
                complexity: 2,
                contextSwitches: 1,
                timeElapsed: 60000,
                completedAt: new Date(Date.now() - 60000),
            },
        ],
    },
    io: {
        create: [
            {
                id: runDbIds.io1,
                nodeName: "node_1",
                nodeInputName: "input",
                data: JSON.stringify({ value: "test input" }),
            },
            {
                id: runDbIds.io2,
                nodeName: "node_3",
                nodeInputName: "output",
                data: JSON.stringify({ result: "test output" }),
            },
        ],
    },
};

/**
 * Failed run
 */
export const failedRunDb: Prisma.RunCreateInput = {
    id: runDbIds.run4,
    name: "Failed Test Run",
    status: "Failed",
    isPrivate: false,
    completedComplexity: 3,
    contextSwitches: 1,
    wasRunAutomatically: false,
    startedAt: new Date(Date.now() - 180000), // 3 minutes ago
    completedAt: new Date(Date.now() - 120000), // 2 minutes ago
    timeElapsed: 60000, // 1 minute
    data: JSON.stringify({ 
        input: "test data",
        error: "Process failed due to invalid input",
    }),
    user: {
        connect: { id: runDbIds.user2 },
    },
};

/**
 * Factory for creating run database fixtures with overrides
 */
export class RunDbFactory {
    static createMinimal(overrides?: Partial<Prisma.RunCreateInput>): Prisma.RunCreateInput {
        return {
            ...minimalRunDb,
            id: generatePK(),
            name: `Test Run ${Date.now()}`,
            ...overrides,
        };
    }

    static createWithResourceVersion(
        resourceVersionId: string,
        overrides?: Partial<Prisma.RunCreateInput>,
    ): Prisma.RunCreateInput {
        return {
            ...runWithResourceVersionDb,
            id: generatePK(),
            name: `Run with Resource ${Date.now()}`,
            resourceVersion: {
                connect: { id: resourceVersionId },
            },
            ...overrides,
        };
    }

    static createComplete(overrides?: Partial<Prisma.RunCreateInput>): Prisma.RunCreateInput {
        return {
            ...completeRunDb,
            id: generatePK(),
            name: `Complete Run ${Date.now()}`,
            steps: {
                create: [
                    {
                        id: generatePK(),
                        name: "Initialize",
                        nodeId: "node_1",
                        status: "Completed",
                        order: 1,
                        complexity: 3,
                        contextSwitches: 1,
                        timeElapsed: 60000,
                        completedAt: new Date(Date.now() - 240000),
                    },
                    {
                        id: generatePK(),
                        name: "Process",
                        nodeId: "node_2",
                        status: "Completed",
                        order: 2,
                        complexity: 5,
                        contextSwitches: 3,
                        timeElapsed: 120000,
                        completedAt: new Date(Date.now() - 120000),
                    },
                    {
                        id: generatePK(),
                        name: "Finalize",
                        nodeId: "node_3",
                        status: "Completed",
                        order: 3,
                        complexity: 2,
                        contextSwitches: 1,
                        timeElapsed: 60000,
                        completedAt: new Date(Date.now() - 60000),
                    },
                ],
            },
            io: {
                create: [
                    {
                        id: generatePK(),
                        nodeName: "node_1",
                        nodeInputName: "input",
                        data: JSON.stringify({ value: "test input" }),
                    },
                    {
                        id: generatePK(),
                        nodeName: "node_3",
                        nodeInputName: "output",
                        data: JSON.stringify({ result: "test output" }),
                    },
                ],
            },
            ...overrides,
        };
    }

    static createFailed(overrides?: Partial<Prisma.RunCreateInput>): Prisma.RunCreateInput {
        return {
            ...failedRunDb,
            id: generatePK(),
            name: `Failed Run ${Date.now()}`,
            ...overrides,
        };
    }

    /**
     * Create run with specific status
     */
    static createWithStatus(
        status: "Scheduled" | "InProgress" | "Paused" | "Completed" | "Failed" | "Cancelled",
        overrides?: Partial<Prisma.RunCreateInput>,
    ): Prisma.RunCreateInput {
        const now = new Date();
        const baseData: Partial<Prisma.RunCreateInput> = {
            status,
        };

        // Set timestamps based on status
        if (status === "InProgress") {
            baseData.startedAt = new Date(now.getTime() - 60000); // Started 1 minute ago
        } else if (status === "Completed") {
            baseData.startedAt = new Date(now.getTime() - 300000); // Started 5 minutes ago
            baseData.completedAt = new Date(now.getTime() - 60000); // Completed 1 minute ago
            baseData.timeElapsed = 240000; // 4 minutes elapsed
        } else if (status === "Failed" || status === "Cancelled") {
            baseData.startedAt = new Date(now.getTime() - 180000); // Started 3 minutes ago
            baseData.completedAt = new Date(now.getTime() - 120000); // Failed/cancelled 2 minutes ago
            baseData.timeElapsed = 60000; // 1 minute elapsed
        }

        return this.createMinimal({
            ...baseData,
            ...overrides,
        });
    }

    /**
     * Create run with user and team
     */
    static createWithOwnership(
        userId: string,
        teamId?: string,
        overrides?: Partial<Prisma.RunCreateInput>,
    ): Prisma.RunCreateInput {
        return {
            ...this.createMinimal(overrides),
            user: { connect: { id: userId } },
            ...(teamId && { team: { connect: { id: teamId } } }),
        };
    }

    /**
     * Create run with steps
     */
    static createWithSteps(
        stepCount = 3,
        overrides?: Partial<Prisma.RunCreateInput>,
    ): Prisma.RunCreateInput {
        const steps = Array.from({ length: stepCount }, (_, index) => ({
            id: generatePK(),
            name: `Step ${index + 1}`,
            nodeId: `node_${index + 1}`,
            status: "Completed" as const,
            order: index + 1,
            complexity: Math.floor(Math.random() * 5) + 1,
            contextSwitches: Math.floor(Math.random() * 3),
            timeElapsed: (Math.floor(Math.random() * 60) + 30) * 1000, // 30-90 seconds
            completedAt: new Date(Date.now() - (stepCount - index) * 60000),
        }));

        return {
            ...this.createMinimal(overrides),
            steps: { create: steps },
        };
    }

    /**
     * Create run with I/O data
     */
    static createWithIO(
        ioData: Array<{ nodeName: string; nodeInputName: string; data: any }>,
        overrides?: Partial<Prisma.RunCreateInput>,
    ): Prisma.RunCreateInput {
        const io = ioData.map(item => ({
            id: generatePK(),
            nodeName: item.nodeName,
            nodeInputName: item.nodeInputName,
            data: typeof item.data === "string" ? item.data : JSON.stringify(item.data),
        }));

        return {
            ...this.createMinimal(overrides),
            io: { create: io },
        };
    }

    /**
     * Create scheduled run
     */
    static createScheduled(
        scheduleId: string,
        overrides?: Partial<Prisma.RunCreateInput>,
    ): Prisma.RunCreateInput {
        return {
            ...this.createMinimal(overrides),
            status: "Scheduled",
            wasRunAutomatically: true,
            schedule: { connect: { id: scheduleId } },
        };
    }
}

/**
 * Helper to seed runs for testing
 */
export async function seedRuns(
    prisma: any,
    options: {
        count?: number;
        userId?: string;
        teamId?: string;
        resourceVersionId?: string;
        statuses?: Array<"Scheduled" | "InProgress" | "Completed" | "Failed" | "Cancelled">;
        withSteps?: boolean;
        withIO?: boolean;
    } = {},
) {
    const {
        count = 3,
        userId,
        teamId,
        resourceVersionId,
        statuses = ["Scheduled", "InProgress", "Completed"],
        withSteps = false,
        withIO = false,
    } = options;

    const runs = [];

    for (let i = 0; i < count; i++) {
        const status = statuses[i % statuses.length];
        const runData = RunDbFactory.createWithStatus(status, {
            name: `Seeded Run ${i + 1}`,
        });

        // Add ownership if provided
        if (userId) {
            runData.user = { connect: { id: userId } };
        }
        if (teamId) {
            runData.team = { connect: { id: teamId } };
        }
        if (resourceVersionId) {
            runData.resourceVersion = { connect: { id: resourceVersionId } };
        }

        // Add steps if requested
        if (withSteps) {
            const stepCount = Math.floor(Math.random() * 3) + 2; // 2-4 steps
            runData.steps = {
                create: Array.from({ length: stepCount }, (_, stepIndex) => ({
                    id: generatePK(),
                    name: `Step ${stepIndex + 1}`,
                    nodeId: `node_${stepIndex + 1}`,
                    status: status === "Completed" ? "Completed" : "InProgress",
                    order: stepIndex + 1,
                    complexity: Math.floor(Math.random() * 5) + 1,
                    contextSwitches: Math.floor(Math.random() * 3),
                    timeElapsed: status === "Completed" ? (Math.floor(Math.random() * 60) + 30) * 1000 : null,
                    completedAt: status === "Completed" ? new Date(Date.now() - (stepCount - stepIndex) * 60000) : null,
                })),
            };
        }

        // Add I/O if requested
        if (withIO) {
            runData.io = {
                create: [
                    {
                        id: generatePK(),
                        nodeName: "input_node",
                        nodeInputName: "input",
                        data: JSON.stringify({ value: `input_${i + 1}`, timestamp: new Date().toISOString() }),
                    },
                    {
                        id: generatePK(),
                        nodeName: "output_node", 
                        nodeInputName: "output",
                        data: JSON.stringify({ result: `output_${i + 1}`, success: status === "Completed" }),
                    },
                ],
            };
        }

        const createdRun = await prisma.run.create({
            data: runData,
            include: {
                steps: true,
                io: true,
                user: true,
                team: true,
                resourceVersion: true,
                schedule: true,
            },
        });

        runs.push(createdRun);
    }

    return runs;
}

/**
 * Helper to create run that matches the shape expected by API responses
 */
export function createApiResponseRun(overrides?: Partial<Prisma.RunCreateInput>) {
    const runData = RunDbFactory.createComplete(overrides);
    return {
        ...runData,
        you: {
            canRead: true,
            canUpdate: true,
            canDelete: true,
        },
        ioCount: 2,
        stepsCount: 3,
    };
}

/**
 * Helper to clean up test runs
 */
export async function cleanupRuns(prisma: any, runIds: string[]) {
    // Delete in correct order for foreign key constraints
    await prisma.runIO.deleteMany({
        where: { runId: { in: runIds } },
    });
    
    await prisma.runStep.deleteMany({
        where: { runId: { in: runIds } },
    });
    
    await prisma.run.deleteMany({
        where: { id: { in: runIds } },
    });
}
