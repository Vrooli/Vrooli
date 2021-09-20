import { gql } from 'graphql-tag';
import { customerSessionFields } from 'graphql/fragment';

export const resetPasswordMutation = gql`
    ${customerSessionFields}
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
        ...customerSessionFields
    }
}
`