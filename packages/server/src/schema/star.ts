import { gql } from 'apollo-server-express';
import { StarSortBy } from '@shared/consts';
import { StarFor, StarInput, StarSearchInput, StarSearchResult, Success } from './types';
import { IWrap } from '../types';
import { Context, rateLimit } from '../middleware';
import { GraphQLResolveInfo } from 'graphql';
import { readManyHelper, StarModel } from '../models';
import { resolveStarTo } from './resolvers';
import { assertRequestFrom } from '../auth/auth';

export const typeDef = gql`
    enum StarSortBy {
        DateUpdatedAsc
        DateUpdatedDesc
    }
    
    enum StarFor {
        Comment
        Organization
        Project
        Routine
        Standard
        Tag
        User
    }   

    union StarTo = Comment | Organization | Project | Routine | Standard | Tag | User

    input StarInput {
        isStar: Boolean!
        starFor: StarFor!
        forId: ID!
    }
    type Star {
        id: ID!
        from: User!
        to: StarTo!
    }
    input StarSearchInput {
        after: String
        excludeTags: Boolean
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

export const resolvers = {
    StarSortBy: StarSortBy,
    StarFor: StarFor,
    StarTo: {
        __resolveType(obj: any) { return resolveStarTo(obj) },
    },
    Query: {
        stars: async (_parent: undefined, { input }: IWrap<StarSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<StarSearchResult> => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 2000, req });
            return readManyHelper({ info, input, model: StarModel, prisma, req, additionalQueries: { userId: userData.id } });
        },
    },
    Mutation: {
        /**
         * Adds or removes a star to an object. A user can only star an object once.
         * @returns 
         */
        star: async (_parent: undefined, { input }: IWrap<StarInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Success> => {
            const userData = assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 1000, req });
            const success = await StarModel.mutate(prisma).star(userData.id, input);
            return { success };
        },
    }
}