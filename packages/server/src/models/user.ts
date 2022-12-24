import { StarModel } from "./star";
import { ViewModel } from "./view";
import { UserSortBy } from "@shared/consts";
import { ProfileUpdateInput, User, UserSearchInput } from '@shared/consts';
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { Prisma } from "@prisma/client";
import { userValidation } from "@shared/validation";
import { SelectWrap } from "../builders/types";

const __typename = 'User' as const;
const suppFields = ['isStarred', 'isViewed'] as const;
export const UserModel: ModelLogic<{
    IsTransferable: false,
    IsVersioned: false,
    GqlCreate: undefined,
    GqlUpdate: ProfileUpdateInput,
    GqlModel: User,
    GqlSearch: UserSearchInput,
    GqlSort: UserSortBy,
    GqlPermission: any,
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
            __typename: 'User',
            comments: 'Comment',
            emails: 'Email',
            // phones: 'Phone',
            projects: 'Project',
            pushDevices: 'PushDevice',
            starredBy: 'User',
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
            schedules: 'UserSchedule',
            smartContractsCreated: 'SmartContract',
            smartContracts: 'SmartContract',
            standardsCreated: 'Standard',
            standards: 'Standard',
            starredBy: 'User',
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
            starredBy: 'user',
        },
        countFields: {
            reportsCount: true,
        },
        supplemental: {
            graphqlFields: suppFields,
            toGraphQL: ({ ids, prisma, userData }) => ({
                isStarred: async () => await StarModel.query.getIsStarreds(prisma, userData?.id, ids, __typename),
                isViewed: async () => await ViewModel.query.getIsVieweds(prisma, userData?.id, ids, __typename),
            }),
        },
    },
    mutate: {
        shape: {
            update: async ({ data, prisma, userData }) => {
                return {
                    // handle: data.handle,
                    // name: data.name ?? undefined,
                    // theme: data.theme ?? undefined,
                    // // // hiddenTags: await TagHiddenModel.mutate(prisma).relationshipBuilder!(userData.id, input, false),
                    // // starred: {
                    // //     create: starredCreate,
                    // //     delete: starredDelete,
                    // // }, TODO
                    // translations: await translationRelationshipBuilder(prisma, userData, data, false),
                } as any
            } 
        },
        yup: userValidation,
    },
    search: {
        defaultSort: UserSortBy.StarsDesc,
        sortBy: UserSortBy,
        searchFields: {
            createdTimeFrame: true,
            minStars: true,
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
        maxObjects: 0,
        permissionsSelect: () => ({
            id: true,
            isPrivate: true,
            languages: { select: { language: true } },
        }),
        permissionResolvers: () => ({}),
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