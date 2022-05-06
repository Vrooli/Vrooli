import { gql } from 'apollo-server-express';
import { ViewSortBy } from './types';

export const typeDef = gql`
    enum ViewSortBy {
        LastViewedAsc
        LastViewedDesc
    }
  
    type View {
        from: User!
        lastViewed: Date!
        title: String!
        to: ProjectOrOrganizationOrRoutineOrStandardOrUser!
    }
 
`

export const resolvers = {
    ViewSortBy: ViewSortBy,
}