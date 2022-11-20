import { ViewSortBy } from '@shared/consts';
import { gql } from 'apollo-server-express';
import { GraphQLResolveInfo } from 'graphql';
import { assertRequestFrom } from '../auth/auth';
import { Context, rateLimit } from '../middleware';
import { CustomError } from '../events/error';
import { getUser, readManyHelper, ViewModel } from '../models';
import { IWrap } from '../types';
import { ViewSearchInput, ViewSearchResult } from './types';

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
            assertRequestFrom(req, { isUser: true });
            await rateLimit({ info, maxUser: 2000, req });
            const userId = getUser(req)?.id;
            if (!userId) throw new CustomError('Unauthorized', 'Must be logged in to view views.', { trace: '0275' });
            return readManyHelper({ info, input, model: ViewModel, prisma, req, additionalQueries: { userId } });
        },
    },
}