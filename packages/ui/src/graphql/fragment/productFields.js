import { gql } from 'graphql-tag';
import { imageFields } from './imageFields';

export const productFields = gql`
    ${imageFields}
    fragment productFields on Product {
        id
        name
        traits {
            name
            value
        }
        images {
            index
            isDisplay
            image {
                ...imageFields
            }
        }
    }
`