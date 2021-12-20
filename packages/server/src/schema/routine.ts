import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { IWrap, RecursivePartial } from '../types';
import { DeleteOneInput, FindByIdInput, ReportInput, Routine, RoutineInput, RoutinesQueryInput } from './types';
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
        externalLink: String
        isAutomatable: Boolean
        inputs: [RoutineInputItemInput!]
        outputs: [RoutineOutputItemInput!]

    }

    type Routine {
        id: ID!
    }

    input RoutineInputItemInput {
        id: ID
    }

    type RoutineInputItem {
        id: ID!
    }

    input RoutineOutputItemInput {
        id: ID
    }

    type RoutineOutputItem {
        id: ID!
    }

    input RoutinesQueryInput {
        first: Int
        skip: Int
    }

    extend type Query {
        routine(input: FindByIdInput!): Routine
        routines(input: RoutinesQueryInput!): [Routine!]!
        routinesCount: Int!
    }

    extend type Mutation {
        addRoutine(input: RoutineInput!): Routine!
        updateRoutine(input: RoutineInput!): Routine!
        deleteRoutine(input: DeleteOneInput!): Boolean!
        reportRoutine(input: ReportInput!): Boolean!
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
        routinesCount: async (_parent: undefined, _args: undefined, context: Context, info: GraphQLResolveInfo): Promise<number> => {
            throw new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        addRoutine: async (_parent: undefined, { input }: IWrap<RoutineInput>, context: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            // Must be logged in
            if (!context.req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        },
        updateRoutine: async (_parent: undefined, { input }: IWrap<RoutineInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<RecursivePartial<Routine>> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            return await RoutineModel(prisma).update(input, info);
        },
        deleteRoutine: async (_parent: undefined, { input }: IWrap<DeleteOneInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            // TODO add extra restrictions
            return await RoutineModel(prisma).delete(input);
        },
        /**
         * Reports a routine. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportRoutine: async (_parent: undefined, { input }: IWrap<ReportInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<boolean> => {
            // Must be logged in
            if (!req.isLoggedIn) throw new CustomError(CODE.Unauthorized);
            return await RoutineModel(prisma).report(input);
        }
    }
}