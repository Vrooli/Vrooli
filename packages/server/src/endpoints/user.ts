import { FindByIdOrHandleInput, ProfileEmailUpdateInput, ProfileUpdateInput, Success, User, UserDeleteInput, UserSearchInput, UserSortBy } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { readManyHelper, readOneHelper, updateHelper } from '../actions';
import { assertRequestFrom } from '../auth/request';
import { CustomError } from '../events/error';
import { rateLimit } from '../middleware';
import { FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { parseICalFile } from '../utils';

export const typeDef = gql`
    enum UserSortBy {
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        BookmarksAsc
        BookmarksDesc
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
        isPrivateBookmarks: Boolean!
        isPrivateVotes: Boolean!
        name: String!
        theme: String
        status: AccountStatus
        bookmarks: Int!
        views: Int!
        apiKeys: [ApiKey!]
        apis: [Api!]!
        apisCount: Int!
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
        translationLanguages: [String!]
        meetingsAttending: [Meeting!]
        meetingsInvited: [MeetingInvite!]
        memberships: [Member!]
        membershipsCount: Int!
        membershipsInvited: [MemberInvite!]
        notesCreated: [Note!]
        notes: [Note!]
        notesCount: Int!
        notifications: [Notification!]
        notificationSubscriptions: [NotificationSubscription!]
        notificationSettings: String
        paymentHistory: [Payment!]
        premium: Premium
        projects: [Project!]
        projectsCount: Int!
        projectsCreated: [Project!]
        pullRequests: [PullRequest!]
        pushDevices: [PushDevice!]
        questionsAnswered: [QuestionAnswer!]
        questionsAsked: [Question!]
        questionsAskedCount: Int!
        quizzesCreated: [Quiz!]
        quizzesTaken: [Quiz!]
        sentReports: [Report!]
        reportsCreated: [Report!]
        reportsReceived: [Report!]!
        reportsReceivedCount: Int!
        reportResponses: [ReportResponse!]
        reputationHistory: [ReputationHistory!]
        roles: [Role!]
        routines: [Routine!]
        routinesCount: Int!
        routinesCreated: [Routine!]
        runProjects: [RunProject!]
        runRoutines: [RunRoutine!]
        focusModes: [FocusMode!]
        smartContracts: [SmartContract!]
        smartContractsCount: Int!
        smartContractsCreated: [SmartContract!]
        standards: [Standard!]
        standardsCount: Int!
        standardsCreated: [Standard!]
        bookmarkedBy: [User!]!
        bookmarked: [Bookmark!]
        tags: [Tag!]
        transfersIncoming: [Transfer!]
        transfersOutgoing: [Transfer!]
        translations: [UserTranslation!]!
        viewed: [View!]
        viewedBy: [View!]
        reacted: [Reaction!]
        wallets: [Wallet!]
        you: UserYou!
    }

    type UserYou {
        canDelete: Boolean!
        canReport: Boolean!
        isBookmarked: Boolean!
        canUpdate: Boolean!
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
        isPrivateBookmarks: Boolean
        isPrivateVotes: Boolean
        notificationSettings: String
        languages: [String!]
        focusModesDelete: [ID!]
        focusModesCreate: [FocusModeCreateInput!]
        focusModesUpdate: [FocusModeUpdateInput!]
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

    input ImportCalendarInput {
        file: Upload!
    }

    input UserSearchInput {
        maxBookmarks: Int
        maxViews: Int
        minBookmarks: Int
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
        importCalendar(input: ImportCalendarInput!): Success!
        # importUserData(input: ImportUserDataInput!): Success!
        exportCalendar: String!
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
        importCalendar: GQLEndpoint<any, Success>;
        // importUserData: GQLEndpoint<ImportUserDataInput, Success>;
        exportCalendar: GQLEndpoint<{}, string>;
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
            await rateLimit({ info, maxUser: 250, req });
            // Add user id to input, since IDs are required for validation checks
            const { id } = assertRequestFrom(req, { isUser: true });
            return updateHelper({ info, input: { ...input, id }, objectType, prisma, req })
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
        importCalendar: async (_, { input }, { prisma, req, res }, info) => {
            await rateLimit({ info, maxUser: 25, req });
            await parseICalFile(input.file);
            throw new CustomError('0999', 'NotImplemented', ['en']);
        },
        exportCalendar: async (_p, _d, { prisma, req, res }, info) => {
            throw new CustomError('0999', 'NotImplemented', ['en']);
        },
        /**
         * Exports user data to a JSON file (created/saved routines, projects, organizations, etc.).
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