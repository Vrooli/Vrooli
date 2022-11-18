import { gql } from 'apollo-server-express';
import { CODE } from '@shared/consts';
import { CustomError } from '../events/error';
import { countHelper, ProfileModel, readManyHelper, readOneHelper, UserModel } from '../models';
import { UserDeleteInput, Success, Profile, ProfileUpdateInput, FindByIdOrHandleInput, UserSearchInput, UserCountInput, UserSearchResult, User, ProfileEmailUpdateInput, UserSortBy } from './types';
import { IWrap, RecursivePartial } from '../types';
import { Context, rateLimit } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';
import { genErrorCode } from '../events/logger';
import { assertRequestFrom, generateSessionJwt } from '../auth/auth';

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

    # Return type for search result
    type UserSearchResult {
        pageInfo: PageInfo!
        edges: [UserEdge!]!
    }

    # Return type for search result edge
    type UserEdge {
        cursor: String!
        node: User!
    }

    # Input for count
    input UserCountInput {
        createdTimeFrame: TimeFrame
        updatedTimeFrame: TimeFrame
    }

    extend type Query {
        profile: Profile!
        user(input: FindByIdOrHandleInput!): User
        users(input: UserSearchInput!): UserSearchResult!
        usersCount(input: UserCountInput!): Int!
    }

    extend type Mutation {
        profileUpdate(input: ProfileUpdateInput!): Profile!
        profileEmailUpdate(input: ProfileEmailUpdateInput!): Profile!
        userDeleteOne(input: UserDeleteInput!): Success!
        exportData: String!
    }
`

export const resolvers = {
    UserSortBy: UserSortBy,
    Query: {
        profile: async (_parent: undefined, _args: undefined, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Profile> | null> => {
            assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 2000, req });
            return ProfileModel.query(prisma).findProfile(req, info);
        },
        user: async (_parent: undefined, { input }: IWrap<FindByIdOrHandleInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<User> | null> => {
            await rateLimit({ info, maxUser: 1000, req });
            return readOneHelper({ info, input, model: UserModel, prisma, req });
        },
        users: async (_parent: undefined, { input }: IWrap<UserSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<UserSearchResult> => {
            await rateLimit({ info, maxUser: 1000, req });
            return readManyHelper({ info, input, model: UserModel, prisma, req });
        },
        usersCount: async (_parent: undefined, { input }: IWrap<UserCountInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ info, maxUser: 1000, req });
            return countHelper({ input, model: UserModel, prisma, req });
        },
    },
    Mutation: {
        profileUpdate: async (_parent: undefined, { input }: IWrap<ProfileUpdateInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Profile> | null> => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 250, req });
            // Update object
            const updated = await ProfileModel.mutate(prisma).updateProfile(userData, input, info);
            if (!updated)
                throw new CustomError(CODE.ErrorUnknown, 'Could not update profile', { code: genErrorCode('0160') });
            // Update session
            const session = await ProfileModel.verify.toSession({ id: userData.id }, prisma, req);
            await generateSessionJwt(res, session);
            return updated;
        },
        profileEmailUpdate: async (_parent: undefined, { input }: IWrap<ProfileEmailUpdateInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Profile> | null> => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 100, req });
            // Update object
            const updated = await ProfileModel.mutate(prisma).updateEmails(userData.id, input, info);
            if (!updated)
                throw new CustomError(CODE.ErrorUnknown, 'Could not update profile', { code: genErrorCode('0162') });
            return updated;
        },
        userDeleteOne: async (_parent: undefined, { input }: IWrap<UserDeleteInput>, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<Success> => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 5, req });
            // TODO anonymize public data
            return await ProfileModel.mutate(prisma).deleteProfile(userData.id, input);
        },
        /**
         * Exports user data to a JSON file (created/saved routines, projects, organizations, etc.).
         * In the future, an import function will be added.
         * @returns JSON of all user data
         */
        exportData: async (_parent: undefined, _args: undefined, { prisma, req, res }: Context, info: GraphQLResolveInfo): Promise<string> => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 5, req });
            return await ProfileModel.port(prisma).exportData(userData.id);
        }
    }
}