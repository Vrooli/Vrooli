import { gql } from 'graphql-tag';

export const submitOrderMutation = gql`
    mutation submitOrder(
        $id: ID!
    ) {
    submitOrder(
        id: $id
    )
}
`