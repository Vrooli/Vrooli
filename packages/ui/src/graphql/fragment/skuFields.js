import { gql } from 'graphql-tag';

export const skuFields = gql`
    fragment skuFields on Sku {
        id
        sku
        isDiscountable
        size
        note
        availability
        price
        status
    }
`