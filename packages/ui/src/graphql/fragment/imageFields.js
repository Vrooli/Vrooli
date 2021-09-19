import { gql } from 'graphql-tag';

export const imageFields = gql`
    fragment imageFields on Image {
        hash
        alt
        description
        files {
            src
            width
            height
        }
    }
`