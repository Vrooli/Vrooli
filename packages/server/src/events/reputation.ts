/**
 * Handles giving reputation to users when some action is performed.
 */

import { DAYS_1_S, generatePK, type ModelType } from "@vrooli/shared";
import { DbProvider } from "../db/provider.js";
import { logger } from "../events/logger.js";
import { CacheService } from "../redisConn.js";

// max iterations for reward–threshold loop
const MAX_REWARD_LOOPS = 1000;

export enum ReputationEvent {
    ObjectDeletedFromReport = "ObjectDeletedFromReport",
    ReportWasAccepted = "ReportWasAccepted",
    ReportWasRejected = "ReportWasRejected",
    PullRequestWasAccepted = "PullRequestWasAccepted",
    PullRequestWasRejected = "PullRequestWasRejected",
    IssueCreatedWasAccepted = "IssueCreatedWasAccepted",
    IssueCreatedWasRejected = "IssueCreatedWasRejected",
    PublicApiCreated = "PublicApiCreated",
    PublicCodeCreated = "PublicCodeCreated",
    PublicProjectCreated = "PublicProjectCreated",
    PublicRoutineCreated = "PublicRoutineCreated",
    PublicStandardCreated = "PublicStandardCreated",
    ReceivedVote = "ReceivedVote",
    ReceivedStar = "ReceivedStar",
    ContributedToReport = "ContributedToReport",
}

const MaxReputationGainPerDay = 100;

// Add constant for contributions needed per reputation point
const REPORT_CONTRIBUTIONS_PER_POINT = 10;
const REPUTATION_REDIS_TTL_S = DAYS_1_S;

/**
 * Checks if an object type is tracked by the repuation system
 * @param objectType The object type to check
 * @returns `Public${objectType}Created` if it's a tracked reputation nevent
 */
export function objectReputationEvent<T extends keyof typeof ModelType>(objectType: T): `Public${T}Created` | null {
    return `Public${objectType}Created` in ReputationEvent ? `Public${objectType}Created` : null;
}

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
export function nextRewardThreshold(curr: number, direction: 1 | -1) {
    // Only adjust curr by 1 for negative-to-positive transitions to preserve digit count
    const diffSigns = curr < 0 && direction > 0;
    // Determine magnitude of curr (1 * 10^(digits - 1)). 
    // NOTE: We move curr by 1 when signs are opposite because not doing so would cause 
    // situations like nextRewardThreshold(-10, 1) to return 0 instead of -9.
    // eslint-disable-next-line no-magic-numbers
    const magnitude = Math.pow(10, (Math.abs(curr + (diffSigns ? direction : 0)) + "").length - 1);
    return direction > 0 ?
        Math.ceil((curr + 1) / magnitude) * magnitude :
        Math.floor((curr - 1) / magnitude) * magnitude;
}

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
export function reputationDeltaVote(v0: number, v1: number): number {
    if (typeof v0 !== "number" || typeof v1 !== "number") {
        logger.error("Invalid vote count", { trace: "0101", v0, v1 });
        return NaN;
    }
    if (v0 === v1) return 0;

    const direction = v1 > v0 ? 1 : -1;
    let reputationDelta = 0;
    let curr = v0;
    let loops = 0;

    function pastTarget() {
        return direction > 0 ? curr > v1 : curr < v1;
    }

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
    } while (loops < MAX_REWARD_LOOPS); // Prevent infinite loops

    logger.error("Infinite loop detected in reputationDeltaVote", { trace: "0102", v0, v1, reputationDelta });
    return reputationDelta * direction;
}

/**
 * Generates the reputation which should be rewarded to a user for receiving a star.
 * Same as votes, but double the magnitude.
 */
export function reputationDeltaStar(v0: number, v1: number): number {
    return reputationDeltaVote(v0, v1) * 2;
}

/**
 * Generates the reputation which should be rewarded to a user for contributing to a report. 
 * Should receive a reputation point every 10 contributions
 * @param totalContributions The total number of contributions you've made to any report
 */
export function reputationDeltaReportContribute(totalContributions: number): number {
    // Reward exactly 1 point each time totalContributions is a multiple of the threshold
    if (totalContributions < REPORT_CONTRIBUTIONS_PER_POINT) {
        return 0;
    }
    return totalContributions % REPORT_CONTRIBUTIONS_PER_POINT === 0 ? 1 : 0;
}

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
    PublicApiCreated: 2,
    PublicCodeCreated: 2,
    PublicProjectCreated: 2,
    PublicRoutineCreated: 2,
    PublicStandardCreated: 2,
};

/**
 * Updates the reputation gained today in Redis
 * @param userId The id of the user
 * @param delta The amount of reputation to add to the user's reputation gained today
 * @returns The new reputation gained today
 */
async function getReputationGainedToday(userId: string, delta: number): Promise<{ permitted: boolean; total: number; applied: number }> {
    const keyBase = `reputation-gain-limit-${userId}`;
    try {
        const redisClient = await CacheService.get().raw();
        // Get current count (or 0)
        const currentStr = await redisClient.get(keyBase);
        const current = currentStr !== null ? parseInt(currentStr, 10) : 0;
        // Compute how much of delta can be applied without exceeding daily limits
        const maxAllowed = delta > 0
            ? MaxReputationGainPerDay - current
            : -MaxReputationGainPerDay - current;
        const applied = Math.sign(maxAllowed) === Math.sign(delta)
            ? (Math.abs(delta) <= Math.abs(maxAllowed) ? delta : maxAllowed)
            : 0;
        if (applied === 0) {
            return { permitted: false, total: current, applied: 0 };
        }
        // Atomically apply the permitted delta and set TTL when first created
        const multi = redisClient.multi();
        multi.incrby(keyBase, applied);
        if (currentStr === null) multi.expire(keyBase, REPUTATION_REDIS_TTL_S);
        await multi.exec();
        const newTotal = current + applied;
        return { permitted: true, total: newTotal, applied };
    } catch (error) {
        logger.warn("Redis unavailable, falling back to database for reputation gain", { trace: "0514", error });
    }
    // Fallback to database
    const reputationAward = await DbProvider.get().award.findFirst({
        where: { userId: BigInt(userId), category: "Reputation" },
    });
    const previousProgress = reputationAward?.progress ?? 0;
    // Compute how much can be applied without exceeding daily limits via DB
    const maxAllowedDb = delta > 0
        ? MaxReputationGainPerDay - previousProgress
        : -MaxReputationGainPerDay - previousProgress;
    const appliedDb = Math.sign(maxAllowedDb) === Math.sign(delta)
        ? (Math.abs(delta) <= Math.abs(maxAllowedDb) ? delta : maxAllowedDb)
        : 0;
    return { permitted: appliedDb !== 0, total: previousProgress + appliedDb, applied: appliedDb };
}

/**
 * Helper function to update the reputation of a user. Not to be used directly.
 * @param delta The amount of reputation to add to the user's reputation
 * @param userId The id of the user
 * @param event The event which caused the reputation change
 * @param objectId1 The id of the first object involved in the event, if applicable
 * @param objectId2 The id of the second object involved in the event, if applicable
 * @param options Optional options for the function
 */
async function updateReputationHelper(
    delta: number,
    userId: string,
    event: ReputationEvent | `${ReputationEvent}`,
    objectId1?: string,
    objectId2?: string,
    options?: { skipRateLimit?: boolean },
) {
    // Skip if no change in reputation to avoid unnecessary operations
    if (delta === 0) {
        return;
    }

    // Branch rate-limit logic based on skipRateLimit flag
    let permitted: boolean;
    let totalReputationToday: number;
    let applied: number;

    if (options?.skipRateLimit) {
        // Bypass daily cap
        permitted = delta !== 0;
        applied = delta;
        try {
            const redisClient = await CacheService.get().raw();
            const currentStr = await redisClient.get(`reputation-gain-limit-${userId}`);
            const current = currentStr !== null ? parseInt(currentStr, 10) : 0;
            totalReputationToday = current + applied;
        } catch (error) {
            logger.warn("Failed to read Redis for skipRateLimit", { trace: "0616", userId, error });
            totalReputationToday = applied;
        }
    } else {
        // Original rate-limited path
        const result = await getReputationGainedToday(userId, delta);
        permitted = result.permitted;
        totalReputationToday = result.total;
        applied = result.applied;
    }

    if (!permitted) {
        // outside daily limits, skip update
        return;
    }

    const amount = applied;

    // Apply updates atomically in a database transaction
    await DbProvider.get().$transaction(async (tx) => {
        // Update the award progress to the clamped total
        await tx.award.updateMany({
            where: { userId: BigInt(userId), category: "Reputation" },
            data: { progress: totalReputationToday },
        });
        // Update the user's actual reputation score
        await tx.user.update({
            where: { id: BigInt(userId) },
            data: { reputation: { increment: amount } },
        });
        // Add to user's reputation history, so they can see why their reputation changed
        await tx.reputation_history.create({
            data: {
                id: generatePK(),
                userId: BigInt(userId),
                amount,
                event,
                objectId1: objectId1 ? BigInt(objectId1) : null,
                objectId2: objectId2 ? BigInt(objectId2) : null,
            },
        });
    }, { isolationLevel: "Serializable" });

    // Sync Redis daily counter to the clamped total to prevent drift
    try {
        const redisClient = await CacheService.get().raw();
        await redisClient.set(
            `reputation-gain-limit-${userId}`,
            totalReputationToday,
            "EX",
            REPUTATION_REDIS_TTL_S,
        );
    } catch (error) {
        logger.warn("Failed to sync reputation limit to Redis", {
            trace: "0615",
            userId,
            clampedTotal: totalReputationToday,
            error,
        });
    }
}

/**
 * Handles updating the reputation score for a user. Repuation is stored as an 
 * award category, so all this function does is determine which actions 
 * give what reputation, and then calls the award category update function.
 */
export function Reputation() {
    return {
        /**
         * Deletes a reputation history entry for an object creation, and updates the user's reputation accordingly
         * @param objectId The id of the object that was deleted
         * @param userId The id of the user who created the object
         */
        unCreateObject: async (objectId: string, userId: string) => {
            // Find the reputation history entry for the object creation
            const historyEntry = await DbProvider.get().reputation_history.findFirst({
                where: {
                    objectId1: BigInt(objectId),
                    userId: BigInt(userId),
                    event: {
                        in: [
                            "PublicApiCreated",
                            "PublicCodeCreated",
                            "PublicProjectCreated",
                            "PublicRoutineCreated",
                            "PublicStandardCreated",
                        ],
                    },
                },
            });
            // If the entry exists, delete it and revert the reputation change properly
            if (historyEntry) {
                await DbProvider.get().reputation_history.delete({
                    where: { id: historyEntry.id },
                });
                // Bypass daily cap when reverting creation
                await updateReputationHelper(
                    -historyEntry.amount,
                    userId,
                    historyEntry.event as ReputationEvent,
                    objectId,
                    undefined,
                    { skipRateLimit: true },
                );
            }
        },
        /**
         * Updates a user's reputation based on an event
         * @param event The event that occurred
         * @param userId Typically the user who performed the event, 
         * but can also be the user who owns the object that was affected
         */
        update: async (
            event: Exclude<ReputationEvent, "ReceivedVote" | "ReceivedStar" | "ContributedToReport"> | `${Exclude<ReputationEvent, "ReceivedVote" | "ReceivedStar" | "ContributedToReport">}`,
            userId: string,
            object1Id?: string,
            object2Id?: string,
        ) => {
            // Determine reputation delta
            const delta = reputationMap[event] || 0;
            // Update reputation
            await updateReputationHelper(delta, userId, event, object1Id, object2Id);
        },
        /**
         * Custom reputation update function for votes
         * @param v0 The original vote count
         * @param v1 The new vote count
         * @param userId The user who received the vote
         */
        updateVote: async (v0: number, v1: number, userId: string) => {
            const delta = reputationDeltaVote(v0, v1);
            if (!isFinite(delta)) return;
            await updateReputationHelper(delta, userId, "ReceivedVote");
        },
        /**
         * Custom reputation update function for bookmarks
         * @param v0 The original star count
         * @param v1 The new star count
         * @param userId The user who received the star
         */
        updateStar: async (v0: number, v1: number, userId: string) => {
            const delta = reputationDeltaStar(v0, v1);
            if (!isFinite(delta)) return;
            await updateReputationHelper(delta, userId, "ReceivedStar");
        },
        /**
         * Custom reputation update function for report contributions
         * @param totalContributions The total number of contributions you've made to all reports
         * @param userId The user who contributed to the report
         */
        updateReportContribute: async (totalContributions: number, userId: string) => {
            const delta = reputationDeltaReportContribute(totalContributions);
            await updateReputationHelper(delta, userId, "ContributedToReport");
        },
    };
}
