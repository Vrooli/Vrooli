import { timeFrameToPrisma, visibilityBuilder } from "../builders";
import { InputMaybe, SessionUser, TimeFrame, VisibilityType } from "../endpoints/types";
import { GraphQLModelType } from "../models/types";

type Maybe<T> = InputMaybe<T> | undefined

/**
 * Creates a partial query for the ID of a one-to-one relation.
 */
const oneToOneId = <RelField extends string>(
    id: InputMaybe<string> | undefined,
    relField: RelField
) => ({ [relField]: { id } });

/**
 * Creates a partial query for multiple IDs of a one-to-one relation.
 */
const oneToOneIds = <RelField extends string>(
    ids: InputMaybe<string[]> | undefined,
    relField: RelField
) => ({ [relField]: { id: { in: ids } } });

/**
 * Creates a partial query for the ID of a one-to-many relation.
 */
const oneToManyId = <RelField extends string>(
    id: InputMaybe<string> | undefined,
    relField: RelField
) => ({ [relField]: { some: { id } } });

/**
 * Creates a partial query for multiple IDs of a one-to-many relation.
 */
const oneToManyIds = <RelField extends string>(
    ids: InputMaybe<string[]> | undefined,
    relField: RelField
) => ({ [relField]: { some: { id: { in: ids } } } });

/**
 * Maps any object's search input field to a partial Prisma query.
 * 
 * Example: SearchMap['languages'] = ({ translations: { some: { language: { in: input.languages } } } })
 * 
 * NOTE: This relies on duplicate keys always having the same query, regardless of the object. 
 * This ensures that the database implementation of various relations (stars, views, etc.) is 
 * consistent across all objects.
 */
export const SearchMap = {
    apiId: (id: Maybe<string>) => oneToOneId(id, 'api'),
    apisId: (id: Maybe<string>) => oneToManyId(id, 'apis'),
    apiVersionId: (id: Maybe<string>) => oneToOneId(id, 'apiVersion'),
    apiVersionsId: (id: Maybe<string>) => oneToManyId(id, 'apiVersions'),
    commentId: (id: Maybe<string>) => oneToOneId(id, 'comment'),
    commentsId: (id: Maybe<string>) => oneToManyId(id, 'comments'),
    completedTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma('completedAt', time),
    closedById: (id: Maybe<string>) => oneToOneId(id, 'closedBy'),
    createdById: (id: Maybe<string>) => oneToOneId(id, 'createdBy'),
    createdTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma('created_at', time),
    directoryListingsId: (id: Maybe<string>) => oneToManyId(id, 'directoryListings'),
    excludeIds: (ids: Maybe<string[]>) => ({ NOT: { id: { in: ids } } }),
    excludeLinkedToTag: (exclude: Maybe<boolean>) => exclude === true ? { tagId: null } : {},
    fromId: (id: Maybe<string>) => oneToOneId(id, 'from'),
    hasCompleteVersion: (hasCompleteVersion: Maybe<boolean>) => ({ hasCompleteVersion }),
    ids: (ids: Maybe<string[]>) => ({ id: { in: ids } }),
    isComplete: (isComplete: Maybe<boolean>) => ({ isComplete }),
    isCompleteWithRoot: (isComplete: Maybe<boolean>) => ({
        AND: [
            { isComplete },
            { root: { isComplete } }
        ]
    }),
    isCompleteWithRootExcludeOwnedByOrganizationId: (ownedByOrganizationId: Maybe<string>) => ({
        OR: [
            { ownedByOrganizationId },
            {
                AND: [
                    { isComplete: true },
                    { root: { isComplete: true } }
                ]
            }
        ]
    }),
    isCompleteWithRootExcludeOwnedByUserId: (ownedByUserId: Maybe<string>) => ({
        OR: [
            { ownedByUserId },
            {
                AND: [
                    { isComplete: true },
                    { root: { isComplete: true } }
                ]
            }
        ]
    }),
    isInternal: (isInternal: Maybe<boolean>) => ({ isInternal }),
    isInternalWithRoot: (isInternal: Maybe<boolean>) => ({
        AND: [
            { isInternal },
            { root: { isInternal } }
        ]
    }),
    isInternalWithRootExcludeOwnedByOrganizationId: (ownedByOrganizationId: Maybe<string>) => ({
        OR: [
            { ownedByOrganizationId },
            {
                AND: [
                    { isInternal: true },
                    { root: { isInternal: true } }
                ]
            }
        ]
    }),
    isInternalWithRootExcludeOwnedByUserId: (ownedByUserId: Maybe<string>) => ({
        OR: [
            { ownedByUserId },
            {
                AND: [
                    { isInternal: true },
                    { root: { isInternal: true } }
                ]
            }
        ]
    }),
    isOpenToNewMembers: () => ({ isOpenToNewMembers: true }),
    issueId: (id: Maybe<string>) => oneToOneId(id, 'issue'),
    issuesId: (id: Maybe<string>) => oneToManyId(id, 'issues'),
    label: (label: Maybe<string>) => ({ label }),
    labelsId: (id: Maybe<string>) => ({ labels: { some: { label: { id } } } }),
    languageIn: (languages: Maybe<string[]>) => ({ language: { in: languages } }),
    lastViewedTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma('lastViewedAt', time),
    maxComplexity: (complexity: Maybe<number>) => ({ complexity: { lte: complexity } }),
    maxScore: (score: Maybe<number>) => ({ score: { lte: score } }),
    maxScoreRoot: (score: Maybe<number>) => ({ root: { score: { lte: score } } }),
    maxSimplicity: (simplicity: Maybe<number>) => ({ simplicity: { lte: simplicity } }),
    maxStars: (stars: Maybe<number>) => ({ stars: { lte: stars } }),
    maxStarsRoot: (stars: Maybe<number>) => ({ root: { stars: { lte: stars } } }),
    maxTimesCompleted: (timesCompleted: Maybe<number>) => ({ timesCompleted: { lte: timesCompleted } }),
    maxViews: (views: Maybe<number>) => ({ views: { lte: views } }),
    maxViewsRoot: (views: Maybe<number>) => ({ root: { views: { lte: views } } }),
    memberUserIds: (ids: Maybe<string[]>) => ({ members: { some: { user: { id: { in: ids } } } } }),
    meetingId: (id: Maybe<string>) => oneToOneId(id, 'meeting'),
    meetingsId: (id: Maybe<string>) => oneToManyId(id, 'meetings'),
    minComplexity: (complexity: Maybe<number>) => ({ complexity: { gte: complexity } }),
    minScore: (score: Maybe<number>) => ({ score: { gte: score } }),
    minScoreRoot: (score: Maybe<number>) => ({ root: { score: { gte: score } } }),
    minSimplicity: (simplicity: Maybe<number>) => ({ simplicity: { gte: simplicity } }),
    minStars: (stars: Maybe<number>) => ({ stars: { gte: stars } }),
    minStarsRoot: (stars: Maybe<number>) => ({ root: { stars: { gte: stars } } }),
    minTimesCompleted: (timesCompleted: Maybe<number>) => ({ timesCompleted: { gte: timesCompleted } }),
    minViews: (views: Maybe<number>) => ({ views: { gte: views } }),
    minViewsRoot: (views: Maybe<number>) => ({ root: { views: { gte: views } } }),
    noteId: (id: Maybe<string>) => oneToOneId(id, 'note'),
    notesId: (id: Maybe<string>) => oneToManyId(id, 'notes'),
    noteVersionId: (id: Maybe<string>) => oneToOneId(id, 'noteVersion'),
    noteVersionsId: (id: Maybe<string>) => oneToManyId(id, 'noteVersions'),
    openToAnyoneWithInvite: () => ({ openToAnyoneWithInvite: true }),
    organizationId: (id: Maybe<string>) => oneToOneId(id, 'organization'),
    organizationsId: (id: Maybe<string>) => oneToManyId(id, 'organizations'),
    ownedByOrganizationId: (id: Maybe<string>) => oneToOneId(id, 'ownedByOrganization'),
    ownedByUserId: (id: Maybe<string>) => oneToOneId(id, 'ownedByUser'),
    parentId: (id: Maybe<string>) => oneToOneId(id, 'parent'),
    postId: (id: Maybe<string>) => oneToOneId(id, 'post'),
    postsId: (id: Maybe<string>) => oneToManyId(id, 'posts'),
    projectId: (id: Maybe<string>) => oneToOneId(id, 'project'),
    projectsId: (id: Maybe<string>) => oneToManyId(id, 'projects'),
    projectVersionId: (id: Maybe<string>) => oneToOneId(id, 'projectVersion'),
    projectVersionsId: (id: Maybe<string>) => oneToManyId(id, 'projectVersions'),
    pullRequestId: (id: Maybe<string>) => oneToOneId(id, 'pullRequest'),
    pullRequestsId: (id: Maybe<string>) => oneToManyId(id, 'pullRequests'),
    questionId: (id: Maybe<string>) => oneToOneId(id, 'question'),
    questionsId: (id: Maybe<string>) => oneToManyId(id, 'questions'),
    questionAnswerId: (id: Maybe<string>) => oneToOneId(id, 'questionAnswer'),
    questionAnswersId: (id: Maybe<string>) => oneToManyId(id, 'questionAnswers'),
    referencedVersionId: (id: Maybe<string>) => ({ referencedVersionId: id }), // Not a relationship, just a field
    reportId: (id: Maybe<string>) => oneToOneId(id, 'report'),
    reportsId: (id: Maybe<string>) => oneToManyId(id, 'reports'),
    resourceListId: (id: Maybe<string>) => oneToOneId(id, 'resourceList'),
    resourceListsId: (id: Maybe<string>) => oneToManyId(id, 'resourceLists'),
    rootId: (id: Maybe<string>) => oneToOneId(id, 'root'),
    routineId: (id: Maybe<string>) => oneToOneId(id, 'routine'),
    routineIds: (ids: Maybe<string[]>) => oneToOneIds(ids, 'routine'),
    routinesId: (id: Maybe<string>) => oneToManyId(id, 'routines'),
    routinesIds: (ids: Maybe<string[]>) => oneToManyIds(ids, 'routines'),
    routineVersionId: (id: Maybe<string>) => oneToOneId(id, 'routineVersion'),
    routineVersionsId: (id: Maybe<string>) => oneToManyId(id, 'routineVersions'),
    showOnOrganizationProfile: () => ({ showOnOrganizationProfile: true }),
    smartContractId: (id: Maybe<string>) => oneToOneId(id, 'smartContract'),
    smartContractsId: (id: Maybe<string>) => oneToManyId(id, 'smartContracts'),
    smartContractVersionId: (id: Maybe<string>) => oneToOneId(id, 'smartContractVersion'),
    smartContractVersionsId: (id: Maybe<string>) => oneToManyId(id, 'smartContractVersions'),
    standardId: (id: Maybe<string>) => oneToOneId(id, 'standard'),
    standardIds: (ids: Maybe<string[]>) => oneToOneIds(ids, 'standard'),
    standardType: (type: Maybe<string>) => type ? ({ type: { contains: type.trim(), mode: 'insensitive' } }) : {},
    standardTypeLatestVersion: (type: Maybe<string>) => type ? ({
        versions: {
            some: {
                isLatest: true,
                type: { contains: type.trim(), mode: 'insensitive' }
            }
        }
    }) : {},
    standardsId: (id: Maybe<string>) => oneToManyId(id, 'standards'),
    standardsIds: (ids: Maybe<string[]>) => oneToManyIds(ids, 'standards'),
    standardVersionId: (id: Maybe<string>) => oneToOneId(id, 'standardVersion'),
    standardVersionsId: (id: Maybe<string>) => oneToManyId(id, 'standardVersions'),
    startedTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma('startedAt', time),
    status: <T>(status: Maybe<T>) => ({ status }),
    tagId: (id: Maybe<string>) => oneToOneId(id, 'tag'),
    tagsId: (id: Maybe<string>) => oneToManyId(id, 'tags'),
    tags: (tags: Maybe<string[]>) => ({ tags: { some: { tag: { tag: { in: tags } } } } }),
    transferId: (id: Maybe<string>) => oneToOneId(id, 'transfer'),
    transfersId: (id: Maybe<string>) => oneToManyId(id, 'transfers'),
    translationLanguages: (languages: Maybe<string[]>) => ({ translations: { some: { language: { in: languages } } } }),
    translationLanguagesLatestVersion: (languages: Maybe<string[]>) => ({
        versions: {
            some: {
                isLatest: true,
                translations: { some: { language: { in: languages } } },
            }
        }
    }),
    updatedTimeFrame: (time: Maybe<TimeFrame>) => timeFrameToPrisma('updated_at', time),
    userId: (id: Maybe<string>) => oneToOneId(id, 'user'),
    usersId: (id: Maybe<string>) => oneToManyId(id, 'users'),
    userScheduleId: (id: Maybe<string>) => oneToOneId(id, 'userSchedule'),
    userSchedulesId: (id: Maybe<string>) => oneToManyId(id, 'userSchedules'),
    visibility: (
        visibility: InputMaybe<VisibilityType> | undefined,
        userData: SessionUser | null | undefined,
        objectType: GraphQLModelType
    ) => visibilityBuilder({ objectType, userData, visibility }),
}