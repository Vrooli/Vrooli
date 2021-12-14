import { gql } from 'apollo-server-express';
import { feedbackNotifyAdmin } from '../worker/email/queue';

export const typeDef = gql`
    input FeedbackInput {
        id: ID
        text: String!
        userId: ID
    }

    extend type Mutation {
        addFeedback(input: FeedbackInput!): Boolean!
    }
`

export const resolvers = {
    Mutation: {
        addFeedback: async (_parent: undefined, args: any, context: any, _info: any) => {
            let from;
            if (args.input.userId) {
                from = await context.prisma.user.findUnique({ where: { id: args.input.userId } });
            }
            feedbackNotifyAdmin(args.input.text, from?.username);
            return true;
        },
    }
}