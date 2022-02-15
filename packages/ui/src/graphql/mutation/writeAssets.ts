import { gql } from 'graphql-tag';

export const writeAssetsMutation = gql`
    mutation writeAssets($input: WriteAssetsInput!) {
        writeAssets(input: $input)
    }
`