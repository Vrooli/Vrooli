import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

export const typeDef = gql`
    input RoutineInput {
        id: ID
    }

    type Routine {
        id: ID!
    }

    extend type Mutation {
        addRoutine(input: RoutineInput!): Routine!
        updateRoutine(input: RoutineInput!): Routine!
        deleteRoutine(id: ID!): Boolean!
    }
`

export const resolvers = {
    Mutation: {
        addRoutine: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        updateRoutine: async (_parent: undefined, args: any, context: any, info: any) => {
            return new CustomError(CODE.NotImplemented);
        },
        deleteRoutine: async (_parent: undefined, args: any, context: any, _info: any) => {
            return new CustomError(CODE.NotImplemented);
        }
    }
}