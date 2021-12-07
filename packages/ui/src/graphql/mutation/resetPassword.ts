import { gql } from 'graphql-tag';
import { userSessionFields } from 'graphql/fragment';

export const resetPasswordMutation = gql`
    ${userSessionFields}
    mutation resetPassword(
        $id: ID!
        $code: String!
        $newPassword: String!
    ) {
    resetPassword(
        id: $id
        code: $code
        newPassword: $newPassword
    ) {
        ...userSessionFields
    }
}
`