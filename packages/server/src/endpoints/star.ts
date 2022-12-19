import { gql } from 'apollo-server-express';
import { StarSortBy } from '@shared/consts';
import { Star, StarFor, StarInput, StarSearchInput, Success } from './types';
import { FindManyResult, GQLEndpoint, UnionResolver } from '../types';
import { rateLimit } from '../middleware';
import { StarModel } from '../models';
import { resolveUnion } from './resolvers';
import { assertRequestFrom } from '../auth/request';
import { readManyHelper } from '../actions';

export const typeDef = gql`
    enum StarSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
    }
    
    enum StarFor {
        Api
        Comment
        Issue
        Note
        Organization
        Post
        Project
        Question
        QuestionAnswer
        Quiz
        Routine
        SmartContract
        Standard
        Tag
        User
    }   

    union StarTo = Api | Comment | Issue | Note | Organization | Project | Post | Question | QuestionAnswer | Quiz | Routine | SmartContract | Standard | Tag | User

    input StarInput {
        isStar: Boolean!
        starFor: StarFor!
        forConnect: ID!
    }
    type Star {
        id: ID!
        by: User!
        to: StarTo!
    }

    input StarSearchInput {
        after: String
        excludeLinkedToTag: Boolean
        ids: [ID!]
        searchString: String
        sortBy: StarSortBy
        take: Int
    }
    type StarSearchResult {
        pageInfo: PageInfo!
        edges: [StarEdge!]!
    }
    type StarEdge {
        cursor: String!
        node: Star!
    }

    extend type Query {
        stars(input: StarSearchInput!): StarSearchResult!
    }

    extend type Mutation {
        star(input: StarInput!): Success!
    }
`

const objectType = 'Star';
export const resolvers: {
    StarSortBy: typeof StarSortBy;
    StarFor: typeof StarFor;
    StarTo: UnionResolver;
    Query: {
        stars: GQLEndpoint<StarSearchInput, FindManyResult<Star>>;
    },
    Mutation: {
        star: GQLEndpoint<StarInput, Success>;
    }
} = {
    StarSortBy,
    StarFor,
    StarTo: { __resolveType(obj: any) { return resolveUnion(obj) } },
    Query: {
        stars: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 2000, req });
            return readManyHelper({ info, input, objectType, prisma, req, additionalQueries: { userId: userData.id } });
        },
    },
    Mutation: {
        /**
         * Adds or removes a star to an object. A user can only star an object once.
         * @returns 
         */
        star: async (_, { input }, { prisma, req }, info) => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            const success = await StarModel.star(prisma, userData, input);
            return { success };
        },
    }
}