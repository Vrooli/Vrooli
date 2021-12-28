import { gql } from 'graphql-tag';
import { resourceFields } from 'graphql/fragment';

export const resourcesQuery = gql`
    ${resourceFields}
    query resources($input: ResourcesQueryInput!) {
        resources(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...resourceFields
                }
            }
        }
    }
`