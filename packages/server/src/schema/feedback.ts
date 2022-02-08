import { feedbackAdd } from '@local/shared';
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
        feedbackAdd(input: FeedbackInput!): Success!
    }
`

export const resolvers = {
    Mutation: {
        feedbackAdd: async (_parent: undefined, { input }: IWrap<FeedbackInput>, { prisma }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Check for valid arguments
            feedbackAdd.validateSync(input, { abortEarly: false });
            // Find user who sent feedback, if any
            let from = input.userId ? await prisma.user.findUnique({ where: { id: input.userId } }) : null;
            // Send feedback to admin
            feedbackNotifyAdmin(input.text, from?.username ?? 'anonymous');
            return { success: true };
        },
    }
}