import { gql } from 'apollo-server-express';
import { VoteInput, Success, VoteFor } from './types';
import { GQLEndpoint, UnionResolver } from '../types';
import { rateLimit } from '../middleware';
import { VoteModel } from '../models';
import { resolveUnion } from './resolvers';
import { assertRequestFrom } from '../auth/request';

export const typeDef = gql`
    enum VoteFor {
        Api
        Comment
        Issue
        Note
        Post
        Project
        Question
        QuestionAnswer
        Quiz
        Routine
        SmartContract
        Standard
    }   

    union VoteTo = Api | Comment | Issue | Note | Post | Project | Question | QuestionAnswer | Quiz | Routine | SmartContract | Standard

    input VoteInput {
        isUpvote: Boolean
        voteFor: VoteFor!
        forConnect: ID!
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

export const resolvers: {
    VoteFor: typeof VoteFor;
    VoteTo: UnionResolver;
    Mutation: {
        vote: GQLEndpoint<VoteInput, Success>;
    }
} = {
    VoteFor,
    VoteTo: { __resolveType(obj: any) { return resolveUnion(obj) } },
    Mutation: {
        /**
         * Adds or removes a vote to an object. A user can only cast one vote per object. So if a user re-votes, 
         * their previous vote is overruled. A user may vote on their own project/routine/etc.
         * @returns 
         */
        vote: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            const success = await VoteModel.vote(prisma, userData, input);
            return { success };
        },
    }
}