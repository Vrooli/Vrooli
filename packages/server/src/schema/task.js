import { gql } from 'apollo-server-express';
import { TABLES } from '../db';
import { CODE, TASK_STATUS } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

const _model = TABLES.Task;

export const typeDef = gql`
    enum TaskStatus {
        Unknown
        Failed
        Active
        Completed
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
    TaskStatus: TASK_STATUS,
    Query: {
        tasks: async (_, args, context) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].findMany({
                where: { status: args.status }
            });
        }
    }
}