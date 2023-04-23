import gql from "graphql-tag";

export const chatParticipantFindMany = gql`
query chatParticipants($input: ChatParticipantSearchInput!) {
  chatParticipants(input: $input) {
    edges {
        cursor
        node {
            user {
                id
                name
                handle
            }
            id
            created_at
            updated_at
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

