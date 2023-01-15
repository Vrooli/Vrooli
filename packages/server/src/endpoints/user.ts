import { gql } from 'apollo-server-express';
import { CustomError } from '../events/error';
import { UserDeleteInput, Success, ProfileUpdateInput, FindByIdOrHandleInput, UserSearchInput, User, ProfileEmailUpdateInput, UserSortBy } from '@shared/consts';
import { FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { rateLimit } from '../middleware';
import { assertRequestFrom } from '../auth/request';
import { readManyHelper, readOneHelper } from '../actions';

export const typeDef = gql`
    enum UserSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
    }

    type User {
        id: ID!
        created_at: Date!
        updated_at: Date
        handle: String
        isPrivate: Boolean!
        isPrivateApis: Boolean!
        isPrivateApisCreated: Boolean!
        isPrivateMemberships: Boolean!
        isPrivateOrganizationsCreated: Boolean!
        isPrivateProjects: Boolean!
        isPrivateProjectsCreated: Boolean!
        isPrivatePullRequests: Boolean!
        isPrivateQuestionsAnswered: Boolean!
        isPrivateQuestionsAsked: Boolean!
        isPrivateQuizzesCreated: Boolean!
        isPrivateRoles: Boolean!
        isPrivateRoutines: Boolean!
        isPrivateRoutinesCreated: Boolean!
        isPrivateSmartContracts: Boolean!
        isPrivateStandards: Boolean!
        isPrivateStandardsCreated: Boolean!
        isPrivateStars: Boolean!
        isPrivateVotes: Boolean!
        name: String!
        theme: String
        status: AccountStatus
        stars: Int!
        views: Int!
        apiKeys: [ApiKey!]
        apis: [Api!]!
        apisCreated: [Api!]
        awards: [Award!]
        comments: [Comment!]
        emails: [Email!]
        organizationsCreate: [Organization!]
        invitedByUser: User
        invitedUsers: [User!]
        issuesCreated: [Issue!]
        issuesClosed: [Issue!]
        labels: [Label!]
        languages: [String!]
        meetingsAttending: [Meeting!]
        meetingsInvited: [MeetingInvite!]
        memberships: [Member!]
        membershipsInvited: [MemberInvite!]
        notesCreated: [Note!]
        notes: [Note!]
        notifications: [Notification!]
        notificationSubscriptions: [NotificationSubscription!]
        notificationSettings: String
        paymentHistory: [Payment!]
        premium: Premium
        projects: [Project!]
        projectsCreated: [Project!]
        pullRequests: [PullRequest!]
        pushDevices: [PushDevice!]
        questionsAnswered: [QuestionAnswer!]
        questionsAsked: [Question!]
        quizzesCreated: [Quiz!]
        quizzesTaken: [Quiz!]
        sentReports: [Report!]
        reportsCreated: [Report!]
        reportsReceived: [Report!]!
        reportsCount: Int!
        reportResponses: [ReportResponse!]
        reputationHistory: [ReputationHistory!]
        roles: [Role!]
        routines: [Routine!]
        routinesCreated: [Routine!]
        runProjects: [RunProject!]
        runRoutines: [RunRoutine!]
        schedules: [UserSchedule!]
        smartContractsCreated: [SmartContract!]
        smartContracts: [SmartContract!]
        standardsCreated: [Standard!]
        standards: [Standard!]
        starredBy: [User!]!
        starred: [Star!]
        stats: StatsUser
        tags: [Tag!]
        transfersIncoming: [Transfer!]
        transfersOutgoing: [Transfer!]
        translations: [UserTranslation!]!
        viewed: [View!]
        viewedBy: [View!]
        voted: [Vote!]
        wallets: [Wallet!]
        you: UserYou!
    }

    type UserYou {
        canDelete: Boolean!
        canEdit: Boolean!
        canReport: Boolean!
        isStarred: Boolean!
        isViewed: Boolean!
    }

    input UserTranslationCreateInput {
        id: ID!
        language: String!
        bio: String
    }
    input UserTranslationUpdateInput {
        id: ID!
        language: String
        bio: String
    }
    type UserTranslation {
        id: ID!
        language: String!
        bio: String
    }

    input ProfileUpdateInput {
        handle: String
        name: String
        theme: String
        isPrivate: Boolean
        isPrivateApis: Boolean
        isPrivateApisCreated: Boolean
        isPrivateMemberships: Boolean
        isPrivateOrganizationsCreated: Boolean
        isPrivateProjects: Boolean
        isPrivateProjectsCreated: Boolean
        isPrivatePullRequests: Boolean
        isPrivateQuestionsAnswered: Boolean
        isPrivateQuestionsAsked: Boolean
        isPrivateQuizzesCreated: Boolean
        isPrivateRoles: Boolean
        isPrivateRoutines: Boolean
        isPrivateRoutinesCreated: Boolean
        isPrivateSmartContracts: Boolean
        isPrivateStandards: Boolean
        isPrivateStandardsCreated: Boolean
        isPrivateStars: Boolean
        isPrivateVotes: Boolean
        notificationSettings: String
        languages: [String!]
        schedulesDelete: [ID!]
        schedulesCreate: [UserScheduleCreateInput!]
        schedulesUpdate: [UserScheduleUpdateInput!]
        translationsDelete: [ID!]
        translationsCreate: [UserTranslationCreateInput!]
        translationsUpdate: [UserTranslationUpdateInput!]
    }

    input ProfileEmailUpdateInput {
        emailsCreate: [EmailCreateInput!]
        emailsDelete: [ID!]
        currentPassword: String!
        newPassword: String
    }

    input UserDeleteInput {
        password: String!
        deletePublicData: Boolean!
    }

    input UserSearchInput {
        minStars: Int
        minViews: Int
        ids: [ID!]
        sortBy: UserSortBy
        searchString: String
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        after: String
        take: Int
        translationLanguages: [String!]
    }

    type UserSearchResult {
        pageInfo: PageInfo!
        edges: [UserEdge!]!
    }

    type UserEdge {
        cursor: String!
        node: User!
    }

    extend type Query {
        profile: User!
        user(input: FindByIdOrHandleInput!): User
        users(input: UserSearchInput!): UserSearchResult!
    }

    extend type Mutation {
        profileUpdate(input: ProfileUpdateInput!): User!
        profileEmailUpdate(input: ProfileEmailUpdateInput!): User!
        userDeleteOne(input: UserDeleteInput!): Success!
        exportData: String!
    }
`

const objectType = 'User';
export const resolvers: {
    UserSortBy: typeof UserSortBy;
    Query: {
        profile: GQLEndpoint<{}, FindOneResult<User>>;
        user: GQLEndpoint<FindByIdOrHandleInput, FindOneResult<User>>;
        users: GQLEndpoint<UserSearchInput, FindManyResult<User>>;
    },
    Mutation: {
        profileUpdate: GQLEndpoint<ProfileUpdateInput, UpdateOneResult<User>>;
        profileEmailUpdate: GQLEndpoint<ProfileEmailUpdateInput, UpdateOneResult<User>>;
        userDeleteOne: GQLEndpoint<UserDeleteInput, Success>;
        exportData: GQLEndpoint<{}, string>;
    }
} = {
    UserSortBy,
    Query: {
        profile: async (_p, _d, { prisma, req }, info) => {
            const { id } = assertRequestFrom(req, { isUser: true });
            return readOneHelper({ info, input: { id }, objectType, prisma, req });
        },
        user: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, objectType, prisma, req });
        },
        users: async (_, { input }, { prisma, req }, info) => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, objectType, prisma, req });
        },
    },
    Mutation: {
        profileUpdate: async (_, { input }, { prisma, req, res }, info) => {
            throw new CustomError('0999', 'NotImplemented', ['en']);
            // const userData = assertRequestFrom(req, { isUser: true });
            // await rateLimit({ info, maxUser: 250, req });
            // // Update object
            // const updated = await ProfileModel.mutate(prisma).updateProfile(userData, input, info);
            // if (!updated)
            //     throw new CustomError('0160', 'ErrorUnknown', req.languages);
            // // Update session
            // const session = await toSession({ id: userData.id }, prisma, req);
            // await generateSessionJwt(res, session);
            // return updated;
        },
        profileEmailUpdate: async (_, { input }, { prisma, req, res }, info) => {
            throw new CustomError('0999', 'NotImplemented', ['en']);
            // const userData = assertRequestFrom(req, { isUser: true });
            // await rateLimit({ info, maxUser: 100, req });
            // // Update object
            // const updated = await ProfileModel.mutate(prisma).updateEmails(userData.id, input, info);
            // if (!updated)
            //     throw new CustomError('0162', 'ErrorUnknown', req.languages);
            // return updated;
        },
        userDeleteOne: async (_, { input }, { prisma, req, res }, info) => {
            throw new CustomError('0999', 'NotImplemented', ['en']);
            // const userData = assertRequestFrom(req, { isUser: true });
            // await rateLimit({ info, maxUser: 5, req });
            // // TODO anonymize public data
            // return await ProfileModel.mutate(prisma).deleteProfile(userData.id, input);
        },
        /**
         * Exports user data to a JSON file (created/saved routines, projects, organizations, etc.).
         * In the future, an import function will be added.
         * @returns JSON of all user data
         */
        exportData: async (_p, _d, { prisma, req, res }, info) => {
            throw new CustomError('0999', 'NotImplemented', ['en']);
            // const userData = assertRequestFrom(req, { isUser: true });
            // await rateLimit({ info, maxUser: 5, req });
            // return await ProfileModel.port(prisma).exportData(userData.id);
        }
    }
}