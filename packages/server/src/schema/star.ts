import { gql } from 'apollo-server-express';
import { CODE, StarSortBy } from '@shared/consts';
import { CustomError } from '../error';
import { StarFor, StarInput, StarSearchInput, StarSearchResult, Success } from './types';
import { IWrap } from '../types';
import { Context } from '../context';
import { GraphQLResolveInfo } from 'graphql';
import { readManyHelper, StarModel } from '../models';
import { rateLimit } from '../rateLimit';
import { genErrorCode } from '../logger';
import { resolveStarTo } from './resolvers';

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
            // Only accessible if logged in and not using an API key
            if (!req.userId || req.apiToken) 
                throw new CustomError(CODE.Unauthorized, 'Must be logged in to query stars', { code: genErrorCode('0252') });
            await rateLimit({ info, max: 2000, req });
            return readManyHelper({ info, input, model: StarModel, prisma, userId: req.userId, additionalQueries: { userId: req.userId } });
        },
    },
    Mutation: {
        /**
         * Adds or removes a star to an object. A user can only star an object once.
         * @returns 
         */
        star: async (_parent: undefined, { input }: IWrap<StarInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<Success> => {
            // Only accessible if logged in and not using an API key
            if (!req.userId || req.apiToken)
                throw new CustomError(CODE.Unauthorized, 'Must be logged in to star', { code: genErrorCode('0157') });
            await rateLimit({ info, max: 1000, byAccountOrKey: true, req });
            const success = await StarModel.mutate(prisma).star(req.userId, input);
            return { success };
        },
    }
}