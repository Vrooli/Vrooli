import { gql } from 'graphql-tag';
import { userFields } from 'graphql/fragment';

export const usersQuery = gql`
    ${userFields}
    query users($input: UserSearchInput!) {
        users(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...userFields
                }
            }
        }
    }
`