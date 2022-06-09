import { gql } from 'apollo-server-express';
import { ViewSortBy } from './types';

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
`

export const resolvers = {
    ViewSortBy: ViewSortBy,
}