import { gql } from 'graphql-tag';
import { customerSessionFields, orderFields, orderItemFields } from 'graphql/fragment';

export const resetPasswordMutation = gql`
    ${customerSessionFields}
    ${orderFields}
    ${orderItemFields}
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
        cart {
            ...orderFields
            items {
                ...orderItemFields
            }
        }
    }
}
`