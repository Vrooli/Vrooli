import { gql } from 'apollo-server-express';
import { TABLES } from '../db';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

const _model = TABLES.Feedback;

export const typeDef = gql`
    input FeedbackInput {
        id: ID
        text: String!
        customerId: ID
    }

    type Feedback {
        id: ID!
        text: String!
        customer: Customer
    }

    extend type Query {
        feedbacks: [Feedback!]!
    }

    extend type Mutation {
        addFeedback(input: FeedbackInput!): Feedback!
        deleteFeedbacks(ids: [ID!]!): Count!
    }
`

export const resolvers = {
    Query: {
        feedbacks: async (_, _args, context, info) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].findMany((new PrismaSelect(info).value));
        }
    },
    Mutation: {
        addFeedback: async (_, args, context, info) => {
            return await context.prisma[_model].create((new PrismaSelect(info).value), { data: { ...args.input } })
        },
        deleteFeedbacks: async (_, args, context) => {
            // Must be admin, or deleting your own
            const specified = await context.prisma[_model].findMany({ where: { id: { in: args.ids } } });
            if (!specified) return new CustomError(CODE.ErrorUnknown);
            const customer_ids = [...new Set(specified.map(s => s.customerId))];
            if (!context.req.isAdmin && (customer_ids.length > 1 || context.req.customerId !== customer_ids[0])) return new CustomError(CODE.Unauthorized);
            return await context.prisma[_model].deleteMany({ where: { id: { in: args.ids } } });
        }
    }
}