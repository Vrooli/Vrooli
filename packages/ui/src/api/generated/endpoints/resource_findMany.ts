import gql from 'graphql-tag';

export const resourceFindMany = gql`
query resources($input: ResourceSearchInput!) {
  resources(input: $input) {
    edges {
        cursor
        node {
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
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

