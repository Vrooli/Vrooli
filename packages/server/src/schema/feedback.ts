import { feedbackSchema } from '@local/shared';
import { gql } from 'apollo-server-express';
import { validateArgs } from '../error';
import { feedbackNotifyAdmin } from '../worker/email/queue';

export const typeDef = gql`
    input FeedbackInput {
        text: String!
        userId: ID
    }

    extend type Mutation {
        addFeedback(input: FeedbackInput!): Boolean!
    }
`

export const resolvers = {
    Mutation: {
        addFeedback: async (_parent: undefined, { input }: any, context: any, _info: any) => {
            // Validate arguments with schema
            const validateError = await validateArgs(feedbackSchema, input);
            if (validateError) return validateError;
            // Find user who sent feedback, if any
            let from = input.userId ? await context.prisma.user({ id: input.userId }) : null;
            // Send feedback to admin
            feedbackNotifyAdmin(input.text, from?.username);
            return true;
        },
    }
}