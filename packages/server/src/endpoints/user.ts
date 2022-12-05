import { gql } from 'apollo-server-express';
import { CustomError } from '../events/error';
import { UserDeleteInput, Success, Profile, ProfileUpdateInput, FindByIdOrHandleInput, UserSearchInput, User, ProfileEmailUpdateInput, UserSortBy } from './types';
import { FindManyResult, FindOneResult, GQLEndpoint, UpdateOneResult } from '../types';
import { rateLimit } from '../middleware';
import { assertRequestFrom } from '../auth/request';
import { ProfileModel } from '../models';
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

    # User information available for you own account
    type Profile {
        id: ID!
        created_at: Date!
        updated_at: Date!
        handle: String
        name: String!
        theme: String!
        status: AccountStatus!
        comments: [Comment!]!
        roles: [Role!]!
        emails: [Email!]!
        wallets: [Wallet!]!
        resourceLists: [ResourceList!]!
        projects: [Project!]!
        projectsCreated: [Project!]!
        routines: [Routine!]!
        routinesCreated: [Routine!]!
        stars: [Star!]!
        starredBy: [User!]!
        starredTags: [Tag!]
        hiddenTags: [TagHidden!]
        sentReports: [Report!]!
        reports: [Report!]!
        translations: [UserTranslation!]!
        votes: [Vote!]!
    }

    # User information available for other accounts
    type User {
        id: ID!
        created_at: Date!
        handle: String
        name: String!
        stars: Int!
        views: Int!
        isStarred: Boolean!
        isViewed: Boolean!
        comments: [Comment!]!
        resourceLists: [ResourceList!]!
        projects: [Project!]!
        projectsCreated: [Project!]!
        starredBy: [User!]!
        reports: [Report!]!
        reportsCount: Int!
        routines: [Routine!]!
        routinesCreated: [Routine!]!
        translations: [UserTranslation!]!
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
        hiddenTagsDelete: [ID!]
        hiddenTagsCreate: [TagHiddenCreateInput!]
        hiddenTagsUpdate: [TagHiddenUpdateInput!]
        resourceListsDelete: [ID!]
        resourceListsCreate: [ResourceListCreateInput!]
        resourceListsUpdate: [ResourceListUpdateInput!]
        starredTagsConnect: [ID!]
        starredTagsDisconnect: [ID!]
        starredTagsCreate: [TagCreateInput!]
        translationsDelete: [ID!]
        translationsCreate: [UserTranslationCreateInput!]
        translationsUpdate: [UserTranslationUpdateInput!]
    }

    input ProfileEmailUpdateInput {
        emailsCreate: [EmailCreateInput!]
        emailsUpdate: [EmailUpdateInput!]
        emailsDelete: [ID!]
        currentPassword: String!
        newPassword: String
    }

    input UserDeleteInput {
        password: String!
        deletePublicData: Boolean!
    }

    input UserSearchInput {
        languages: [String!]
        minStars: Int
        minViews: Int
        organizationId: ID
        projectId: ID
        routineId: ID
        reportId: ID
        standardId: ID
        ids: [ID!]
        resourceLists: [String!]
        resourceTypes: [ResourceUsedFor!]
        sortBy: UserSortBy
        searchString: String
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
        after: String
        take: Int
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
        profile: Profile!
        user(input: FindByIdOrHandleInput!): User
        users(input: UserSearchInput!): UserSearchResult!
    }

    extend type Mutation {
        profileUpdate(input: ProfileUpdateInput!): Profile!
        profileEmailUpdate(input: ProfileEmailUpdateInput!): Profile!
        userDeleteOne(input: UserDeleteInput!): Success!
        exportData: String!
    }
`

const objectType = 'User';
export const resolvers: {
    UserSortBy: typeof UserSortBy;
    Query: {
        profile: GQLEndpoint<{}, FindOneResult<Profile>>;
        user: GQLEndpoint<FindByIdOrHandleInput, FindOneResult<User>>;
        users: GQLEndpoint<UserSearchInput, FindManyResult<User>>;
    },
    Mutation: {
        profileUpdate: GQLEndpoint<ProfileUpdateInput, UpdateOneResult<Profile>>;
        profileEmailUpdate: GQLEndpoint<ProfileEmailUpdateInput, UpdateOneResult<Profile>>;
        userDeleteOne: GQLEndpoint<UserDeleteInput, Success>;
        exportData: GQLEndpoint<{}, string>;
    }
} = {
    UserSortBy,
    Query: {
        profile: async (_p, _d, { prisma, req }, info) => {
            assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 2000, req });
            return ProfileModel.query.findProfile(prisma, req, info);
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