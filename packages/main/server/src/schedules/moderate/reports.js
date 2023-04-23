import { ReportSuggestedAction } from "@local/consts";
import { uppercaseFirstLetter } from "@local/utils";
import pkg, { ReportStatus } from "@prisma/client";
import { findFirstRel } from "../../builders";
import { logger, Trigger } from "../../events";
import { getLogic } from "../../getters";
const { PrismaClient } = pkg;
const MIN_REP = {
    Delete: 500,
    FalseReport: 100,
    HideUntilFixed: 100,
    NonIssue: 100,
    SuspendUser: 2500,
};
const DEFAULT_TIMEOUT = 1000 * 60 * 60 * 24 * 7;
const bestAction = (list) => {
    const filtered = list.filter(([action, rep]) => rep >= MIN_REP[action]);
    if (filtered.length === 0)
        return null;
    if (filtered.length === 1)
        return filtered[0][0];
    const sorted = filtered.sort((a, b) => b[1] - a[1]);
    const highest = sorted.filter(([_, rep]) => rep === sorted[0][1]);
    if (highest.length === 1)
        return highest[0][0];
    const severities = [
        ReportSuggestedAction.NonIssue,
        ReportSuggestedAction.HideUntilFixed,
        ReportSuggestedAction.FalseReport,
        ReportSuggestedAction.Delete,
        ReportSuggestedAction.SuspendUser,
    ];
    const sortedSeverities = severities.sort((a, b) => {
        const aIndex = highest.findIndex(([action]) => action === a);
        const bIndex = highest.findIndex(([action]) => action === b);
        return aIndex - bIndex;
    });
    return sortedSeverities[0];
};
const actionToStatus = (action) => {
    switch (action) {
        case ReportSuggestedAction.Delete:
            return ReportStatus.ClosedDeleted;
        case ReportSuggestedAction.FalseReport:
            return ReportStatus.ClosedFalseReport;
        case ReportSuggestedAction.HideUntilFixed:
            return ReportStatus.ClosedHidden;
        case ReportSuggestedAction.NonIssue:
            return ReportStatus.ClosedNonIssue;
        case ReportSuggestedAction.SuspendUser:
            return ReportStatus.ClosedSuspended;
    }
};
const softDeletableTypes = [
    "ApiVersion",
    "NoteVersion",
    "Post",
    "ProjectVersion",
    "RoutineVersion",
    "SmartContractVersion",
    "StandardVersion",
];
const nonHideableTypes = [
    "Comment",
    "Issue",
    "Tag",
];
const moderateReport = async (report, prisma) => {
    let acceptedAction = null;
    const sumsMap = report?.responses
        ?.reduce((acc, cur) => {
        const action = cur.actionSuggested;
        const rep = cur.createdBy.reputation;
        if (acc[action]) {
            acc[action] += rep;
        }
        else {
            acc[action] = rep;
        }
        return acc;
    }, {}) ?? {};
    acceptedAction = bestAction(Object.entries(sumsMap));
    if (!acceptedAction && (Date.now() - report.created_at.getTime()) > DEFAULT_TIMEOUT) {
        const amountToAdd = Object.values(MIN_REP).reduce((acc, cur) => Math.max(acc, cur), 0);
        const bumpedActionsList = Object.entries(sumsMap).map(([action, rep]) => [action, rep + amountToAdd]);
        acceptedAction = bestAction(bumpedActionsList);
    }
    if (acceptedAction) {
        const status = actionToStatus(acceptedAction);
        await prisma.report.update({
            where: { id: report.id },
            data: { status },
        });
        const [objectField, objectData] = findFirstRel(report, [
            "apiVersion",
            "comment",
            "issue",
            "noteVersion",
            "organization",
            "post",
            "projectVersion",
            "question",
            "routineVersion",
            "smartContractVersion",
            "standardVersion",
            "tag",
            "user",
        ]).map(([type, data]) => [uppercaseFirstLetter(type ?? ""), data]);
        if (!objectField || !objectData) {
            logger.error("Failed to complete report moderation. Object likely deleted", { trace: "0409", reportId: report.id, objectField, objectData });
            return;
        }
        const objectType = uppercaseFirstLetter(objectField);
        let objectOwner = null;
        if (objectType.endsWith("Version")) {
            if (objectData.root.ownedByOrganization) {
                objectOwner = { __typename: "Organization", id: objectData.root.ownedByOrganization.id };
            }
            else if (objectData.root.ownedByUser) {
                objectOwner = { __typename: "User", id: objectData.root.ownedByUser.id };
            }
        }
        else if (["Post"].includes(objectType)) {
            if (objectData.organization) {
                objectOwner = { __typename: "Organization", id: objectData.organization.id };
            }
            else if (objectData.user) {
                objectOwner = { __typename: "User", id: objectData.user.id };
            }
        }
        else if (["Issue", "Question", "Tag"].includes(objectType)) {
            if (objectData.createdBy) {
                objectOwner = { __typename: "User", id: objectData.owner.id };
            }
        }
        else if (["Organization"].includes(objectType)) {
            objectOwner = { __typename: "Organization", id: objectData.id };
        }
        else if (["User"].includes(objectType)) {
            objectOwner = { __typename: "User", id: objectData.id };
        }
        if (!objectOwner) {
            logger.error("Failed to complete report moderation. Owner not found", { trace: "0410", reportId: report.id, objectType, objectData });
            return;
        }
        await Trigger(prisma, ["en"]).reportActivity({
            objectId: objectData.id,
            objectType: objectType,
            objectOwner,
            reportContributors: report.responses.map(r => r.createdBy?.id).filter(id => id),
            reportCreatedById: report.createdBy?.id ?? null,
            reportId: report.id,
            reportStatus: status,
            userUpdatingReportId: null,
        });
        const { delegate } = getLogic(["delegate"], objectType, ["en"], "moderateReport");
        switch (acceptedAction) {
            case ReportSuggestedAction.Delete:
                if (softDeletableTypes.includes(objectType)) {
                    await delegate(prisma).update({
                        where: { id: objectData.id },
                        data: { isDeleted: true },
                    });
                }
                else {
                    await delegate(prisma).delete({ where: { id: objectData.id } });
                }
                break;
            case ReportSuggestedAction.FalseReport:
                break;
            case ReportSuggestedAction.HideUntilFixed:
                if (nonHideableTypes.includes(objectType)) {
                    logger.error("Failed to complete report moderation. Object cannot be hidden", { trace: "0411", reportId: report.id, objectType, objectData });
                    return;
                }
                await delegate(prisma).update({
                    where: { id: objectData.id },
                    data: { isPrivate: true },
                });
                break;
            case ReportSuggestedAction.NonIssue:
                break;
            case ReportSuggestedAction.SuspendUser:
                break;
        }
    }
};
const versionedObjectQuery = {
    id: true,
    root: {
        select: {
            ownedByOrganization: {
                select: { id: true },
            },
            ownedByUser: {
                select: { id: true },
            },
        },
    },
};
const nonVersionedObjectQuery = {
    id: true,
    ownedByOrganization: {
        select: { id: true },
    },
    ownedByUser: {
        select: { id: true },
    },
};
const nonVersionedObjectQuery2 = {
    id: true,
    organization: {
        select: { id: true },
    },
    user: {
        select: { id: true },
    },
};
const nonVersionedObjectQuery3 = {
    id: true,
    createdBy: {
        select: {
            id: true,
        },
    },
};
export const checkReportResponses = async () => {
    const prisma = new PrismaClient();
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const batch = await prisma.report.findMany({
            where: {
                status: ReportStatus.Open,
            },
            select: {
                id: true,
                created_at: true,
                apiVersion: { select: versionedObjectQuery },
                comment: { select: nonVersionedObjectQuery },
                issue: { select: nonVersionedObjectQuery3 },
                noteVersion: { select: versionedObjectQuery },
                organization: { select: { id: true } },
                post: { select: nonVersionedObjectQuery2 },
                projectVersion: { select: versionedObjectQuery },
                question: { select: nonVersionedObjectQuery3 },
                routineVersion: { select: versionedObjectQuery },
                smartContractVersion: { select: versionedObjectQuery },
                standardVersion: { select: versionedObjectQuery },
                tag: { select: nonVersionedObjectQuery3 },
                user: { select: { id: true } },
                createdBy: { select: { id: true } },
                responses: {
                    select: {
                        id: true,
                        actionSuggested: true,
                        createdBy: {
                            select: { reputation: true },
                        },
                    },
                },
            },
            skip,
            take: batchSize,
        });
        skip += batchSize;
        currentBatchSize = batch.length;
        for (const report of batch) {
            await moderateReport(report, prisma);
        }
    } while (currentBatchSize === batchSize);
    await prisma.$disconnect();
};
//# sourceMappingURL=reports.js.map