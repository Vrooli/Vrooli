import { type ModelType, type ResourceSubType, type ResourceType, type ScheduleFor, type TimeFrame, VisibilityType, lowercaseFirstLetter } from "@vrooli/shared";
import { type PeriodType } from "@prisma/client";
import { type RequestService } from "../auth/request.js";
import { timeFrameToPrisma } from "../builders/timeFrame.js";
import { visibilityBuilderPrisma } from "../builders/visibilityBuilder.js";

/**
 * Creates a partial query for the ID of a one-to-one relation.
 */
function oneToOneId<RelField extends string>(id: string, relField: RelField) {
    return {
        [relField]: { id },
    } as const;
}

/**
 * Creates a partial query for multiple IDs of a one-to-one relation.
 */
function oneToOneIds<RelField extends string>(ids: string[], relField: RelField) {
    return {
        [relField]: { id: { in: ids } },
    } as const;
}

/**
 * Creates a partial query for the ID of a one-to-many relation.
 */
function oneToManyId<RelField extends string>(id: string, relField: RelField) {
    return {
        [relField]: { some: { id } },
    } as const;
}

/**
 * Creates a partial query for multiple IDs of a one-to-many relation.
 */
function oneToManyIds<RelField extends string>(ids: string[], relField: RelField) {
    return {
        [relField]: { some: { id: { in: ids } } },
    } as const;
}

/**
 * Supplemental data passed into SearchMap. Useful for building queries that need 
 * access to the user's session data, the search visibility, etc.
 */
type RequestData = {
    /** The object type being queried */
    objectType: ModelType | `${ModelType}`;
    /** The current request */
    req: Parameters<typeof RequestService.assertRequestFrom>[0],
    /** Full search input query */
    searchInput: { [x: string]: any };
    /** The visibility of the query */
    visibility: VisibilityType;
};

type SearchFunction = (inputField: unknown, requestData: RequestData) => object;

type SearchMapType<T> = {
    [P in keyof T]: SearchFunction;
};

/**
 * Maps any object's search input field to a partial Prisma query.
 * 
 * Example: SearchMap['languages'] = ({ translations: { some: { language: { in: input.languages } } } })
 * 
 * NOTE: This relies on duplicate keys always having the same query, regardless of the object. 
 * This ensures that the database implementation of various relations (bookmarks, views, etc.) is 
 * consistent across all objects.
 */
export const SearchMap: { [key in string]?: SearchFunction } = {
    bookmarksContainsId: (id: string) => ({
        bookmarks: {
            some: {
                OR: [
                    { api: { id } },
                    { code: { id } },
                    { comment: { id } },
                    { issue: { id } },
                    { note: { id } },
                    { project: { id } },
                    { routine: { id } },
                    { standard: { id } },
                    { tag: { id } },
                    { team: { id } },
                    { user: { id } },
                ],
            },
        },
    }),
    calledByRoutineVersionId: (id: string) => oneToManyId(id, "calledByRoutineVersion"),
    cardLast4: (cardLast4: string) => ({ cardLast4 }),
    chatId: (id: string) => oneToOneId(id, "chat"),
    chatMessageId: (id: string) => oneToOneId(id, "chatMessage"),
    codeLanguage: (codeLanguage: string) => codeLanguage ? ({ codeLanguage: { contains: codeLanguage.trim(), mode: "insensitive" } }) : {},
    codeLanguageLatestVersion: (language: string) => language ? ({
        versions: {
            some: {
                isLatest: true,
                codeLanguage: { contains: language.trim(), mode: "insensitive" },
            },
        },
    }) : {},
    commentId: (id: string) => oneToOneId(id, "comment"),
    commentsId: (id: string) => oneToManyId(id, "comments"),
    completedTimeFrame: (time: TimeFrame) => timeFrameToPrisma("completedAt", time),
    closedById: (id: string) => oneToOneId(id, "closedBy"),
    createdById: (id: string) => oneToOneId(id, "createdBy"),
    createdByIdRoot: (id: string) => ({ root: oneToOneId(id, "createdBy") }),
    createdTimeFrame: (time: TimeFrame) => timeFrameToPrisma("createdAt", time),
    creatorId: (id: string) => oneToOneId(id, "creator"),
    currency: (currency: string) => ({ currency }),
    endTimeFrame: (time: TimeFrame) => timeFrameToPrisma("endTime", time),
    excludeIds: (ids: string[]) => ({ NOT: { id: { in: ids } } }),
    excludeLinkedToTag: (exclude: boolean) => exclude === true ? { tagId: null } : {},
    fromId: (id: string) => oneToOneId(id, "from"),
    fromTeamId: (id: string) => oneToOneId(id, "fromTeam"),
    hasAcceptedAnswer: (hasAcceptedAnswer: boolean) => ({ hasAcceptedAnswer }),
    hasCompleteVersion: (hasCompleteVersion: boolean) => ({ hasCompleteVersion }),
    ids: (ids: string[]) => ({ id: { in: ids } }),
    isAdmin: (isAdmin: boolean) => ({ isAdmin }),
    isBot: (isBot: boolean) => ({ isBot }),
    isBotDepictingPerson: (isBotDepictingPerson: boolean) => ({ isBotDepictingPerson }),
    isComplete: (isComplete: boolean) => ({ isComplete }),
    isCompleteWithRoot: (isComplete: boolean) => ({
        AND: [
            { isComplete },
            { root: { hasCompleteVersion: isComplete } },
        ],
    }),
    isCompleteWithRootExcludeOwnedByTeamId: (ownedByTeamId: string) => ({
        OR: [
            { ownedByTeamId },
            {
                AND: [
                    { isComplete: true },
                    { root: { hasCompleteVersion: true } },
                ],
            },
        ],
    }),
    isCompleteWithRootExcludeOwnedByUserId: (ownedByUserId: string) => ({
        OR: [
            { ownedByUserId },
            {
                AND: [
                    { isComplete: true },
                    { root: { hasCompleteVersion: true } },
                ],
            },
        ],
    }),
    isInternal: (isInternal: boolean) => ({ isInternal }),
    isInternalWithRoot: (isInternal: boolean) => ({ root: { isInternal } }),
    isInternalWithRootExcludeOwnedByTeamId: (ownedByTeamId: string) => ({
        OR: [
            { ownedByTeamId },
            { root: { isInternal: true } },
        ],
    }),
    isInternalWithRootExcludeOwnedByUserId: (ownedByUserId: string) => ({
        OR: [
            { ownedByUserId },
            { root: { isInternal: true } },
        ],
    }),
    isExternalWithRootExcludeOwnedByTeamId: (ownedByTeamId: string) => ({
        OR: [
            { ownedByTeamId },
            { root: { isInternal: false } },
        ],
    }),
    isExternalWithRootExcludeOwnedByUserId: (ownedByUserId: string) => ({
        OR: [
            { ownedByUserId },
            { root: { isInternal: false } },
        ],
    }),
    isLatest: (isLatest: boolean) => ({ isLatest }),
    isMergedOrRejected: (isMergedOrRejected: boolean) => ({ isMergedOrRejected }),
    isOpenToNewMembers: (isOpenToNewMembers: boolean) => ({ isOpenToNewMembers }),
    isPinned: (isPinned: boolean) => ({ isPinned }),
    issueId: (id: string) => oneToOneId(id, "issue"),
    issuesId: (id: string) => oneToManyId(id, "issues"),
    label: (label: string) => ({ label }),
    labelsId: (id: string) => ({ labels: { some: { label: { id } } } }),
    labelsIds: (ids: string[]) => ({ labels: { some: { label: { id: { in: ids } } } } }),
    languageIn: (languages: string[]) => ({ language: { in: languages } }),
    lastViewedTimeFrame: (time: TimeFrame) => timeFrameToPrisma("lastViewedAt", time),
    latestVersionResourceSubType: (resourceSubType: ResourceSubType, { visibility }) => {
        // If visibility is "Public", then we must use "isLatestPublic" flag
        if (visibility === VisibilityType.Public) {
            return { versions: { some: { isLatestPublic: true, resourceSubType } } };
        }
        // Otherwise, use "isLatest" flag or "isLatestPublic" flag. The visibility builder will handle omitting objects you're not allowed to see.
        // TODO probably flawed if using visibility "All"
        return {
            versions: {
                some: {
                    OR: [
                        { isLatest: true },
                        { isLatestPublic: true },
                    ],
                    resourceSubType,
                },
            },
        };
    },
    latestVersionResourceSubTypes: (resourceSubTypes: ResourceSubType[], { visibility }) => {
        // If visibility is "Public", then we must use "isLatestPublic" flag
        if (visibility === VisibilityType.Public) {
            return { versions: { some: { isLatestPublic: true, resourceSubType: { in: resourceSubTypes } } } };
        }
        // Otherwise, use "isLatest" flag or "isLatestPublic" flag. The visibility builder will handle omitting objects you're not allowed to see.
        // TODO probably flawed if using visibility "All"
        return {
            versions: {
                some: {
                    OR: [
                        { isLatest: true },
                        { isLatestPublic: true },
                    ],
                    resourceSubType: { in: resourceSubTypes },
                },
            },
        };
    },
    /**
     * Example: limitTo(["Routine", "Standard"]) => ({ OR: [{ routineId: { NOT: null } }, { standardId: { NOT: null } }] })
     */
    limitTo: (limitTo: string[]) => limitTo ? ({
        OR: limitTo.map((t) => ({ [`${lowercaseFirstLetter(t)}Id`]: { not: null } })),
    }) : {},
    listLabel: (label: string) => ({ list: { label } }),
    listId: (id: string) => oneToOneId(id, "list"),
    maxAmount: (amount: number) => ({ amount: { lte: amount } }),
    maxBookmarks: (bookmarks: number) => ({ bookmarks: { lte: bookmarks } }),
    maxBookmarksRoot: (bookmarks: number) => ({ root: { bookmarks: { lte: bookmarks } } }),
    maxComplexity: (complexity: number) => ({ complexity: { lte: complexity } }),
    maxPointsEarned: (pointsEarned: number) => ({ pointsEarned: { lte: pointsEarned } }),
    maxScore: (score: number) => ({ score: { lte: score } }),
    maxScoreRoot: (score: number) => ({ root: { score: { lte: score } } }),
    maxTimesCompleted: (timesCompleted: number) => ({ timesCompleted: { lte: timesCompleted } }),
    maxViews: (views: number) => ({ views: { lte: views } }),
    maxViewsRoot: (views: number) => ({ root: { views: { lte: views } } }),
    memberInTeamId: (id: string) => ({ memberships: { some: { team: { id } } } }),
    memberUserIds: (ids: string[]) => ({ members: { some: { user: { id: { in: ids } } } } }),
    meetingId: (id: string) => oneToOneId(id, "meeting"),
    meetingsId: (id: string) => oneToManyId(id, "meetings"),
    minAmount: (amount: number) => ({ amount: { gte: amount } }),
    minBookmarks: (bookmarks: number) => ({ bookmarks: { gte: bookmarks } }),
    minBookmarksRoot: (bookmarks: number) => ({ root: { bookmarks: { gte: bookmarks } } }),
    minComplexity: (complexity: number) => ({ complexity: { gte: complexity } }),
    minPointsEarned: (pointsEarned: number) => ({ pointsEarned: { gte: pointsEarned } }),
    minScore: (score: number) => ({ score: { gte: score } }),
    minScoreRoot: (score: number) => ({ root: { score: { gte: score } } }),
    minTimesCompleted: (timesCompleted: number) => ({ timesCompleted: { gte: timesCompleted } }),
    minViews: (views: number) => ({ views: { gte: views } }),
    minViewsRoot: (views: number) => ({ root: { views: { gte: views } } }),
    nodeType: (nodeType: string) => nodeType ? ({ nodeType: { contains: nodeType.trim(), mode: "insensitive" } }) : {},
    notInChatId: (id: string) => id ? ({ NOT: { chats: { some: { id } } } }) : {}, // TODO should probably validate that you can read the participants in this chat, so that you can't figure out who's in a chat by finding out everyone who's not
    notInvitedToTeamId: (id: string) => ({ membershipsInvited: { none: { team: { id } } } }),
    notMemberInTeamId: (id: string) => ({ memberships: { none: { team: { id } } } }),
    objectId: (id: string) => oneToOneId(id, "object"),
    objectType: (objectType: string) => objectType ? ({ objectType: { contains: objectType.trim(), mode: "insensitive" } }) : {},
    openToAnyoneWithInvite: () => ({ openToAnyoneWithInvite: true }),
    ownedByTeamId: (id: string) => oneToOneId(id, "ownedByTeam"),
    ownedByTeamIdRoot: (id: string) => ({ root: oneToOneId(id, "createdBy") }),
    ownedByUserId: (id: string) => oneToOneId(id, "ownedByUser"),
    ownedByUserIdRoot: (id: string) => ({ root: oneToOneId(id, "createdBy") }),
    parentId: (id: string) => oneToOneId(id, "parent"),
    periodTimeFrame: (time: TimeFrame) => timeFrameToPrisma("periodEnd", time),
    periodType: (periodType: PeriodType) => ({ periodType }),
    pullRequestId: (id: string) => oneToOneId(id, "pullRequest"),
    pullRequestsId: (id: string) => oneToManyId(id, "pullRequests"),
    referencedVersionId: (id: string) => ({ referencedVersionId: id }), // Not a relationship, just a field
    reminderListId: (id: string) => oneToOneId(id, "reminderList"),
    reportId: (id: string) => oneToOneId(id, "report"),
    reportsId: (id: string) => oneToManyId(id, "reports"),
    responseId: (id: string) => oneToOneId(id, "response"),
    rootId: (id: string) => oneToOneId(id, "root"),
    rootResourceType: (resourceType: ResourceType) => ({ root: { resourceType } }),
    resourceType: (resourceType: ResourceType) => ({ resourceType }),
    resourceTypes: (resourceTypes: ResourceType[]) => ({ resourceType: { in: resourceTypes } }),
    runTeamId: (id: string) => ({ run: { team: { id } } }),
    runUserId: (id: string) => ({ run: { user: { id } } }),
    scheduleEndTimeFrame: (time: TimeFrame) => ({ schedule: timeFrameToPrisma("endTime", time) }),
    scheduleStartTimeFrame: (time: TimeFrame) => ({ schedule: timeFrameToPrisma("startTime", time) }),
    scheduleFor: (scheduleFor: ScheduleFor) => {
        switch (scheduleFor) {
            case "Meeting": return { meetings: { some: {} } };
            case "Run": return { runs: { some: {} } };
            default: return {};
        }
    },
    scheduleForUserId: (userId: string) => userId ? ({
        meetings: {
            some: {
                OR: [
                    {
                        team: {
                            members: {
                                some: {
                                    user: {
                                        id: userId,
                                    },
                                },
                            },
                        },
                    },
                    {
                        invites: {
                            some: {
                                user: {
                                    id: userId,
                                },
                            },
                        },
                    },
                    {
                        attendees: {
                            some: {
                                user: {
                                    id: userId,
                                },
                            },
                        },
                    },
                ],
            },
        },
        runs: {

        },
    }) : {},
    showOnTeamProfile: () => ({ showOnTeamProfile: true }),
    silent: (silent: boolean) => ({ silent }),
    startTimeFrame: (time: TimeFrame) => timeFrameToPrisma("startTime", time),
    startedTimeFrame: (time: TimeFrame) => timeFrameToPrisma("startedAt", time),
    status: <T>(status: T) => ({ status }),
    statuses: <T>(statuses: T[]) => ({ status: { in: statuses } }),
    tagId: (id: string) => oneToOneId(id, "tag"),
    tagsId: (id: string) => oneToManyId(id, "tags"),
    tags: (tags: string[]) => ({ tags: { some: { tag: { tag: { in: tags } } } } }),
    tagsRoot: (tags: string[]) => ({ root: { tags: { some: { tag: { tag: { in: tags } } } } } }),
    teamId: (id: string) => oneToOneId(id, "team"),
    teamsId: (id: string) => oneToManyId(id, "teams"),
    timeZone: (timeZone: string) => timeZone ? ({ timeZone: { contains: timeZone.trim(), mode: "insensitive" } }) : {},
    toId: (id: string) => oneToOneId(id, "to"),
    toTeamId: (id: string) => oneToOneId(id, "toTeam"),
    toUserId: (id: string) => oneToOneId(id, "toUser"),
    transferId: (id: string) => oneToOneId(id, "transfer"),
    transfersId: (id: string) => oneToManyId(id, "transfers"),
    translationLanguages: (languages: string[]) => ({ translations: { some: { language: { in: languages } } } }),
    translationLanguagesLatestVersion: (languages: string[]) => ({
        versions: {
            some: {
                isLatest: true,
                translations: { some: { language: { in: languages } } },
            },
        },
    }),
    updatedTimeFrame: (time: TimeFrame) => timeFrameToPrisma("updatedAt", time),
    userId: (id: string) => oneToOneId(id, "user"),
    usersId: (id: string) => oneToManyId(id, "users"),
    visibility: (visibility: VisibilityType | null | undefined, { objectType, req, searchInput }) => visibilityBuilderPrisma({ objectType, req, searchInput, visibility }).query,
} as SearchMapType<typeof SearchMap>;
