import { gql } from 'graphql-tag';

export const resourcesCountQuery = gql`
    query resourcesCount($input: ResourceCountInput!) {
        resourcesCount(input: $input)
    }
`