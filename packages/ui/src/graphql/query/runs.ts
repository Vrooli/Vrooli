import { gql } from 'graphql-tag';
import { listRunFields } from 'graphql/fragment';

export const runsQuery = gql`
    ${listRunFields}
    query runs($input: RunSearchInput!) {
        runs(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listRunFields
                }
            }
        }
    }
`