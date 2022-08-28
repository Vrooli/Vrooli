import { gql } from 'graphql-tag';
import { listViewFields } from 'graphql/fragment';

export const viewsQuery = gql`
    ${listViewFields}
    query views($input: ViewSearchInput!) {
        views(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listViewFields
                }
            }
        }
    }
`