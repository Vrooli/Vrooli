import { feedbackCreate } from '@local/shared';
import { gql } from 'apollo-server-express';
import { IWrap } from 'types';
import { feedbackNotifyAdmin } from '../worker/email/queue';
import { FeedbackInput, Success } from './types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';

export const typeDef = gql`
    input FeedbackInput {
        text: String!
        userId: ID
    }

    extend type Mutation {
        feedbackCreate(input: FeedbackInput!): Success!
    }
`

export const resolvers = {
    Mutation: {
        feedbackCreate: async (_parent: undefined, { input }: IWrap<FeedbackInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Check for valid arguments
            feedbackCreate.validateSync(input, { abortEarly: false });
            // Find user who sent feedback, if any
            let from = input.userId ? await prisma.user.findUnique({ where: { id: input.userId } }) : null;
            // Send feedback to admin
            feedbackNotifyAdmin(input.text, from?.name ?? 'anonymous');
            return { success: true };
        },
    }
}