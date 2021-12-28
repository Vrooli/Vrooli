import { gql } from 'graphql-tag';

export const readOpenGraphQuery = gql`
    query readOpenGraph($url: String!) {
        readOpenGraph(url: $url) {
            site
            title
            description
            imageUrl
        }
    }
`