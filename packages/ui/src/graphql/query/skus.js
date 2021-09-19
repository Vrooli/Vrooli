import { gql } from 'graphql-tag';
import { discountFields, productFields, skuFields } from 'graphql/fragment';

export const skusQuery = gql`
    ${skuFields}
    ${productFields}
    ${discountFields}
    query SkusQuery(
        $ids: [ID!]
        $sortBy: SkuSortBy
        $searchString: String
        $onlyInStock: Boolean
    ) {
        skus(
            ids: $ids
            sortBy: $sortBy
            searchString: $searchString
            onlyInStock: $onlyInStock
        ) {
            ...skuFields
            discounts {
                discount {
                    ...discountFields
                }
            }
            product {
                ...productFields
            }
        }
    }
`