import { gql } from 'graphql-tag';

export const deleteProductsMutation = gql`
    mutation deleteProducts(
        $ids: [ID!]!
    ) {
    deleteProducts(
        ids: $ids
    ) {
        count
    }
}
`