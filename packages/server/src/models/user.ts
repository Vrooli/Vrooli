import { MaxObjects, ProfileUpdateInput, User, UserSearchInput, UserSortBy, userValidation, UserYou } from "@local/shared";
import { Prisma } from "@prisma/client";
import { noNull, shapeHelper } from "../builders";
import { SelectWrap } from "../builders/types";
import { PrismaType } from "../types";
import { defaultPermissions, translationShapeHelper } from "../utils";
import { getSingleTypePermissions } from "../validators";
import { BookmarkModel } from "./bookmark";
import { ModelLogic } from "./types";
import { ViewModel } from "./view";

const __typename = 'User' as const;
type Permissions = Pick<UserYou, 'canDelete' | 'canUpdate' | 'canReport'>
const suppFields = ['you'] as const;
export const UserModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: ProfileUpdateInput,
    GqlModel: User,
    GqlSearch: UserSearchInput,
    GqlSort: UserSortBy,
    GqlPermission: Permissions,
    PrismaCreate: Prisma.userUpsertArgs['create'],
    PrismaUpdate: Prisma.userUpsertArgs['update'],
    PrismaModel: Prisma.userGetPayload<SelectWrap<Prisma.userSelect>>,
    PrismaSelect: Prisma.userSelect,
    PrismaWhere: Prisma.userWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.user,
    display: {
        select: () => ({ id: true, name: true }),
        label: (select) => select.name ?? '',
    },
    format: {
        gqlRelMap: {
            __typename,
            comments: 'Comment',
            emails: 'Email',
            focusModes: 'FocusMode',
            labels: 'Label',
            // phones: 'Phone',
            projects: 'Project',
            pushDevices: 'PushDevice',
            bookmarkedBy: 'User',
            reportsCreated: 'Report',
            reportsReceived: 'Report',
            routines: 'Routine',
        },
        prismaRelMap: {
            __typename,
            apis: 'Api',
            apiKeys: 'ApiKey',
            comments: 'Comment',
            emails: 'Email',
            organizationsCreated: 'Organization',
            phones: 'Phone',
            posts: 'Post',
            focusModes: 'FocusMode',
            invitedByUser: 'User',
            invitedUsers: 'User',
            issuesCreated: 'Issue',
            issuesClosed: 'Issue',
            labels: 'Label',
            meetingsAttending: 'Meeting',
            meetingsInvited: 'MeetingInvite',
            pushDevices: 'PushDevice',
            notifications: 'Notification',
            memberships: 'Member',
            projectsCreated: 'Project',
            projects: 'Project',
            pullRequests: 'PullRequest',
            questionAnswered: 'QuestionAnswer',
            questionsAsked: 'Question',
            quizzesCreated: 'Quiz',
            quizzesTaken: 'QuizAttempt',
            reportsCreated: 'Report',
            reportsReceived: 'Report',
            reportResponses: 'ReportResponse',
            routinesCreated: 'Routine',
            routines: 'Routine',
            runProjects: 'RunProject',
            runRoutines: 'RunRoutine',
            smartContractsCreated: 'SmartContract',
            smartContracts: 'SmartContract',
            standardsCreated: 'Standard',
            standards: 'Standard',
            bookmarkedBy: 'User',
            tags: 'Tag',
            transfersIncoming: 'Transfer',
            transfersOutgoing: 'Transfer',
            notesCreated: 'Note',
            notes: 'Note',
            wallets: 'Wallet',
            stats: 'StatsUser',
        },
        joinMap: {
            meetingsAttending: 'user',
            bookmarkedBy: 'user',
        },
        countFields: {
            apisCount: true,
            membershipsCount: true,
            notesCount: true,
            projectsCount: true,
            questionsAskedCount: true,
            reportsReceivedCount: true,
            routinesCount: true,
            smartContractsCount: true,
            standardsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: async ({ ids, prisma, userData }) => {
                return {
                    you: {
                        ...(await getSingleTypePermissions<Permissions>(__typename, ids, prisma, userData)),
                        isBookmarked: await BookmarkModel.query.getIsBookmarkeds(prisma, userData?.id, ids, __typename),
                        isViewed: await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
                    }
                }
            },
        },
    },
    mutate: {
        shape: {
            update: async ({ data, ...rest }) => ({
                handle: data.handle ?? null,
                name: noNull(data.name),
                theme: noNull(data.theme),
                isPrivate: noNull(data.isPrivate),
                isPrivateApis: noNull(data.isPrivateApis),
                isPrivateApisCreated: noNull(data.isPrivateApisCreated),
                isPrivateMemberships: noNull(data.isPrivateMemberships),
                isPrivateOrganizationsCreated: noNull(data.isPrivateOrganizationsCreated),
                isPrivateProjects: noNull(data.isPrivateProjects),
                isPrivateProjectsCreated: noNull(data.isPrivateProjectsCreated),
                isPrivatePullRequests: noNull(data.isPrivatePullRequests),
                isPrivateQuestionsAnswered: noNull(data.isPrivateQuestionsAnswered),
                isPrivateQuestionsAsked: noNull(data.isPrivateQuestionsAsked),
                isPrivateQuizzesCreated: noNull(data.isPrivateQuizzesCreated),
                isPrivateRoles: noNull(data.isPrivateRoles),
                isPrivateRoutines: noNull(data.isPrivateRoutines),
                isPrivateRoutinesCreated: noNull(data.isPrivateRoutinesCreated),
                isPrivateSmartContracts: noNull(data.isPrivateSmartContracts),
                isPrivateStandards: noNull(data.isPrivateStandards),
                isPrivateStandardsCreated: noNull(data.isPrivateStandardsCreated),
                isPrivateBookmarks: noNull(data.isPrivateBookmarks),
                isPrivateVotes: noNull(data.isPrivateVotes),
                notificationSettings: data.notificationSettings ?? null,
                // languages: TODO!!!
                ...(await shapeHelper({ relation: 'focusModes', relTypes: ['Create', 'Update', 'Delete'], isOneToOne: false, isRequired: false, objectType: 'FocusMode', parentRelationshipName: 'user', data, ...rest })),
                ...(await translationShapeHelper({ relTypes: ['Create', 'Update', 'Delete'], isRequired: false, data, ...rest })),
            }),
        },
        yup: userValidation,
    },
    search: {
        defaultSort: UserSortBy.BookmarksDesc,
        sortBy: UserSortBy,
        searchFields: {
            createdTimeFrame: true,
            maxBookmarks: true,
            maxViews: true,
            minBookmarks: true,
            minViews: true,
            translationLanguages: true,
            updatedTimeFrame: true,
        },
        searchStringQuery: () => ({
            OR: [
                'transBioWrapped',
                'nameWrapped',
                'handleWrapped',
            ]
        }),
    },
    validate: {
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        permissionsSelect: () => ({
            id: true,
            isPrivate: true,
            languages: { select: { language: true } },
        }),
        permissionResolvers: defaultPermissions,
        owner: (data) => ({ User: data }),
        isDeleted: () => false,
        isPublic: (data) => data.isPrivate === false,
        profanityFields: ['name', 'handle'],
        visibility: {
            private: { isPrivate: true },
            public: { isPrivate: false },
            owner: (userId) => ({ id: userId }),
        },
        // createMany.forEach(input => lineBreaksCheck(input, ['bio'], 'LineBreaksBio'));
    },
})