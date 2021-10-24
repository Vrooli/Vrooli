import { gql } from 'graphql-tag';
import { customerContactFields } from 'graphql/fragment';

export const addCustomerMutation = gql`
    ${customerContactFields}
    mutation addCustomer(
        $input: CustomerInput!
    ) {
    addCustomer(
        input: $input
    ) {
        ...customerContactFields
        status
        roles {
            role {
                title
            }
        }
    }
}
`