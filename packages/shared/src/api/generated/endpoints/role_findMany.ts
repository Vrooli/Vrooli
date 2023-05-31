import gql from "graphql-tag";

export const roleFindMany = gql`
query roles($input: RoleSearchInput!) {
  roles(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            updated_at
            name
            permissions
            membersCount
            translations {
                id
                language
                description
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

