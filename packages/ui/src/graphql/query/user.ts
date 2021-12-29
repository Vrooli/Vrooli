import { gql } from 'graphql-tag';
import { userFields } from 'graphql/fragment';

export const userQuery = gql`
    ${userFields}
    query user($input: FindByIdInput!) {
        user(input: $input) {
            ...userFields
        }
    }
`