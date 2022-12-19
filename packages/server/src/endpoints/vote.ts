import { gql } from 'apollo-server-express';
import { VoteInput, Success, VoteFor, VoteSortBy, VoteSearchInput, Vote } from './types';
import { FindManyResult, GQLEndpoint, UnionResolver } from '../types';
import { rateLimit } from '../middleware';
import { VoteModel } from '../models';
import { resolveUnion } from './resolvers';
import { assertRequestFrom } from '../auth/request';
import { readManyHelper } from '../actions';

export const typeDef = gql`
    enum VoteSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
    }

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
        by: User!
        to: VoteTo!
    }

    input VoteSearchInput {
        after: String
        excludeLinkedToTag: Boolean
        ids: [ID!]
        searchString: String
        sortBy: VoteSortBy
        take: Int
    }

    type VoteSearchResult {
        pageInfo: PageInfo!
        edges: [VoteEdge!]!
    }

    type VoteEdge {
        cursor: String!
        node: Vote!
    }

    extend type Query {
        votes(input: VoteSearchInput!): VoteSearchResult!
    }

    extend type Mutation {
        vote(input: VoteInput!): Success!
    }
`

const objectType = 'Vote';
export const resolvers: {
    VoteSortBy: typeof VoteSortBy,
    VoteFor: typeof VoteFor,
    VoteTo: UnionResolver,
    Query: {
        votes: GQLEndpoint<VoteSearchInput, FindManyResult<Vote>>;
    },
    Mutation: {
        vote: GQLEndpoint<VoteInput, Success>;
    }
} = {
    VoteSortBy,
    VoteFor,
    VoteTo: { __resolveType(obj: any) { return resolveUnion(obj) } },
    Query: {
        votes: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { userId: userData.id } });
        },
    },
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