import { gql } from 'graphql-tag';
import { orderFields, orderItemFields } from 'graphql/fragment';

export const updateOrderMutation = gql`
    ${orderFields}
    ${orderItemFields}
    mutation updateOrder(
        $input: OrderInput!
    ) {
    updateOrder(
        input: $input
    ) {
        ...orderFields
        items {
            ...orderItemFields
        }
    }
}
`