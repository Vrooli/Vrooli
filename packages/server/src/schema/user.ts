import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { UserModel } from '../models';
import { DeleteUserInput, ReportInput, UpdateUserInput, User } from './types';
import { IWrap, RecursivePartial } from '../types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`

    input UserInput {
        id: ID
        username: String
        pronouns: String
        emails: [EmailInput!]
        theme: String
        status: AccountStatus
    }

    type User {
        id: ID!
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
        sentReports: [Report!]!
        reports: [Report!]!
        votedComments: [Comment!]!
        votedByTag: [Tag!]!
    }

    input UpdateUserInput {
        data: UserInput!
        currentPassword: String!
        newPassword: String
    }

    input DeleteUserInput {
        id: ID!
        password: String!
    }

    extend type Query {
        profile: User!
    }

    extend type Mutation {
        updateUser(input: UpdateUserInput!): User!
        deleteUser(input: DeleteUserInput!): Boolean
        reportUser(input: ReportInput!): Boolean!
        exportData: String!
    }
`

export const resolvers = {
    Query: {
        profile: async (_parent: undefined, _args: undefined, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<User> | null> => {
            // TODO add restrictions to what data can be queried
            return await UserModel(prisma).findById({ id: req.userId ?? '' }, info);
        }
    },
    Mutation: {
        updateUser: async (_parent: undefined, { input }: IWrap<UpdateUserInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<User>> => {
            // Must be updating your own
            if (req.userId !== input.data.id) throw new CustomError(CODE.Unauthorized);
            // Check for correct password
            let user = await prisma.user.findUnique({ where: { id: input.data.id } });
            if (!user) throw new CustomError(CODE.InvalidArgs);
            if (!UserModel(prisma).validatePassword(input.currentPassword, user)) throw new CustomError(CODE.BadCredentials);
            // Update user
            return await UserModel(prisma).upsertUser(input.data, info);
        },
        deleteUser: async (_parent: undefined, { input }: IWrap<DeleteUserInput>, { prisma, req }: any, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be deleting your own
            if (req.userId !== input.id) throw new CustomError(CODE.Unauthorized);
            // Check for correct password
            let user = await prisma.user.findUnique({ where: { id: input.id } });
            if (!user) throw new CustomError(CODE.InvalidArgs);
            if (!UserModel(prisma).validatePassword(input.password, user)) throw new CustomError(CODE.BadCredentials);
            // Delete user
            return await UserModel(prisma).delete(user.id);
        },
        /**
         * Reports a user. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportUser: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await UserModel(prisma).report(input);
        },
        /**
         * Exports user data to a JSON file (created/saved routines, projects, organizations, etc.).
         * In the future, an import function will be added.
         * @returns JSON of all user data
         */
        exportData: async (_parent: undefined, _args: undefined, context: Context, _info: GraphQLResolveInfo): Promise<string> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        }
    }
}