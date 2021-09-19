import { gql } from 'graphql-tag';
import { orderFields, orderItemFields, customerContactFields } from 'graphql/fragment';

export const ordersQuery = gql`
    ${orderFields}
    ${orderItemFields}
    ${customerContactFields}
    query Orders(
        $ids: [ID!]
        $status: OrderStatus
        $searchString: String
    ) {
        orders(
            ids: $ids
            status: $status
            searchString: $searchString
        ) {
            ...orderFields
            items {
                ...orderItemFields
            }
            customer {
                ...customerContactFields
            }
        }
    }
`