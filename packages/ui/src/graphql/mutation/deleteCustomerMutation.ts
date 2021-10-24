import { gql } from 'graphql-tag';

export const deleteCustomerMutation = gql`
    mutation deleteCustomer(
        $id: ID!
    ) {
    deleteCustomer(
        id: $id
    )
}
`