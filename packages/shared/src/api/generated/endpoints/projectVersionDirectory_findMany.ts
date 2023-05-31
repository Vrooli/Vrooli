import gql from "graphql-tag";

export const projectVersionDirectoryFindMany = gql`
query projectVersionDirectories($input: ProjectVersionDirectorySearchInput!) {
  projectVersionDirectories(input: $input) {
    edges {
        cursor
        node {
            translations {
                id
                language
                description
                name
            }
            id
            created_at
            updated_at
            childOrder
            isRoot
            projectVersion {
                id
                complexity
                isLatest
                isPrivate
                versionIndex
                versionLabel
                root {
                    id
                    isPrivate
                }
                translations {
                    id
                    language
                    description
                    name
                }
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

