import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { countHelper, ProfileModel, readManyHelper, readOneHelper, UserModel } from '../models';
import { UserDeleteInput, Success, Profile, ProfileUpdateInput, FindByIdOrHandleInput, UserSearchInput, UserCountInput, UserSearchResult, User, ProfileEmailUpdateInput, UserSortBy } from './types';
import { IWrap, RecursivePartial } from '../types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { rateLimit } from '../rateLimit';
import { genErrorCode } from '../logger';

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
        history: [Log!]!
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
        routines: [Routine!]!
        routinesCreated: [Routine!]!
        translations: [UserTranslation!]!
    }

    input UserTranslationCreateInput {
        id: ID
        language: String!
        bio: String
    }
    input UserTranslationUpdateInput {
        id: ID!
        language: String!
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
        resourceListsCreate: [ResourceCreateInput!]
        resourceListsUpdate: [ResourceUpdateInput!]
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
        profile: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Profile> | null> => {
            if (!context.req.userId) 
                throw new CustomError(CODE.Unauthorized, 'Must be logged in to query profile', { code: genErrorCode('0158') });
            await rateLimit({ context, info, max: 2000, byAccount: true });
            return ProfileModel(context.prisma).findProfile(context.req.userId, info);
        },
        user: async (_parent: undefined, { input }: IWrap<FindByIdOrHandleInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<User> | null> => {
            await rateLimit({ context, info, max: 1000 });
            return readOneHelper(context.req.userId, input, info, UserModel(context.prisma));
        },
        users: async (_parent: undefined, { input }: IWrap<UserSearchInput>, context: Context, info: GraphQLResolveInfo): Promise<UserSearchResult> => {
            await rateLimit({ context, info, max: 1000 });
            return readManyHelper(context.req.userId, input, info, UserModel(context.prisma));
        },
        usersCount: async (_parent: undefined, { input }: IWrap<UserCountInput>, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            await rateLimit({ context, info, max: 1000 });
            return countHelper(input, UserModel(context.prisma));
        },
    },
    Mutation: {
        profileUpdate: async (_parent: undefined, { input }: IWrap<ProfileUpdateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Profile> | null> => {
            if (!context.req.userId) 
                throw new CustomError(CODE.Unauthorized, 'Must be logged in to update profile', { code: genErrorCode('0159') });
            await rateLimit({ context, info, max: 250, byAccount: true });
            // Update object
            const updated = await ProfileModel(context.prisma).updateProfile(context.req.userId, input, info);
            if (!updated) 
                throw new CustomError(CODE.ErrorUnknown, 'Could not update profile', { code: genErrorCode('0160') });
            return updated;
        },
        profileEmailUpdate: async (_parent: undefined, { input }: IWrap<ProfileEmailUpdateInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Profile> | null> => {
            if (!context.req.userId) 
                throw new CustomError(CODE.Unauthorized, 'Must be logged in to update profile', { code: genErrorCode('0161') });
            await rateLimit({ context, info, max: 100, byAccount: true });
            // Update object
            const updated = await ProfileModel(context.prisma).updateEmails(context.req.userId, input, info);
            if (!updated) 
                throw new CustomError(CODE.ErrorUnknown, 'Could not update profile', { code: genErrorCode('0162') });
            return updated;
        },
        userDeleteOne: async (_parent: undefined, { input }: IWrap<UserDeleteInput>, context: Context, info: GraphQLResolveInfo): Promise<Success> => {
            if (!context.req.userId) 
                throw new CustomError(CODE.Unauthorized, 'Must be logged in to delete account', { code: genErrorCode('0163') });
            await rateLimit({ context, info, max: 5 });
            return await ProfileModel(context.prisma).deleteProfile(context.req.userId, input);
        },
        /**
         * Exports user data to a JSON file (created/saved routines, projects, organizations, etc.).
         * In the future, an import function will be added.
         * @returns JSON of all user data
         */
        exportData: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<string> => {
            if (!context.req.userId) 
                throw new CustomError(CODE.Unauthorized, 'Must be logged in to export data', { code: genErrorCode('0164') });
            await rateLimit({ context, info, max: 5, byAccount: true });
            return await ProfileModel(context.prisma).exportData(context.req.userId);
        }
    }
}