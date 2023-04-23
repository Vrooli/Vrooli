import { initializeRedis } from "../redisConn";
import { logger } from "./logger";
export var ReputationEvent;
(function (ReputationEvent) {
    ReputationEvent["ObjectDeletedFromReport"] = "ObjectDeletedFromReport";
    ReputationEvent["ReportWasAccepted"] = "ReportWasAccepted";
    ReputationEvent["ReportWasRejected"] = "ReportWasRejected";
    ReputationEvent["PullRequestWasAccepted"] = "PullRequestWasAccepted";
    ReputationEvent["PullRequestWasRejected"] = "PullRequestWasRejected";
    ReputationEvent["IssueCreatedWasAccepted"] = "IssueCreatedWasAccepted";
    ReputationEvent["IssueCreatedWasRejected"] = "IssueCreatedWasRejected";
    ReputationEvent["AnsweredQuestion"] = "AnsweredQuestion";
    ReputationEvent["AnsweredQuestionWasAccepted"] = "AnsweredQuestionWasAccepted";
    ReputationEvent["PublicApiCreated"] = "PublicApiCreated";
    ReputationEvent["PublicProjectCreated"] = "PublicProjectCreated";
    ReputationEvent["PublicRoutineCreated"] = "PublicRoutineCreated";
    ReputationEvent["PublicSmartContractCreated"] = "PublicSmartContractCreated";
    ReputationEvent["PublicStandardCreated"] = "PublicStandardCreated";
    ReputationEvent["ReceivedVote"] = "ReceivedVote";
    ReputationEvent["ReceivedStar"] = "ReceivedStar";
    ReputationEvent["ContributedToReport"] = "ContributedToReport";
})(ReputationEvent || (ReputationEvent = {}));
const MaxReputationGainPerDay = 100;
export const objectReputationEvent = (objectType) => {
    return `Public${objectType}Created` in ReputationEvent ? `Public${objectType}Created` : null;
};
export const reputationDeltaVote = (v0, v1) => {
    if (v0 === v1)
        return 0;
    const magnitude = v0 === 0 ? 1 : Math.floor(Math.log10(Math.abs(v0)));
    const isMultiple = v1 % Math.pow(10, magnitude) === 0;
    if (v1 > v0)
        return isMultiple ? 1 : 0;
    return isMultiple ? -1 : 0;
};
export const reputationDeltaStar = (v0, v1) => {
    return reputationDeltaVote(v0, v1) * 2;
};
export const reputationDeltaReportContribute = (totalContributions) => {
    return Math.floor(totalContributions / 10);
};
const reputationMap = {
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
export async function getReputationGainedToday(userId, delta) {
    const keyBase = `reputation-gain-limit-${userId}`;
    try {
        const client = await initializeRedis();
        const updatedReputationCount = await client.incrBy(keyBase, delta);
        if (updatedReputationCount === delta) {
            await client.expire(keyBase, 60 * 60 * 24);
        }
        return updatedReputationCount;
    }
    catch (error) {
        logger.error("Error occured while connecting or accessing redis server", { trace: "0279", error });
        return Number.MAX_SAFE_INTEGER;
    }
}
const updateReputationHelper = async (delta, prisma, userId, event, objectId1, objectId2) => {
    const totalReputationToday = await getReputationGainedToday(userId, delta);
    let updatedReputation = totalReputationToday;
    if (totalReputationToday > MaxReputationGainPerDay) {
        updatedReputation = MaxReputationGainPerDay;
    }
    else if (totalReputationToday < -MaxReputationGainPerDay) {
        updatedReputation = -MaxReputationGainPerDay;
    }
    if (totalReputationToday - delta > MaxReputationGainPerDay || totalReputationToday - delta < -MaxReputationGainPerDay) {
        return;
    }
    await prisma.award.update({
        where: { userId_category: { userId, category: "Reputation" } },
        data: { progress: updatedReputation },
    });
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
export const Reputation = () => ({
    unCreateObject: async (prisma, objectId, userId) => {
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
    update: async (event, prisma, userId, object1Id, object2Id) => {
        const delta = reputationMap[event] || 0;
        await updateReputationHelper(delta, prisma, userId, event, object1Id, object2Id);
    },
    updateVote: async (v0, v1, prisma, userId) => {
        const delta = reputationDeltaVote(v0, v1);
        await updateReputationHelper(delta, prisma, userId, "ReceivedVote");
    },
    updateStar: async (v0, v1, prisma, userId) => {
        const delta = reputationDeltaStar(v0, v1);
        await updateReputationHelper(delta, prisma, userId, "ReceivedStar");
    },
    updateReportContribute: async (totalContributions, prisma, userId) => {
        const delta = reputationDeltaReportContribute(totalContributions);
        await updateReputationHelper(delta, prisma, userId, "ContributedToReport");
    },
});
//# sourceMappingURL=reputation.js.map