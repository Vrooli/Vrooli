import { gql } from 'graphql-tag';
import { listUserFields } from 'graphql/fragment';

export const usersQuery = gql`
    ${listUserFields}
    query users($input: UserSearchInput!) {
        users(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listUserFields
                }
            }
        }
    }
`