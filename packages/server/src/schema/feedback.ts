import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { PrismaSelect } from '@paljs/plugins';

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
        feedbacks: async (_parent: undefined, _args: any, context: any, info: any) => {
            // Must be admin
            if (!context.req.isAdmin) return new CustomError(CODE.Unauthorized);
            return await context.prisma.feedback.findMany((new PrismaSelect(info).value));
        }
    },
    Mutation: {
        addFeedback: async (_parent: undefined, args: any, context: any, info: any) => {
            return await context.prisma.feedback.create((new PrismaSelect(info).value), { data: { ...args.input } })
        },
        deleteFeedbacks: async (_parent: undefined, args: any, context: any, _info: any) => {
            // Must be admin, or deleting your own
            const specified = await context.prisma.feedback.findMany({ where: { id: { in: args.ids } } });
            if (!specified) return new CustomError(CODE.ErrorUnknown);
            const customer_ids = [...new Set(specified.map((s: any) => s.customerId))];
            if (!context.req.isAdmin && (customer_ids.length > 1 || context.req.customerId !== customer_ids[0])) return new CustomError(CODE.Unauthorized);
            return await context.prisma.feedback.deleteMany({ where: { id: { in: args.ids } } });
        }
    }
}