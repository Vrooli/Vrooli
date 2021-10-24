import { gql } from 'graphql-tag';

export const readAssetsQuery = gql`
    query readAssets(
        $files: [String!]!
    ) {
        readAssets(
        files: $files
    )
  }
`