// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient, RunStepStatus } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";

interface RunStepRelationConfig extends RelationConfig {
    withRun?: { runId: string };
    withResourceVersion?: { resourceVersionId: string };
}

/**
 * Enhanced database fixture factory for RunStep model
 * Provides comprehensive testing capabilities for run execution steps
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for various step statuses
 * - Execution timing tracking
 * - Complexity and context switch metrics
 * - Resource version association
 * - Step ordering management
 * - Predefined test scenarios
 */
export class RunStepDbFactory extends EnhancedDatabaseFactory<
    Prisma.run_stepCreateInput,
    Prisma.run_stepCreateInput,
    Prisma.run_stepInclude,
    Prisma.run_stepUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    constructor(prisma: PrismaClient) {
        super("RunStep", prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.run_step;
    }

    /**
     * Get complete test fixtures for RunStep model
     */
    protected getFixtures(): DbTestFixtures<Prisma.run_stepCreateInput, Prisma.run_stepUpdateInput> {
        const now = new Date();
        const oneMinuteAgo = new Date(now.getTime() - 60 * 1000);
        const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000);
        const tenMinutesAgo = new Date(now.getTime() - 10 * 60 * 1000);

        return {
            minimal: {
                id: this.generateId(),
                name: "Test Step",
                nodeId: nanoid(),
                resourceInId: nanoid(),
                order: 0,
                status: RunStepStatus.InProgress,
                complexity: 10,
                contextSwitches: 0,
                run: {
                    connect: { id: "run_id" },
                },
            },
            complete: {
                id: this.generateId(),
                name: "Complete Test Step",
                nodeId: nanoid(),
                resourceInId: nanoid(),
                order: 1,
                status: RunStepStatus.Completed,
                complexity: 25,
                contextSwitches: 3,
                startedAt: fiveMinutesAgo,
                completedAt: oneMinuteAgo,
                timeElapsed: 240000, // 4 minutes in milliseconds
                run: {
                    connect: { id: "run_id" },
                },
                resourceVersion: {
                    connect: { id: "resource_version_id" },
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, name, nodeId, resourceInId, order, status, run
                    complexity: 0,
                    contextSwitches: 0,
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    name: null, // Should be string
                    nodeId: 123, // Should be string UUID
                    resourceInId: true, // Should be string UUID
                    order: "1", // Should be number
                    status: "InProgress", // Should be enum value
                    complexity: "10", // Should be number
                    contextSwitches: -1, // Should be non-negative
                    runId: "string-not-bigint", // Should be BigInt
                },
                tooLongName: {
                    id: this.generateId(),
                    name: "a".repeat(129), // Exceeds 128 character limit
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 0,
                    status: RunStepStatus.InProgress,
                    complexity: 0,
                    contextSwitches: 0,
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                invalidTimeRange: {
                    id: this.generateId(),
                    name: "Invalid Time Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 0,
                    status: RunStepStatus.Completed,
                    complexity: 0,
                    contextSwitches: 0,
                    startedAt: now,
                    completedAt: oneMinuteAgo, // Completed before started
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                invalidUuid: {
                    id: this.generateId(),
                    name: "Invalid UUID Step",
                    nodeId: "not-a-uuid",
                    resourceInId: "also-not-a-uuid",
                    order: 0,
                    status: RunStepStatus.InProgress,
                    complexity: 0,
                    contextSwitches: 0,
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                negativeOrder: {
                    id: this.generateId(),
                    name: "Negative Order Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: -1, // Should be non-negative
                    status: RunStepStatus.InProgress,
                    complexity: 0,
                    contextSwitches: 0,
                    run: {
                        connect: { id: "run_id" },
                    },
                },
            },
            edgeCases: {
                pendingStep: {
                    id: this.generateId(),
                    name: "Pending Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 0,
                    status: RunStepStatus.Pending,
                    complexity: 0,
                    contextSwitches: 0,
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                inProgressStep: {
                    id: this.generateId(),
                    name: "In Progress Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 1,
                    status: RunStepStatus.InProgress,
                    complexity: 15,
                    contextSwitches: 2,
                    startedAt: fiveMinutesAgo,
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                completedStep: {
                    id: this.generateId(),
                    name: "Completed Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 2,
                    status: RunStepStatus.Completed,
                    complexity: 20,
                    contextSwitches: 1,
                    startedAt: tenMinutesAgo,
                    completedAt: fiveMinutesAgo,
                    timeElapsed: 300000, // 5 minutes
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                failedStep: {
                    id: this.generateId(),
                    name: "Failed Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 3,
                    status: RunStepStatus.Failed,
                    complexity: 5,
                    contextSwitches: 5,
                    startedAt: tenMinutesAgo,
                    completedAt: fiveMinutesAgo,
                    timeElapsed: 300000,
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                skippedStep: {
                    id: this.generateId(),
                    name: "Skipped Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 4,
                    status: RunStepStatus.Skipped,
                    complexity: 0,
                    contextSwitches: 0,
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                highComplexityStep: {
                    id: this.generateId(),
                    name: "High Complexity Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 5,
                    status: RunStepStatus.Completed,
                    complexity: 100,
                    contextSwitches: 15,
                    startedAt: new Date(now.getTime() - 60 * 60 * 1000), // 1 hour ago
                    completedAt: new Date(now.getTime() - 30 * 60 * 1000), // 30 minutes ago
                    timeElapsed: 1800000, // 30 minutes
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                quickStep: {
                    id: this.generateId(),
                    name: "Quick Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 6,
                    status: RunStepStatus.Completed,
                    complexity: 1,
                    contextSwitches: 0,
                    startedAt: new Date(now.getTime() - 5000), // 5 seconds ago
                    completedAt: new Date(now.getTime() - 1000), // 1 second ago
                    timeElapsed: 4000, // 4 seconds
                    run: {
                        connect: { id: "run_id" },
                    },
                },
                highOrderStep: {
                    id: this.generateId(),
                    name: "High Order Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 999,
                    status: RunStepStatus.Pending,
                    complexity: 0,
                    contextSwitches: 0,
                    run: {
                        connect: { id: "run_id" },
                    },
                },
            },
            updates: {
                minimal: {
                    status: RunStepStatus.Completed,
                    completedAt: now,
                },
                complete: {
                    status: RunStepStatus.Completed,
                    complexity: 50,
                    contextSwitches: 8,
                    completedAt: now,
                    timeElapsed: 600000, // 10 minutes
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.run_stepCreateInput>): Prisma.run_stepCreateInput {
        return {
            id: this.generateId(),
            name: `Step_${nanoid()}`,
            nodeId: nanoid(),
            resourceInId: nanoid(),
            order: 0,
            status: RunStepStatus.Pending,
            complexity: 0,
            contextSwitches: 0,
            run: {
                connect: { id: "default_run_id" },
            },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.run_stepCreateInput>): Prisma.run_stepCreateInput {
        const now = new Date();
        const fiveMinutesAgo = new Date(now.getTime() - 5 * 60 * 1000);
        
        return {
            id: this.generateId(),
            name: `Complete_Step_${nanoid()}`,
            nodeId: nanoid(),
            resourceInId: nanoid(),
            order: 0,
            status: RunStepStatus.Completed,
            complexity: 25,
            contextSwitches: 3,
            startedAt: fiveMinutesAgo,
            completedAt: now,
            timeElapsed: 300000, // 5 minutes
            run: {
                connect: { id: "default_run_id" },
            },
            resourceVersion: {
                connect: { id: "default_resource_version_id" },
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            simpleStep: {
                name: "simpleStep",
                description: "Simple step with basic execution",
                config: {
                    overrides: {
                        name: "Simple Processing Step",
                        complexity: 10,
                        status: RunStepStatus.Completed,
                        startedAt: new Date(Date.now() - 60000),
                        completedAt: new Date(),
                        timeElapsed: 60000,
                    },
                },
            },
            complexStep: {
                name: "complexStep",
                description: "Complex step with high complexity and context switches",
                config: {
                    overrides: {
                        name: "Complex Algorithm Step",
                        complexity: 75,
                        contextSwitches: 10,
                        status: RunStepStatus.InProgress,
                        startedAt: new Date(Date.now() - 600000), // 10 minutes ago
                    },
                    withResourceVersion: { resourceVersionId: "resource_version_id" },
                },
            },
            failedStep: {
                name: "failedStep",
                description: "Step that failed during execution",
                config: {
                    overrides: {
                        name: "Failed Validation Step",
                        complexity: 30,
                        contextSwitches: 5,
                        status: RunStepStatus.Failed,
                        startedAt: new Date(Date.now() - 120000), // 2 minutes ago
                        completedAt: new Date(Date.now() - 60000), // 1 minute ago
                        timeElapsed: 60000,
                    },
                },
            },
            longRunningStep: {
                name: "longRunningStep",
                description: "Long running step still in progress",
                config: {
                    overrides: {
                        name: "Long Data Processing",
                        complexity: 150,
                        contextSwitches: 20,
                        status: RunStepStatus.InProgress,
                        startedAt: new Date(Date.now() - 3600000), // 1 hour ago
                    },
                },
            },
            quickStep: {
                name: "quickStep",
                description: "Quick step that completed rapidly",
                config: {
                    overrides: {
                        name: "Quick Validation",
                        complexity: 1,
                        contextSwitches: 0,
                        status: RunStepStatus.Completed,
                        startedAt: new Date(Date.now() - 5000), // 5 seconds ago
                        completedAt: new Date(Date.now() - 1000), // 1 second ago
                        timeElapsed: 4000,
                    },
                },
            },
        };
    }

    /**
     * Create specific step types
     */
    async createPendingStep(runId: string, order: number): Promise<Prisma.run_step> {
        return await this.createMinimal({
            order,
            status: RunStepStatus.Pending,
            run: { connect: { id: runId } },
        });
    }

    async createInProgressStep(runId: string, order: number): Promise<Prisma.run_step> {
        return await this.createMinimal({
            order,
            status: RunStepStatus.InProgress,
            startedAt: new Date(),
            run: { connect: { id: runId } },
        });
    }

    async createCompletedStep(
        runId: string, 
        order: number, 
        complexity = 10,
    ): Promise<Prisma.run_step> {
        const now = new Date();
        const startTime = new Date(now.getTime() - complexity * 10000); // 10s per complexity point
        
        return await this.createMinimal({
            order,
            status: RunStepStatus.Completed,
            complexity,
            startedAt: startTime,
            completedAt: now,
            timeElapsed: now.getTime() - startTime.getTime(),
            run: { connect: { id: runId } },
        });
    }

    async createFailedStep(runId: string, order: number): Promise<Prisma.run_step> {
        const now = new Date();
        const startTime = new Date(now.getTime() - 60000); // 1 minute ago
        
        return await this.createMinimal({
            order,
            status: RunStepStatus.Failed,
            startedAt: startTime,
            completedAt: now,
            timeElapsed: 60000,
            run: { connect: { id: runId } },
        });
    }

    protected getDefaultInclude(): Prisma.run_stepInclude {
        return {
            run: {
                select: {
                    id: true,
                    name: true,
                    status: true,
                },
            },
            resourceVersion: {
                select: {
                    id: true,
                    publicId: true,
                    versionString: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.run_stepCreateInput,
        config: RunStepRelationConfig,
        tx: PrismaClient,
    ): Promise<Prisma.run_stepCreateInput> {
        const data = { ...baseData };

        // Handle run relationship
        if (config.withRun) {
            data.run = {
                connect: { id: config.withRun.runId },
            };
        }

        // Handle resource version relationship
        if (config.withResourceVersion) {
            data.resourceVersion = {
                connect: { id: config.withResourceVersion.resourceVersionId },
            };
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.run_step): Promise<string[]> {
        const violations: string[] = [];
        
        // Check name length
        if (record.name.length > 128) {
            violations.push("Step name exceeds 128 character limit");
        }

        // Check order
        if (record.order < 0) {
            violations.push("Step order must be non-negative");
        }

        // Check complexity
        if (record.complexity < 0) {
            violations.push("Step complexity must be non-negative");
        }

        // Check context switches
        if (record.contextSwitches < 0) {
            violations.push("Context switches must be non-negative");
        }

        // Check time constraints
        if (record.startedAt && record.completedAt && record.startedAt >= record.completedAt) {
            violations.push("Start time must be before completion time");
        }

        // Check status consistency
        if (record.status === RunStepStatus.Completed && !record.completedAt) {
            violations.push("Completed steps must have completion time");
        }

        if (record.status === RunStepStatus.Failed && !record.completedAt) {
            violations.push("Failed steps must have completion time");
        }

        if (record.status === RunStepStatus.InProgress && !record.startedAt) {
            violations.push("In-progress steps must have start time");
        }

        if (record.status === RunStepStatus.Pending && record.startedAt) {
            violations.push("Pending steps should not have start time");
        }

        // Check time elapsed consistency
        if (record.timeElapsed && record.startedAt && record.completedAt) {
            const calculatedElapsed = record.completedAt.getTime() - record.startedAt.getTime();
            if (Math.abs(record.timeElapsed - calculatedElapsed) > 1000) { // Allow 1s tolerance
                violations.push("Time elapsed does not match start and completion times");
            }
        }

        // Check UUID format
        const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;
        if (!uuidRegex.test(record.nodeId)) {
            violations.push("Node ID must be a valid UUID");
        }

        if (!uuidRegex.test(record.resourceInId)) {
            violations.push("Resource In ID must be a valid UUID");
        }

        // Check run association
        if (!record.runId) {
            violations.push("RunStep must belong to a run");
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {};
    }

    protected async deleteRelatedRecords(
        record: Prisma.run_step,
        remainingDepth: number,
        tx: PrismaClient,
        includeOnly?: string[],
    ): Promise<void> {
        // RunStep has no dependent records to delete
    }

    /**
     * Create a sequence of steps for a run
     */
    async createStepSequence(
        runId: string,
        stepConfigs: Array<{
            name: string;
            status: RunStepStatus;
            complexity?: number;
            contextSwitches?: number;
        }>,
    ): Promise<Prisma.run_step[]> {
        const now = new Date();
        
        return await this.createMany(
            stepConfigs.map((config, i) => {
                const startTime = new Date(now.getTime() - (stepConfigs.length - i) * 60000);
                const endTime = config.status === RunStepStatus.Completed || config.status === RunStepStatus.Failed
                    ? new Date(startTime.getTime() + 30000)
                    : undefined;
                
                return {
                    name: config.name,
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: i,
                    status: config.status,
                    complexity: config.complexity || 10,
                    contextSwitches: config.contextSwitches || 0,
                    startedAt: config.status !== RunStepStatus.Pending ? startTime : undefined,
                    completedAt: endTime,
                    timeElapsed: endTime ? endTime.getTime() - startTime.getTime() : undefined,
                    run: { connect: { id: runId } },
                };
            }),
        );
    }

    /**
     * Create test steps for various scenarios
     */
    async createTestSteps(runId: string): Promise<{
        pending: Prisma.run_step;
        inProgress: Prisma.run_step;
        completed: Prisma.run_step;
        failed: Prisma.run_step;
        skipped: Prisma.run_step;
    }> {
        const [pending, inProgress, completed, failed, skipped] = await Promise.all([
            this.createPendingStep(runId, 0),
            this.createInProgressStep(runId, 1),
            this.createCompletedStep(runId, 2),
            this.createFailedStep(runId, 3),
            this.createMinimal({
                order: 4,
                status: RunStepStatus.Skipped,
                run: { connect: { id: runId } },
            }),
        ]);

        return {
            pending,
            inProgress,
            completed,
            failed,
            skipped,
        };
    }

    /**
     * Create a step with specific timing
     */
    async createTimedStep(
        runId: string,
        order: number,
        duration: number, // in milliseconds
        status: RunStepStatus = RunStepStatus.Completed,
    ): Promise<Prisma.run_step> {
        const endTime = new Date();
        const startTime = new Date(endTime.getTime() - duration);
        
        return await this.createMinimal({
            order,
            status,
            startedAt: startTime,
            completedAt: status === RunStepStatus.Completed || status === RunStepStatus.Failed ? endTime : undefined,
            timeElapsed: status === RunStepStatus.Completed || status === RunStepStatus.Failed ? duration : undefined,
            run: { connect: { id: runId } },
        });
    }
}

// Export factory creator function
export const createRunStepDbFactory = (prisma: PrismaClient) => 
    new RunStepDbFactory(prisma);

// Export the class for type usage
export { RunStepDbFactory as RunStepDbFactoryClass };
