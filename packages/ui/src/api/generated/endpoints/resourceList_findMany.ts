import gql from 'graphql-tag';

export const resourceListFindMany = gql`
query resourceLists($input: ResourceListSearchInput!) {
  resourceLists(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            translations {
                id
                language
                description
                name
            }
            resources {
                id
                index
                link
                usedFor
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

