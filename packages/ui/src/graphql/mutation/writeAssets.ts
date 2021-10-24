import { gql } from 'graphql-tag';

export const writeAssetsMutation = gql`
    mutation writeAssets(
        $files: [Upload!]!
    ) {
        writeAssets(
            files: $files
        )
    }
`