import { feedbackCreate } from '@shared/validation';
import { gql } from 'apollo-server-express';
import { IWrap } from '../types';
import { FeedbackInput, Success } from './types';
import { Context, rateLimit } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';
import { feedbackNotifyAdmin } from '../notify';

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
        feedbackCreate: async (_parent: undefined, { input }: IWrap<FeedbackInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Success> => {
            await rateLimit({ info, maxUser: 250, req });
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