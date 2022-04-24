import { gql } from 'graphql-tag';
import { listStandardFields } from 'graphql/fragment';

export const standardsQuery = gql`
    ${listStandardFields}
    query standards($input: StandardSearchInput!) {
        standards(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listStandardFields
                }
            }
        }
    }
`