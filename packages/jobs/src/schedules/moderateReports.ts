import { DbProvider, ModelMap, PrismaDelegate, Trigger, batch, findFirstRel, logger } from "@local/server";
import { ModelType, ReportStatus, ReportSuggestedAction, uppercaseFirstLetter } from "@local/shared";
import pkg, { Prisma } from "@prisma/client";

// Constants for calculating when a moderation action for a report should be accepted
// Minimum reputation sum of users who have suggested a specific moderation action
const MIN_REP: { [key in ReportSuggestedAction]: number } = {
    Delete: 500,
    FalseReport: 100,
    HideUntilFixed: 100,
    NonIssue: 100,
    SuspendUser: 2500,
};
// If report does not meet the minimum reputation for any of the above actions, this is the timeout before 
// the highest voted action is accepted
const DEFAULT_TIMEOUT = 1000 * 60 * 60 * 24 * 7; // 1 week

/**
 * Determines best action to pick from a list of [action, reputation] pairs. 
 * Priorities are as follows:
 * 1. The highest score among the actions that meet the minimum reputation
 * 2. The least severe action if there is a tie in 1. Least to most servere orders are:
 *     - NonIssue
 *     - HideUntilFixed
 *     - FalseReport
 *     - Delete
 *     - SuspendUser 
 * 3. Null if there are no actions that meet the minimum reputation
 *   
 * @param list The list of [action, reputation] pairs
 * @returns The best action to pick or null if there is no best action
 */
function bestAction(list: [ReportSuggestedAction, number][]): ReportSuggestedAction | null {
    // Filter out actions that don't meet the minimum reputation
    const filtered = list.filter(([action, rep]) => rep >= MIN_REP[action]);
    // If there are no actions that meet the minimum reputation, return null
    if (filtered.length === 0) return null;
    // If there is only one action that meets the minimum reputation, return it
    if (filtered.length === 1) return filtered[0][0];
    // Sort the actions by reputation
    const sorted = filtered.sort((a, b) => b[1] - a[1]);
    // Filter out actions that don't match the reputation of the first action 
    // in the sorted list (i.e. the highest reputation)
    const highest = sorted.filter(([_, rep]) => rep === sorted[0][1]);
    // If there is only one action with the highest reputation, return it
    if (highest.length === 1) return highest[0][0];
    // Sort the actions by severity
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
    // Return the least severe action
    return sortedSeverities[0];
};

/**
 * Maps ReportSuggestedAction to ReportStatus
 */
function actionToStatus(action: ReportSuggestedAction): ReportStatus {
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

/**
 * Types that can be soft-deleted
 */
const softDeletableTypes = [
    "ApiVersion",
    "CodeVersion",
    "NoteVersion",
    "Post",
    "ProjectVersion",
    "RoutineVersion",
    "StandardVersion",
];

/**
 * Types that don't support being hidden (i.e. don't have "isPrivate" field)
 */
const nonHideableTypes = [
    "Comment",
    "Issue",
    "Tag",
];

/**
 * Checks if a report should be closed, with its suggested actions executed.
 * If so, performs the moderation action and triggers appropriate event.
 */
async function moderateReport(
    report: pkg.report & {
        responses: (pkg.report_response & { createdBy: pkg.user })[]
    },
): Promise<void> {
    let acceptedAction: ReportSuggestedAction | null = null;
    // Group responses by action and sum reputation
    const sumsMap = report?.responses
        ?.reduce((acc, cur) => {
            const action = cur.actionSuggested;
            const rep = cur.createdBy.reputation;
            if (acc[action]) {
                acc[action] += rep;
            } else {
                acc[action] = rep;
            }
            return acc;
        }, {} as Record<ReportSuggestedAction, number>) ?? {};
    // Try to find a valid action
    acceptedAction = bestAction(Object.entries(sumsMap) as [ReportSuggestedAction, number][]);
    // If no action was found, check if the report has been open for too long
    if (!acceptedAction && (Date.now() - report.createdAt.getTime()) > DEFAULT_TIMEOUT) {
        // Add a constant to each reputation score so that they all meet the minimum reputation requirement
        const amountToAdd = Object.values(MIN_REP).reduce((acc, cur) => Math.max(acc, cur), 0);
        const bumpedActionsList = Object.entries(sumsMap).map(([action, rep]) => [action, rep + amountToAdd]) as [ReportSuggestedAction, number][];
        // Try to find a valid action again
        acceptedAction = bestAction(bumpedActionsList);
    }
    // If accepted
    if (acceptedAction) {
        // Update report
        const status = actionToStatus(acceptedAction);
        await DbProvider.get().report.update({
            where: { id: report.id },
            data: { status },
        });
        // Find the object that was reported.
        // Must capitalize the first letter of the object type to match the __typename
        const [objectField, objectData] = findFirstRel(report, [
            "apiVersion",
            "comment",
            "codeVersion",
            "issue",
            "noteVersion",
            "post",
            "projectVersion",
            "routineVersion",
            "standardVersion",
            "tag",
            "team",
            "user",
        ]).map(([type, data]) => [uppercaseFirstLetter(type ?? ""), data]) as [string, any];
        if (!objectField || !objectData) {
            logger.error("Failed to complete report moderation. Object likely deleted", { trace: "0409", reportId: report.id, objectField, objectData });
            return;
        }
        const objectType = uppercaseFirstLetter(objectField);
        // Find the object's owner
        let objectOwner: { __typename: "Team" | "User", id: string } | null = null;
        if (objectType.endsWith("Version")) {
            if (objectData.root.ownedByTeam) {
                objectOwner = { __typename: "Team", id: objectData.root.ownedByTeam.id };
            }
            else if (objectData.root.ownedByUser) {
                objectOwner = { __typename: "User", id: objectData.root.ownedByUser.id };
            }
        }
        else if (["Post"].includes(objectType)) {
            if (objectData.team) {
                objectOwner = { __typename: "Team", id: objectData.team.id };
            }
            else if (objectData.user) {
                objectOwner = { __typename: "User", id: objectData.user.id };
            }
        }
        else if (["Issue", "Tag"].includes(objectType)) {
            if (objectData.createdBy) {
                objectOwner = { __typename: "User", id: objectData.owner.id };
            }
        }
        else if (["Team"].includes(objectType)) {
            objectOwner = { __typename: "Team", id: objectData.id };
        }
        else if (["User"].includes(objectType)) {
            objectOwner = { __typename: "User", id: objectData.id };
        }
        if (!objectOwner) {
            logger.error("Failed to complete report moderation. Owner not found", { trace: "0410", reportId: report.id, objectType, objectData });
            return;
        }
        // Trigger activity
        await Trigger(["en"]).reportActivity({
            objectId: objectData.id,
            objectType: objectType as ModelType,
            objectOwner,
            reportContributors: report.responses.map(r => r.createdBy?.id).filter(id => id) as string[],
            reportCreatedById: (report as any).createdBy?.id ?? null,
            reportId: report.id,
            reportStatus: status,
            userUpdatingReportId: null,
        });
        // Get Prisma table for the object type
        const { dbTable } = ModelMap.getLogic(["dbTable"], objectType as ModelType);
        // Perform moderation action
        switch (acceptedAction) {
            // How delete works:
            // If the object is a version, delete the version. DO NOT delete the root object.
            // If the object can be soft-deleted, soft-delete it.
            case ReportSuggestedAction.Delete:
                if (softDeletableTypes.includes(objectType)) {
                    await (DbProvider.get()[dbTable] as PrismaDelegate).update({
                        where: { id: objectData.id },
                        data: { isDeleted: true },
                    });
                }
                else {
                    await (DbProvider.get()[dbTable] as PrismaDelegate).delete({ where: { id: objectData.id } });
                }
                break;
            case ReportSuggestedAction.FalseReport:
                // Do nothing
                break;
            case ReportSuggestedAction.HideUntilFixed:
                // Make sure the object can be hidden
                if (nonHideableTypes.includes(objectType)) {
                    logger.error("Failed to complete report moderation. Object cannot be hidden", { trace: "0411", reportId: report.id, objectType, objectData });
                    return;
                }
                // Hide the object
                await (DbProvider.get()[dbTable] as PrismaDelegate).update({
                    where: { id: objectData.id },
                    data: { isPrivate: true },
                });
                break;
            case ReportSuggestedAction.NonIssue:
                // Do nothing
                break;
            case ReportSuggestedAction.SuspendUser:
                //TODO - implement. Can set account status to HardLockout, 
                // but should have timeout that increases with each suspension. 
                // Also, this cannot be available for objects owned by teams. 
                // So need to come up with similarly severe punishment for orgs.
                break;
        }
    }
};

/**
 * Partial query to get data for a versioned object
 */
const versionedObjectQuery = {
    id: true,
    root: {
        select: {
            ownedByTeam: {
                select: { id: true },
            },
            ownedByUser: {
                select: { id: true },
            },
        },
    },
} as const;

/**
 * Partial query to get data for a non-versioned object
 */
const nonVersionedObjectQuery = {
    id: true,
    ownedByTeam: {
        select: { id: true },
    },
    ownedByUser: {
        select: { id: true },
    },
} as const;

/**
 * Partial query for non-versioned objects that don't use 
 * "ownedBy" prefix for the owner field
 */
const nonVersionedObjectQuery2 = {
    id: true,
    team: {
        select: { id: true },
    },
    user: {
        select: { id: true },
    },
} as const;

/**
 * Partial query for non-versioned objects that only have a 
 * createdBy field
 */
const nonVersionedObjectQuery3 = {
    id: true,
    createdBy: {
        select: {
            id: true,
        },
    },
} as const;

/**
 * Handles the moderation of content through report suggestions. It works like this: 
 * 1. A user reports some object with a reason and optional details. The reporter can delete their own report at any time.
 * 2. Other users (cannot be report creator or object owner) suggest a moderation action. These include:
 *      - Deleting the object
 *      - Marking the report as a false report
 *      - Hiding the object until a new version is published that fixes the issue
 *      - Marking the report as a non-issue
 *      - Suspending the object owner (for how long depends on previous suspensions)
 * 3. When enough suggestions are made (accounting for the reputation of the report creator and suggestors), 
 *   the report is automatically accepted and the object is moderated accordingly.
 * 4. Notifications are sent to the relevant users when a decision is made, and reputation scores are updated.
 */
export async function moderateReports() {
    try {
        await batch<Prisma.reportFindManyArgs>({
            objectType: "Report",
            processBatch: async (batch) => {
                Promise.all(batch.map(async (report) => {
                    await moderateReport(report);
                }));
            },
            select: {
                id: true,
                createdAt: true,
                apiVersion: { select: versionedObjectQuery },
                comment: { select: nonVersionedObjectQuery },
                codeVersion: { select: versionedObjectQuery },
                issue: { select: nonVersionedObjectQuery3 },
                noteVersion: { select: versionedObjectQuery },
                post: { select: nonVersionedObjectQuery2 },
                projectVersion: { select: versionedObjectQuery },
                routineVersion: { select: versionedObjectQuery },
                standardVersion: { select: versionedObjectQuery },
                tag: { select: nonVersionedObjectQuery3 },
                team: { select: { id: true } },
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
            where: {
                status: ReportStatus.Open,
            },
        });
    } catch (error) {
        logger.error("moderateReports caught error", { error, trace: "0464" });
    }
};
