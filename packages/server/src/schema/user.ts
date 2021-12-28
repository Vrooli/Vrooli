import { gql } from 'apollo-server-express';
import { CODE, USER_SORT_BY } from '@local/shared';
import { CustomError } from '../error';
import { UserModel } from '../models';
import { UserDeleteInput, ReportInput, Success, Profile, ProfileUpdateInput } from './types';
import { IWrap, RecursivePartial } from '../types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    enum UserSortBy {
        AlphabeticalAsc
        AlphabeticalDesc
        CommentsAsc
        CommentsDesc
        DateCreatedAsc
        DateCreatedDesc
        DateUpdatedAsc
        DateUpdatedDesc
        StarsAsc
        StarsDesc
    }

    input UserInput {
        id: ID
        username: String
        pronouns: String
        emails: [EmailInput!]
        theme: String
        status: AccountStatus
    }

    # User information available for you own account
    type Profile {
        id: ID!
        created_at: Date!
        updated_at: Date!
        username: String
        pronouns: String!
        theme: String!
        emailVerified: Boolean!
        status: AccountStatus!
        comments: [Comment!]!
        roles: [Role!]!
        emails: [Email!]!
        wallets: [Wallet!]!
        resources: [Resource!]!
        projects: [Project!]!
        starredComments: [Comment!]!
        starredProjects: [Project!]!
        starredOrganizations: [Organization!]!
        starredResources: [Resource!]!
        starredRoutines: [Routine!]!
        starredStandards: [Standard!]!
        starredTags: [Tag!]!
        starredUsers: [User!]!
        starredBy: [User!]!
        sentReports: [Report!]!
        reports: [Report!]!
        votedComments: [Comment!]!
        votedByTag: [Tag!]!
    }

    # User information available for other accounts
    type User {
        id: ID!
        created_at: Date!
        username: String
        pronouns: String!
        comments: [Comment!]!
        roles: [Role!]!
        resources: [Resource!]!
        projects: [Project!]!
        starredBy: [User!]!
        reports: [Report!]!
    }

    input ProfileUpdateInput {
        data: UserInput!
        currentPassword: String!
        newPassword: String
    }

    input UserDeleteInput {
        id: ID!
        password: String!
    }

    input UserSearchInput {
        ids: [ID!]
        sortBy: UserSortBy
        searchString: String
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

    extend type Query {
        profile: Profile!
        user(input: FindByIdInput!): User
        users(input: UserSearchInput!): UserSearchResult!
    }

    extend type Mutation {
        profileUpdate(input: ProfileUpdateInput!): Profile!
        userDeleteOne(input: UserDeleteInput!): Success!
        userReport(input: ReportInput!): Success!
        exportData: String!
    }
`

export const resolvers = {
    UserSortBy: USER_SORT_BY,
    Query: {
        profile: async (_parent: undefined, _args: undefined, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<any> | null> => {
            // Query database
            const dbModel = await UserModel(prisma).findById({ id: req.userId ?? '' }, info);
            // Format data
            return dbModel ? UserModel().toGraphQLProfile(dbModel) : null;
        }
    },
    Mutation: {
        profileUpdate: async (_parent: undefined, { input }: IWrap<ProfileUpdateInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Profile> | null> => {
            // Must be updating your own
            if (req.userId !== input.data.id) throw new CustomError(CODE.Unauthorized);
            // Check for correct password
            let user = await prisma.user.findUnique({ where: { id: input.data.id } });
            if (!user) throw new CustomError(CODE.InvalidArgs);
            if (!UserModel(prisma).validatePassword(input.currentPassword, user)) throw new CustomError(CODE.BadCredentials);
            // Update user
            let dbModel = await UserModel(prisma).upsertUser(input.data, info);
            // Format data
            return dbModel ? UserModel().toGraphQLProfile(dbModel) : null;
        },
        userDeleteOne: async (_parent: undefined, { input }: IWrap<UserDeleteInput>, { prisma, req }: any, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be deleting your own
            if (req.userId !== input.id) throw new CustomError(CODE.Unauthorized);
            // Check for correct password
            let user = await prisma.user.findUnique({ where: { id: input.id } });
            if (!user) throw new CustomError(CODE.InvalidArgs);
            if (!UserModel(prisma).validatePassword(input.password, user)) throw new CustomError(CODE.BadCredentials);
            // Delete user
            const success = await UserModel(prisma).delete(user.id);
            return { success }
        },
        /**
         * Reports a user. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
        userReport: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            const success = await UserModel(prisma).report(input);
            return { success }
        },
        /**
         * Exports user data to a JSON file (created/saved routines, projects, organizations, etc.).
         * In the future, an import function will be added.
         * @returns JSON of all user data
         */
        exportData: async (_parent: undefined, _args: undefined, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<string> => {
            // Must be logged in
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            return await UserModel(prisma).exportData(req.userId);
        }
    }
}