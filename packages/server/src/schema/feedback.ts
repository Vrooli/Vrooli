import { gql } from 'apollo-server-express';
import { feedbackNotifyAdmin } from '../worker/email/queue';

export const typeDef = gql`
    input FeedbackInput {
        id: ID
        text: String!
        customerId: ID
    }

    extend type Mutation {
        addFeedback(input: FeedbackInput!): Boolean!
    }
`

export const resolvers = {
    Mutation: {
        addFeedback: async (_parent: undefined, args: any, context: any, info: any) => {
            let from;
            if (args.input.customerId) {
                from = await context.prisma.customer.findUnique({ where: { id: args.input.customerId } });
            }
            feedbackNotifyAdmin(args.input.text, from?.username);
            return true;
        },
    }
}