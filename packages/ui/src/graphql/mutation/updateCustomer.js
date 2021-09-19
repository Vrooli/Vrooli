import { gql } from 'graphql-tag';
import { customerSessionFields } from 'graphql/fragment';

export const updateCustomerMutation = gql`
    ${customerSessionFields}
    mutation updateCustomer(
        $input: CustomerInput!
        $currentPassword: String!
        $newPassword: String
    ) {
    updateCustomer(
        input: $input
        currentPassword: $currentPassword
        newPassword: $newPassword
    ) {
        ...customerSessionFields
    }
}
`