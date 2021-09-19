import { gql } from 'graphql-tag';

export const changeCustomerStatusMutation = gql`
    mutation changeCustomerStatus(
        $id: ID!
        $status: AccountStatus!
    ) {
    changeCustomerStatus(
        id: $id
        status: $status
    )
}
`