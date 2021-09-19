import { gql } from 'graphql-tag';

export const updateImagesMutation = gql`
    mutation updateImages(
        $data: [ImageUpdate!]!
        $deleting: [String!]
        $label: String
    ) {
    updateImages(
        data: $data
        deleting: $deleting
        label: $label
    )
}
`