import { gql } from 'graphql-tag';
import { productFields, skuFields } from 'graphql/fragment';

export const updateProductMutation = gql`
    ${productFields}
    ${skuFields}
    mutation updateProduct(
        $input: ProductInput!
    ) {
    updateProduct(
        input: $input
    ) {
        ...productFields
        skus {
            ...skuFields
            discounts {
                discount {
                    id
                    discount
                    title
                    comment
                    terms
                }
            }
        }
    }
}
`