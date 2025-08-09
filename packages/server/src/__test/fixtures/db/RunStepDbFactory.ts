/* eslint-disable no-magic-numbers */
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { type Prisma, type PrismaClient, RunStepStatus, type run_step } from "@prisma/client";
import { nanoid } from "@vrooli/shared";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type {
    DbTestFixtures,
    RelationConfig,
    TestScenario,
} from "./types.js";

interface RunStepRelationConfig extends RelationConfig {
    withRun?: { runId: bigint };
    withResourceVersion?: { resourceVersionId: bigint };
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
    run_step,
    Prisma.run_stepCreateInput,
    Prisma.run_stepInclude,
    Prisma.run_stepUpdateInput
> {
    protected scenarios: Record<string, TestScenario> = {};
    // Store commonly used IDs for fixtures
    private fixtureRunId: bigint;
    private defaultRunId: bigint;
    private fixtureResourceVersionId: bigint;
    private defaultResourceVersionId: bigint;

    constructor(prisma: PrismaClient) {
        super("RunStep", prisma);
        // Generate valid IDs that can be reused in fixtures
        this.fixtureRunId = this.generateId();
        this.defaultRunId = this.generateId();
        this.fixtureResourceVersionId = this.generateId();
        this.defaultResourceVersionId = this.generateId();
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
                    connect: { id: this.fixtureRunId },
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
                    connect: { id: this.fixtureRunId },
                },
                resourceVersion: {
                    connect: { id: this.fixtureResourceVersionId },
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, name, nodeId, resourceInId, order, status, run
                    complexity: 0,
                    contextSwitches: 0,
                } as any,
                invalidTypes: {
                    id: BigInt(123456789), // Correct: using BigInt
                    name: null as any, // Should be string
                    nodeId: 123 as any, // Should be string UUID
                    resourceInId: true as any, // Should be string UUID
                    order: "1" as any, // Should be number
                    status: "InProgress" as any, // Should be enum value
                    complexity: "10" as any, // Should be number
                    contextSwitches: -1, // Should be non-negative
                    runId: BigInt(123456789), // Correct: using BigInt
                } as any,
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
                        connect: { id: this.fixtureRunId },
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
                        connect: { id: this.fixtureRunId },
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
                        connect: { id: this.fixtureRunId },
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
                        connect: { id: this.fixtureRunId },
                    },
                },
            },
            edgeCases: {
                notStartedStep: {
                    id: this.generateId(),
                    name: "Not Started Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 0,
                    status: RunStepStatus.InProgress,
                    complexity: 0,
                    contextSwitches: 0,
                    run: {
                        connect: { id: this.fixtureRunId },
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
                        connect: { id: this.fixtureRunId },
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
                        connect: { id: this.fixtureRunId },
                    },
                },
                stoppedStep: {
                    id: this.generateId(),
                    name: "Stopped Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 3,
                    status: RunStepStatus.Skipped,
                    complexity: 5,
                    contextSwitches: 5,
                    startedAt: tenMinutesAgo,
                    completedAt: fiveMinutesAgo,
                    timeElapsed: 300000,
                    run: {
                        connect: { id: this.fixtureRunId },
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
                        connect: { id: this.fixtureRunId },
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
                        connect: { id: this.fixtureRunId },
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
                        connect: { id: this.fixtureRunId },
                    },
                },
                highOrderStep: {
                    id: this.generateId(),
                    name: "High Order Step",
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: 999,
                    status: RunStepStatus.InProgress,
                    complexity: 0,
                    contextSwitches: 0,
                    run: {
                        connect: { id: this.fixtureRunId },
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
            status: RunStepStatus.InProgress,
            complexity: 0,
            contextSwitches: 0,
            run: {
                connect: { id: this.defaultRunId },
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
                connect: { id: this.defaultRunId },
            },
            resourceVersion: {
                connect: { id: this.defaultResourceVersionId },
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
                    withResourceVersion: { resourceVersionId: this.fixtureResourceVersionId },
                },
            },
            skippedStep: {
                name: "skippedStep",
                description: "Step that was skipped during execution",
                config: {
                    overrides: {
                        name: "Skipped Validation Step",
                        complexity: 30,
                        contextSwitches: 5,
                        status: RunStepStatus.Skipped,
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
    async createNotStartedStep(runId: bigint, order: number): Promise<run_step> {
        return await this.createMinimal({
            order,
            status: RunStepStatus.InProgress,
            startedAt: undefined,
            run: { connect: { id: runId } },
        });
    }

    async createInProgressStep(runId: bigint, order: number): Promise<run_step> {
        return await this.createMinimal({
            order,
            status: RunStepStatus.InProgress,
            startedAt: new Date(),
            run: { connect: { id: runId } },
        });
    }

    async createCompletedStep(
        runId: bigint,
        order: number,
        complexity = 10,
    ): Promise<run_step> {
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

    async createSkippedStep(runId: bigint, order: number): Promise<run_step> {
        return await this.createMinimal({
            order,
            status: RunStepStatus.Skipped,
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
                    versionLabel: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.run_stepCreateInput,
        config: RunStepRelationConfig,
        _tx: PrismaClient,
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

    protected async checkModelConstraints(record: run_step): Promise<string[]> {
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

        if (record.status === RunStepStatus.Skipped && record.startedAt) {
            violations.push("Skipped steps should not have start time");
        }

        // Note: InProgress steps without startedAt represent not-started steps, which is valid
        // Only add violation if explicitly checking for started in-progress steps

        // InProgress steps without start time represent not-started steps
        if (record.status === RunStepStatus.InProgress && record.completedAt) {
            violations.push("In-progress steps should not have completion time");
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
        record: run_step,
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
        runId: bigint,
        stepConfigs: Array<{
            name: string;
            status: RunStepStatus;
            complexity?: number;
            contextSwitches?: number;
        }>,
    ): Promise<run_step[]> {
        const now = new Date();

        const results: run_step[] = [];
        for (let i = 0; i < stepConfigs.length; i++) {
            const config = stepConfigs[i];
            const startTime = new Date(now.getTime() - (stepConfigs.length - i) * 60000);
            const endTime = config.status === RunStepStatus.Completed
                ? new Date(startTime.getTime() + 30000)
                : undefined;

            const result = await this.createMinimal({
                name: config.name,
                nodeId: nanoid(),
                resourceInId: nanoid(),
                order: i,
                status: config.status,
                complexity: config.complexity || 10,
                contextSwitches: config.contextSwitches || 0,
                startedAt: config.status === RunStepStatus.Completed || config.status === RunStepStatus.InProgress ? startTime : undefined,
                completedAt: endTime,
                timeElapsed: endTime ? endTime.getTime() - startTime.getTime() : undefined,
                run: { connect: { id: runId } },
            });
            results.push(result);
        }
        return results;
    }

    /**
     * Create test steps for various scenarios
     */
    async createTestSteps(runId: bigint): Promise<{
        notStarted: run_step;
        inProgress: run_step;
        completed: run_step;
        skipped: run_step;
    }> {
        const [notStarted, inProgress, completed, skipped] = await Promise.all([
            this.createNotStartedStep(runId, 0),
            this.createInProgressStep(runId, 1),
            this.createCompletedStep(runId, 2),
            this.createSkippedStep(runId, 3),
        ]);

        return {
            notStarted,
            inProgress,
            completed,
            skipped,
        };
    }

    /**
     * Create a step with specific timing
     */
    async createTimedStep(
        runId: bigint,
        order: number,
        duration: number, // in milliseconds
        status: RunStepStatus = RunStepStatus.Completed,
    ): Promise<run_step> {
        const endTime = new Date();
        const startTime = new Date(endTime.getTime() - duration);

        return await this.createMinimal({
            order,
            status,
            startedAt: startTime,
            completedAt: status === RunStepStatus.Completed ? endTime : undefined,
            timeElapsed: status === RunStepStatus.Completed ? duration : undefined,
            run: { connect: { id: runId } },
        });
    }
}

// Export factory creator function
export const createRunStepDbFactory = (prisma: PrismaClient) =>
    new RunStepDbFactory(prisma);

// Export the class for type usage
export { RunStepDbFactory as RunStepDbFactoryClass };
