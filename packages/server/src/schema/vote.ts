import { gql } from 'apollo-server-express';
import { CODE, VoteFor } from '@local/shared';
import { CustomError } from '../error';
import { VoteInput, Success } from './types';
import { IWrap } from 'types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { VoteModel } from '../models';

export const typeDef = gql`
    enum VoteFor {
        Comment
        Project
        Routine
        Standard
        Tag
    }   

    input VoteInput {
        isUpvote: Boolean!
        voteFor: VoteFor!
        forId: ID!
    }

    input VoteRemoveInput {
        voteFor: VoteFor!
        forId: ID!
    }

    extend type Mutation {
        voteAdd(input: VoteInput!): Success!
        voteRemove(input: VoteRemoveInput!): Success!
    }
`

export const resolvers = {
    VoteFor: VoteFor,
    Mutation: {
        /**
         * Adds a vote to an object. A user can only cast one vote per object. So if a user re-votes, 
         * their previous vote is overruled. A user may vote on their own project/routine/etc.
         * @returns 
         */
        voteAdd: async (_parent: undefined, { input }: IWrap<VoteInput>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            const success = await VoteModel(prisma).castVote(req.userId, input);
            return { success };
        },
        /**
         * Removes a vote from an object
         * @returns 
         */
        voteRemove: async (_parent: undefined, { input }: IWrap<any>, { prisma, req }: Context, _info: GraphQLResolveInfo): Promise<Success> => {
            // Must be logged in with an account
            if (!req.isLoggedIn || !req.userId) throw new CustomError(CODE.Unauthorized);
            const success = await VoteModel(prisma).removeVote(req.userId, input);
            return { success };
        },
    }
}