import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

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

    extend type Query {
        routine(id: ID!): Routine
        routines(first: Int, skip: Int): [Routine!]!
        routinesCount: Int!
    }

    extend type Mutation {
        addRoutine(input: RoutineInput!): Routine!
        updateRoutine(input: RoutineInput!): Routine!
        deleteRoutine(id: ID!): Boolean!
        reportRoutine(id: ID!): Boolean!
    }
`

export const resolvers = {
    Query: {
        routine: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        routines: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        routinesCount: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
    },
    Mutation: {
        addRoutine: async (_parent: undefined, args: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            return new CustomError(CODE.NotImplemented);
        },
        updateRoutine: async (_parent: undefined, args: any, context: any, info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            return new CustomError(CODE.NotImplemented);
        },
        deleteRoutine: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            return new CustomError(CODE.NotImplemented);
        },
        /**
         * Reports a routine. After enough reports, it will be deleted.
         * Related objects will not be deleted.
         * @returns True if report was successfully recorded
         */
         reportRoutine: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Must be logged in
            if (!context.req.isLoggedIn) return new CustomError(CODE.Unauthorized);
            throw new CustomError(CODE.NotImplemented);
        }
    }
}