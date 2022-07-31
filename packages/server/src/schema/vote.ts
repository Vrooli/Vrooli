import { gql } from 'apollo-server-express';
import { CODE } from '@local/shared';
import { CustomError } from '../error';
import { VoteInput, Success, VoteFor } from './types';
import { IWrap } from 'types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { VoteModel } from '../models';
import { rateLimit } from '../rateLimit';
import { genErrorCode } from '../logger';
import { resolveVoteTo } from './resolvers';

export const typeDef = gql`
    enum VoteFor {
        Comment
        Project
        Routine
        Standard
        Tag
    }   

    union VoteTo = Comment | Project | Routine | Standard | Tag

    input VoteInput {
        isUpvote: Boolean
        voteFor: VoteFor!
        forId: ID!
    }
    type Vote {
        isUpvote: Boolean
        from: User!
        to: VoteTo!
    }

    extend type Mutation {
        vote(input: VoteInput!): Success!
    }
`

export const resolvers = {
    VoteFor: VoteFor,
    VoteTo: {
        __resolveType(obj: any) { return resolveVoteTo(obj) },
    },
    Mutation: {
        /**
         * Adds or removes a vote to an object. A user can only cast one vote per object. So if a user re-votes, 
         * their previous vote is overruled. A user may vote on their own project/routine/etc.
         * @returns 
         */
        vote: async (_parent: undefined, { input }: IWrap<VoteInput>, context: Context, info: GraphQLResolveInfo): Promise<Success> => {
            if (!context.req.userId) 
                throw new CustomError(CODE.Unauthorized, 'Must be logged in to vote', { code: genErrorCode('0165') });
            await rateLimit({ context, info, max: 1000, byAccountOrKey: true });
            const success = await VoteModel.mutate(context.prisma).vote(context.req.userId, input);
            return { success };
        },
    }
}