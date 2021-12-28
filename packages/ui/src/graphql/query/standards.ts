import { gql } from 'graphql-tag';
import { standardFields } from 'graphql/fragment';

export const standardsQuery = gql`
    ${standardFields}
    query standards($input: StandardsQueryInput!) {
        standards(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...standardFields
                }
            }
        }
    }
`