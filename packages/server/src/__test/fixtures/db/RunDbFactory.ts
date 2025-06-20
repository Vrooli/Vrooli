import { generatePK, nanoid } from "@vrooli/shared";
import { type Prisma, type PrismaClient, RunStatus, RunStepStatus } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
    TestScenario,
} from "./types.js";
import { runConfigFixtures } from "@vrooli/shared/__test/fixtures/config";

interface RunRelationConfig extends RelationConfig {
    withUser?: { userId: string };
    withTeam?: { teamId: string };
    withSchedule?: { scheduleId: string };
    withResourceVersion?: { resourceVersionId: string };
    withSteps?: boolean | number;
    withIO?: boolean | number;
}

/**
 * Enhanced database fixture factory for Run model
 * Provides comprehensive testing capabilities for routine execution runs
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for various run statuses
 * - Step tracking and completion
 * - Input/output data management
 * - Resource version association
 * - Schedule integration
 * - Predefined test scenarios
 * - Execution metrics tracking
 */
export class RunDbFactory extends EnhancedDatabaseFactory<
    Prisma.run,
    Prisma.runCreateInput,
    Prisma.runInclude,
    Prisma.runUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('Run', prisma);
        this.initializeScenarios();
    }

    protected getPrismaDelegate() {
        return this.prisma.run;
    }

    /**
     * Get complete test fixtures for Run model
     */
    protected getFixtures(): DbTestFixtures<Prisma.runCreateInput, Prisma.runUpdateInput> {
        const now = new Date();
        const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000);
        const twoHoursAgo = new Date(now.getTime() - 2 * 60 * 60 * 1000);

        return {
            minimal: {
                id: generatePK().toString(),
                name: "Test Run",
                status: RunStatus.Scheduled,
                isPrivate: false,
                wasRunAutomatically: false,
                completedComplexity: 0,
                contextSwitches: 0,
            },
            complete: {
                id: generatePK().toString(),
                name: "Complete Test Run",
                status: RunStatus.InProgress,
                isPrivate: false,
                wasRunAutomatically: true,
                completedComplexity: 50,
                contextSwitches: 3,
                data: JSON.stringify(runConfigFixtures.complete),
                startedAt: twoHoursAgo,
                timeElapsed: 3600000, // 1 hour in milliseconds
                user: {
                    connect: { id: "user_id" }
                },
                resourceVersion: {
                    connect: { id: "resource_version_id" }
                },
                steps: {
                    create: [
                        {
                            id: generatePK().toString(),
                            name: "Step 1",
                            nodeId: nanoid(),
                            resourceInId: nanoid(),
                            order: 0,
                            status: RunStepStatus.Completed,
                            complexity: 10,
                            contextSwitches: 1,
                            startedAt: twoHoursAgo,
                            completedAt: oneHourAgo,
                            timeElapsed: 3600000,
                        },
                        {
                            id: generatePK().toString(),
                            name: "Step 2",
                            nodeId: nanoid(),
                            resourceInId: nanoid(),
                            order: 1,
                            status: RunStepStatus.InProgress,
                            complexity: 15,
                            contextSwitches: 0,
                            startedAt: oneHourAgo,
                        },
                    ],
                },
                io: {
                    create: [
                        {
                            id: generatePK().toString(),
                            nodeInputName: "input1",
                            nodeName: "Node 1",
                            data: JSON.stringify({ value: "test input" }),
                        },
                        {
                            id: generatePK().toString(),
                            nodeInputName: "output1",
                            nodeName: "Node 1",
                            data: JSON.stringify({ result: "processed" }),
                        },
                    ],
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, name, status
                    isPrivate: false,
                    wasRunAutomatically: false,
                },
                invalidTypes: {
                    id: "not-a-bigint",
                    name: null, // Should be string
                    status: "InProgress", // Should be enum value
                    isPrivate: "false", // Should be boolean
                    completedComplexity: "50", // Should be number
                    contextSwitches: -1, // Should be non-negative
                },
                invalidData: {
                    id: generatePK().toString(),
                    name: "Invalid Data Run",
                    status: RunStatus.Failed,
                    isPrivate: false,
                    wasRunAutomatically: false,
                    completedComplexity: 0,
                    contextSwitches: 0,
                    data: "not-json", // Invalid JSON
                },
                invalidTimeRange: {
                    id: generatePK().toString(),
                    name: "Invalid Time Run",
                    status: RunStatus.Completed,
                    isPrivate: false,
                    wasRunAutomatically: false,
                    completedComplexity: 0,
                    contextSwitches: 0,
                    startedAt: now,
                    completedAt: oneHourAgo, // Completed before started
                },
                conflictingOwnership: {
                    id: generatePK().toString(),
                    name: "Conflicting Ownership",
                    status: RunStatus.Scheduled,
                    isPrivate: false,
                    wasRunAutomatically: false,
                    completedComplexity: 0,
                    contextSwitches: 0,
                    // Both user and team ownership
                    user: {
                        connect: { id: "user_id" }
                    },
                    team: {
                        connect: { id: "team_id" }
                    },
                },
            },
            edgeCases: {
                scheduledRun: {
                    id: generatePK().toString(),
                    name: "Scheduled Run",
                    status: RunStatus.Scheduled,
                    isPrivate: false,
                    wasRunAutomatically: true,
                    completedComplexity: 0,
                    contextSwitches: 0,
                    schedule: {
                        connect: { id: "schedule_id" }
                    },
                },
                failedRun: {
                    id: generatePK().toString(),
                    name: "Failed Run",
                    status: RunStatus.Failed,
                    isPrivate: false,
                    wasRunAutomatically: false,
                    completedComplexity: 25,
                    contextSwitches: 5,
                    startedAt: twoHoursAgo,
                    timeElapsed: 1800000, // 30 minutes
                    data: JSON.stringify(runConfigFixtures.withErrors),
                },
                completedRun: {
                    id: generatePK().toString(),
                    name: "Completed Run",
                    status: RunStatus.Completed,
                    isPrivate: false,
                    wasRunAutomatically: true,
                    completedComplexity: 100,
                    contextSwitches: 10,
                    startedAt: new Date(now.getTime() - 3 * 60 * 60 * 1000),
                    completedAt: oneHourAgo,
                    timeElapsed: 7200000, // 2 hours
                    data: JSON.stringify(runConfigFixtures.withCompletedDecisions),
                },
                privateRun: {
                    id: generatePK().toString(),
                    name: "Private Run",
                    status: RunStatus.InProgress,
                    isPrivate: true,
                    wasRunAutomatically: false,
                    completedComplexity: 0,
                    contextSwitches: 0,
                    user: {
                        connect: { id: "user_id" }
                    },
                },
                complexDataRun: {
                    id: generatePK().toString(),
                    name: "Complex Data Run",
                    status: RunStatus.InProgress,
                    isPrivate: false,
                    wasRunAutomatically: true,
                    completedComplexity: 75,
                    contextSwitches: 8,
                    data: JSON.stringify(runConfigFixtures.withSubcontexts),
                    startedAt: oneHourAgo,
                    timeElapsed: 3600000,
                },
                manyStepsRun: {
                    id: generatePK().toString(),
                    name: "Many Steps Run",
                    status: RunStatus.InProgress,
                    isPrivate: false,
                    wasRunAutomatically: false,
                    completedComplexity: 200,
                    contextSwitches: 20,
                    steps: {
                        create: Array.from({ length: 20 }, (_, i) => ({
                            id: generatePK().toString(),
                            name: `Step ${i + 1}`,
                            nodeId: nanoid(),
                            resourceInId: nanoid(),
                            order: i,
                            status: i < 10 ? RunStepStatus.Completed : RunStepStatus.InProgress,
                            complexity: 10,
                            contextSwitches: 1,
                            startedAt: new Date(now.getTime() - (20 - i) * 5 * 60 * 1000),
                            completedAt: i < 10 ? new Date(now.getTime() - (20 - i) * 5 * 60 * 1000 + 4 * 60 * 1000) : undefined,
                            timeElapsed: i < 10 ? 240000 : undefined,
                        })),
                    },
                },
            },
            updates: {
                minimal: {
                    status: RunStatus.InProgress,
                    startedAt: now,
                },
                complete: {
                    status: RunStatus.Completed,
                    completedAt: now,
                    timeElapsed: 7200000,
                    completedComplexity: 150,
                    contextSwitches: 15,
                    data: JSON.stringify(runConfigFixtures.withCompletedDecisions),
                },
            },
        };
    }

    protected generateMinimalData(overrides?: Partial<Prisma.runCreateInput>): Prisma.runCreateInput {
        return {
            id: generatePK().toString(),
            name: `Run_${nanoid(8)}`,
            status: RunStatus.Scheduled,
            isPrivate: false,
            wasRunAutomatically: false,
            completedComplexity: 0,
            contextSwitches: 0,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.runCreateInput>): Prisma.runCreateInput {
        const now = new Date();
        const oneHourAgo = new Date(now.getTime() - 60 * 60 * 1000);
        
        return {
            id: generatePK().toString(),
            name: `Complete_Run_${nanoid(8)}`,
            status: RunStatus.InProgress,
            isPrivate: false,
            wasRunAutomatically: true,
            completedComplexity: 50,
            contextSwitches: 5,
            data: JSON.stringify(runConfigFixtures.complete),
            startedAt: oneHourAgo,
            timeElapsed: 3600000,
            steps: {
                create: [
                    {
                        id: generatePK().toString(),
                        name: "Initial Step",
                        nodeId: nanoid(),
                        resourceInId: nanoid(),
                        order: 0,
                        status: RunStepStatus.Completed,
                        complexity: 25,
                        contextSwitches: 2,
                        startedAt: oneHourAgo,
                        completedAt: now,
                        timeElapsed: 3600000,
                    },
                ],
            },
            io: {
                create: [
                    {
                        id: generatePK().toString(),
                        nodeInputName: "input",
                        nodeName: "Start Node",
                        data: JSON.stringify({ initialData: "test" }),
                    },
                ],
            },
            ...overrides,
        };
    }

    /**
     * Initialize test scenarios
     */
    protected initializeScenarios(): void {
        this.scenarios = {
            automatedRun: {
                name: "automatedRun",
                description: "Automated run triggered by schedule",
                config: {
                    overrides: {
                        name: "Automated Daily Process",
                        wasRunAutomatically: true,
                        status: RunStatus.InProgress,
                        data: JSON.stringify(runConfigFixtures.autoExecutionConfig),
                    },
                    withSchedule: { scheduleId: "schedule_id" },
                    withSteps: 3,
                },
            },
            manualRun: {
                name: "manualRun",
                description: "Manual run started by user",
                config: {
                    overrides: {
                        name: "Manual Data Processing",
                        wasRunAutomatically: false,
                        status: RunStatus.InProgress,
                        data: JSON.stringify(runConfigFixtures.manualExecutionConfig),
                    },
                    withUser: { userId: "user_id" },
                    withSteps: 5,
                    withIO: 4,
                },
            },
            failedRun: {
                name: "failedRun",
                description: "Run that failed during execution",
                config: {
                    overrides: {
                        name: "Failed Process",
                        status: RunStatus.Failed,
                        completedComplexity: 30,
                        contextSwitches: 8,
                        data: JSON.stringify(runConfigFixtures.withErrors),
                    },
                    withSteps: 2,
                },
            },
            completedRun: {
                name: "completedRun",
                description: "Successfully completed run",
                config: {
                    overrides: {
                        name: "Completed Process",
                        status: RunStatus.Completed,
                        completedComplexity: 100,
                        contextSwitches: 12,
                        startedAt: new Date(Date.now() - 2 * 60 * 60 * 1000),
                        completedAt: new Date(),
                        timeElapsed: 7200000,
                        data: JSON.stringify(runConfigFixtures.withCompletedDecisions),
                    },
                    withSteps: 10,
                    withIO: 20,
                },
            },
            complexRun: {
                name: "complexRun",
                description: "Complex run with many branches and decisions",
                config: {
                    overrides: {
                        name: "Complex Multi-Branch Process",
                        status: RunStatus.InProgress,
                        completedComplexity: 250,
                        contextSwitches: 25,
                        data: JSON.stringify(runConfigFixtures.withActiveBranches),
                    },
                    withResourceVersion: { resourceVersionId: "resource_version_id" },
                    withSteps: 15,
                    withIO: 30,
                },
            },
        };
    }

    /**
     * Create specific run types
     */
    async createScheduledRun(scheduleId: string): Promise<Prisma.run> {
        return await this.createMinimal({
            status: RunStatus.Scheduled,
            wasRunAutomatically: true,
            schedule: { connect: { id: scheduleId } },
        });
    }

    async createInProgressRun(userId?: string, teamId?: string): Promise<Prisma.run> {
        const ownership = userId 
            ? { user: { connect: { id: userId } } }
            : teamId 
            ? { team: { connect: { id: teamId } } }
            : {};

        return await this.createWithRelations({
            overrides: {
                status: RunStatus.InProgress,
                startedAt: new Date(),
                ...ownership,
            },
            withSteps: 3,
        });
    }

    async createCompletedRun(userId: string): Promise<Prisma.run> {
        const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000);
        
        return await this.seedScenario('completedRun');
    }

    async createFailedRun(errorData: any = {}): Promise<Prisma.run> {
        return await this.createMinimal({
            status: RunStatus.Failed,
            data: JSON.stringify({ error: errorData }),
            startedAt: new Date(Date.now() - 30 * 60 * 1000),
            timeElapsed: 1800000,
        });
    }

    protected getDefaultInclude(): Prisma.runInclude {
        return {
            user: true,
            team: true,
            schedule: true,
            resourceVersion: {
                select: {
                    id: true,
                    publicId: true,
                    versionString: true,
                },
            },
            steps: {
                orderBy: { order: 'asc' },
            },
            io: true,
            _count: {
                select: {
                    steps: true,
                    io: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.runCreateInput,
        config: RunRelationConfig,
        tx: any
    ): Promise<Prisma.runCreateInput> {
        let data = { ...baseData };

        // Handle user relationship
        if (config.withUser) {
            data.user = {
                connect: { id: config.withUser.userId }
            };
        }

        // Handle team relationship
        if (config.withTeam) {
            data.team = {
                connect: { id: config.withTeam.teamId }
            };
        }

        // Handle schedule relationship
        if (config.withSchedule) {
            data.schedule = {
                connect: { id: config.withSchedule.scheduleId }
            };
            data.wasRunAutomatically = true;
        }

        // Handle resource version relationship
        if (config.withResourceVersion) {
            data.resourceVersion = {
                connect: { id: config.withResourceVersion.resourceVersionId }
            };
        }

        // Handle steps
        if (config.withSteps) {
            const stepCount = typeof config.withSteps === 'number' ? config.withSteps : 1;
            const startTime = data.startedAt || new Date(Date.now() - 60 * 60 * 1000);
            
            data.steps = {
                create: Array.from({ length: stepCount }, (_, i) => ({
                    id: generatePK().toString(),
                    name: `Step ${i + 1}`,
                    nodeId: nanoid(),
                    resourceInId: nanoid(),
                    order: i,
                    status: i === 0 ? RunStepStatus.InProgress : RunStepStatus.Pending,
                    complexity: 10,
                    contextSwitches: 0,
                    startedAt: i === 0 ? startTime : undefined,
                })),
            };
        }

        // Handle IO
        if (config.withIO) {
            const ioCount = typeof config.withIO === 'number' ? config.withIO : 1;
            
            data.io = {
                create: Array.from({ length: ioCount }, (_, i) => ({
                    id: generatePK().toString(),
                    nodeInputName: i % 2 === 0 ? `input${i/2}` : `output${Math.floor(i/2)}`,
                    nodeName: `Node ${Math.floor(i/2)}`,
                    data: JSON.stringify({ 
                        type: i % 2 === 0 ? 'input' : 'output',
                        value: `data_${i}` 
                    }),
                })),
            };
        }

        return data;
    }

    protected async checkModelConstraints(record: Prisma.run): Promise<string[]> {
        const violations: string[] = [];
        
        // Check ownership (must have either user or team, not both)
        if (record.userId && record.teamId) {
            violations.push('Run cannot belong to both user and team');
        }

        // Check time constraints
        if (record.startedAt && record.completedAt && record.startedAt >= record.completedAt) {
            violations.push('Start time must be before completion time');
        }

        // Check status consistency
        if (record.status === RunStatus.Scheduled && record.startedAt) {
            violations.push('Scheduled runs should not have start time');
        }

        if (record.status === RunStatus.Completed && !record.completedAt) {
            violations.push('Completed runs must have completion time');
        }

        if (record.status === RunStatus.InProgress && !record.startedAt) {
            violations.push('In-progress runs must have start time');
        }

        // Check complexity and context switches
        if (record.completedComplexity < 0) {
            violations.push('Completed complexity cannot be negative');
        }

        if (record.contextSwitches < 0) {
            violations.push('Context switches cannot be negative');
        }

        // Check data validity
        if (record.data) {
            try {
                JSON.parse(record.data);
            } catch {
                violations.push('Run data must be valid JSON');
            }
        }

        // Check steps
        if (record.steps) {
            const orderSet = new Set<number>();
            for (const step of record.steps) {
                if (orderSet.has(step.order)) {
                    violations.push('Step orders must be unique within a run');
                }
                orderSet.add(step.order);

                if (step.startedAt && step.completedAt && step.startedAt >= step.completedAt) {
                    violations.push('Step start time must be before completion time');
                }

                if (step.status === RunStepStatus.Completed && !step.completedAt) {
                    violations.push('Completed steps must have completion time');
                }
            }
        }

        return violations;
    }

    protected getCascadeInclude(): any {
        return {
            steps: true,
            io: true,
        };
    }

    protected async deleteRelatedRecords(
        record: Prisma.run,
        remainingDepth: number,
        tx: any,
        includeOnly?: string[]
    ): Promise<void> {
        const shouldDelete = (relation: string) => 
            !includeOnly || includeOnly.includes(relation);

        // Delete steps
        if (shouldDelete('steps') && record.steps?.length) {
            await tx.run_step.deleteMany({
                where: { runId: record.id },
            });
        }

        // Delete IO
        if (shouldDelete('io') && record.io?.length) {
            await tx.run_io.deleteMany({
                where: { runId: record.id },
            });
        }
    }

    /**
     * Create a run with specific step configuration
     */
    async createRunWithSteps(
        stepConfigs: Array<{
            name: string;
            status: RunStepStatus;
            complexity?: number;
        }>,
        userId?: string
    ): Promise<Prisma.run> {
        const now = new Date();
        const steps = stepConfigs.map((config, i) => ({
            id: generatePK().toString(),
            name: config.name,
            nodeId: nanoid(),
            resourceInId: nanoid(),
            order: i,
            status: config.status,
            complexity: config.complexity || 10,
            contextSwitches: 0,
            startedAt: config.status !== RunStepStatus.Pending ? new Date(now.getTime() - (stepConfigs.length - i) * 10 * 60 * 1000) : undefined,
            completedAt: config.status === RunStepStatus.Completed ? now : undefined,
            timeElapsed: config.status === RunStepStatus.Completed ? 600000 : undefined,
        }));

        return await this.createComplete({
            user: userId ? { connect: { id: userId } } : undefined,
            steps: { create: steps },
        });
    }

    /**
     * Create test runs for different scenarios
     */
    async createTestRuns(userId: string): Promise<{
        scheduled: Prisma.run;
        inProgress: Prisma.run;
        completed: Prisma.run;
        failed: Prisma.run;
        automated: Prisma.run;
    }> {
        const [scheduled, inProgress, completed, failed, automated] = await Promise.all([
            this.createMinimal({ 
                user: { connect: { id: userId } },
                status: RunStatus.Scheduled,
            }),
            this.createInProgressRun(userId),
            this.createCompletedRun(userId),
            this.createFailedRun(),
            this.seedScenario('automatedRun'),
        ]);

        return {
            scheduled,
            inProgress,
            completed,
            failed,
            automated: automated as unknown as Prisma.run,
        };
    }

    /**
     * Create a run with IO data
     */
    async createRunWithIO(
        inputs: Record<string, any>,
        outputs: Record<string, any>,
        userId: string
    ): Promise<Prisma.run> {
        const ioData = [
            ...Object.entries(inputs).map(([key, value]) => ({
                id: generatePK().toString(),
                nodeInputName: key,
                nodeName: "Input Node",
                data: JSON.stringify(value),
            })),
            ...Object.entries(outputs).map(([key, value]) => ({
                id: generatePK().toString(),
                nodeInputName: key,
                nodeName: "Output Node",
                data: JSON.stringify(value),
            })),
        ];

        return await this.createComplete({
            user: { connect: { id: userId } },
            io: { create: ioData },
        });
    }
}

// Export factory creator function
export const createRunDbFactory = (prisma: PrismaClient) => 
    RunDbFactory.getInstance('Run', prisma);

// Export the class for type usage
export { RunDbFactory as RunDbFactoryClass };