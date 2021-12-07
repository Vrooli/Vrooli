import { gql } from 'graphql-tag';
import { userSessionFields } from 'graphql/fragment';

export const updateUserMutation = gql`
    ${userSessionFields}
    mutation updateUser(
        $input: UserInput!
        $currentPassword: String!
        $newPassword: String
    ) {
    updateUser(
        input: $input
        currentPassword: $currentPassword
        newPassword: $newPassword
    ) {
        ...userSessionFields
    }
}
`