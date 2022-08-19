import { CODE } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { GraphQLResolveInfo } from 'graphql';
import { Context } from '../context';
import { CustomError } from '../error';
import { genErrorCode } from '../logger';
import { readManyHelper, ViewModel } from '../models';
import { rateLimit } from '../rateLimit';
import { IWrap } from '../types';
import { ViewSearchInput, ViewSearchResult, ViewSortBy } from './types';

export const typeDef = gql`
    enum ViewSortBy {
        LastViewedAsc
        LastViewedDesc
    }
  
    type View {
        id: ID!
        from: User!
        lastViewed: Date!
        title: String!
        to: ProjectOrOrganizationOrRoutineOrStandardOrUser!
    }

    input ViewSearchInput {
        after: String
        lastViewedTimeFrame: TimeFrame
        ids: [ID!]
        searchString: String
        sortBy: ViewSortBy
        take: Int
    }

    type ViewSearchResult {
        pageInfo: PageInfo!
        edges: [ViewEdge!]!
    }

    type ViewEdge {
        cursor: String!
        node: View!
    }

    extend type Query {
        views(input: ViewSearchInput!): ViewSearchResult!
    }
`

export const resolvers = {
    ViewSortBy: ViewSortBy,
    Query: {
        views: async (_parent: undefined, { input }: IWrap<ViewSearchInput>, { prisma, req }: Context, info: GraphQLResolveInfo): Promise<ViewSearchResult> => {
            // Only accessible if logged in and not using an API key
            if (!req.userId || req.apiToken) 
                throw new CustomError(CODE.Unauthorized, 'Must be logged in to query views', { code: genErrorCode('0253') });
            await rateLimit({ info, max: 2000, req });
            return readManyHelper({ info, input, model: ViewModel, prisma, userId: req.userId, additionalQueries: { userId: req.userId } });
        },
    },
}