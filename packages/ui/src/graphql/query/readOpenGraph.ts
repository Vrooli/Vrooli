import { gql } from 'graphql-tag';

export const readOpenGraphQuery = gql`
    query readOpenGraph($input: ReadOpenGraphInput!) {
        readOpenGraph(input: $input) {
            site
            title
            description
            imageUrl
        }
    }
`