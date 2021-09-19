import { gql } from 'graphql-tag';

export const discountFields = gql`
    fragment discountFields on Discount {
        id
        discount
        title
        comment
        terms
    }
`