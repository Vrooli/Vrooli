import { gql } from 'graphql-tag';
import { userSessionFields } from 'graphql/fragment';

export const updateUserMutation = gql`
    ${userSessionFields}
    mutation updateUser($input: UpdateUserInput!) {
    updateUser(input: $input) {
        ...userSessionFields
    }
}
`