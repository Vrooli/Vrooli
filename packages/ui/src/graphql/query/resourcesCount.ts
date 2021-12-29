import { gql } from 'graphql-tag';

export const resourcesCountQuery = gql`
    query resourcesCount {
        resourcesCount
    }
`