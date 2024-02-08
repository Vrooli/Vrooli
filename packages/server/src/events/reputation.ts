/**
 * Handles giving reputation to users when some action is performed.
 */

import { GqlModelType } from "@local/shared";
import { logger } from "../events/logger";
import { withRedis } from "../redisConn";
import { PrismaType } from "../types";

export enum ReputationEvent {
    ObjectDeletedFromReport = "ObjectDeletedFromReport",
    ReportWasAccepted = "ReportWasAccepted",
    ReportWasRejected = "ReportWasRejected",
    PullRequestWasAccepted = "PullRequestWasAccepted",
    PullRequestWasRejected = "PullRequestWasRejected",
    IssueCreatedWasAccepted = "IssueCreatedWasAccepted",
    IssueCreatedWasRejected = "IssueCreatedWasRejected",
    AnsweredQuestion = "AnsweredQuestion",
    AnsweredQuestionWasAccepted = "AnsweredQuestionWasAccepted",
    PublicApiCreated = "PublicApiCreated",
    PublicProjectCreated = "PublicProjectCreated",
    PublicRoutineCreated = "PublicRoutineCreated",
    PublicSmartContractCreated = "PublicSmartContractCreated",
    PublicStandardCreated = "PublicStandardCreated",
    ReceivedVote = "ReceivedVote",
    ReceivedStar = "ReceivedStar",
    ContributedToReport = "ContributedToReport",
}

const MaxReputationGainPerDay = 100;

/**
 * Checks if an object type is tracked by the repuation system
 * @param objectType The object type to check
 * @returns `Public${objectType}Created` if it's a tracked reputation nevent
 */
export const objectReputationEvent = <T extends keyof typeof GqlModelType>(objectType: T): `Public${T}Created` | null => {
    return `Public${objectType}Created` in ReputationEvent ? `Public${objectType}Created` : null;
};

/**
 * Finds the next threshold for rewarding a reputation point,
 * given the current vote count and the direction of the change.
 * 
 * Examples:
 * - If curr is 0 and direction is 1, returns 1
 * - If curr is 0 and direction is -1, returns -1
 * - If curr is 10 and direction is 1, returns 20
 * - If curr is 10 and direction is -1, returns 0
 * - If curr is 99 and direction is 1, returns 100
 * - If curr is 99 and direction is -1, returns 90
 * - If curr is 100 and direction is 1, returns 200
 * - If curr is -1 and direction is -1, returns -2
 * - If curr is -10 and direction is -1, returns -20
 * - If curr is -999 and direction is -1, returns -1000
 */
export const nextRewardThreshold = (curr: number, direction: 1 | -1) => {
    const diffSigns = (curr < 0 && direction > 0) || (curr > 0 && direction < 0);
    // Determine magnitude of curr (1 * 10^(digits - 1)). 
    // NOTE: We move curr by 1 when signs are opposite because not doing so would cause 
    // situations like nextRewardThreshold(-10, 1) to return 0 instead of -9.
    const magnitude = Math.pow(10, (Math.abs(curr + (diffSigns ? direction : 0)) + "").length - 1);
    return direction > 0 ?
        Math.ceil((curr + 1) / magnitude) * magnitude :
        Math.floor((curr - 1) / magnitude) * magnitude;
};

/**
 * Generates the reputation which should be rewarded to a user for receiving a vote. 
 * Given v0 is the original vote count and v1 is the new vote count:
 * - If v0 === v1, returns 0
 * - If v0 is from -9 to 9 and v1 > v0, then the user gets 1 reputation
 * - If v0 is from -9 to 9 and v1 < v0, then the user loses 1 reputation
 * - If v0 is from -99 to -10 or 10 to 99 and v1 > v0, then the user gets 1 reputation if v1 is a multiple of 10, otherwise 0
 * - If v0 is from -99 to -10 or 10 to 99 and v1 < v0, then the user loses 1 reputation if v1 is a multiple of 10, otherwise 0
 * - If v0 is from -999 to -100 or 100 to 999 and v1 > v0, then the user gets 1 reputation if v1 is a multiple of 100, otherwise 0
 * - If v0 is from -999 to -100 or 100 to 999 and v1 < v0, then the user loses 1 reputation if v1 is a multiple of 100, otherwise 0
 * , and so on.
 * @returns The amount of reputation to reward the user, or NaN if an error occurred
 */
export const reputationDeltaVote = (v0: number, v1: number): number => {
    if (typeof v0 !== "number" || typeof v1 !== "number") {
        logger.error("Invalid vote count", { trace: "0101", v0, v1 });
        return NaN;
    }
    if (v0 === v1) return 0;

    const pastTarget = () => direction > 0 ? curr > v1 : curr < v1;

    const direction = v1 > v0 ? 1 : -1;
    let reputationDelta = 0;
    let curr = v0;
    let loops = 0;

    do {
        curr = nextRewardThreshold(curr, direction);
        // If the current number is past the target, return current reputation delta
        if (pastTarget()) {
            return reputationDelta * direction; // Make sure it has the correct sign
        }
        // Otherwise, increment the reputation delta
        reputationDelta++;
        // Track the number of loops to prevent infinite loops
        loops++;
    } while (loops < 1000); // Prevent infinite loops

    logger.error("Infinite loop detected in reputationDeltaVote", { trace: "0102", v0, v1, reputationDelta });
    return reputationDelta;
};

/**
 * Generates the reputation which should be rewarded to a user for receiving a star.
 * Same as votes, but double the magnitude.
 */
export const reputationDeltaStar = (v0: number, v1: number): number => {
    return reputationDeltaVote(v0, v1) * 2;
};

/**
 * Generates the reputation which should be rewarded to a user for contributing to a report. 
 * Should receive a reputation point every 10 contributions
 * @param totalContributions The total number of contributions you've made to any report
 */
export const reputationDeltaReportContribute = (totalContributions: number): number => {
    return Math.floor(totalContributions / 10);
};

/**
 * Maps reputation events to the amount of reputation gained. Events which 
 * don't have a fixed amount of reputation gained (e.g. votes) are handled separately.
 */
const reputationMap: { [key in ReputationEvent]?: number } = {
    ObjectDeletedFromReport: -10,
    ReportWasAccepted: 2,
    ReportWasRejected: -1,
    PullRequestWasAccepted: 3,
    PullRequestWasRejected: -1,
    IssueCreatedWasAccepted: 3,
    IssueCreatedWasRejected: -1,
    AnsweredQuestion: 1,
    AnsweredQuestionWasAccepted: 3,
    PublicApiCreated: 2,
    PublicProjectCreated: 2,
    PublicRoutineCreated: 2,
    PublicSmartContractCreated: 2,
    PublicStandardCreated: 2,
};

/**
 * Updates the reputation gained today in Redis
 * @param userId The id of the user
 * @param delta The amount of reputation to add to the user's reputation gained today
 * @returns The new reputation gained today
 */
export async function getReputationGainedToday(userId: string, delta: number): Promise<number> {
    const keyBase = `reputation-gain-limit-${userId}`;
    // Initialize with max value, so on fail we don't increase reputation in the non-Redis database
    let reputationGainedToday = Number.MAX_SAFE_INTEGER;
    await withRedis({
        process: async (redisClient) => {
            // Increment the reputation gained today by the given amount
            reputationGainedToday = await redisClient.incrBy(keyBase, delta);
            // If key is new, set it to expire in 24 hours
            if (reputationGainedToday === delta) {
                await redisClient.expire(keyBase, 60 * 60 * 24);
            }
        },
        trace: "0514",
    });
    return reputationGainedToday;
}

/**
 * Helper function to update the reputation of a user. Not to be used directly.
 * @param delta The amount of reputation to add to the user's reputation
 * @param prisma The prisma client
 * @param userId The id of the user
 * @param event The event which caused the reputation change
 * @param objectId1 The id of the first object involved in the event, if applicable
 * @param objectId2 The id of the second object involved in the event, if applicable
 */
const updateReputationHelper = async (
    delta: number,
    prisma: PrismaType,
    userId: string,
    event: ReputationEvent | `${ReputationEvent}`,
    objectId1?: string,
    objectId2?: string,
) => {
    // Determine how much reputation the user will have gained today. 
    // Should not be able to gain more than 100 reputation per day, 
    // or lose more than 100 reputation per day.
    const totalReputationToday = await getReputationGainedToday(userId, delta);
    let updatedReputation = totalReputationToday;
    if (totalReputationToday > MaxReputationGainPerDay) {
        updatedReputation = MaxReputationGainPerDay;
    }
    else if (totalReputationToday < -MaxReputationGainPerDay) {
        updatedReputation = -MaxReputationGainPerDay;
    }
    // If already past the max or min reputation, don't update
    if (totalReputationToday - delta > MaxReputationGainPerDay || totalReputationToday - delta < -MaxReputationGainPerDay) {
        return;
    }
    // Update the user's reputation
    await prisma.award.update({
        where: { userId_category: { userId, category: "Reputation" } },
        data: { progress: updatedReputation },
    });
    // Also add to user's reputation history, so they can see why their reputation changed
    const amount = totalReputationToday > MaxReputationGainPerDay || totalReputationToday < -MaxReputationGainPerDay ?
        totalReputationToday - delta : delta;
    await prisma.reputation_history.create({
        data: {
            userId,
            amount,
            event,
            objectId1,
            objectId2,
        },
    });
};

/**
 * Handles updating the reputation score for a user. Repuation is stored as an 
 * award category, so all this function does is determine which actions 
 * give what reputation, and then calls the award category update function.
 */
export const Reputation = () => ({
    /**
     * Deletes a reputation history entry for an object creation, and updates the user's reputation accordingly
     * @param prisma The prisma client
     * @param objectId The id of the object that was deleted
     * @param userId The id of the user who created the object
     */
    unCreateObject: async (prisma: PrismaType, objectId: string, userId: string) => {
        // Find the reputation history entry for the object creation
        const historyEntry = await prisma.reputation_history.findFirst({
            where: {
                objectId1: objectId,
                userId,
                event: {
                    in: [
                        "PublicApiCreated",
                        "PublicProjectCreated",
                        "PublicRoutineCreated",
                        "PublicSmartContractCreated",
                        "PublicStandardCreated",
                    ],
                },
            },
        });
        // If the entry exists, delete it and decrease the user's reputation
        if (historyEntry) {
            await prisma.reputation_history.delete({
                where: { id: historyEntry.id },
            });
            await prisma.user.update({
                where: { id: userId },
                data: { reputation: { decrement: historyEntry.amount } },
            });
        }
    },
    /**
     * Updates a user's reputation based on an event
     * @param event The event that occurred
     * @param prisma The prisma client
     * @param userId Typically the user who performed the event, 
     * but can also be the user who owns the object that was affected
     */
    update: async (
        event: Exclude<ReputationEvent, "ReceivedVote" | "ReceivedStar" | "ContributedToReport"> | `${Exclude<ReputationEvent, "ReceivedVote" | "ReceivedStar" | "ContributedToReport">}`,
        prisma: PrismaType,
        userId: string,
        object1Id?: string,
        object2Id?: string,
    ) => {
        // Determine reputation delta
        const delta = reputationMap[event] || 0;
        // Update reputation
        await updateReputationHelper(delta, prisma, userId, event, object1Id, object2Id);
    },
    /**
     * Custom reputation update function for votes
     * @param v0 The original vote count
     * @param v1 The new vote count
     * @param prisma The prisma client
     * @param userId The user who received the vote
     */
    updateVote: async (v0: number, v1: number, prisma: PrismaType, userId: string) => {
        const delta = reputationDeltaVote(v0, v1);
        if (isNaN(delta)) return;
        await updateReputationHelper(delta, prisma, userId, "ReceivedVote");
    },
    /**
     * Custom reputation update function for bookmarks
     * @param v0 The original star count
     * @param v1 The new star count
     * @param prisma The prisma client
     * @param userId The user who received the star
     */
    updateStar: async (v0: number, v1: number, prisma: PrismaType, userId: string) => {
        const delta = reputationDeltaStar(v0, v1);
        if (isNaN(delta)) return;
        await updateReputationHelper(delta, prisma, userId, "ReceivedStar");
    },
    /**
     * Custom reputation update function for report contributions
     * @param totalContributions The total number of contributions you've made to all reports
     * @param prisma The prisma client
     * @param userId The user who contributed to the report
     */
    updateReportContribute: async (totalContributions: number, prisma: PrismaType, userId: string) => {
        const delta = reputationDeltaReportContribute(totalContributions);
        await updateReputationHelper(delta, prisma, userId, "ContributedToReport");
    },
});
