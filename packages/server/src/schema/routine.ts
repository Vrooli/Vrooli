import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from '../types';
import { Count, DeleteOneInput, FindByIdInput, ReportInput, Routine, RoutineInput, RoutinesQueryInput, Success } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { RoutineModel } from '../models';

export const typeDef = gql`
    input RoutineInput {
        id: ID
        version: String
        title: String
        description: String
        instructions: String
        isAutomatable: Boolean
        inputs: [RoutineInputItemInput!]
        outputs: [RoutineOutputItemInput!]
    }

    type Routine {
        id: ID!
        created_at: Date!
        updated_at: Date!
        version: String
        title: String
        description: String
        instructions: String
        isAutomatable: Boolean
        inputs: [RoutineInputItem!]!
        outputs: [RoutineOutputItem!]!
        nodes: [Node!]!
        contextualResources: [Resource!]!
        externalResources: [Resource!]!
        donationResources: [Resource!]!
        tags: [Tag!]!
        users: [User!]!
        organizations: [Organization!]!
        starredBy: [User!]!
        parent: Routine
        forks: [Routine!]!
        nodeLists: [NodeRoutineList!]!
        reports: [Report!]!
        comments: [Comment!]!
    }

    input RoutineInputItemInput {
        id: ID
        routineId: ID!
        standardId: ID
    }

    type RoutineInputItem {
        id: ID!
        routine: Routine!
        standard: Standard!
    }

    input RoutineOutputItemInput {
        id: ID
        routineId: ID!
        standardId: ID
    }

    type RoutineOutputItem {
        id: ID!
        routine: Routine!
        standard: Standard!
    }

    input RoutinesQueryInput {
        first: Int
        skip: Int
    }

    extend type Query {
        routine(input: FindByIdInput!): Routine
        routines(input: RoutinesQueryInput!): [Routine!]!
        routinesCount: Count!
    }

    extend type Mutation {
        routineAdd(input: RoutineInput!): Routine!
        routineUpdate(input: RoutineInput!): Routine!
        routineDeleteOne(input: DeleteOneInput!): Success!
        routineReport(input: ReportInput!): Success!
    }
`

export const resolvers = {
    Query: {
        routine: async (_parent: undefined, { input }: IWrap<FindByIdInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            throw new CustomError(CODE.NotImplemented);
        },
        routines: async (_parent: undefined, { input }: IWrap<RoutinesQueryInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>[]> => {
            throw new CustomError(CODE.NotImplemented);
        },
        routinesCount: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<Count> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        routineAdd: async (_parent: undefined, { input }: IWrap<RoutineInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        },
        routineUpdate: async (_parent: undefined, { input }: IWrap<RoutineInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            return await RoutineModel(prisma).update(input, info);
        },
        routineDeleteOne: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            const success = await RoutineModel(prisma).delete(input);
            return { success };
        },
        /**
         * Reports a routine. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         routineReport: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await RoutineModel(prisma).report(input);
        }
    }
}