import { gql } from 'graphql-tag';
import { skuFields } from './skuFields';
import { productFields } from './productFields';
import { discountFields } from './discountFields';

export const orderItemFields = gql`
    ${skuFields}
    ${productFields}
    ${discountFields}
    fragment orderItemFields on OrderItem {
        id
        quantity
        sku {
            ...skuFields
            product {
                ...productFields
            }
            discounts {
                discount {
                    ...discountFields
                }
            }
        }
    }
`