/**
 * Specialized factory for creating SwarmExecutionTask test data with proper TierExecutionRequest structure
 */

import { generatePK, generatePublicId, type SessionUser } from "@vrooli/shared";
import { createValidTaskData } from "../../../tasks/taskFactory.js";
import { QueueTaskType, type SwarmExecutionTask } from "../../../tasks/taskTypes.js";

/**
 * Creates a valid SwarmExecutionTask with the nested TierExecutionRequest structure
 * 
 * @param overrides - Partial SwarmExecutionTask data to override defaults
 * @returns Complete SwarmExecutionTask with all required properties
 */
export function createSwarmTask(overrides: Partial<SwarmExecutionTask> = {}): SwarmExecutionTask {
    const userId = overrides.context?.userData?.id || generatePK().toString();
    const swarmId = overrides.input?.swarmId || generatePK().toString();

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
        publicId: generatePublicId(),
        updatedAt: new Date(),
        session: {
            __typename: "SessionUserSession",
            id: generatePK().toString(),
            lastRefreshAt: new Date(),
        },
    };

    const baseTask = createValidTaskData(QueueTaskType.SWARM_EXECUTION, {}, {
        id: overrides.id || swarmId,
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
            swarmId: overrides.input?.swarmId,
            chatId: overrides.input?.chatId,
            goal: overrides.input?.goal || "Complete the test objectives efficiently",
            teamConfiguration: overrides.input?.teamConfiguration,
            availableTools: overrides.input?.availableTools || [
                { name: "calculator", description: "Perform mathematical calculations" },
                { name: "researcher", description: "Research and gather information" },
            ],
            executionConfig: overrides.input?.executionConfig || {
                model: "gpt-4",
                temperature: 0.7,
                parallelExecutionLimit: 3,
            },
            userData: { ...defaultUserData, ...overrides.input?.userData },
            ...overrides.input,
        },
        allocation: {
            maxCredits: "1000",
            maxDurationMs: 300000, // 5 minutes
            maxMemoryMB: 512,
            maxConcurrentSteps: 5,
            ...overrides.allocation,
        },
        options: overrides.options,
    } as SwarmExecutionTask;
}

/**
 * Helper to create partial SwarmExecutionTask data for specific test scenarios
 */
export const swarmTaskScenarios = {
    /** Create a SwarmTask for a new swarm */
    newSwarm: (goal: string): Partial<SwarmExecutionTask> => {
        const userId = generatePK().toString();
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
            publicId: generatePublicId(),
            updatedAt: new Date(),
            session: {
                __typename: "SessionUserSession",
                id: generatePK().toString(),
                lastRefreshAt: new Date(),
            },
        };

        return {
            input: {
                goal,
                teamConfiguration: {
                    leaderAgentId: generatePK().toString(),
                    preferredTeamSize: 3,
                    requiredSkills: ["research", "analysis", "communication"],
                },
                userData: defaultUserData,
            },
        };
    },

    /** Create a SwarmTask for an existing swarm/chat */
    existingSwarm: (swarmId: string, chatId: string): Partial<SwarmExecutionTask> => {
        const userId = generatePK().toString();
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
            publicId: generatePublicId(),
            updatedAt: new Date(),
            session: {
                __typename: "SessionUserSession",
                id: generatePK().toString(),
                lastRefreshAt: new Date(),
            },
        };

        return {
            input: {
                swarmId,
                chatId,
                goal: "Continue the existing swarm's objectives",
                userData: defaultUserData,
            },
        };
    },

    /** Create a SwarmTask with premium user */
    premiumUser: (): Partial<SwarmExecutionTask> => ({
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
                publicId: generatePublicId(),
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

    /** Create a SwarmTask with limited resources */
    limitedResources: (): Partial<SwarmExecutionTask> => {
        const userId = generatePK().toString();
        const defaultUserData: SessionUser = {
            __typename: "SessionUser",
            id: userId,
            name: "testUser",
            hasPremium: false,
            hasReceivedPhoneVerificationReward: false,
            languages: ["en"],
            theme: "light",
            credits: "100",
            phoneNumberVerified: false,
            publicId: generatePublicId(),
            updatedAt: new Date(),
            session: {
                __typename: "SessionUserSession",
                id: generatePK().toString(),
                lastRefreshAt: new Date(),
            },
        };

        return {
            allocation: {
                maxCredits: "100",
                maxDurationMs: 60000, // 1 minute
                maxMemoryMB: 256,
                maxConcurrentSteps: 1,
            },
            input: {
                goal: "Complete simple task with limited resources",
                availableTools: [
                    { name: "calculator", description: "Basic calculations only" },
                ],
                executionConfig: {
                    model: "gpt-3.5-turbo",
                    temperature: 0.5,
                    parallelExecutionLimit: 1,
                },
                userData: defaultUserData,
            },
        };
    },
};
