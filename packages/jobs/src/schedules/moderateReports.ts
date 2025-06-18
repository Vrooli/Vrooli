import { type Prisma } from "@prisma/client";
import type { PrismaDelegate } from "@vrooli/server";
import { DbProvider, ModelMap, Trigger, batch, findFirstRel, logger } from "@vrooli/server";
import { ModelType, type ModelType as ModelTypeType } from "@vrooli/shared";
import { ReportStatus, ReportSuggestedAction, WEEKS_1_MS, uppercaseFirstLetter } from "@vrooli/shared";

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
const DEFAULT_TIMEOUT = WEEKS_1_MS; // 1 week equivalent

/**
 * Type guard to check if a value is a valid ModelType
 */
function isValidModelType(value: string): value is ModelTypeType {
    return (Object.values(ModelType) as string[]).includes(value);
}

/**
 * Type guard to check if a value is a valid ReportSuggestedAction
 */
function isValidReportSuggestedAction(value: string): value is ReportSuggestedAction {
    return (Object.values(ReportSuggestedAction) as string[]).includes(value);
}

/**
 * Type guard to validate a [string, number] entry as [ReportSuggestedAction, number]
 */
function isValidReportActionEntry(entry: [string, unknown]): entry is [ReportSuggestedAction, number] {
    return typeof entry[0] === 'string' && 
           isValidReportSuggestedAction(entry[0]) && 
           typeof entry[1] === 'number';
}

/**
 * Type guard to check if a value is a PrismaDelegate
 */
function isPrismaDelegate(value: unknown): value is PrismaDelegate {
    return value !== null && 
           typeof value === "object" && 
           "update" in value &&
           typeof value.update === "function";
}

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

// Declare select shape and payload type for reports
const reportSelect = {
    id: true,
    createdAt: true,
    resourceVersion: { select: versionedObjectQuery },
    chatMessage: { select: nonVersionedObjectQuery3 },
    comment: { select: nonVersionedObjectQuery },
    issue: { select: nonVersionedObjectQuery3 },
    tag: { select: nonVersionedObjectQuery3 },
    team: { select: { id: true } },
    user: { select: { id: true } },
    createdBy: { select: { id: true } },
    responses: {
        select: {
            id: true,
            actionSuggested: true,
            createdBy: {
                select: { id: true, reputation: true },
            },
        },
    },
} as const;
type ReportPayload = Prisma.reportGetPayload<{ select: typeof reportSelect }>;

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
    if (filtered.length === 1 && filtered[0] && filtered[0].length > 0) return filtered[0][0];
    // Sort the actions by reputation
    const sorted = filtered.sort((a, b) => b[1] - a[1]);
    // Filter out actions that don't match the reputation of the first action 
    // in the sorted list (i.e. the highest reputation)
    const highest = sorted.length > 0 && sorted[0] && sorted[0].length > 1 ? sorted.filter(([_, rep]) => rep === sorted[0][1]) : [];
    // If there is only one action with the highest reputation, return it
    if (highest.length === 1 && highest[0] && highest[0].length > 0) return highest[0][0];
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
}

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
}

/**
 * Types that can be soft-deleted
 */
const softDeletableTypes = [
    "ResourceVersion",
];

/**
 * Types that don't support being hidden (i.e. don't have "isPrivate" field)
 */
const nonHideableTypes = [
    "ChatMessage",
    "Comment",
    "Issue",
    "Tag",
];

/**
 * Checks if a report should be closed, with its suggested actions executed.
 * If so, performs the moderation action and triggers appropriate event.
 */
async function moderateReport(report: ReportPayload): Promise<void> {
    let acceptedAction: ReportSuggestedAction | null = null;
    // Group responses by action and sum reputation
    const sumsMap: Partial<Record<ReportSuggestedAction, number>> = {};
    
    if (report?.responses && Array.isArray(report.responses)) {
        for (const response of report.responses) {
            const action = response.actionSuggested;
            const rep = response.createdBy?.reputation;
            
            // Validate action and reputation
            if (isValidReportSuggestedAction(action) && typeof rep === 'number') {
                if (sumsMap[action]) {
                    sumsMap[action] = (sumsMap[action] || 0) + rep;
                } else {
                    sumsMap[action] = rep;
                }
            }
        }
    }
    // Try to find a valid action
    // Create properly typed entries from the sumsMap
    const entries = Object.entries(sumsMap);
    const reportActionValues = Object.values(ReportSuggestedAction);
    
    // Validate that all enum values are strings
    if (!reportActionValues.every(value => typeof value === 'string')) {
        logger.error("ReportSuggestedAction enum contains non-string values", {
            enumValues: reportActionValues,
            trace: "0223_enum_validation"
        });
        return;
    }
    
    // Filter and validate entries more safely
    const typedEntries: [ReportSuggestedAction, number][] = [];
    for (const entry of entries) {
        if (isValidReportActionEntry(entry)) {
            typedEntries.push(entry);
        }
    }
    acceptedAction = bestAction(typedEntries);
    // If no action was found, check if the report has been open for too long
    if (!acceptedAction && (Date.now() - report.createdAt.getTime()) > DEFAULT_TIMEOUT) {
        // Add a constant to each reputation score so that they all meet the minimum reputation requirement
        const amountToAdd = Object.values(MIN_REP).reduce((acc, cur) => Math.max(acc, cur), 0);
        // Build bumped actions list with proper validation
        const bumpedActionsList: [ReportSuggestedAction, number][] = [];
        for (const [action, rep] of Object.entries(sumsMap)) {
            if (isValidReportSuggestedAction(action) && typeof rep === 'number') {
                bumpedActionsList.push([action, rep + amountToAdd]);
            }
        }
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
        const relResult = findFirstRel(report, [
            "chatMessage",
            "comment",
            "issue",
            "resourceVersion",
            "tag",
            "team",
            "user",
        ]);
        const objectFieldRaw = relResult.length > 0 ? relResult[0] : null;
        const objectData = relResult.length > 1 ? relResult[1] : null;
        if (!objectFieldRaw || !objectData) {
            logger.error("Failed to complete report moderation. Object likely deleted", { trace: "0409", reportId: report.id, objectField: objectFieldRaw, objectData });
            return;
        }
        
        // Validate that objectFieldRaw is a string before processing
        if (typeof objectFieldRaw !== 'string') {
            logger.error("Invalid object field type", { trace: "0409_field_type", reportId: report.id, objectFieldType: typeof objectFieldRaw });
            return;
        }
        
        const objectField = uppercaseFirstLetter(objectFieldRaw);
        const objectType = objectField;
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
        else if (["Comment"].includes(objectType)) {
            if (objectData.ownedByTeam) {
                objectOwner = { __typename: "Team", id: objectData.ownedByTeam.id };
            }
            else if (objectData.ownedByUser) {
                objectOwner = { __typename: "User", id: objectData.ownedByUser.id };
            }
        }
        else if (["ChatMessage", "Issue", "Tag"].includes(objectType)) {
            if (objectData.createdBy && objectData.createdBy.id) {
                objectOwner = { __typename: "User", id: objectData.createdBy.id };
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
        
        // Validate objectType is a valid ModelType
        if (!isValidModelType(objectType)) {
            logger.error("Invalid ModelType for report moderation", { 
                trace: "0412", 
                reportId: report.id, 
                objectType 
            });
            return;
        }
        
        // Trigger activity
        await Trigger(["en"]).reportActivity({
            objectId: objectData.id,
            objectType: objectType,
            objectOwner,
            reportContributors: report.responses.map(r => r.createdBy?.id?.toString()).filter((id): id is string => typeof id === 'string'),
            reportCreatedById: report.createdBy?.id?.toString() ?? null,
            reportId: report.id.toString(),
            reportStatus: status,
            userUpdatingReportId: null,
        });
        // Get Prisma table for the object type
        const { dbTable } = ModelMap.getLogic(["dbTable"], objectType);
        // Perform moderation action
        switch (acceptedAction) {
            // How delete works:
            // If the object is a version, delete the version. DO NOT delete the root object.
            // If the object can be soft-deleted, soft-delete it.
            case ReportSuggestedAction.Delete:
                const dbModelForDelete = DbProvider.get()[dbTable];
                if (!isPrismaDelegate(dbModelForDelete)) {
                    logger.error("Invalid database model for delete operation", {
                        objectType,
                        dbTable,
                        reportId: report.id,
                    });
                    return;
                }
                
                // Validate that the update method exists
                if (typeof dbModelForDelete.update !== 'function') {
                    logger.error("Database model does not have update method", {
                        objectType,
                        dbTable,
                        reportId: report.id,
                        availableMethods: Object.getOwnPropertyNames(dbModelForDelete).filter(prop => typeof dbModelForDelete[prop] === 'function'),
                    });
                    return;
                }
                
                if (softDeletableTypes.includes(objectType)) {
                    await dbModelForDelete.update({
                        where: { id: objectData.id },
                        data: { isDeleted: true },
                    });
                }
                else {
                    await dbModelForDelete.delete({ where: { id: objectData.id } });
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
                
                const dbModelForHide = DbProvider.get()[dbTable];
                if (!isPrismaDelegate(dbModelForHide)) {
                    logger.error("Invalid database model for hide operation", {
                        objectType,
                        dbTable,
                        reportId: report.id,
                    });
                    return;
                }
                
                // Hide the object
                await dbModelForHide.update({
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
}


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
export async function moderateReports(): Promise<void> {
    try {
        await batch<Prisma.reportFindManyArgs, ReportPayload>({
            objectType: "Report",
            processBatch: async (batch) => {
                await Promise.all(batch.map(async (report) => {
                    await moderateReport(report);
                }));
            },
            select: reportSelect,
            where: {
                status: ReportStatus.Open,
            },
        });
    } catch (error) {
        logger.error("moderateReports caught error", { error, trace: "0464" });
    }
}
