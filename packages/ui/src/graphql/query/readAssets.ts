import { gql } from 'graphql-tag';

export const readAssetsQuery = gql`
    query readAssets($input: ReadAssetsInput!) {
        readAssets(input: $input)
    }
`