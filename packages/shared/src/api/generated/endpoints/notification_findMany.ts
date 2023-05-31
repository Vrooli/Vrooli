import gql from "graphql-tag";

export const notificationFindMany = gql`
query notifications($input: NotificationSearchInput!) {
  notifications(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            category
            isRead
            title
            description
            link
            imgLink
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

