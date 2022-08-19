import { gql } from 'graphql-tag';
import { listStarFields } from 'graphql/fragment';

export const starsQuery = gql`
    ${listStarFields}
    query stars($input: StarSearchInput!) {
        stars(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listStarFields
                }
            }
        }
    }
`