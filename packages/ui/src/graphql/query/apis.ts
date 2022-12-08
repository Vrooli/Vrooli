import { gql } from 'graphql-tag';
import { listApiFields } from 'graphql/fragment';

export const apisQuery = gql`
    ${listApiFields}
    query apis($input: ApiSearchInput!) {
        apis(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listApiFields
                }
            }
        }
    }
`