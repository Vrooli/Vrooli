import { GqlModelType, InputMaybe, lowercaseFirstLetter, TimeFrame, VisibilityType } from "@local/shared";
import { PeriodType } from "@prisma/client";
import { timeFrameToPrisma } from "../builders/timeFrameToPrisma";
import { visibilityBuilder } from "../builders/visibilityBuilder";
import { SessionUserToken } from "../types";

type Maybe<T> = InputMaybe<T> | undefined

/**
 * Creates a partial query for the ID of a one-to-one relation.
 */
const oneToOneId = <RelField extends string>(
    id: InputMaybe<string> | undefined,
    relField: RelField,
) => ({ [relField]: { id } });

/**
 * Creates a partial query for multiple IDs of a one-to-one relation.
 */
const oneToOneIds = <RelField extends string>(
    ids: InputMaybe<string[]> | undefined,
    relField: RelField,
) => ({ [relField]: { id: { in: ids } } });

/**
 * Creates a partial query for the ID of a one-to-many relation.
 */
const oneToManyId = <RelField extends string>(
    id: InputMaybe<string> | undefined,
    relField: RelField,
) => ({ [relField]: { some: { id } } });

/**
 * Creates a partial query for multiple IDs of a one-to-many relation.
 */
const oneToManyIds = <RelField extends string>(
    ids: InputMaybe<string[]> | undefined,
    relField: RelField,
) => ({ [relField]: { some: { id: { in: ids } } } });

/**
 * Maps any object's search input field to a partial Prisma query.
 * 
 * Example: SearchMap['languages'] = ({ translations: { some: { language: { in: input.languages } } } })
 * 
 * NOTE: This relies on duplicate keys always having the same query, regardless of the object. 
 * This ensures that the database implementation of various relations (bookmarks, views, etc.) is 
 * consistent across all objects.
 */
export const SearchMap = {
    apiId: (id: Maybe<string>) => oneToOneId(id, "api"),
    apisId: (id: Maybe<string>) => oneToManyId(id, "apis"),
    apiVersionId: (id: Maybe<string>) => oneToOneId(id, "apiVersion"),
    apiVersionsId: (id: Maybe<string>) => oneToManyId(id, "apiVersions"),
    bookmarksContainsId: (id: Maybe<string>) => ({
        bookmarks: {
            some: {
                OR: [
                    { api: { id } },
                    { comment: { id } },
                    { issue: { id } },
                    { note: { id } },
                    { organization: { id } },
                    { post: { id } },
                    { project: { id } },
                    { question: { id } },
                    { questionAnswer: { id } },
                    { quiz: { id } },
                    { routine: { id } },
                    { smartContract: { id } },
                    { standard: { id } },
                    { tag: { id } },
                    { user: { id } },
                ],
            },
        },
    }),
    cardLast4: (cardLast4: Maybe<string>) => ({ cardLast4 }),
    chatId: (id: Maybe<string>) => oneToOneId(id, "chat"),
    chatMessageId: (id: Maybe<string>) => oneToOneId(id, "chatMessage"),
    commentId: (id: Maybe<string>) => oneToOneId(id, "comment"),
    commentsId: (id: Maybe<string>) => oneToManyId(id, "comments"),
    completedTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma("completedAt", time),
    closedById: (id: Maybe<string>) => oneToOneId(id, "closedBy"),
    createdById: (id: Maybe<string>) => oneToOneId(id, "createdBy"),
    createdByIdRoot: (id: Maybe<string>) => ({ root: oneToOneId(id, "createdBy") }),
    createdTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma("created_at", time),
    currency: (currency: Maybe<string>) => ({ currency }),
    directoryListingsId: (id: Maybe<string>) => oneToManyId(id, "directoryListings"),
    endTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma("endTime", time),
    excludeIds: (ids: Maybe<string[]>) => ({ NOT: { id: { in: ids } } }),
    excludeLinkedToTag: (exclude: Maybe<boolean>) => exclude === true ? { tagId: null } : {},
    focusModeId: (id: Maybe<string>) => oneToOneId(id, "focusMode"),
    focusModesId: (id: Maybe<string>) => oneToManyId(id, "focusModes"),
    fromId: (id: Maybe<string>) => oneToOneId(id, "from"),
    fromOrganizationId: (id: Maybe<string>) => oneToOneId(id, "fromOrganization"),
    hasAcceptedAnswer: (hasAcceptedAnswer: Maybe<boolean>) => ({ hasAcceptedAnswer }),
    hasCompleteVersion: (hasCompleteVersion: Maybe<boolean>) => ({ hasCompleteVersion }),
    ids: (ids: Maybe<string[]>) => ({ id: { in: ids } }),
    isBot: (isBot: Maybe<boolean>) => ({ isBot }),
    isComplete: (isComplete: Maybe<boolean>) => ({ isComplete }),
    isCompleteWithRoot: (isComplete: Maybe<boolean>) => ({
        AND: [
            { isComplete },
            { root: { hasCompleteVersion: isComplete } },
        ],
    }),
    isCompleteWithRootExcludeOwnedByOrganizationId: (ownedByOrganizationId: Maybe<string>) => ({
        OR: [
            { ownedByOrganizationId },
            {
                AND: [
                    { isComplete: true },
                    { root: { hasCompleteVersion: true } },
                ],
            },
        ],
    }),
    isCompleteWithRootExcludeOwnedByUserId: (ownedByUserId: Maybe<string>) => ({
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
    isInternal: (isInternal: Maybe<boolean>) => ({ isInternal }),
    isInternalWithRoot: (isInternal: Maybe<boolean>) => ({ root: { isInternal } }),
    isInternalWithRootExcludeOwnedByOrganizationId: (ownedByOrganizationId: Maybe<string>) => ({
        OR: [
            { ownedByOrganizationId },
            { root: { isInternal: true } },
        ],
    }),
    isInternalWithRootExcludeOwnedByUserId: (ownedByUserId: Maybe<string>) => ({
        OR: [
            { ownedByUserId },
            { root: { isInternal: true } },
        ],
    }),
    isExternalWithRootExcludeOwnedByOrganizationId: (ownedByOrganizationId: Maybe<string>) => ({
        OR: [
            { ownedByOrganizationId },
            { root: { isInternal: false } },
        ],
    }),
    isExternalWithRootExcludeOwnedByUserId: (ownedByUserId: Maybe<string>) => ({
        OR: [
            { ownedByUserId },
            { root: { isInternal: false } },
        ],
    }),
    isMergedOrRejected: (isMergedOrRejected: Maybe<boolean>) => ({ isMergedOrRejected }),
    isOpenToNewMembers: (isOpenToNewMembers: Maybe<boolean>) => ({ isOpenToNewMembers }),
    isPinned: (isPinned: Maybe<boolean>) => ({ isPinned }),
    issueId: (id: Maybe<string>) => oneToOneId(id, "issue"),
    issuesId: (id: Maybe<string>) => oneToManyId(id, "issues"),
    label: (label: Maybe<string>) => ({ label }),
    labelsId: (id: Maybe<string>) => ({ labels: { some: { label: { id } } } }),
    labelsIds: (ids: Maybe<string[]>) => ({ labels: { some: { label: { id: { in: ids } } } } }),
    languageIn: (languages: Maybe<string[]>) => ({ language: { in: languages } }),
    lastViewedTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma("lastViewedAt", time),
    /**
     * Example: limitTo(["Routine", "Standard"]) => ({ OR: [{ routineId: { NOT: null } }, { standardId: { NOT: null } }] })
     */
    limitTo: (limitTo: Maybe<string[]>) => limitTo ? ({
        OR: limitTo.map((t) => ({ [`${lowercaseFirstLetter(t)}Id`]: { not: null } })),
    }) : {},
    listLabel: (label: Maybe<string>) => ({ list: { label } }),
    listId: (id: Maybe<string>) => oneToOneId(id, "list"),
    maxAmount: (amount: Maybe<number>) => ({ amount: { lte: amount } }),
    maxBookmarks: (bookmarks: Maybe<number>) => ({ bookmarks: { lte: bookmarks } }),
    maxBookmarksRoot: (bookmarks: Maybe<number>) => ({ root: { bookmarks: { lte: bookmarks } } }),
    maxComplexity: (complexity: Maybe<number>) => ({ complexity: { lte: complexity } }),
    maxPointsEarned: (pointsEarned: Maybe<number>) => ({ pointsEarned: { lte: pointsEarned } }),
    maxScore: (score: Maybe<number>) => ({ score: { lte: score } }),
    maxScoreRoot: (score: Maybe<number>) => ({ root: { score: { lte: score } } }),
    maxSimplicity: (simplicity: Maybe<number>) => ({ simplicity: { lte: simplicity } }),
    maxTimesCompleted: (timesCompleted: Maybe<number>) => ({ timesCompleted: { lte: timesCompleted } }),
    maxViews: (views: Maybe<number>) => ({ views: { lte: views } }),
    maxViewsRoot: (views: Maybe<number>) => ({ root: { views: { lte: views } } }),
    memberInOrganizationId: (id: Maybe<string>) => ({ memberships: { some: { organization: { id } } } }),
    memberUserIds: (ids: Maybe<string[]>) => ({ members: { some: { user: { id: { in: ids } } } } }),
    meetingId: (id: Maybe<string>) => oneToOneId(id, "meeting"),
    meetingsId: (id: Maybe<string>) => oneToManyId(id, "meetings"),
    minAmount: (amount: Maybe<number>) => ({ amount: { gte: amount } }),
    minBookmarks: (bookmarks: Maybe<number>) => ({ bookmarks: { gte: bookmarks } }),
    minBookmarksRoot: (bookmarks: Maybe<number>) => ({ root: { bookmarks: { gte: bookmarks } } }),
    minComplexity: (complexity: Maybe<number>) => ({ complexity: { gte: complexity } }),
    minPointsEarned: (pointsEarned: Maybe<number>) => ({ pointsEarned: { gte: pointsEarned } }),
    minScore: (score: Maybe<number>) => ({ score: { gte: score } }),
    minScoreRoot: (score: Maybe<number>) => ({ root: { score: { gte: score } } }),
    minSimplicity: (simplicity: Maybe<number>) => ({ simplicity: { gte: simplicity } }),
    minTimesCompleted: (timesCompleted: Maybe<number>) => ({ timesCompleted: { gte: timesCompleted } }),
    minViews: (views: Maybe<number>) => ({ views: { gte: views } }),
    minViewsRoot: (views: Maybe<number>) => ({ root: { views: { gte: views } } }),
    nodeType: (nodeType: Maybe<string>) => nodeType ? ({ nodeType: { contains: nodeType.trim(), mode: "insensitive" } }) : {},
    notInChatId: (id: Maybe<string>) => id ? ({ NOT: { chats: { some: { id } } } }) : {}, // TODO should probably validate that you can read the participants in this chat, so that you can't figure out who's in a chat by finding out everyone who's not
    noteId: (id: Maybe<string>) => oneToOneId(id, "note"),
    notesId: (id: Maybe<string>) => oneToManyId(id, "notes"),
    noteVersionId: (id: Maybe<string>) => oneToOneId(id, "noteVersion"),
    noteVersionsId: (id: Maybe<string>) => oneToManyId(id, "noteVersions"),
    objectId: (id: Maybe<string>) => oneToOneId(id, "object"),
    objectType: (objectType: Maybe<string>) => objectType ? ({ objectType: { contains: objectType.trim(), mode: "insensitive" } }) : {},
    openToAnyoneWithInvite: () => ({ openToAnyoneWithInvite: true }),
    organizationId: (id: Maybe<string>) => oneToOneId(id, "organization"),
    organizationsId: (id: Maybe<string>) => oneToManyId(id, "organizations"),
    ownedByOrganizationId: (id: Maybe<string>) => oneToOneId(id, "ownedByOrganization"),
    ownedByOrganizationIdRoot: (id: Maybe<string>) => ({ root: oneToOneId(id, "createdBy") }),
    ownedByUserId: (id: Maybe<string>) => oneToOneId(id, "ownedByUser"),
    ownedByUserIdRoot: (id: Maybe<string>) => ({ root: oneToOneId(id, "createdBy") }),
    parentId: (id: Maybe<string>) => oneToOneId(id, "parent"),
    periodTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma("periodEnd", time),
    periodType: (periodType: Maybe<PeriodType>) => ({ periodType }),
    postId: (id: Maybe<string>) => oneToOneId(id, "post"),
    postsId: (id: Maybe<string>) => oneToManyId(id, "posts"),
    projectId: (id: Maybe<string>) => oneToOneId(id, "project"),
    projectsId: (id: Maybe<string>) => oneToManyId(id, "projects"),
    projectVersionId: (id: Maybe<string>) => oneToOneId(id, "projectVersion"),
    projectVersionsId: (id: Maybe<string>) => oneToManyId(id, "projectVersions"),
    pullRequestId: (id: Maybe<string>) => oneToOneId(id, "pullRequest"),
    pullRequestsId: (id: Maybe<string>) => oneToManyId(id, "pullRequests"),
    questionId: (id: Maybe<string>) => oneToOneId(id, "question"),
    questionsId: (id: Maybe<string>) => oneToManyId(id, "questions"),
    questionAnswerId: (id: Maybe<string>) => oneToOneId(id, "questionAnswer"),
    questionAnswersId: (id: Maybe<string>) => oneToManyId(id, "questionAnswers"),
    quizId: (id: Maybe<string>) => oneToOneId(id, "quiz"),
    quizAttemptId: (id: Maybe<string>) => oneToOneId(id, "quizAttempt"),
    quizQuestionId: (id: Maybe<string>) => oneToOneId(id, "quizQuestion"),
    referencedVersionId: (id: Maybe<string>) => ({ referencedVersionId: id }), // Not a relationship, just a field
    reminderListId: (id: Maybe<string>) => oneToOneId(id, "reminderList"),
    repostedFromIds: (ids: Maybe<string[]>) => oneToManyIds(ids, "repostedFrom"),
    reportId: (id: Maybe<string>) => oneToOneId(id, "report"),
    reportsId: (id: Maybe<string>) => oneToManyId(id, "reports"),
    resourceListId: (id: Maybe<string>) => oneToOneId(id, "resourceList"),
    resourceListsId: (id: Maybe<string>) => oneToManyId(id, "resourceLists"),
    responseId: (id: Maybe<string>) => oneToOneId(id, "response"),
    rootId: (id: Maybe<string>) => oneToOneId(id, "root"),
    routineId: (id: Maybe<string>) => oneToOneId(id, "routine"),
    routineIds: (ids: Maybe<string[]>) => oneToOneIds(ids, "routine"),
    routinesId: (id: Maybe<string>) => oneToManyId(id, "routines"),
    routinesIds: (ids: Maybe<string[]>) => oneToManyIds(ids, "routines"),
    routineVersionId: (id: Maybe<string>) => oneToOneId(id, "routineVersion"),
    routineVersionsId: (id: Maybe<string>) => oneToManyId(id, "routineVersions"),
    runProjectOrganizationId: (id: Maybe<string>) => ({ runProject: { organization: { id } } }),
    runProjectUserId: (id: Maybe<string>) => ({ runProject: { user: { id } } }),
    runRoutineOrganizationId: (id: Maybe<string>) => ({ runRoutine: { organization: { id } } }),
    runRoutineUserId: (id: Maybe<string>) => ({ runRoutine: { user: { id } } }),
    scheduleEndTimeFrame: (time: Maybe<TimeFrame>) => ({ schedule: timeFrameToPrisma("endTime", time) }),
    scheduleStartTimeFrame: (time: Maybe<TimeFrame>) => ({ schedule: timeFrameToPrisma("startTime", time) }),
    scheduleForUserId: (userId: Maybe<string>) => userId ? ({
        focusModes: {
            some: {
                user: {
                    id: userId,
                },
            },
        },
        meetings: {
            some: {
                OR: [
                    {
                        organization: {
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
        runProjects: {

        },
        runRoutines: {

        },
    }) : {},
    showOnOrganizationProfile: () => ({ showOnOrganizationProfile: true }),
    silent: (silent: Maybe<boolean>) => ({ silent }),
    smartContractId: (id: Maybe<string>) => oneToOneId(id, "smartContract"),
    smartContractsId: (id: Maybe<string>) => oneToManyId(id, "smartContracts"),
    smartContractType: (smartContractType: Maybe<string>) => smartContractType ? ({ smartContractType: { contains: smartContractType.trim(), mode: "insensitive" } }) : {},
    smartContractVersionId: (id: Maybe<string>) => oneToOneId(id, "smartContractVersion"),
    smartContractVersionsId: (id: Maybe<string>) => oneToManyId(id, "smartContractVersions"),
    standardId: (id: Maybe<string>) => oneToOneId(id, "standard"),
    standardIds: (ids: Maybe<string[]>) => oneToOneIds(ids, "standard"),
    standardType: (standardType: Maybe<string>) => standardType ? ({ standardType: { contains: standardType.trim(), mode: "insensitive" } }) : {},
    standardTypeLatestVersion: (type: Maybe<string>) => type ? ({
        versions: {
            some: {
                isLatest: true,
                type: { contains: type.trim(), mode: "insensitive" },
            },
        },
    }) : {},
    standardsId: (id: Maybe<string>) => oneToManyId(id, "standards"),
    standardsIds: (ids: Maybe<string[]>) => oneToManyIds(ids, "standards"),
    standardVersionId: (id: Maybe<string>) => oneToOneId(id, "standardVersion"),
    standardVersionsId: (id: Maybe<string>) => oneToManyId(id, "standardVersions"),
    startTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma("startTime", time),
    startedTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma("startedAt", time),
    status: <T>(status: Maybe<T>) => ({ status }),
    statuses: <T>(statuses: Maybe<T[]>) => ({ status: { in: statuses } }),
    tagId: (id: Maybe<string>) => oneToOneId(id, "tag"),
    tagsId: (id: Maybe<string>) => oneToManyId(id, "tags"),
    tags: (tags: Maybe<string[]>) => ({ tags: { some: { tag: { tag: { in: tags } } } } }),
    tagsRoot: (tags: Maybe<string[]>) => ({ root: { tags: { some: { tag: { tag: { in: tags } } } } } }),
    timeZone: (timeZone: Maybe<string>) => timeZone ? ({ timeZone: { contains: timeZone.trim(), mode: "insensitive" } }) : {},
    toId: (id: Maybe<string>) => oneToOneId(id, "to"),
    toOrganizationId: (id: Maybe<string>) => oneToOneId(id, "toOrganization"),
    toUserId: (id: Maybe<string>) => oneToOneId(id, "toUser"),
    transferId: (id: Maybe<string>) => oneToOneId(id, "transfer"),
    transfersId: (id: Maybe<string>) => oneToManyId(id, "transfers"),
    translationLanguages: (languages: Maybe<string[]>) => ({ translations: { some: { language: { in: languages } } } }),
    translationLanguagesLatestVersion: (languages: Maybe<string[]>) => ({
        versions: {
            some: {
                isLatest: true,
                translations: { some: { language: { in: languages } } },
            },
        },
    }),
    updatedTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma("updated_at", time),
    userId: (id: Maybe<string>) => oneToOneId(id, "user"),
    usersId: (id: Maybe<string>) => oneToManyId(id, "users"),
    visibility: (
        visibility: InputMaybe<VisibilityType> | undefined,
        userData: SessionUserToken | null | undefined,
        objectType: `${GqlModelType}`,
    ) => visibilityBuilder({ objectType, userData, visibility }),
};
