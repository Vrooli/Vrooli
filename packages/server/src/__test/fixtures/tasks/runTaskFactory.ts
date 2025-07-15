/**
 * Specialized factory for creating RunTask test data with proper TierExecutionRequest structure
 */

import { generatePK, TaskStatus, type RunConfig, type RunTriggeredFrom, type SessionUser } from "@vrooli/shared";
import { type RunTask, QueueTaskType } from "../../../tasks/taskTypes.js";
import { createValidTaskData } from "../../../tasks/taskFactory.js";

/**
 * Creates a valid RunTask with the nested TierExecutionRequest structure
 * 
 * @param overrides - Partial RunTask data to override defaults
 * @returns Complete RunTask with all required properties
 */
export function createRunTask(overrides: Partial<RunTask> = {}): RunTask {
    const userId = overrides.context?.userData?.id || generatePK().toString();
    const swarmId = overrides.context?.swarmId || generatePK().toString();
    const runId = overrides.input?.runId || generatePK().toString();
    const resourceVersionId = overrides.input?.resourceVersionId || generatePK().toString();

    const defaultUserData: SessionUser = {
        __typename: "SessionUser",
        id: userId,
        name: "testUser",
        hasPremium: false,
        hasReceivedPhoneVerificationReward: false,
        languages: ["en"],
        theme: "light",
        credits: "1000",
        phoneNumberVerified: false,
        publicId: generatePK().toString(),
        updatedAt: new Date(),
        session: {
            __typename: "SessionUserSession",
            id: generatePK().toString(),
            lastRefreshAt: new Date(),
        },
    };

    const defaultConfig: RunConfig = {
        botConfig: {
            strategy: "reasoning",
            model: "gpt-4",
        },
        limits: {
            maxSteps: 50,
            maxTime: 300000, // 5 minutes
        },
    };

    const baseTask = createValidTaskData(QueueTaskType.RUN_START, {}, {
        id: overrides.id || runId,
        userId,
    });

    return {
        ...baseTask,
        context: {
            swarmId,
            parentSwarmId: overrides.context?.parentSwarmId,
            userData: { ...defaultUserData, ...overrides.context?.userData },
            timestamp: overrides.context?.timestamp || new Date(),
            ...overrides.context,
        },
        input: {
            runId,
            resourceVersionId,
            config: overrides.input?.config || defaultConfig,
            formValues: overrides.input?.formValues || {},
            isNewRun: overrides.input?.isNewRun !== undefined ? overrides.input.isNewRun : true,
            runFrom: overrides.input?.runFrom || "Trigger" as RunTriggeredFrom,
            startedById: overrides.input?.startedById || userId,
            status: overrides.input?.status || TaskStatus.Scheduled,
            ...overrides.input,
        },
        allocation: {
            maxCredits: "1000",
            maxDurationMs: 300000,
            maxMemoryMB: 512,
            maxConcurrentSteps: 5,
            ...overrides.allocation,
        },
        options: overrides.options,
    } as RunTask;
}

/**
 * Helper to create partial RunTask data for specific test scenarios
 */
export const runTaskScenarios = {
    /** Create a RunTask for a new run */
    newRun: (userId?: string): Partial<RunTask> => ({
        input: {
            runId: generatePK().toString(),
            resourceVersionId: generatePK().toString(),
            isNewRun: true,
            runFrom: "RunView" as RunTriggeredFrom,
            startedById: userId || generatePK().toString(),
            status: TaskStatus.Scheduled,
        },
    }),
    
    /** Create a RunTask for resuming an existing run */
    resumeRun: (runId: string): Partial<RunTask> => ({
        input: {
            runId,
            resourceVersionId: generatePK().toString(),
            isNewRun: false,
            runFrom: "Resume" as RunTriggeredFrom,
            startedById: generatePK().toString(),
            status: TaskStatus.InProgress,
        },
    }),
    
    /** Create a RunTask with premium user */
    premiumUser: (): Partial<RunTask> => ({
        context: {
            userData: {
                __typename: "SessionUser",
                id: generatePK().toString(),
                name: "premiumUser",
                hasPremium: true,
                hasReceivedPhoneVerificationReward: true,
                languages: ["en"],
                theme: "dark",
                credits: "5000",
                phoneNumberVerified: true,
                publicId: generatePK().toString(),
                updatedAt: new Date(),
                session: {
                    __typename: "SessionUserSession",
                    id: generatePK().toString(),
                    lastRefreshAt: new Date(),
                },
            },
        },
        allocation: {
            maxCredits: "5000", // Higher limit for premium
            maxDurationMs: 600000, // 10 minutes
            maxMemoryMB: 1024,
            maxConcurrentSteps: 10,
        },
    }),
};