import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import pkg from '@prisma/client';
const { TaskStatus } = pkg;

export const typeDef = gql`
    enum TaskStatus {
        UNKNOWN
        FAILED
        ACTIVE
        INACTIVE
        COMPLETED
    }

    type Task {
        id: ID!
        taskId: Int!
        name: String!
        status: TaskStatus!
        description: String
        result: String
        resultCode: Int
    }

    extend type Query {
        tasks(ids: [ID!], status: TaskStatus): [Task!]!
    }
`

export const resolvers = {
    TaskStatus: TaskStatus,
    Query: {
        tasks: async (_parent, args, context, _info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma.task.findMany({
                where: { status: args.status }
            });
        }
    }
}