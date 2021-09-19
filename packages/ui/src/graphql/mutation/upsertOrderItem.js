import { gql } from 'graphql-tag';
import { orderItemFields } from 'graphql/fragment';

export const upsertOrderItemMutation = gql`
    ${orderItemFields}
    mutation upsertOrderItem(
        $quantity: Int!
        $orderId: ID
        $skuId: ID!
    ) {
    upsertOrderItem(
        quantity: $quantity
        orderId: $orderId
        skuId: $skuId
    ) {
        ...orderItemFields
    }
}
`