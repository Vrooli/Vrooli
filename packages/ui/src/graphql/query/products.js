import { gql } from 'graphql-tag';
import { productFields, skuFields } from 'graphql/fragment';

export const productsQuery = gql`
    ${productFields}
    ${skuFields}
    query ProductsQuery(
        $ids: [ID!]
        $sortBy: SkuSortBy
        $searchString: String
    ) {
        products(
            ids: $ids
            sortBy: $sortBy
            searchString: $searchString
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