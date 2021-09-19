import { gql } from 'graphql-tag';
import { imageFields } from 'graphql/fragment/imageFields';

export const imagesByLabelQuery = gql`
    ${imageFields}
    query ImagesByLabel(
        $label: String!
    ) {
        imagesByLabel(
            label: $label
        ) {
            ...imageFields
        }
    }
`